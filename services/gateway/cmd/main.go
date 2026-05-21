package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pree-it/pree-it/pkg/config"
	"github.com/pree-it/pree-it/pkg/logger"
	"github.com/pree-it/pree-it/pkg/middleware"
)

func main() {
	log := logger.New("gateway")
	defer log.Sync()

	port := config.MustGet("GATEWAY_PORT")
	jwtSecret := config.MustGet("jwt_secret") // /run/secrets/jwt_secret

	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())
	r.Use(middleware.RateLimit())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "gateway"})
	})

	// Public Routes
	public := r.Group("/api/v1")
	{
		public.POST("/auth/register", proxyToAuth)
		public.POST("/auth/login", proxyToAuth)
		public.POST("/auth/refresh", proxyToAuth)
	}

	//  Protected Routes
	protected := r.Group("/api/v1")
	protected.Use(middleware.JWTAuth(jwtSecret))
	{
		protected.POST("/auth/logout", proxyToAuth)

		protected.GET("/users/me", stubHandler("user profile"))
		protected.GET("/users/:id", stubHandler("user by id"))
		protected.PUT("/users/me", stubHandler("update profile"))

		protected.POST("/conversations", stubHandler("create conversation"))
		protected.GET("/conversations", stubHandler("list conversations"))
		protected.GET("/conversations/:id", stubHandler("get conversation"))
		protected.GET("/conversations/:id/messages", stubHandler("list messages"))
		protected.POST("/conversations/:id/messages", stubHandler("send message"))
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("gateway listening on :" + port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error: " + err.Error())
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("forced shutdown: " + err.Error())
	}
	log.Info("gateway stopped")
}

func proxyToAuth(c *gin.Context) {
	authURL := config.MustGet("AUTH_SERVICE_URL")
	c.JSON(http.StatusNotImplemented, gin.H{
		"message":  "auth proxy not yet wired",
		"upstream": authURL + c.Request.URL.Path,
	})
}

func stubHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"message": name + " — not yet implemented",
			"user_id": c.GetString("user_id"),
		})
	}
}
