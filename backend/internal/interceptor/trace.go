// Package interceptor provides Connect RPC interceptors for ant.
// trace.go implements trace_id injection per ADR-0011 (OpenTelemetry-compatible).
package interceptor

import (
	"context"
	"net/http"

	"github.com/google/uuid"
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

// TraceInterceptor is an HTTP middleware that injects OTel-compatible trace_id
// into the request context. It reads from the incoming `traceparent` header
// (W3C Trace Context); if missing, generates a new UUID-based trace_id.
//
// The trace_id is also added as a response header for client-side correlation.
func TraceInterceptor(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID, spanID := extractOrGenerate(r.Header.Get("traceparent"))

			ctx := context.WithValue(r.Context(), TraceIDKey, traceID)
			ctx = context.WithValue(ctx, SpanIDKey, spanID)

			// Inject trace_id into response header for client-side correlation
			w.Header().Set("X-Trace-ID", traceID)

			// Create a request-scoped logger with trace fields
			reqLogger := logger.With(
				zap.String("trace_id", traceID),
				zap.String("span_id", spanID),
			)

			// Store logger in context so downstream handlers can use it
			ctx = withLogger(ctx, reqLogger)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractOrGenerateTrace parses a W3C traceparent header and returns (traceID, spanID).
// Falls back to UUID generation if the header is missing or malformed.
//
// W3C traceparent format: 00-{trace_id}-{span_id}-{trace_flags}
// trace_id: 32 hex chars, span_id: 16 hex chars
func extractOrGenerate(traceparent string) (traceID, spanID string) {
	if len(traceparent) >= 55 && traceparent[0:3] == "00-" {
		// Basic parsing: 00-{32}-{16}-{2}
		if len(traceparent) >= 55 {
			traceID = traceparent[3:35]
			spanID = traceparent[36:52]
			if isValidHex(traceID) && isValidHex(spanID) {
				return traceID, spanID
			}
		}
	}
	// Fallback: generate new UUIDs
	traceID = uuid.New().String()
	spanID = uuid.New().String()[:16]
	return traceID, spanID
}

func isValidHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// ── Logger propagation ───────────────────────────────────────────────

type loggerKey struct{}

func withLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

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
