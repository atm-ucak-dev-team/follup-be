package infra

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/atm-ucak/follup/internal/domain"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool creates a new PostgreSQL connection pool
//
// Note: PostgreSQL is optional in MVP. If connection fails, returns nil instead of error.
// This allows the application to start without PostgreSQL and enables future persistence
// implementations to inject the pool without restructuring infrastructure bootstrap.
//
// Usage in future repositories:
//   - Inject *pgxpool.Pool into repository constructors
//   - Use pool for database operations when implementing persistence layers
//   - Pool will be automatically closed during graceful shutdown
//
// DSN format: postgres://user:password@localhost:5432/dbname
func NewPostgresPool(cfg *domain.Config) *pgxpool.Pool {
	if cfg == nil || cfg.PostgresDSN == "" {
		log.Println("PostgreSQL DSN not provided, skipping PostgreSQL initialization")
		return nil
	}

	// Create connection pool
	pool, err := pgxpool.New(context.Background(), cfg.PostgresDSN)
	if err != nil {
		log.Printf("Failed to create PostgreSQL pool: %v. PostgreSQL will be unavailable.", err)
		return nil
	}

	// Validate connection with Ping
	if err := pool.Ping(context.Background()); err != nil {
		log.Printf("PostgreSQL ping failed: %v. PostgreSQL will be unavailable.", err)
		pool.Close()
		return nil
	}

	log.Println("PostgreSQL connection pool initialized successfully")
	return pool
}

// ClosePostgresPool closes the PostgreSQL connection pool gracefully
//
// Should be called during application shutdown sequence in main.go:
//
//	if postgresPool != nil {
//	    infra.ClosePostgresPool(postgresPool)
//	}
func ClosePostgresPool(pool *pgxpool.Pool) {
	if pool != nil {
		log.Println("Closing PostgreSQL connection pool")
		pool.Close()
	}
}

// RunMigrations applies pending database migrations using golang-migrate.
//
// DSN format: postgres://user:password@host:port/dbname?sslmode=...
// The function converts the DSN to pgx5:// scheme required by the migrate driver.
func RunMigrations(dsn string) error {
	if dsn == "" {
		return fmt.Errorf("DSN is empty, skipping migrations")
	}

	// Convert postgres:// or postgresql:// to pgx5:// for golang-migrate pgx v5 driver
	migrateDSN := dsn
	migrateDSN = strings.Replace(migrateDSN, "postgresql://", "pgx5://", 1)
	migrateDSN = strings.Replace(migrateDSN, "postgres://", "pgx5://", 1)

	m, err := migrate.New("file://cmd/server/db/migrations", migrateDSN)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}
