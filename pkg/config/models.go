package config

import "time"

type Config struct {
	// Runtime environment
	AppName string
	Env     string // "development" | "production" | "test"

	// Service ports - each service reads its own port key
	GatewayPort string
	AuthPort    string

	// PostgreSQL - host/port are non-secret; credentials from secrets/
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string // from secret
	PostgresPassword string // from secret
	PostgresDB       string // from secret

	// Redis - host/port non-secret; password from secret
	RedisHost     string
	RedisPort     string
	RedisPassword string // from secret

	// NATS
	NatsURL string

	// JWT - from secret
	JWTSecret string // from secret

	// Inter-service URLs
	AuthServiceURL string

	// Observability
	OTLPEndpoint string // "http://alloy:4317" - empty disables tracing
	ServiceName  string

	// Timeouts
	ShutdownTimeout  time.Duration
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
}
