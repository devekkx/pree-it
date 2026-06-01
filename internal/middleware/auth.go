package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/devekkx/pree-it-auth/internal/token"
	"github.com/gin-gonic/gin"
)

// JWTAuth validates a Bearer token on every request and injects claims
// into the Gin context. It does NOT issue tokens - that is auth-service's job.
// The gateway calls this middleware; auth-service handlers do not need it.
func JWTAuth(tm *token.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			abortUnauthorized(c, "authorization header required")
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			abortUnauthorized(c, "authorization header must be 'Bearer <token>'")
			return
		}

		claims, err := tm.ParseAccessToken(parts[1])
		if err != nil {
			switch {
			case errors.Is(err, token.ErrTokenExpired):
				abortUnauthorized(c, "token expired")
			default:
				abortUnauthorized(c, "invalid token")
			}
			return
		}

		// Inject into context for downstream handlers.
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Next()
	}
}

func abortUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
}
