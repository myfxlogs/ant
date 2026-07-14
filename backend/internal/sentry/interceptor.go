package sentry

import (
	"context"

	"connectrpc.com/connect"
	"github.com/getsentry/sentry-go"
)

// ErrorInterceptor captures RPC errors and reports them to Sentry.
// Only errors with Code >= CodeInternal are reported (excludes
// client errors like Unauthenticated, InvalidArgument, NotFound).
type ErrorInterceptor struct {
	enabled bool
}

// NewErrorInterceptor creates a Sentry error capture interceptor.
// Sentry must be initialized before calling this.
func NewErrorInterceptor() *ErrorInterceptor {
	return &ErrorInterceptor{enabled: sentry.HasHubOnContext(context.Background()) || sentry.CurrentHub().Client() != nil}
}

func (i *ErrorInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		resp, err := next(ctx, req)
		if err != nil && i.shouldCapture(err) {
			hub := sentry.GetHubFromContext(ctx)
			if hub == nil {
				hub = sentry.CurrentHub()
			}
			hub.CaptureException(err)
		}
		return resp, err
	}
}

func (i *ErrorInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *ErrorInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		err := next(ctx, conn)
		if err != nil && i.shouldCapture(err) {
			hub := sentry.GetHubFromContext(ctx)
			if hub == nil {
				hub = sentry.CurrentHub()
			}
			hub.CaptureException(err)
		}
		return err
	}
}

// shouldCapture returns true for server-side errors worth tracking.
// Client errors (4xx) are excluded — they indicate user mistakes, not bugs.
func (i *ErrorInterceptor) shouldCapture(err error) bool {
	if !i.enabled {
		return false
	}
	ce, ok := err.(*connect.Error)
	if !ok {
		return true
	}
	switch ce.Code() {
	case connect.CodeUnauthenticated,
		connect.CodePermissionDenied,
		connect.CodeInvalidArgument,
		connect.CodeNotFound,
		connect.CodeAlreadyExists,
		connect.CodeFailedPrecondition,
		connect.CodeResourceExhausted:
		return false
	default:
		return true
	}
}
