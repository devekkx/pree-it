package observability

import (
	"context"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	otellog "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger returns a Zap logger that writes structured JSON to stdout
// and mirrors every entry into the OTel log pipeline (→ Loki via Collector).
// Must be called after observability.Setup() so the global logger provider
// is already registered.
func NewLogger(serviceName string) (*zap.Logger, error) {
	//  Stdout core
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "timestamp"
	encCfg.MessageKey = "message"
	encCfg.LevelKey = "level"
	encCfg.CallerKey = "caller"

	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encCfg),
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	)

	//  OTel bridge core
	// Sends every Zap log entry to the OTel LoggerProvider → Collector → Loki.
	// trace_id and span_id are attached automatically by the bridge when a
	// span is active in the context — enabling log↔trace correlation in Grafana.
	otelCore := otelzap.NewCore(
		serviceName,
		otelzap.WithLoggerProvider(otellog.GetLoggerProvider()),
	)

	combined := zapcore.NewTee(stdoutCore, otelCore)

	return zap.New(
		combined,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	), nil
}

// WithContext returns a child logger with trace_id and span_id fields
// extracted from the active OTel span in ctx.
// These fields match the derivedFields regex in the Loki datasource config,
// rendering a "View Trace in Tempo" button in Grafana Explore.
func WithContext(ctx context.Context, log *zap.Logger) *zap.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return log
	}
	return log.With(
		zap.String("trace_id", span.SpanContext().TraceID().String()),
		zap.String("span_id", span.SpanContext().SpanID().String()),
	)
}
