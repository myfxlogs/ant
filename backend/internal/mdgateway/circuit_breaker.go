package mdgateway

import (
	"fmt"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed: return "closed"
	case StateOpen: return "open"
	case StateHalfOpen: return "half_open"
	default: return "unknown"
	}
}

// CircuitBreaker implements a sliding-window circuit breaker per broker endpoint.
// Scoped per-broker (not per-account) because failure correlation is at the
// network layer (Netflix Hystrix / resilience4j pattern).
type CircuitBreaker struct {
	failureThreshold int
	successThreshold int
	cooldown         time.Duration

	mu            sync.Mutex
	state         State
	failures      int
	successes     int
	openedAt      time.Time
	onStateChange func(from, to State)
}

func NewCircuitBreaker(failureThreshold, successThreshold int, cooldown time.Duration) *CircuitBreaker {
	if failureThreshold <= 0 { failureThreshold = 5 }
	if successThreshold <= 0 { successThreshold = 2 }
	if cooldown <= 0 { cooldown = 30 * time.Second }
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		cooldown:         cooldown,
		state:            StateClosed,
	}
}

func (c *CircuitBreaker) Allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.state {
	case StateClosed: return true
	case StateOpen:
		if time.Since(c.openedAt) >= c.cooldown {
			prev := c.state
			c.state = StateHalfOpen
			c.successes = 0
			c.fireStateChange(prev, StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen: return true
	default: return false
	}
}

func (c *CircuitBreaker) OnSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.state == StateHalfOpen {
		c.successes++
		if c.successes >= c.successThreshold {
			prev := c.state
			c.state = StateClosed
			c.failures = 0
			c.fireStateChange(prev, StateClosed)
		}
	}
}

func (c *CircuitBreaker) OnFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures++
	if c.state == StateHalfOpen {
		prev := c.state
		c.state = StateOpen
		c.openedAt = Clk.Now()
		c.fireStateChange(prev, StateOpen)
	} else if c.failures >= c.failureThreshold && c.state == StateClosed {
		prev := c.state
		c.state = StateOpen
		c.openedAt = Clk.Now()
		c.fireStateChange(prev, StateOpen)
	}
}

func (c *CircuitBreaker) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// SetOnStateChange registers a callback fired on state transitions.
// Called outside the lock to avoid deadlocks.
func (c *CircuitBreaker) SetOnStateChange(fn func(from, to State)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onStateChange = fn
}

// fireStateChange invokes the state-change callback. Must be called while holding c.mu.
func (c *CircuitBreaker) fireStateChange(from, to State) {
	if c.onStateChange != nil && from != to {
		cb := c.onStateChange
		go cb(from, to)
	}
}

// BrokerKey constructs the circuit breaker key from broker identity.
func BrokerKey(broker, host, port string) string {
	return fmt.Sprintf("%s|%s:%s", broker, host, port)
}
