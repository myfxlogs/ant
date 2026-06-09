package trace

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTracer_DisabledByDefault(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	tr := New()
	if tr.Enabled() {
		t.Error("tracer should be disabled when OTEL_EXPORTER_OTLP_ENDPOINT is empty")
	}
	ctx, span := tr.StartSpan(context.Background(), "test")
	if span == nil {
		t.Error("span should not be nil (no-op span returns valid object)")
	}
	span.End()
	_ = ctx
	_ = tr.Shutdown(context.Background())
}

func TestTracer_EnabledWithEndpoint(t *testing.T) {
	// Uses NewWithProvider to avoid depending on global OTel provider state.
	provider := sdktrace.NewTracerProvider()
	tr := NewWithProvider(provider)
	if !tr.Enabled() {
		t.Error("tracer should be enabled with explicit provider")
	}
	_ = tr.Shutdown(context.Background())
}
