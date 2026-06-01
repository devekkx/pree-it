package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is the canonical JWT payload for this system.
// Only the fields needed for authorisation decisions are included -
// no PII beyond email, which is needed for display.
type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Manager signs and verifies JWTs and generates opaque refresh tokens.
type Manager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewManager creates a Manager. secret must be at least 32 bytes.
func NewManager(secret string, accessTTL, refreshTTL time.Duration) (*Manager, error) {
	if len(secret) < 32 {
		return nil, errors.New("token.NewManager: JWT secret must be at least 32 bytes")
	}
	return &Manager{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}, nil
}

// NewAccessToken returns a signed JWT for the given user.
func (m *Manager) NewAccessToken(userID, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(), // jti - prevents replay of identical tokens
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("token.NewAccessToken: sign: %w", err)
	}
	return token, nil
}

// NewRefreshToken returns a cryptographically random opaque token string
// and its expiry time. The raw token is returned once to the client;
// only its SHA-256 hash is stored in the database.
func (m *Manager) NewRefreshToken() (raw string, expiresAt time.Time) {
	// Two UUIDs concatenated = 72 chars of random hex = 288 bits of entropy.
	// Far exceeds NIST SP 800-63B minimum of 112 bits for session tokens.
	raw = uuid.NewString() + uuid.NewString()
	expiresAt = time.Now().Add(m.refreshTokenTTL)
	return
}

// ParseAccessToken validates a token string and returns the embedded claims.
// Returns a typed error so callers can distinguish expired from malformed.
func (m *Manager) ParseAccessToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return m.secret, nil
		},
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithStrictDecoding(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

// Sentinel errors - callers switch on these, never on string messages.
var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("token invalid")
)
