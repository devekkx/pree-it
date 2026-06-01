package db

import (
	"context"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds all tuning parameters for the pgxpool connection pool.
// Call defaults() or use NewPool which calls it automatically.
type PoolConfig struct {
	DSN          string
	MaxConns     int32
	MinConns     int32
	MaxConnLife  time.Duration
	MaxConnIdle  time.Duration
	HealthPeriod time.Duration
}

func (c *PoolConfig) applyDefaults() {
	if c.MaxConns == 0 {
		c.MaxConns = 25
	}
	if c.MinConns == 0 {
		c.MinConns = 5
	}
	if c.MaxConnLife == 0 {
		c.MaxConnLife = 30 * time.Minute
	}
	if c.MaxConnIdle == 0 {
		c.MaxConnIdle = 10 * time.Minute
	}
	if c.HealthPeriod == 0 {
		c.HealthPeriod = 1 * time.Minute
	}
}

// NewPool creates a fully configured *pgxpool.Pool.
// It attaches an OTel tracer so every SQL statement emits a child span,
// pings once to verify connectivity, and closes the pool on ping failure
// so the caller never holds a broken pool.
func NewPool(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("db.NewPool: DSN must not be empty")
	}
	cfg.applyDefaults()

	pcfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db.NewPool: parse DSN: %w", err)
	}

	pcfg.MaxConns = cfg.MaxConns
	pcfg.MinConns = cfg.MinConns
	pcfg.MaxConnLifetime = cfg.MaxConnLife
	pcfg.MaxConnIdleTime = cfg.MaxConnIdle
	pcfg.HealthCheckPeriod = cfg.HealthPeriod

	// Every query becomes a child OTel span visible in Tempo.
	pcfg.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("db.NewPool: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db.NewPool: initial ping failed: %w", err)
	}

	return pool, nil
}
