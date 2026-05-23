package faultisol

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type IsolatedExecutor struct {
	name           string
	circuitBreaker *CircuitBreaker
	timeoutConfig  TimeoutConfig
	fields         []zap.Field
	onPanic        func(interface{})
	onCircuitOpen  func()
}

type IsolatedExecutorOption func(*IsolatedExecutor)

func WithCircuitBreaker(cb *CircuitBreaker) IsolatedExecutorOption {
	return func(e *IsolatedExecutor) {
		e.circuitBreaker = cb
	}
}

func WithTimeoutConfig(config TimeoutConfig) IsolatedExecutorOption {
	return func(e *IsolatedExecutor) {
		e.timeoutConfig = config
	}
}

func WithExecutorFields(fields ...zap.Field) IsolatedExecutorOption {
	return func(e *IsolatedExecutor) {
		e.fields = fields
	}
}

func WithPanicCallback(callback func(interface{})) IsolatedExecutorOption {
	return func(e *IsolatedExecutor) {
		e.onPanic = callback
	}
}

func WithCircuitOpenCallback(callback func()) IsolatedExecutorOption {
	return func(e *IsolatedExecutor) {
		e.onCircuitOpen = callback
	}
}

func NewIsolatedExecutor(name string, opts ...IsolatedExecutorOption) *IsolatedExecutor {
	e := &IsolatedExecutor{
		name:          name,
		timeoutConfig: DefaultTimeoutConfig(),
		fields:        []zap.Field{},
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

type ExecuteOptions struct {
	Timeout   time.Duration
	Operation string
}

type ExecuteOption func(*ExecuteOptions)

func WithTimeout(timeout time.Duration) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Timeout = timeout
	}
}

func WithOperation(op string) ExecuteOption {
	return func(o *ExecuteOptions) {
		o.Operation = op
	}
}

func (e *IsolatedExecutor) Execute(ctx context.Context, fn func(context.Context) error, opts ...ExecuteOption) error {
	options := &ExecuteOptions{
		Timeout:   e.timeoutConfig.Default,
		Operation: e.name,
	}
	for _, opt := range opts {
		opt(options)
	}

	if e.circuitBreaker != nil && !e.circuitBreaker.Allow() {
		if e.onCircuitOpen != nil {
			e.onCircuitOpen()
		}
		return &CircuitBreakerError{Name: e.name}
	}

	ctx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()

	result := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic recovered: %v", r)
				if e.circuitBreaker != nil {
					e.circuitBreaker.RecordFailure()
				}
				if e.onPanic != nil {
					e.onPanic(r)
				}
				result <- err
			}
		}()

		result <- fn(ctx)
	}()

	select {
	case err := <-result:
		if err != nil {
			if e.circuitBreaker != nil {
				e.circuitBreaker.RecordFailure()
			}
			return err
		}
		if e.circuitBreaker != nil {
			e.circuitBreaker.RecordSuccess()
		}
		return nil
	case <-ctx.Done():
		if e.circuitBreaker != nil {
			e.circuitBreaker.RecordFailure()
		}
		return &TimeoutError{
			Operation: options.Operation,
			Timeout:   options.Timeout,
		}
	}
}

func (e *IsolatedExecutor) ExecuteWithValue(ctx context.Context, fn func(context.Context) (interface{}, error), opts ...ExecuteOption) (interface{}, error) {
	options := &ExecuteOptions{
		Timeout:   e.timeoutConfig.Default,
		Operation: e.name,
	}
	for _, opt := range opts {
		opt(options)
	}

	if e.circuitBreaker != nil && !e.circuitBreaker.Allow() {
		if e.onCircuitOpen != nil {
			e.onCircuitOpen()
		}
		return nil, &CircuitBreakerError{Name: e.name}
	}

	ctx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()

	type result struct {
		value interface{}
		err   error
	}

	resultChan := make(chan result, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic recovered: %v", r)
				if e.circuitBreaker != nil {
					e.circuitBreaker.RecordFailure()
				}
				if e.onPanic != nil {
					e.onPanic(r)
				}
				resultChan <- result{nil, err}
			}
		}()

		value, err := fn(ctx)
		resultChan <- result{value, err}
	}()

	select {
	case res := <-resultChan:
		if res.err != nil {
			if e.circuitBreaker != nil {
				e.circuitBreaker.RecordFailure()
			}
			return nil, res.err
		}
		if e.circuitBreaker != nil {
			e.circuitBreaker.RecordSuccess()
		}
		return res.value, nil
	case <-ctx.Done():
		if e.circuitBreaker != nil {
			e.circuitBreaker.RecordFailure()
		}
		return nil, &TimeoutError{
			Operation: options.Operation,
			Timeout:   options.Timeout,
		}
	}
}
