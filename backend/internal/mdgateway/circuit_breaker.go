// Package mdgateway — circuit breaker for gateway connections.
// Implements a simple sliding-window circuit breaker that opens after
// consecutive failures and half-opens after a cooldown period.
package mdgateway

import (
	"sync"
	"time"
)

// CircuitState represents the breaker state.
type CircuitState int

const (
	CircuitClosed   CircuitState = iota // normal operation
	CircuitOpen                         // rejecting requests
	CircuitHalfOpen                     // testing recovery
)

// CircuitBreakerConfig holds breaker thresholds.
type CircuitBreakerConfig struct {
	FailureThreshold int           // consecutive failures to open (default 5)
	SuccessThreshold int           // consecutive successes to close (default 2)
	CooldownPeriod   time.Duration // wait before half-open (default 30s)
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		CooldownPeriod:   30 * time.Second,
	}
}

// CircuitBreaker protects a gateway from cascading failures.
type CircuitBreaker struct {
	cfg CircuitBreakerConfig

	mu             sync.Mutex
	state          CircuitState
	failureCount   int
	successCount   int
	lastFailureAt  time.Time
	lastStateChange time.Time
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.CooldownPeriod <= 0 {
		cfg.CooldownPeriod = 30 * time.Second
	}
	return &CircuitBreaker{
		cfg:   cfg,
		state: CircuitClosed,
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Allow returns true if the request should proceed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastStateChange) >= cb.cfg.CooldownPeriod {
			cb.state = CircuitHalfOpen
			cb.lastStateChange = time.Now()
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	return false
}

// RecordSuccess reports a successful operation.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount = 0
	case CircuitHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.cfg.SuccessThreshold {
			cb.state = CircuitClosed
			cb.failureCount = 0
			cb.successCount = 0
			cb.lastStateChange = time.Now()
		}
	}
}

// RecordFailure reports a failed operation.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureAt = time.Now()

	switch cb.state {
	case CircuitClosed:
		cb.failureCount++
		if cb.failureCount >= cb.cfg.FailureThreshold {
			cb.state = CircuitOpen
			cb.successCount = 0
			cb.lastStateChange = time.Now()
		}
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.successCount = 0
		cb.lastStateChange = time.Now()
	}
}
