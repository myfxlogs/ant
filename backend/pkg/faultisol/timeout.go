package faultisol

import (
	"context"
	"fmt"
	"time"
)

type TimeoutConfig struct {
	Connect   time.Duration
	Query     time.Duration
	Trading   time.Duration
	Subscribe time.Duration
	Default   time.Duration
}

func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Connect:   60 * time.Second,
		Query:     30 * time.Second,
		Trading:   30 * time.Second,
		Subscribe: 10 * time.Second,
		Default:   30 * time.Second,
	}
}

type TimeoutController struct {
	config TimeoutConfig
}

func NewTimeoutController(config TimeoutConfig) *TimeoutController {
	return &TimeoutController{config: config}
}

func (tc *TimeoutController) WithConnectTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, tc.config.Connect)
}

func (tc *TimeoutController) WithQueryTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, tc.config.Query)
}

func (tc *TimeoutController) WithTradingTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, tc.config.Trading)
}

func (tc *TimeoutController) WithSubscribeTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, tc.config.Subscribe)
}

func (tc *TimeoutController) WithDefaultTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, tc.config.Default)
}

type TimeoutError struct {
	Operation string
	Timeout   time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("operation '%s' timed out after %v", e.Operation, e.Timeout)
}

func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*TimeoutError)
	return ok
}

func WrapWithTimeout(ctx context.Context, operation string, timeout time.Duration, fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- fn(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return &TimeoutError{
			Operation: operation,
			Timeout:   timeout,
		}
	}
}
