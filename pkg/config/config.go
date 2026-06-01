// Loads all configuration at startup into a typed struct.
// Sources (highest precedence first):
//   1. Docker secret file at /run/secrets/<lowercase_key>
//   2. Environment variable
//   3. Default value
//
// Load() panics on any missing required value — services fail
// immediately at boot with a clear message, not silently later.

package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const secretsDir = "/run/secrets"

// Load reads all configuration and returns a validated *Config.
// Panics on any missing required field.
func Load() *Config {
	return &Config{
		AppName: get("APP_NAME", "pree-it"),
		Env:     get("ENV", "development"),

		GatewayPort: mustGet("GATEWAY_PORT"),
		AuthPort:    mustGet("AUTH_PORT"),

		PostgresHost:     get("POSTGRES_HOST", "localhost"),
		PostgresPort:     get("POSTGRES_PORT", "5432"),
		PostgresUser:     mustGet("postgres_user"),
		PostgresPassword: mustGet("postgres_password"),
		PostgresDB:       mustGet("postgres_db"),

		RedisHost:     get("REDIS_HOST", "localhost"),
		RedisPort:     get("REDIS_PORT", "6379"),
		RedisPassword: mustGet("redis_password"),

		NatsURL: mustGet("NATS_URL"),

		JWTSecret:      mustGet("jwt_secret"),
		AuthServiceURL: mustGet("AUTH_SERVICE_URL"),

		OTLPEndpoint: get("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		ServiceName:  get("OTEL_SERVICE_NAME", "unknown"),

		ShutdownTimeout:  mustDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
		HTTPReadTimeout:  mustDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		HTTPWriteTimeout: mustDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
		HTTPIdleTimeout:  mustDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
	}
}

// PostgresDSN builds a pgx-compatible connection string.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.PostgresUser, c.PostgresPassword,
		c.PostgresHost, c.PostgresPort,
		c.PostgresDB,
	)
}

// RedisAddr returns "host:port" for go-redis.
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

// IsProd reports whether the service is running in production mode.
func (c *Config) IsProd() bool { return c.Env == "production" }

// TracingEnabled reports whether an OTLP endpoint is configured.
func (c *Config) TracingEnabled() bool { return c.OTLPEndpoint != "" }

// private helpers

// get returns the first non-empty value: secret → env → fallback.
func get(key, fallback string) string {
	if v := readSecret(key); v != "" {
		return v
	}
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// mustGet panics when no value is found for key.
func mustGet(key string) string {
	if v := get(key, ""); v != "" {
		return v
	}
	panic(fmt.Sprintf(
		"[config] required value missing for %q — set env var or create secrets/%s",
		key, strings.ToLower(key),
	))
}

// mustDuration reads a duration string or returns the fallback.
func mustDuration(key string, fallback time.Duration) time.Duration {
	raw := get(key, "")
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

// readSecret reads /run/secrets/<lowercase_key>, trimming trailing newlines.
// Returns empty string if the file does not exist — caller decides if required.
func readSecret(key string) string {
	data, err := os.ReadFile(fmt.Sprintf("%s/%s", secretsDir, strings.ToLower(key)))
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(data), "\r\n")
}
