package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx" driver for database/sql
	"github.com/pressly/goose/v3"
)

// RunMigrations applies all pending Goose migrations at service startup.
// It derives a database/sql connection from the pool's config string
// so we don't need a separate DSN - the pool is the source of truth.
// Safe to call on every startup - Goose is idempotent.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	// Derive a stdlib DSN from the pool's resolved config.
	// pgxpool.Config.ConnString() is not available; use ConnConfig directly.
	connStr := pool.Config().ConnConfig.ConnString()

	sqlDB, err := sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("migrate: open stdlib connection: %w", err)
	}
	defer sqlDB.Close()

	// Verify the connection is actually usable before running migrations.
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("migrate: ping: %w", err)
	}

	goose.SetBaseFS(nil)    // use OS filesystem - migrations dir is baked into the image
	goose.SetVerbose(false) // structured logs only; don't print to stdout

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("migrate: set dialect: %w", err)
	}

	if err := goose.UpContext(ctx, sqlDB, dir); err != nil {
		return fmt.Errorf("migrate: up: %w", err)
	}

	return nil
}
