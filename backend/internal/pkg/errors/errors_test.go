package errors

import (
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
)

func TestNew_Success(t *testing.T) {
	t.Parallel()
	appErr := New(UserNotFound, fmt.Errorf("user not in db"))
	if appErr.Code != UserNotFound {
		t.Errorf("expected code %d, got %d", UserNotFound, appErr.Code)
	}
}

func TestNew_UnknownCodeFallsBack(t *testing.T) {
	t.Parallel()
	appErr := New(99999, nil)
	if appErr.Message != errorMessages[UnknownError] {
		t.Errorf("expected fallback to unknown error message")
	}
}

func TestNewWithMessage_CustomMessage(t *testing.T) {
	t.Parallel()
	appErr := NewWithMessage(OrderRejected, "custom reason", fmt.Errorf("detail"))
	if appErr.Code != OrderRejected {
		t.Errorf("expected code %d, got %d", OrderRejected, appErr.Code)
	}
	if appErr.Message != "custom reason" {
		t.Errorf("expected custom message, got %q", appErr.Message)
	}
}

func TestAppError_Error(t *testing.T) {
	t.Parallel()
	appErr := New(InvalidParameter, fmt.Errorf("field X"))
	if appErr.Error() == "" {
		t.Fatal("expected non-empty error string")
	}
}

func TestAppError_Unwrap(t *testing.T) {
	t.Parallel()
	inner := fmt.Errorf("inner")
	appErr := New(InternalError, inner)
	if !errors.Is(appErr, inner) {
		t.Fatal("expected Unwrap to return inner error")
	}
}

func TestGetMessage(t *testing.T) {
	t.Parallel()
	if GetMessage(TokenExpired) == "" {
		t.Fatal("expected non-empty message")
	}
	if GetMessage(99999) != errorMessages[UnknownError] {
		t.Fatal("expected fallback")
	}
}

func TestToConnectError(t *testing.T) {
	t.Parallel()
	appErr := New(Unauthorized, fmt.Errorf("bad token"))
	ce := ToConnectError(appErr)
	if ce.Code() != connect.CodeUnauthenticated {
		t.Errorf("expected Unauthenticated, got %v", ce.Code())
	}
}

func TestToConnectError_UnknownCode(t *testing.T) {
	t.Parallel()
	appErr := New(99999, nil)
	ce := ToConnectError(appErr)
	if ce.Code() != connect.CodeInternal {
		t.Errorf("expected Internal fallback, got %v", ce.Code())
	}
}

func TestErrorCodeUniqueness(t *testing.T) {
	t.Parallel()
	seen := make(map[int]bool)
	for code := range errorMessages {
		if seen[code] {
			t.Errorf("duplicate error code: %d", code)
		}
		seen[code] = true
	}
}
