package usermgr

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"anttrader/internal/interceptor"
)

// GetUserID extracts the user ID from context using the existing auth interceptor.
// Returns empty string if not authenticated.
func GetUserID(ctx context.Context) string {
	return interceptor.GetUserID(ctx)
}

// LoggerWithUser returns a logger with user_id field if present in context.
// Does NOT create a new logger — adds user_id via With() only when authenticated.
func LoggerWithUser(ctx context.Context, log *zap.Logger) *zap.Logger {
	uid := GetUserID(ctx)
	if uid == "" {
		return log
	}
	return log.With(zap.String("user_id", uid))
}

// SpanWithUser adds user_id to the current tracing span.
// No-op if there is no active span or userID is empty.
func SpanWithUser(ctx context.Context, userID string) {
	if userID == "" {
		return
	}
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attribute.String("user_id", userID))
	}
}
