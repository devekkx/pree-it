// Entry point for the auth service.
// Responsibilities: load config, wire dependencies, start server, handle shutdown.

package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/devekkx/pree-it/pkg/config"
	"github.com/devekkx/pree-it/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	log := logger.New("auth")
	defer log.Sync() //nolint:errcheck

	cfg := config.Load()

	// Repository─
	repo, err := repository.NewPostgres(context.Background(), cfg)
	if err != nil {
		log.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer repo.Close()

	// Service
	svc := service.NewAuthService(repo, cfg)

	// Handler
	h := handler.NewAuthHandler(svc, log)

	// HTTP Server
	srv := server.New(cfg, log, h)

	// Start
	go func() {
		log.Info("auth service starting", zap.String("port", cfg.AuthPort))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down auth service…")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("unclean shutdown", zap.Error(err))
		os.Exit(1)
	}

	log.Info("auth service stopped cleanly")
}
