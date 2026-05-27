package infra

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunSeed(ctx context.Context, pool *pgxpool.Pool, seedPath string) error {
	if pool == nil {
		return fmt.Errorf("postgres pool is nil, skipping seed")
	}

	data, err := os.ReadFile(seedPath)
	if err != nil {
		return fmt.Errorf("read seed file: %w", err)
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	for _, stmt := range splitSQL(string(data)) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("seed statement failed: %w\nSQL: %s", err, stmt[:min(len(stmt), 200)])
		}
	}

	log.Printf("Seed data loaded from %s", seedPath)
	return nil
}

func splitSQL(sql string) []string {
	raw := strings.Split(sql, ";")
	result := make([]string, 0, len(raw))
	for _, block := range raw {
		lines := strings.Split(block, "\n")
		var cleaned []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "--") {
				continue
			}
			cleaned = append(cleaned, line)
		}
		if len(cleaned) > 0 {
			result = append(result, strings.Join(cleaned, "\n"))
		}
	}
	return result
}
