package service

import (
	"time"

	"github.com/devekkx/pree-it/auth/internal/domain"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	bcryptCost      = 12
)

// TokenPair is returned to the caller after a successful auth operation.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds until the access token expires
}

// AuthService implements authentication business logic.
type AuthService struct {
	users  domain.UserRepository
	tokens domain.TokenRepository
	secret []byte
}
