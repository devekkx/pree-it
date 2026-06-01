package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/devekkx/pree-it-auth/internal/service"
	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Login handles POST /api/v1/auth/login.
// The refresh token is set as an HttpOnly Secure cookie scoped to the
// refresh endpoint - it is never exposed in the response body.
func Login(svc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}

		_, pair, err := svc.Login(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrInvalidCredentials) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
			return
		}

		setRefreshCookie(c, pair.RefreshToken, pair.ExpiresAt)

		c.JSON(http.StatusOK, gin.H{
			"access_token": pair.AccessToken,
			"token_type":   "Bearer",
			"expires_in":   900, // 15 minutes in seconds
		})
	}
}

// setRefreshCookie writes the refresh token as a hardened HttpOnly cookie.
func setRefreshCookie(c *gin.Context, token string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge <= 0 {
		maxAge = 0
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"refresh_token",        // name
		token,                  // value
		maxAge,                 // maxAge
		"/api/v1/auth/refresh", // path - scoped tightly, not root
		"",                     // domain - set to your domain in prod
		true,                   // secure - HTTPS only
		true,                   // httpOnly - no JS access
	)
}
