// Package interceptor provides Connect RPC interceptors for ant.
// trace.go implements trace_id injection per ADR-0011 (OpenTelemetry-compatible).
package interceptor

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type traceCtxKey string

const (
	// TraceIDKey is the context key for trace_id.
	TraceIDKey traceCtxKey = "trace_id"
	// SpanIDKey is the context key for span_id (W3C compatible).
	SpanIDKey traceCtxKey = "span_id"
)

// TraceIDFromContext extracts the W3C-compatible trace_id from ctx.
// Returns empty string if not found.
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(TraceIDKey).(string); ok {
		return v
	}
	return ""
}

// SpanIDFromContext extracts the span_id from ctx.
func SpanIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(SpanIDKey).(string); ok {
		return v
	}
	return ""
}

// Note: W3C trace context propagation is now handled by the
// connectrpc.com/otelconnect interceptor (OTLP gRPC binary encoding).
// TraceInterceptor was removed in favor of the standard OTel SDK.
// The tools below (InjectNATSTraceHeaders, LogFields) remain for
// cross-service propagation and structured logging enrichment.

// ── Logger propagation ───────────────────────────────────────────────

type loggerKey struct{}

// LoggerFromContext extracts the request-scoped zap.Logger from ctx.
// Falls back to the global zap logger if not found.
func LoggerFromContext(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok && l != nil {
		return l
	}
	return zap.L()
}

// InjectNATSTraceHeaders copies W3C trace context from ctx into NATS headers.
func InjectNATSTraceHeaders(ctx context.Context, h map[string][]string) {
	if h == nil {
		return
	}
	traceID := TraceIDFromContext(ctx)
	spanID := SpanIDFromContext(ctx)
	if traceID == "" {
		return
	}
	// W3C traceparent: 00-{traceID}-{spanID}-01
	tp := "00-" + traceID + "-"
	if spanID != "" {
		tp += spanID
	} else {
		tp += "0000000000000000"
	}
	tp += "-01"
	h["traceparent"] = []string{tp}
}

// LogFields returns zap fields with trace_id and span_id from ctx.
// Use this when you need to add trace fields to a logger that wasn't
// created via LoggerFromContext.
func LogFields(ctx context.Context) []zapcore.Field {
	return []zapcore.Field{
		zap.String("trace_id", TraceIDFromContext(ctx)),
		zap.String("span_id", SpanIDFromContext(ctx)),
	}
}
