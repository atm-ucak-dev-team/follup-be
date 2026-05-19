package infra

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/bomanarakasura/jira-email-automation/internal/domain"
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
//   if postgresPool != nil {
//       infra.ClosePostgresPool(postgresPool)
//   }
func ClosePostgresPool(pool *pgxpool.Pool) {
	if pool != nil {
		log.Println("Closing PostgreSQL connection pool")
		pool.Close()
	}
}