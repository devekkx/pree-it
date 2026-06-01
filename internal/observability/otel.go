package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otellog "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// SDK holds the three OTel providers. Call Shutdown on process exit.
type SDK struct {
	tp *sdktrace.TracerProvider
	mp *sdkmetric.MeterProvider
	lp *sdklog.LoggerProvider
}

// Shutdown flushes all pending telemetry and closes exporters.
// Should be deferred immediately after Setup returns.
func (s *SDK) Shutdown(ctx context.Context) {
	// Flush traces first - they may reference active spans from metrics/logs.
	if err := s.tp.Shutdown(ctx); err != nil {
		fmt.Printf("otel: trace provider shutdown error: %v\n", err)
	}
	if err := s.mp.Shutdown(ctx); err != nil {
		fmt.Printf("otel: metric provider shutdown error: %v\n", err)
	}
	if err := s.lp.Shutdown(ctx); err != nil {
		fmt.Printf("otel: log provider shutdown error: %v\n", err)
	}
}

// Setup initialises traces, metrics, and logs - all exported to the
// OTel Collector over gRPC - and registers the global providers.
// Must be called before any instrumented code runs.
func Setup(ctx context.Context, collectorEndpoint, serviceName string) (*SDK, error) {
	conn, err := grpc.NewClient(
		collectorEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("otel: dial collector %q: %w", collectorEndpoint, err)
	}

	res, err := sdkresource.New(ctx,
		sdkresource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("0.1.0"),
			attribute.String("deployment.environment", "production"),
		),
		sdkresource.WithOS(),
		sdkresource.WithProcess(),
		sdkresource.WithContainer(),
	)
	if err != nil {
		return nil, fmt.Errorf("otel: build resource: %w", err)
	}

	//  Traces
	traceExp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("otel: trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExp,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		// Respect upstream sampling decisions; otherwise sample 10%.
		sdktrace.WithSampler(
			sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1)),
		),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	//  Metrics
	metricExp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("otel: metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(metricExp,
				sdkmetric.WithInterval(15*time.Second),
			),
		),
	)
	otel.SetMeterProvider(mp)

	//  Logs
	logExp, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("otel: log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(logExp,
				sdklog.WithExportInterval(5*time.Second),
			),
		),
	)
	otellog.SetLoggerProvider(lp)

	return &SDK{tp: tp, mp: mp, lp: lp}, nil
}
