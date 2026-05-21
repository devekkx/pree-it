// Domain layer: entities, repository interface, and sentinel errors.
// This package has zero external dependencies  it is the innermost
// layer and nothing here ever imports from handler, service, or repository.

package domain

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors
// Defined here so every layer (service, handler) can use errors.Is()
// without importing a sibling package.

var (
	ErrEmailTaken         = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrUserNotFound       = errors.New("user not found")
)

// Entities

// User represents an authenticated identity in the system.
type User struct {
	ID            string
	Email         string
	PasswordHash  string
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// RefreshToken represents a persisted opaque refresh token (stored hashed).
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

// IsExpired reports whether the refresh token is past its expiry time.
func (t *RefreshToken) IsExpired() bool {
	return time.Now().UTC().After(t.ExpiresAt)
}

// IsValid reports whether the token can be used to issue a new pair.
func (t *RefreshToken) IsValid() bool {
	return !t.Revoked && !t.IsExpired()
}

// Repository interface
// Defined in the domain so the service layer depends only on this interface,
// not on any concrete database package. Implementations live in internal/repository.

// UserRepository defines all persistence operations for User entities.
type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
}

// TokenRepository defines all persistence operations for RefreshToken entities.
type TokenRepository interface {
	Store(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	FindByHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAll(ctx context.Context, userID string) error
}
