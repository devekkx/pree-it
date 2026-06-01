// Wraps go.uber.org/zap with a service-tagged logger.
// Development: coloured human-readable output.
// Production:  JSON lines - ready for Loki ingestion via Alloy.

package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New returns a *zap.Logger tagged with the given service name.
// It reads ENV from the environment directly so it can be called
// before the full config struct is built.
func New(service string) *zap.Logger {
	var cfg zap.Config

	if os.Getenv("ENV") == "production" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	log, err := cfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		panic("logger: failed to initialise: " + err.Error())
	}

	return log.With(zap.String("service", service))
}
