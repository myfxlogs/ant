package faultisol

import (
	"context"
	"fmt"
	"sync"
	"time"

	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

type CircuitState int

const (
	StateClosed CircuitState = iota
	StateOpen
	StateHalfOpen
)

type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitState
	failureCount     int
	failureThreshold int
	successThreshold int
	successCount     int
	cooldownPeriod   time.Duration
	lastFailureTime  time.Time
	name             string
}

type CircuitBreakerConfig struct {
	Name             string
	FailureThreshold int
	SuccessThreshold int
	CooldownPeriod   time.Duration
}

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
		name:             cfg.Name,
		state:            StateClosed,
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		cooldownPeriod:   cfg.CooldownPeriod,
	}
}

func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureTime) > cb.cooldownPeriod {
			cb.state = StateHalfOpen
			cb.successCount = 0
			logger.Info("Circuit breaker entering half-open state",
				zap.String("name", cb.name))
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.successCount = 0
			logger.Info("Circuit breaker closed",
				zap.String("name", cb.name))
		}
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		cb.state = StateOpen
		logger.Warn("Circuit breaker reopened",
			zap.String("name", cb.name))
		return
	}

	if cb.failureCount >= cb.failureThreshold {
		cb.state = StateOpen
		logger.Warn("Circuit breaker opened",
			zap.String("name", cb.name),
			zap.Int("failure_count", cb.failureCount))
	}
}

type CircuitBreakerError struct {
	Name string
}

func (e *CircuitBreakerError) Error() string {
	return fmt.Sprintf("circuit breaker '%s' is open", e.Name)
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.Allow() {
		return &CircuitBreakerError{Name: cb.name}
	}

	err := fn()
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}

func (cb *CircuitBreaker) ExecuteWithContext(ctx context.Context, fn func(context.Context) error) error {
	if !cb.Allow() {
		return &CircuitBreakerError{Name: cb.name}
	}

	err := fn(ctx)
	if err != nil {
		cb.RecordFailure()
		return err
	}

	cb.RecordSuccess()
	return nil
}
