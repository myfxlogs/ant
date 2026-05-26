// Package trace provides OpenTelemetry initialization for the ant data pipeline.
// ADR-0010 §2.3: spans cover normalize → quality → dedup → aggregate → publish → chwrite.
// Enabled via OTEL_EXPORTER_OTLP_ENDPOINT env var; disabled by default.
package trace

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Tracer wraps the OpenTelemetry tracer for the ant pipeline.
type Tracer struct {
	provider *sdktrace.TracerProvider
	enabled  bool
}

// Span is a thin wrapper around trace.Span.
type Span struct {
	inner trace.Span
}

// New creates a tracer. If OTEL_EXPORTER_OTLP_ENDPOINT is set, OTel SDK
// is initialized with OTLP gRPC exporter; otherwise traces are no-ops.
func New() *Tracer {
	ep := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if ep == "" {
		return &Tracer{enabled: false}
	}

	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint(ep),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return &Tracer{enabled: false}
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.01)), // 1% sampling for real-time path
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return &Tracer{provider: provider, enabled: true}
}

// NewWithProvider creates a tracer backed by a user-supplied TracerProvider.
// Use this for testing with tracetest.InMemoryExporter or custom sampling.
// L-2: enables e2e verification of pipeline spans without an external collector.
func NewWithProvider(provider *sdktrace.TracerProvider) *Tracer {
	if provider == nil {
		return &Tracer{enabled: false}
	}
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return &Tracer{provider: provider, enabled: true}
}

// ForceFlush flushes all pending spans to the exporter. For testing.
func (t *Tracer) ForceFlush(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.ForceFlush(ctx)
	}
	return nil
}

// StartSpan begins a new span. If disabled, returns a no-op span.
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	if !t.enabled {
		return ctx, &Span{}
	}
	tr := otel.Tracer("ant-mdgateway")
	ctx, span := tr.Start(ctx, name)
	return ctx, &Span{inner: span}
}

// End closes a span. Safe to call on disabled spans.
func (s *Span) End() {
	if s.inner != nil {
		s.inner.End()
	}
}

// Enabled reports whether tracing is active.
func (t *Tracer) Enabled() bool {
	return t.enabled
}

// Shutdown flushes pending spans and stops the exporter.
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}
