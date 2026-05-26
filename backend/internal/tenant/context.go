// Package tenant provides per-tenant context propagation and rate limiting (M10-BASE-A4).
//
// Key constraint: Prometheus metrics must NOT carry tenant_id as a label (cardinality).
// tenant_id is propagated via logs, traces, and in-memory LRU bookkeeping only.
package tenant

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ctxKey struct{}

var tenantIDKey ctxKey

// WithTenantID adds tenant_id to a context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// GetTenantID extracts tenant_id from context.
func GetTenantID(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}

// LoggerWithTenant returns a logger with tenant_id field if present in context.
func LoggerWithTenant(ctx context.Context, log *zap.Logger) *zap.Logger {
	tid := GetTenantID(ctx)
	if tid == "" {
		return log
	}
	return log.With(zap.String("tenant_id", tid))
}

// SpanWithTenant adds tenant_id to the current tracing span.
func SpanWithTenant(ctx context.Context, tenantID string) {
	if tenantID == "" {
		return
	}
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attribute.String("tenant_id", tenantID))
	}
}
