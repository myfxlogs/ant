// Package trace provides OpenTelemetry initialization for the ant platform.
// ADR-0010 §2.3: spans cover normalize → quality → dedup → aggregate → publish → chwrite.
// Enabled via OTEL_EXPORTER_OTLP_ENDPOINT env var; disabled by default.
//
// Architecture:
//   - InitGlobalProvider() is called once in main.go — sets the global TracerProvider.
//   - New() creates a tracer from the global provider (used by mdgateway pipeline).
//   - connectrpc.com/otelconnect interceptor uses the global provider for RPC spans.
//   - All share one TracerProvider → unified trace context across HTTP + pipeline.

package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
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

// InitGlobalProvider initializes the OTel SDK global TracerProvider and
// TextMapPropagator. Returns a shutdown function that flushes pending spans.
//
// Call once in main.go before any tracing occurs. When endpoint is empty,
// tracing is disabled and the returned shutdown is a no-op.
//
// Both the connectrpc.com/otelconnect interceptor (ConnectRPC spans) and
// the mdgateway pipeline (data-quality spans) share this single provider.
func InitGlobalProvider(endpoint string) (func(context.Context) error, error) {
	if endpoint == "" {
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func(context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.01)),
	)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return provider.Shutdown, nil
}

// New creates a tracer from the global TracerProvider.
// Used by the mdgateway pipeline; ConnectRPC uses the global provider directly
// via the otelconnect interceptor.
func New() *Tracer {
	provider, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	if !ok || provider == nil {
		return &Tracer{enabled: false}
	}
	return &Tracer{provider: provider, enabled: true}
}

// NewWithProvider creates a tracer backed by a user-supplied TracerProvider.
// Use for testing with tracetest.InMemoryExporter or custom sampling.
func NewWithProvider(provider *sdktrace.TracerProvider) *Tracer {
	if provider == nil {
		return &Tracer{enabled: false}
	}
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return &Tracer{provider: provider, enabled: true}
}

// ForceFlush flushes all pending spans. For testing.
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
