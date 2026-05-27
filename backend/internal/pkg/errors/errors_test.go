package errors

import (
	stderrors "errors"
	"fmt"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	t.Parallel()
	e := New(UserNotFound, fmt.Errorf("user 123 not in db"))
	got := e.Error()
	want := "[1001] errors.user_not_found: user 123 not in db"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAppError_Error_Nil(t *testing.T) {
	t.Parallel()
	e := New(NotFound, nil)
	got := e.Error()
	want := "[5] errors.not_found"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAppError_Unwrap(t *testing.T) {
	t.Parallel()
	sentinel := fmt.Errorf("original error")
	e := New(InternalError, sentinel)
	if !stderrors.Is(e, sentinel) {
		t.Error("stderrors.Is should find sentinel via Unwrap")
	}
}

func TestAppError_Unwrap_Nil(t *testing.T) {
	t.Parallel()
	e := New(Forbidden, nil)
	if e.Unwrap() != nil {
		t.Error("Unwrap() should return nil when inner error is nil")
	}
}

func TestAppError_Unwrap_AsType(t *testing.T) {
	t.Parallel()
	e := New(InternalError, fmt.Errorf("wrapped"))
	var appErr *AppError
	if !stderrors.As(e, &appErr) {
		t.Error("stderrors.As should succeed on *AppError")
		return
	}
	if appErr.Code != InternalError {
		t.Errorf("expected code %d, got %d", InternalError, appErr.Code)
	}
}
