package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a structured logger tagged with the service name.
// Development: human-friendly coloured output.
// Production: JSON for Loki ingestion via Alloy.
func New(service string) *zap.Logger {
	var cfg zap.Config

	if os.Getenv("ENV") == "production" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := cfg.Build()
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}

	return logger.With(zap.String("service", service))
}
