package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store composes the sqlc-generated Queries with transaction support.
//
// IMPORTANT: *Queries is embedded ANONYMOUSLY (no field name before the type).
// This promotes all methods of *Queries directly onto Store:
//
//	store.GetUserByEmail(...)      promoted from (*Queries).GetUserByEmail
//	store.CreateUser(...)          promoted from (*Queries).CreateUser
//	store.DeleteExpiredTokens(...) promoted from (*Queries).DeleteExpiredTokens
//	store.InvalidateTokenFamily(…) promoted from (*Queries).InvalidateTokenFamily
//	... and every other *Queries method
//
// store.Queries still works as a field selector to obtain the *Queries
// pointer when you need to pass it explicitly (e.g. to issuePairTx).
type Store struct {
	*Queries // anonymous embed - ALL Queries methods promoted
	pool     *pgxpool.Pool
}

// NewStore wraps pool in a Store ready for use.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		Queries: New(pool), // New(pool) returns *Queries; assigned to the anonymous field
		pool:    pool,
	}
}

// ExecTx runs fn inside a serializable transaction.
//
// The *Queries passed to fn is bound to the transaction - every query
// executed via q inside fn participates in the same transaction.
// ExecTx automatically rolls back on any error and commits on success.
//
// Usage in auth_service.go:
//
//	err = store.ExecTx(ctx, func(q *db.Queries) error {
//	    if _, err := q.MarkRefreshTokenUsed(ctx, id); err != nil {
//	        return err
//	    }
//	    _, err = q.CreateRefreshToken(ctx, params)
//	    return err
//	})
func (s *Store) ExecTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return fmt.Errorf("store.ExecTx: begin: %w", err)
	}

	// Always attempt rollback. pgx makes Rollback on an already-committed
	// transaction a cheap no-op, so this defer is unconditionally safe.
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(New(tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store.ExecTx: commit: %w", err)
	}
	return nil
}
