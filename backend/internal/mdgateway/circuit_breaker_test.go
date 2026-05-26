package mdgateway

import (
	"testing"
	"time"
)

func TestCircuitBreakerDefaultThresholds(t *testing.T) {
	cb := NewCircuitBreaker(0, 0, 0)
	if cb.State() != StateClosed {
		t.Fatal("new circuit breaker must start in closed state")
	}
	if cb.failureThreshold != 5 {
		t.Fatalf("default failure threshold = %d, want 5", cb.failureThreshold)
	}
	if cb.successThreshold != 2 {
		t.Fatalf("default success threshold = %d, want 2", cb.successThreshold)
	}
}

func TestCircuitBreakerStateString(t *testing.T) {
	if s := StateClosed.String(); s != "closed" {
		t.Fatalf("StateClosed = %q", s)
	}
	if s := StateOpen.String(); s != "open" {
		t.Fatalf("StateOpen = %q", s)
	}
	if s := StateHalfOpen.String(); s != "half_open" {
		t.Fatalf("StateHalfOpen = %q", s)
	}
}

func TestCircuitBreakerClosedAllows(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, time.Second)
	for i := 0; i < 10; i++ {
		if !cb.Allow() {
			t.Fatalf("closed state must allow all calls, denied at call %d", i)
		}
	}
}

func TestCircuitBreakerTripsOpen(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, time.Second)
	// Record failures but stay under threshold.
	cb.OnFailure()
	cb.OnFailure()
	if cb.State() != StateClosed {
		t.Fatal("must stay closed below failure threshold")
	}
	// Third failure trips open.
	cb.OnFailure()
	if cb.State() != StateOpen {
		t.Fatalf("must trip open at failure threshold, got %s", cb.State())
	}
}

func TestCircuitBreakerOpenBlocks(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, 10*time.Second)
	cb.OnFailure()
	cb.OnFailure()
	if cb.Allow() {
		t.Fatal("open breaker must block calls")
	}
}

func TestCircuitBreakerHalfOpenProbe(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, 50*time.Millisecond)
	cb.OnFailure()
	cb.OnFailure()
	if cb.State() != StateOpen {
		t.Fatal("must be open after failures")
	}

	// Wait for cooldown.
	time.Sleep(60 * time.Millisecond)

	// First call after cooldown enters half-open.
	if !cb.Allow() {
		t.Fatal("half-open must allow probe call")
	}
	if cb.State() != StateHalfOpen {
		t.Fatalf("must transition to half-open after cooldown, got %s", cb.State())
	}
}

func TestCircuitBreakerHalfOpenClose(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, time.Millisecond)
	cb.OnFailure()
	cb.OnFailure()
	time.Sleep(5 * time.Millisecond)
	cb.Allow() // enters half-open

	cb.OnSuccess()
	if cb.State() != StateHalfOpen {
		t.Fatal("must stay half-open until success threshold met")
	}
	cb.OnSuccess()
	if cb.State() != StateClosed {
		t.Fatalf("must close after success threshold, got %s", cb.State())
	}
}

func TestCircuitBreakerHalfOpenFailureReopens(t *testing.T) {
	cb := NewCircuitBreaker(2, 2, time.Millisecond)
	cb.OnFailure()
	cb.OnFailure()
	time.Sleep(5 * time.Millisecond)
	cb.Allow() // enters half-open

	cb.OnFailure()
	if cb.State() != StateOpen {
		t.Fatalf("single failure in half-open must reopen, got %s", cb.State())
	}
}

func TestCircuitBreakerFailuresInHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(5, 3, time.Millisecond)
	// Trip open.
	for i := 0; i < 5; i++ {
		cb.OnFailure()
	}
	time.Sleep(5 * time.Millisecond)
	cb.Allow() // half-open

	// Fail in half-open → reopens immediately.
	cb.OnFailure()
	if cb.State() != StateOpen {
		t.Fatal("must reopen on half-open failure")
	}

	// Wait cooldown, probe succeeds.
	time.Sleep(5 * time.Millisecond)
	cb.Allow()

	// Three successes should close.
	for i := 0; i < 3; i++ {
		cb.OnSuccess()
	}
	if cb.State() != StateClosed {
		t.Fatalf("must close after success threshold, got %s", cb.State())
	}
}

func TestCircuitBreakerClosedSuccessResetsNothing(t *testing.T) {
	cb := NewCircuitBreaker(3, 2, time.Second)
	cb.OnSuccess()
	cb.OnSuccess()
	if cb.State() != StateClosed {
		t.Fatal("success in closed must not change state")
	}
}

func TestCircuitBreakerConcurrent(t *testing.T) {
	cb := NewCircuitBreaker(10, 5, 50*time.Millisecond)
	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				cb.Allow()
				cb.OnSuccess()
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	// All successes on closed → must still be closed.
	if cb.State() != StateClosed {
		t.Fatalf("concurrent success must leave breaker closed, got %s", cb.State())
	}
}

func TestCircuitBreakerConcurrentTripsOpen(t *testing.T) {
	cb := NewCircuitBreaker(10, 5, time.Second)
	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				cb.OnFailure()
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
	if cb.State() != StateOpen {
		t.Fatalf("concurrent failures must trip breaker, got %s", cb.State())
	}
}

func TestBrokerKey(t *testing.T) {
	k := BrokerKey("ic_markets", "mt4grpc3.mtapi.io", "443")
	if k != "ic_markets|mt4grpc3.mtapi.io:443" {
		t.Fatalf("BrokerKey = %q, want ic_markets|mt4grpc3.mtapi.io:443", k)
	}

	k2 := BrokerKey("oanda", "localhost", "8080")
	if k2 != "oanda|localhost:8080" {
		t.Fatalf("BrokerKey = %q", k2)
	}
}

func TestCircuitBreakerOpenedAt(t *testing.T) {
	before := time.Now()
	cb := NewCircuitBreaker(1, 1, time.Second)
	cb.OnFailure()

	if cb.openedAt.Before(before) {
		t.Fatal("openedAt must be >= Before failure")
	}
}

func TestCircuitBreakerCooldownNotElapsed(t *testing.T) {
	cb := NewCircuitBreaker(1, 1, 5*time.Second)
	cb.OnFailure()

	// Immediately check — cooldown not elapsed.
	if cb.Allow() {
		t.Fatal("must block when cooldown not elapsed")
	}
}
