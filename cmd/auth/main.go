package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devekkx/pree-it-auth/internal/config"
	"github.com/devekkx/pree-it-auth/internal/db"
	"github.com/devekkx/pree-it-auth/internal/handler"
	"github.com/devekkx/pree-it-auth/internal/middleware"
	"github.com/devekkx/pree-it-auth/internal/observability"
	"github.com/devekkx/pree-it-auth/internal/service"
	"github.com/devekkx/pree-it-auth/internal/token"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	//  Bootstrap order
	// 1. Load config (panics if secrets missing)
	// 2. Init OTel SDK (must precede logger so bridge is registered)
	// 3. Init logger (uses OTel global provider)
	// 4. Init infrastructure (postgres, redis)
	// 5. Run migrations
	// 6. Wire application
	// 7. Start HTTP server
	// 8. Graceful shutdown

	cfg := config.Load()

	//  OTel
	shutdownCtx := context.Background()
	sdk, err := observability.Setup(shutdownCtx, cfg.OTELEndpoint, cfg.ServiceName)
	if err != nil {
		// Can't use the logger yet  write directly.
		os.Stderr.WriteString("fatal: otel setup: " + err.Error() + "\n")
		os.Exit(1)
	}

	//  Logger
	log, err := observability.NewLogger(cfg.ServiceName)
	if err != nil {
		os.Stderr.WriteString("fatal: logger setup: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	log.Info("starting", zap.String("service", cfg.ServiceName), zap.String("env", cfg.Env))

	//  PostgreSQL
	pool, err := db.NewPool(shutdownCtx, db.PoolConfig{DSN: cfg.PostgresDSN})
	if err != nil {
		log.Fatal("postgres pool", zap.Error(err))
	}
	defer pool.Close()

	if err := db.RunMigrations(shutdownCtx, pool, "db/migrations"); err != nil {
		log.Fatal("migrations", zap.Error(err))
	}

	//  Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           0,
		PoolSize:     20,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	defer rdb.Close()

	if err := rdb.Ping(shutdownCtx).Err(); err != nil {
		log.Fatal("redis ping", zap.Error(err))
	}

	//  Application wiring
	store := db.NewStore(pool)

	tokenMgr, err := token.NewManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	if err != nil {
		log.Fatal("token manager", zap.Error(err))
	}

	authSvc := service.New(store, tokenMgr)

	//  Router
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		gin.RecoveryWithWriter(os.Stderr),
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.NewRateLimiter(rdb, 20, 10*time.Second).Middleware(), // auth endpoints are sensitive
	)

	// Health  unauthenticated, not rate-limited
	r.GET("/health/live", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/health/ready", healthReady(rdb, pool))

	// Auth routes
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", handler.Register(authSvc))
		auth.POST("/login", handler.Login(authSvc))
		auth.POST("/refresh", handler.Refresh(authSvc))
		auth.POST("/logout", handler.Logout(authSvc))
	}

	//  Token cleanup background job
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	go runTokenCleanup(cleanupCtx, store, log)

	//  HTTP server
	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16, // 64 KiB
	}

	go func() {
		log.Info("listening", zap.String("addr", cfg.ListenAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	//  Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info("shutdown signal received", zap.String("signal", sig.String()))

	shutdownTimeout, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownTimeout); err != nil {
		log.Error("http shutdown error", zap.Error(err))
	}

	sdk.Shutdown(shutdownTimeout)
	log.Info("service stopped cleanly")
}

// healthReady checks both Redis and Postgres.
func healthReady(rdb *redis.Client, pool interface{ Ping(context.Context) error }) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		checks := map[string]string{}
		code := http.StatusOK

		if err := rdb.Ping(ctx).Err(); err != nil {
			checks["redis"] = "unhealthy"
			code = http.StatusServiceUnavailable
		} else {
			checks["redis"] = "healthy"
		}

		if err := pool.Ping(ctx); err != nil {
			checks["postgres"] = "unhealthy"
			code = http.StatusServiceUnavailable
		} else {
			checks["postgres"] = "healthy"
		}

		c.JSON(code, gin.H{
			"status": func() string {
				if code == http.StatusOK {
					return "ready"
				}
				return "degraded"
			}(),
			"checks": checks,
		})
	}
}

// runTokenCleanup deletes expired refresh tokens on a fixed interval.
func runTokenCleanup(ctx context.Context, store *db.Store, log *zap.Logger) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := store.DeleteExpiredTokens(ctx); err != nil {
				log.Warn("token cleanup failed", zap.Error(err))
			} else {
				log.Info("expired refresh tokens cleaned")
			}
		case <-ctx.Done():
			return
		}
	}
}
