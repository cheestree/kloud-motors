package observability

import (
	"context"
	"log/slog"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func InitTracing(ctx context.Context, logger *slog.Logger, serviceName string) func(context.Context) error {
	exporter, err := texporter.New()
	if err != nil {
		logger.Warn("failed to initialize Cloud Trace exporter; tracing disabled", "service", serviceName, "error", err)
		return func(context.Context) error { return nil }
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		logger.Warn("failed to build OpenTelemetry resource; tracing disabled", "service", serviceName, "error", err)
		_ = exporter.Shutdown(ctx)
		return func(context.Context) error { return nil }
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("OpenTelemetry tracing enabled", "service", serviceName)

	return tracerProvider.Shutdown
}
