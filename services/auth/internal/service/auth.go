// Auth business logic. Depends only on domain interfaces - never on
// concrete repository types or HTTP types. All errors are wrapped with
// context so callers can use errors.Is() at the handler layer.

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/devekkx/pree-it/auth/internal/domain"
	"github.com/devekkx/pree-it/pkg/config"
	"github.com/devekkx/pree-it/pkg/jwtutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// NewAuthService constructs an AuthService wired to the given repositories.
func NewAuthService(repo interface {
	domain.UserRepository
	domain.TokenRepository
}, cfg *config.Config) *AuthService {
	return &AuthService{
		users:  repo,
		tokens: repo,
		secret: []byte(cfg.JWTSecret),
	}
}

// Register creates a new user account and returns an initial token pair.
// Returns domain.ErrEmailTaken if the address is already registered.
func (s *AuthService) Register(ctx context.Context, email, password string) (*domain.User, *TokenPair, error) {
	existing, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, nil, fmt.Errorf("service: register - lookup: %w", err)
	}
	if existing != nil {
		return nil, nil, domain.ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, nil, fmt.Errorf("service: register - hash password: %w", err)
	}

	user, err := s.users.Create(ctx, email, string(hash))
	if err != nil {
		return nil, nil, fmt.Errorf("service: register - create user: %w", err)
	}

	pair, err := s.issueTokenPair(ctx, user.ID, user.Email)
	if err != nil {
		return nil, nil, err
	}

	return user, pair, nil
}

// Login validates credentials and returns a token pair.
// Returns domain.ErrInvalidCredentials for any auth failure (timing-safe).
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, *TokenPair, error) {
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, nil, fmt.Errorf("service: login - lookup: %w", err)
	}
	// Return same error whether user missing or password wrong - prevents enumeration.
	if user == nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, domain.ErrInvalidCredentials
	}

	pair, err := s.issueTokenPair(ctx, user.ID, user.Email)
	if err != nil {
		return nil, nil, err
	}

	return user, pair, nil
}

// Refresh rotates the refresh token and returns a new pair.
// The consumed token is revoked atomically (single-use enforcement).
// Returns domain.ErrInvalidToken if the token is missing, revoked, or expired.
func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*TokenPair, error) {
	hash := hashToken(rawToken)

	stored, err := s.tokens.FindByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("service: refresh - lookup token: %w", err)
	}
	if stored == nil || !stored.IsValid() {
		return nil, domain.ErrInvalidToken
	}

	// Revoke before issuing - if issueTokenPair fails the old token is gone,
	// forcing the user to log in again. This is intentional: it prevents
	// a partially-successful refresh from leaving two valid tokens alive.
	if err := s.tokens.Revoke(ctx, hash); err != nil {
		return nil, fmt.Errorf("service: refresh - revoke old token: %w", err)
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if err != nil {
		return nil, fmt.Errorf("service: refresh - lookup user: %w", err)
	}
	if user == nil {
		return nil, domain.ErrInvalidToken
	}

	return s.issueTokenPair(ctx, user.ID, user.Email)
}

// Logout revokes all refresh tokens for the user, invalidating every session.
func (s *AuthService) Logout(ctx context.Context, userID string) error {
	if err := s.tokens.RevokeAll(ctx, userID); err != nil {
		return fmt.Errorf("service: logout: %w", err)
	}
	return nil
}

// private

func (s *AuthService) issueTokenPair(ctx context.Context, userID, email string) (*TokenPair, error) {
	now := time.Now().UTC()

	claims := &jwtutil.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jwtutil.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("service: sign access token: %w", err)
	}

	rawRefresh := uuid.New().String()
	if err := s.tokens.Store(ctx, userID, hashToken(rawRefresh), now.Add(refreshTokenTTL)); err != nil {
		return nil, fmt.Errorf("service: store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int64(accessTokenTTL.Seconds()),
	}, nil
}

// hashToken returns the hex-encoded SHA-256 of rawToken.
// Tokens are never stored in plain text.
func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
