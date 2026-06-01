package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store embeds the generated Queries and adds transaction support.
// All application code should interact with *Store, never with the pool directly.
type Store struct {
	*Queries
	pool *pgxpool.Pool
}

// NewStore wraps pool in a Store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		Queries: New(pool),
		pool:    pool,
	}
}

// ExecTx executes fn inside a serializable transaction.
// It automatically rolls back on any error returned by fn or on panic,
// and commits only when fn returns nil.
func (s *Store) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return fmt.Errorf("store.ExecTx: begin: %w", err)
	}

	// Ensure rollback is always attempted if we don't commit.
	defer func() {
		// pgx Rollback on an already-committed tx is a no-op - safe to call unconditionally.
		_ = tx.Rollback(ctx)
	}()

	if err := fn(New(tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store.ExecTx: commit: %w", err)
	}
	return nil
}
