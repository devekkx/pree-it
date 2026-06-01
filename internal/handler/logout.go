package handler

import (
	"net/http"

	"github.com/devekkx/pree-it-auth/internal/service"
	"github.com/gin-gonic/gin"
)

// Logout handles POST /api/v1/auth/logout.
// Always returns 204 - even if the token is already gone (idempotent).
func Logout(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie("refresh_token")
		if err == nil && raw != "" {
			// Best-effort invalidation - log but don't surface the error.
			_ = svc.Logout(c.Request.Context(), raw)
		}

		clearRefreshCookie(c)
		c.Status(http.StatusNoContent)
	}
}
