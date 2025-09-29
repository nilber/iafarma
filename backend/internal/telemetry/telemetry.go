package telemetry

import (
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTelemetry initializes OpenTelemetry (optional service)
// Returns (shutdown function, enabled, error)
func InitTelemetry() (func(), bool, error) {
	ctx := context.Background()

	// Check if telemetry is enabled via environment variable
	enableTelemetry := strings.ToLower(os.Getenv("ENABLE_TELEMETRY"))
	if enableTelemetry != "true" && enableTelemetry != "1" {
		// Telemetry is disabled, return noop shutdown function
		return func() {}, false, nil
	}

	// Check if telemetry endpoint is configured
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		// Telemetry is disabled, return noop shutdown function
		return func() {}, false, nil
	}

	// Create OTLP HTTP exporter
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(), // Use HTTPS in production
	)
	if err != nil {
		// Log warning but don't fail the application
		return func() {}, false, err
	}

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(os.Getenv("OTEL_SERVICE_NAME")),
			semconv.ServiceVersion(os.Getenv("OTEL_SERVICE_VERSION")),
		),
	)
	if err != nil {
		return func() {}, false, err
	}

	// Create trace provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global trace provider
	otel.SetTracerProvider(tp)

	// Return shutdown function
	return func() {
		tp.Shutdown(ctx)
	}, true, nil
}
