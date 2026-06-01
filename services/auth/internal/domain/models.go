// Sentinel errors
// Defined here so every layer (service, handler) can use errors.Is()
// without importing a sibling package.

package domain

import (
	"errors"
	"time"
)

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
