package handler

import (
	"errors"
	"net/http"

	"github.com/devekkx/pree-it-auth/internal/service"
	"github.com/gin-gonic/gin"
)

// Refresh handles POST /api/v1/auth/refresh.
// Reads the refresh token from the HttpOnly cookie only - never from body/header.
func Refresh(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie("refresh_token")
		if err != nil || raw == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
			return
		}

		pair, err := svc.Refresh(c.Request.Context(), raw)
		if err != nil {
			// Always clear the cookie on any refresh failure.
			clearRefreshCookie(c)

			switch {
			case errors.Is(err, service.ErrTokenReuse):
				// Reuse means potential compromise - respond with 401
				// and a message that prompts re-login without leaking details.
				c.JSON(http.StatusUnauthorized, gin.H{"error": "session invalidated, please log in again"})
			case errors.Is(err, service.ErrTokenExpired):
				c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			default:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			}
			return
		}

		setRefreshCookie(c, pair.RefreshToken, pair.ExpiresAt)

		c.JSON(http.StatusOK, gin.H{
			"access_token": pair.AccessToken,
			"token_type":   "Bearer",
			"expires_in":   900,
		})
	}
}

// clearRefreshCookie immediately expires the refresh token cookie.
func clearRefreshCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", "", -1, "/api/v1/auth/refresh", "", true, true)
}
