package service

import (
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker open: engine temporarily unavailable")

type CircuitBreakerState int

const (
	CircuitClosed CircuitBreakerState = iota
	CircuitOpen
	CircuitHalfOpen
)

type CircuitBreaker struct {
	mu            sync.Mutex
	state         CircuitBreakerState
	failCount     int
	successCount  int
	threshold     int
	recoveryAfter time.Duration
	halfOpenMax   int
	openedAt      time.Time
}

func NewCircuitBreaker(threshold int, recoveryAfter time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:         CircuitClosed,
		threshold:     threshold,
		recoveryAfter: recoveryAfter,
		halfOpenMax:   1,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	if cb == nil {
		return true
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.openedAt) >= cb.recoveryAfter {
			cb.state = CircuitHalfOpen
			cb.successCount = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	return true
}

func (cb *CircuitBreaker) RecordSuccess() {
	if cb == nil {
		return
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.halfOpenMax {
			cb.state = CircuitClosed
			cb.failCount = 0
			cb.successCount = 0
		}
	case CircuitClosed:
		cb.failCount = 0
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	if cb == nil {
		return
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		cb.failCount++
		if cb.failCount >= cb.threshold {
			cb.state = CircuitOpen
			cb.openedAt = time.Now()
		}
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.openedAt = time.Now()
		cb.failCount = 0
	}
}

func (cb *CircuitBreaker) State() CircuitBreakerState {
	if cb == nil {
		return CircuitClosed
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == CircuitOpen && time.Since(cb.openedAt) >= cb.recoveryAfter {
		return CircuitHalfOpen
	}
	return cb.state
}

func (cb *CircuitBreaker) Reset() {
	if cb == nil {
		return
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = CircuitClosed
	cb.failCount = 0
	cb.successCount = 0
}
