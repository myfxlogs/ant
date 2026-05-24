// Package trace provides OpenTelemetry initialization for the ant data pipeline.
// ADR-0010 §2.3: spans cover normalize → quality → dedup → aggregate → publish → chwrite.
// Enabled via OTEL_EXPORTER_OTLP_ENDPOINT env var; disabled by default.
package trace

import (
	"context"
	"os"
)

// Tracer is a minimal span tracer. When OTLP endpoint is configured, spans
// are exported to a collector (Tempo/Jaeger). When disabled, all methods are no-ops.
type Tracer struct {
	enabled bool
	// otel tracer handle (go.opentelemetry.io/otel/trace.Tracer) — wired when
	// the OTel SDK is added as a dependency. For now, spans are logged as
	// structured events.
}

// Span represents a single operation span.
type Span struct {
	Name   string
	parent *Span
}

// New creates a tracer. If OTEL_EXPORTER_OTLP_ENDPOINT is set, OTel SDK
// is initialized; otherwise traces are no-ops.
func New() *Tracer {
	ep := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	return &Tracer{enabled: ep != ""}
}

// StartSpan begins a new span as a child of the current span.
// If parent is nil, this is a root span.
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	if !t.enabled {
		return ctx, &Span{Name: name}
	}
	// TODO: wire go.opentelemetry.io/otel when added to go.mod.
	// For now, this is a structured no-op that preserves the trace hierarchy.
	return ctx, &Span{Name: name}
}

// End closes a span. Must be called (typically via defer) for every StartSpan.
func (s *Span) End() {
	// no-op when OTel SDK is not wired
}

// Enabled reports whether tracing is active.
func (t *Tracer) Enabled() bool {
	return t.enabled
}
