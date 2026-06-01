// Assembles the Gin router and http.Server for the auth service.
// Separating server construction from main.go makes it testable -
// tests can call New() and bind to an ephemeral port.

package server

import (
	"net/http"

	"github.com/devekkx/pree-it/auth/internal/handler"
	"github.com/devekkx/pree-it/pkg/config"
	"github.com/devekkx/pree-it/pkg/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// New builds and returns a fully configured *http.Server.
func New(cfg *config.Config, log *zap.Logger, h *handler.AuthHandler) *http.Server {
	if cfg.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middleware - order matters
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())

	// Observability
	r.GET("/healthz", healthz(cfg))
	r.GET("/readyz", readyz())

	// Auth routes
	v1 := r.Group("/api/v1/auth")
	{
		v1.POST("/register", h.Register)
		v1.POST("/login", h.Login)
		v1.POST("/refresh", h.Refresh)
		v1.POST("/logout", middleware.JWTAuth(cfg.JWTSecret), h.Logout)
	}

	return &http.Server{
		Addr:         ":" + cfg.AuthPort,
		Handler:      r,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}
}

func healthz(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "auth",
			"env":     cfg.Env,
		})
	}
}

// readyz can be extended to probe postgres/redis before returning 200.
func readyz() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
