package config

import (
	"fmt"
	"os"
	"time"

	"github.com/devekkx/pree-it-auth/pkg/secrets"
)

// Config holds all resolved configuration for the auth service.
// Secrets are never stored as plain env vars - always read from files.
type Config struct {
	// Runtime
	Env         string
	ServiceName string
	ListenAddr  string

	// Resolved DSN - built from parts + secret password
	PostgresDSN string

	// Redis
	RedisAddr     string
	RedisPassword string

	// NATS
	NATSUrl      string
	NATSPassword string

	// JWT
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	// Observability
	OTELEndpoint string
}

// Load reads all configuration and resolves secrets.
// Panics immediately if any required secret is missing - fail fast at startup.
func Load() *Config {
	cfg := &Config{
		Env:             env("APP_ENV", "development"),
		ServiceName:     env("SERVICE_NAME", "preeit-auth"),
		ListenAddr:      env("LISTEN_ADDR", ":8081"),
		RedisAddr:       env("REDIS_ADDR", "redis:6379"),
		NATSUrl:         env("NATS_URL", "nats://nats:4222"),
		OTELEndpoint:    env("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317"),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	// All of these panic if the secret file is missing or empty.
	cfg.JWTSecret = secrets.MustRead("jwt_secret")
	cfg.RedisPassword = secrets.MustRead("redis_password")
	cfg.NATSPassword = secrets.MustRead("nats_password")
	cfg.PostgresDSN = buildDSN()

	return cfg
}

func buildDSN() string {
	password := secrets.MustRead("postgres_password")
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		env("POSTGRES_HOST", "postgres"),
		env("POSTGRES_PORT", "5432"),
		env("POSTGRES_USER", "preeit_admin"),
		password,
		env("POSTGRES_DB", "preeit"),
		env("POSTGRES_SSLMODE", "disable"),
	)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
