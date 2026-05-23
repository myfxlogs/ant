package faultisol

import (
	"context"
	"errors"
	"testing"
	"time"

	"anttrader/pkg/logger"
)

func init() {
	logger.Init(&logger.Config{Level: "info", Format: "json", Output: "stdout"})
}

func TestCircuitBreaker_New(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		CooldownPeriod:   10 * time.Second,
	})

	if cb.State() != StateClosed {
		t.Error("New circuit breaker should be in closed state")
	}
}

func TestCircuitBreaker_Allow(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 3,
		CooldownPeriod:   100 * time.Millisecond,
	})

	if !cb.Allow() {
		t.Error("Circuit breaker should allow requests when closed")
	}

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if cb.State() != StateOpen {
		t.Error("Circuit breaker should be open after threshold failures")
	}

	if cb.Allow() {
		t.Error("Circuit breaker should not allow requests when open")
	}

	time.Sleep(150 * time.Millisecond)

	if !cb.Allow() {
		t.Error("Circuit breaker should allow requests after cooldown")
	}

	if cb.State() != StateHalfOpen {
		t.Error("Circuit breaker should be in half-open state after cooldown")
	}
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 2,
		SuccessThreshold: 2,
		CooldownPeriod:   100 * time.Millisecond,
	})

	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != StateOpen {
		t.Error("Circuit breaker should be open")
	}

	time.Sleep(150 * time.Millisecond)

	cb.Allow()
	cb.RecordSuccess()
	cb.RecordSuccess()

	if cb.State() != StateClosed {
		t.Error("Circuit breaker should be closed after success threshold")
	}
}

func TestCircuitBreaker_Execute(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 2,
		CooldownPeriod:   100 * time.Millisecond,
	})

	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Execute should succeed: %v", err)
	}

	callCount := 0
	err = cb.Execute(func() error {
		callCount++
		return errors.New("test error")
	})
	if err == nil {
		t.Error("Execute should return error")
	}

	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errors.New("test error")
		})
	}

	err = cb.Execute(func() error {
		t.Error("Should not be called when circuit is open")
		return nil
	})

	var cbErr *CircuitBreakerError
	if !errors.As(err, &cbErr) {
		t.Error("Should return CircuitBreakerError when open")
	}
}

func TestSafeCall_NoPanic(t *testing.T) {
	result := SafeCall(func() (interface{}, error) {
		return "success", nil
	})

	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}
	if result.Value != "success" {
		t.Errorf("Expected 'success', got %v", result.Value)
	}
}

func TestSafeCall_WithError(t *testing.T) {
	testErr := errors.New("test error")
	result := SafeCall(func() (interface{}, error) {
		return nil, testErr
	})

	if result.Error != testErr {
		t.Errorf("Expected test error, got %v", result.Error)
	}
}

func TestTimeoutController(t *testing.T) {
	config := DefaultTimeoutConfig()
	tc := NewTimeoutController(config)

	t.Run("connect timeout", func(t *testing.T) {
		ctx, cancel := tc.WithConnectTimeout(context.Background())
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("Context should have deadline")
		}
		if time.Until(deadline) > 60*time.Second {
			t.Error("Connect timeout should be around 60 seconds")
		}
	})

	t.Run("query timeout", func(t *testing.T) {
		ctx, cancel := tc.WithQueryTimeout(context.Background())
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("Context should have deadline")
		}
		if time.Until(deadline) > 30*time.Second {
			t.Error("Query timeout should be around 30 seconds")
		}
	})
}
