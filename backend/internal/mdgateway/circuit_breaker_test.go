package mdgateway

import (
	"context"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	if cb == nil {
		t.Fatal("NewCircuitBreaker returned nil")
	}
	if cb.State() != CircuitClosed {
		t.Fatal("expected Closed state initially")
	}
}

func TestCircuitBreaker_OpenOnFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		CooldownPeriod:   100 * time.Millisecond,
	})

	// Allow initially
	if !cb.Allow() {
		t.Fatal("should allow initially")
	}

	// Record 3 failures → open
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if cb.Allow() {
		t.Fatal("should not allow after 3 failures")
	}
	if cb.State() != CircuitOpen {
		t.Fatal("should be Open")
	}
}

func TestCircuitBreaker_HalfOpenAfterCooldown(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		CooldownPeriod:   50 * time.Millisecond,
	})

	cb.RecordFailure()
	if cb.Allow() {
		t.Fatal("should be open after failure")
	}

	time.Sleep(100 * time.Millisecond)

	// After cooldown, should half-open
	if !cb.Allow() {
		t.Fatal("should half-open after cooldown")
	}
	if cb.State() != CircuitHalfOpen {
		t.Fatalf("expected HalfOpen, got %d", cb.State())
	}
}

func TestCircuitBreaker_CloseAfterSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		CooldownPeriod:   10 * time.Millisecond,
	})

	cb.RecordFailure()
	time.Sleep(20 * time.Millisecond)
	cb.Allow() // half-open

	cb.RecordSuccess()
	cb.RecordSuccess()

	if cb.State() != CircuitClosed {
		t.Fatalf("expected Closed after 2 successes, got %d", cb.State())
	}
}

func TestManager_CircuitBreaker(t *testing.T) {
	m := NewEmptyManager()
	cfg := AccountConfig{
		Broker:   "demo",
		Platform: "mt4",
		Login:    "12345",
		Password: "pw",
		Server:   "srv",
		Host:     "h",
		Port:     "443",
	}
	m.AddGateway(cfg)

	if m.CircuitState("demo-12345") != CircuitClosed {
		t.Fatal("expected CircuitClosed initially")
	}

	// Verify gateway exists
	conns := m.Connections()
	if _, ok := conns["demo-12345"]; !ok {
		t.Fatal("gateway not found")
	}

	// Remove gateway also cleans breaker
	m.RemoveGateway("demo-12345")
	if len(m.Connections()) != 0 {
		t.Fatal("gateway not removed")
	}
}

func TestManager_ConnectGateway_NotFound(t *testing.T) {
	m := NewEmptyManager()
	err := m.ConnectGateway(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing gateway")
	}
}

func TestManager_HealthCheckGateway_NotFound(t *testing.T) {
	m := NewEmptyManager()
	err := m.HealthCheckGateway(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing gateway")
	}
}

func TestManager_SetMetrics(t *testing.T) {
	m := NewEmptyManager()
	metrics := NewMDMetrics(nil)
	m.SetMetrics(metrics)
	if m.metrics == nil {
		t.Fatal("metrics should be set")
	}
}
