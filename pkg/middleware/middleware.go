// pree-it/pkg/middleware/middleware.go
//
// Reusable Gin middleware for every pree-it HTTP service.

package middleware

import (
	"net/http"
	"strings"

	"github.com/devekkx/pree-it/pkg/httputil"
	"github.com/devekkx/pree-it/pkg/jwtutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// contextKey is an unexported type for context keys set by this package,
// preventing collisions with keys from other packages.
type contextKey string

const (
	KeyUserID    contextKey = "user_id"
	KeyEmail     contextKey = "email"
	KeyRequestID contextKey = "request_id"
)

// RequestID injects a unique X-Request-ID into every request and response.
// Propagates an existing ID from upstream if present.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		c.Set(string(KeyRequestID), id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

// JWTAuth validates Bearer tokens and sets user_id and email on the context.
// Aborts with 401 if the token is missing, malformed, or expired.
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if raw == "" {
			httputil.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(raw, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			httputil.Unauthorized(c, "authorization header must be: Bearer <token>")
			c.Abort()
			return
		}

		claims, err := jwtutil.Parse(parts[1], secret)
		if err != nil {
			httputil.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(string(KeyUserID), claims.UserID)
		c.Set(string(KeyEmail), claims.Email)
		c.Next()
	}
}

// CORS sets permissive headers for development.
// In production, replace "*" with the specific allowed origin(s).
func CORS(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Recovery wraps gin.Recovery to return structured JSON instead of plain text.
func Recovery() gin.HandlerFunc {
	return gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, err any) {
		httputil.InternalError(c)
	})
}
