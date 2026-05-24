package trace

import (
	"context"
	"testing"
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
	if span.Name != "test" {
		t.Errorf("span name = %q, want %q", span.Name, "test")
	}
	span.End()
	_ = ctx
}

func TestTracer_EnabledWithEndpoint(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	tr := New()
	if !tr.Enabled() {
		t.Error("tracer should be enabled when OTEL_EXPORTER_OTLP_ENDPOINT is set")
	}
}
