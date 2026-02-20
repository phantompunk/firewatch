package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		slog.Error("failed to create migrations table", "err", err)
		os.Exit(1)
	}

	migrationsDir := envOr("MIGRATIONS_DIR", "migrations")
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		slog.Error("failed to read migrations", "err", err)
		os.Exit(1)
	}
	sort.Strings(files)

	for _, f := range files {
		version := strings.TrimSuffix(filepath.Base(f), ".sql")

		var exists bool
		_ = pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`,
			version,
		).Scan(&exists)
		if exists {
			continue
		}

		sql, err := os.ReadFile(f)
		if err != nil {
			slog.Error("failed to read migration", "file", f, "err", err)
			os.Exit(1)
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			slog.Error("migration failed", "version", version, "err", err)
			os.Exit(1)
		}

		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`, version,
		); err != nil {
			slog.Error("failed to record migration", "version", version, "err", err)
			os.Exit(1)
		}

		fmt.Printf("applied: %s\n", version)
	}

	fmt.Println("migrations complete")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
