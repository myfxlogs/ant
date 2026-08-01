package mdgateway

import (
	"errors"
	"testing"
)

func TestClassifyConnError_NilError(t *testing.T) {
	t.Parallel()
	if c := ClassifyConnError(nil); c != ErrTransient {
		t.Fatalf("expected ErrTransient for nil error, got %d", c)
	}
}

func TestClassifyConnError_AuthErrors(t *testing.T) {
	t.Parallel()
	authErrors := []string{
		"invalid_account",
		"code=1001",
		"code=8",
		"invalid_credentials",
		"wrong password",
		"access denied",
		"invalid password",
		"not authorized",
		"account disabled",
		"WRONG PASSWORD",
		"Access Denied: invalid_account",
		"error code=1001: auth failed",
	}
	for _, msg := range authErrors {
		c := ClassifyConnError(errors.New(msg))
		if c != ErrAuth {
			t.Errorf("ClassifyConnError(%q) = %d, want ErrAuth", msg, c)
		}
	}
}

func TestClassifyConnError_HostErrors(t *testing.T) {
	t.Parallel()
	hostErrors := []string{
		"no such host",
		"dns resolution failed",
		"connection refused",
		"connect: connection refused",
		"no route to host",
		"network is unreachable",
		"name resolution error",
		"failed to connect",
		"dial tcp 1.2.3.4:443: connect: connection refused",
	}
	for _, msg := range hostErrors {
		c := ClassifyConnError(errors.New(msg))
		if c != ErrHost {
			t.Errorf("ClassifyConnError(%q) = %d, want ErrHost", msg, c)
		}
	}
}

func TestClassifyConnError_TransientErrors(t *testing.T) {
	t.Parallel()
	transientErrors := []string{
		"timeout",
		"EOF",
		"connection reset by peer",
		"context deadline exceeded",
		"service unavailable",
		"broker is temporarily down",
	}
	for _, msg := range transientErrors {
		c := ClassifyConnError(errors.New(msg))
		if c != ErrTransient {
			t.Errorf("ClassifyConnError(%q) = %d, want ErrTransient", msg, c)
		}
	}
}

func TestIsHostError(t *testing.T) {
	t.Parallel()
	// Positive cases
	hostMsgs := []string{
		"no such host",
		"dns error",
		"connection refused",
		"no route to host",
		"network is unreachable",
		"name resolution failure",
		"failed to connect to server",
		"dial tcp: connection refused",
	}
	for _, msg := range hostMsgs {
		if !isHostError(errors.New(msg), msg) {
			t.Errorf("isHostError(%q) should be true", msg)
		}
	}

	// Negative cases
	nonHostMsgs := []string{
		"timeout",
		"EOF",
		"invalid password",
		"connection reset",
	}
	for _, msg := range nonHostMsgs {
		if isHostError(errors.New(msg), msg) {
			t.Errorf("isHostError(%q) should be false", msg)
		}
	}
}

func TestClassifyConnError_AuthPriority(t *testing.T) {
	t.Parallel()
	// Auth keywords should take priority over host keywords.
	c := ClassifyConnError(errors.New("connection refused: invalid_account"))
	if c != ErrAuth {
		t.Fatalf("auth should take priority, got %d", c)
	}
}

func TestBoolToStr(t *testing.T) {
	t.Parallel()
	if boolToStr(true) != "1" {
		t.Fatal("expected 1 for true")
	}
	if boolToStr(false) != "0" {
		t.Fatal("expected 0 for false")
	}
}

func TestCircuitBreaker_SetOnStateChange(t *testing.T) {
	// 1 failure threshold, 1 success threshold, 1h cooldown.
	cb := NewCircuitBreaker(1, 1, 3600000000000)
	done := make(chan struct{})
	var fromState, toState State
	cb.SetOnStateChange(func(from, to State) {
		fromState, toState = from, to
		close(done)
	})
	cb.OnFailure()
	// Wait for async callback to fire.
	<-done
	if cb.State() != StateOpen {
		t.Fatal("expected StateOpen after 1 failure with threshold=1")
	}
	if fromState != StateClosed || toState != StateOpen {
		t.Fatalf("expected callback with Closed→Open, got %v→%v", fromState, toState)
	}
}
