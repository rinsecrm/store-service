package tracing

import (
	"context"
	"fmt"

	"github.com/rinsecrm/store-service/core/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

var (
	tracer         trace.Tracer
	tracerProvider *sdktrace.TracerProvider
)

// Config holds tracing configuration
type Config struct {
	ServiceName string
	TempoHost   string
	Version     string
}

// Start initializes the tracing system
func Start(config Config) error {
	tracer = otel.Tracer(config.ServiceName)

	if config.TempoHost == "" {
		logging.Info("No Tempo host configured, tracing will be no-op")
		return nil
	}

	ctx := context.Background()
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(config.TempoHost),
		otlptracegrpc.WithDialOption(grpc.WithBlock()),
	)
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}
	logging.Info("Created OTLP Exporter")

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.Version),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}
	logging.Info("Created OTLP Resource")

	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	logging.Info("Tracing initialized successfully")
	return nil
}

// Stop gracefully shuts down the tracing system
func Stop(ctx context.Context) error {
	if tracerProvider == nil {
		return nil
	}
	if err := tracerProvider.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown TracerProvider: %w", err)
	}
	logging.Info("Tracing shutdown successfully")
	return nil
}

// StartSpan starts a new span
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	if tracer == nil {
		// Return no-op span if tracing is not configured
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, name)
}

// StartSpanWithOptions starts a new span with additional options
func StartSpanWithOptions(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tracer == nil {
		// Return no-op span if tracing is not configured
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, name, opts...)
}

// GetTracer returns the current tracer instance
func GetTracer() trace.Tracer {
	if tracer == nil {
		// Return a no-op tracer if none is configured
		return trace.NewNoopTracerProvider().Tracer("noop")
	}
	return tracer
}

// IsEnabled returns true if tracing is properly configured
func IsEnabled() bool {
	return tracerProvider != nil
}
