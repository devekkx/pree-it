// Reads configuration from Docker secrets first, env vars second.
// Docker secrets are mounted as files at /run/secrets/<name>.
// MustGet panics at startup if a required value is missing —
// services fail fast with a clear message rather than misbehaving.

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const secretsDir = "/run/secrets"

// Get returns the value for key, checking (in order):
//  1. Docker secret at /run/secrets/<lowercase_key>
//  2. Environment variable KEY
//  3. The provided fallback
func Get(key, fallback string) string {
	if val := readSecret(key); val != "" {
		return val
	}
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// MustGet returns the value for key or panics if not found anywhere.
func MustGet(key string) string {
	val := Get(key, "")
	if val == "" {
		panic(fmt.Sprintf(
			"required config missing: %q — set env var or create secrets/%s",
			key, strings.ToLower(key),
		))
	}
	return val
}

// GetInt returns the value for key as an int, or fallback if missing/invalid.
func GetInt(key string, fallback int) int {
	raw := Get(key, "")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

// BuildPostgresDSN constructs a DSN from individual secret/env values.
// Each credential can be a separate Docker secret.
func BuildPostgresDSN() string {
	host := Get("POSTGRES_HOST", "localhost")
	port := Get("POSTGRES_PORT", "5432")
	user := MustGet("postgres_user")
	password := MustGet("postgres_password")
	db := MustGet("postgres_db")
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, db,
	)
}

// BuildRedisAddr constructs a Redis address from host + port config.
func BuildRedisAddr() string {
	host := Get("REDIS_HOST", "localhost")
	port := Get("REDIS_PORT", "6379")
	return fmt.Sprintf("%s:%s", host, port)
}

// readSecret reads a Docker secret file for the given key.
// Returns empty string if the file does not exist.
func readSecret(key string) string {
	path := fmt.Sprintf("%s/%s", secretsDir, strings.ToLower(key))
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(data), "\n\r")
}
