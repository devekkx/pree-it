// JWT claim type and parsing helper shared across services.
// Keeping claims here avoids circular imports between middleware
// and the auth service.

package jwtutil

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

const Issuer = "preeit"

// Claims is the payload embedded in every pree-it access token.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Parse validates tokenStr with secret and returns the embedded Claims.
// Returns a descriptive error if the token is malformed, expired, or
// signed with the wrong key.
func Parse(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("jwtutil: unexpected signing method %q", t.Header["alg"])
			}
			return []byte(secret), nil
		},
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("jwtutil: invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("jwtutil: token claims are invalid")
	}

	return claims, nil
}
