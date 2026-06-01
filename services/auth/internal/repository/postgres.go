// PostgreSQL implementations of domain.UserRepository and domain.TokenRepository.
// This package is the only place that knows about pgx — no database types
// leak into the domain or service layers.

package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/devekkx/pree-it/auth/internal/domain"
	"github.com/devekkx/pree-it/pkg/config"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres implements both domain.UserRepository and domain.TokenRepository.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres opens a connection pool and verifies connectivity.
// The caller is responsible for calling Close() when done.
func NewPostgres(ctx context.Context, cfg *config.Config) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, cfg.PostgresDSN())
	if err != nil {
		return nil, fmt.Errorf("repository: open pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("repository: ping postgres: %w", err)
	}

	return &Postgres{pool: pool}, nil
}

// Close releases all connections in the pool.
func (p *Postgres) Close() { p.pool.Close() }

// UserRepository

func (p *Postgres) Create(ctx context.Context, email, passwordHash string) (*domain.User, error) {
	var u domain.User
	err := p.pool.QueryRow(ctx,
		`INSERT INTO auth_schema.users (email, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, email, password_hash, email_verified, created_at, updated_at`,
		email, passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository: create user: %w", err)
	}
	return &u, nil
}

func (p *Postgres) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := p.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, email_verified, created_at, updated_at
		 FROM auth_schema.users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find user by email: %w", err)
	}
	return &u, nil
}

func (p *Postgres) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var u domain.User
	err := p.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, email_verified, created_at, updated_at
		 FROM auth_schema.users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find user by id: %w", err)
	}
	return &u, nil
}

// TokenRepository

func (p *Postgres) Store(ctx context.Context, userID, tokenHash string, expiresAt interface{}) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO auth_schema.refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("repository: store refresh token: %w", err)
	}
	return nil
}

func (p *Postgres) FindByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := p.pool.QueryRow(ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked, created_at
		 FROM auth_schema.refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Revoked, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("repository: find refresh token: %w", err)
	}
	return &t, nil
}

func (p *Postgres) Revoke(ctx context.Context, tokenHash string) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE auth_schema.refresh_tokens SET revoked = TRUE WHERE token_hash = $1`,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("repository: revoke token: %w", err)
	}
	return nil
}

func (p *Postgres) RevokeAll(ctx context.Context, userID string) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE auth_schema.refresh_tokens SET revoked = TRUE WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("repository: revoke all tokens: %w", err)
	}
	return nil
}
