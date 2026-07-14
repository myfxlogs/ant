package sentry

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

func TestCaptureError_NilError(t *testing.T) {
	t.Parallel()
	CaptureError(nil)
}

func TestCapturePanic_NoPanic(t *testing.T) {
	t.Parallel()
	CapturePanic(nil)
}

func TestErrorInterceptor_ShouldCapture_ClientErrors(t *testing.T) {
	t.Parallel()
	i := &ErrorInterceptor{enabled: true}
	clientErrors := []connect.Code{
		connect.CodeUnauthenticated,
		connect.CodePermissionDenied,
		connect.CodeInvalidArgument,
		connect.CodeNotFound,
		connect.CodeAlreadyExists,
		connect.CodeFailedPrecondition,
		connect.CodeResourceExhausted,
	}
	for _, code := range clientErrors {
		err := connect.NewError(code, errors.New("test"))
		if i.shouldCapture(err) {
			t.Errorf("expected shouldCapture=false for %s", code)
		}
	}
}

func TestErrorInterceptor_ShouldCapture_ServerErrors(t *testing.T) {
	t.Parallel()
	i := &ErrorInterceptor{enabled: true}
	serverErrors := []connect.Code{
		connect.CodeInternal,
		connect.CodeUnknown,
		connect.CodeDataLoss,
		connect.CodeUnavailable,
	}
	for _, code := range serverErrors {
		err := connect.NewError(code, errors.New("test"))
		if !i.shouldCapture(err) {
			t.Errorf("expected shouldCapture=true for %s", code)
		}
	}
}

func TestErrorInterceptor_ShouldCapture_NonConnectError(t *testing.T) {
	t.Parallel()
	i := &ErrorInterceptor{enabled: true}
	err := errors.New("plain error")
	if !i.shouldCapture(err) {
		t.Error("expected shouldCapture=true for non-connect error")
	}
}

func TestErrorInterceptor_Disabled(t *testing.T) {
	t.Parallel()
	i := &ErrorInterceptor{enabled: false}
	err := connect.NewError(connect.CodeInternal, errors.New("test"))
	if i.shouldCapture(err) {
		t.Error("expected shouldCapture=false when disabled")
	}
}

func TestErrorInterceptor_WrapUnary_NoError(t *testing.T) {
	t.Parallel()
	i := &ErrorInterceptor{enabled: false}
	called := false
	next := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		return nil, nil
	}
	wrapped := i.WrapUnary(next)
	_, err := wrapped(context.Background(), connect.NewRequest(&emptypbMsg{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected next to be called")
	}
}

type emptypbMsg struct{}

func (m *emptypbMsg) Reset()         {}
func (m *emptypbMsg) String() string { return "" }
func (m *emptypbMsg) ProtoMessage()  {}

func TestNewErrorInterceptor(t *testing.T) {
	t.Parallel()
	i := NewErrorInterceptor()
	if i == nil {
		t.Fatal("expected non-nil interceptor")
	}
}

func TestSentryInit_NoDSN(t *testing.T) {
	t.Setenv("SENTRY_DSN", "")
	cleanup := Init(zap.NewNop())
	if cleanup == nil {
		t.Fatal("expected non-nil cleanup function")
	}
	cleanup()
}

func TestSentryInit_InvalidDSN(t *testing.T) {
	t.Setenv("SENTRY_DSN", "not-a-valid-dsn")
	cleanup := Init(zap.NewNop())
	if cleanup == nil {
		t.Fatal("expected non-nil cleanup function even on init failure")
	}
	cleanup()
}

func TestCaptureError_WithTags(t *testing.T) {
	t.Parallel()
	err := errors.New("test error")
	CaptureError(err, map[string]string{"key": "value"})
}

func TestCapturePanic_WithPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected re-panic after CapturePanic")
		}
	}()
	func() {
		defer CapturePanic(nil)
		panic("test panic")
	}()
}

func TestErrorInterceptor_WrapStreamingClient(t *testing.T) {
	t.Parallel()
	i := &ErrorInterceptor{enabled: false}
	fn := i.WrapStreamingClient(func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		return nil
	})
	if fn == nil {
		t.Fatal("expected non-nil wrapped function")
	}
}

func TestErrorInterceptor_WrapStreamingHandler_NoError(t *testing.T) {
	t.Parallel()
	i := &ErrorInterceptor{enabled: false}
	called := false
	next := func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		called = true
		return nil
	}
	wrapped := i.WrapStreamingHandler(next)
	err := wrapped(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected next to be called")
	}
}

func TestSentryHub(t *testing.T) {
	t.Parallel()
	hub := sentry.CurrentHub()
	if hub == nil {
		t.Fatal("expected non-nil current hub")
	}
}
