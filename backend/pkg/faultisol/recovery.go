package faultisol

import (
	"fmt"

	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

type RecoveryResult struct {
	Value interface{}
	Error error
	Panic interface{}
}

func SafeCall(fn func() (interface{}, error)) *RecoveryResult {
	result := &RecoveryResult{}

	defer func() {
		if r := recover(); r != nil {
			result.Panic = r
			result.Error = fmt.Errorf("panic recovered: %v", r)
			logger.Error("Panic recovered in safe call",
				zap.Any("panic", r))
		}
	}()

	value, err := fn()
	result.Value = value
	result.Error = err
	return result
}

func SafeCallWithLog(fn func() (interface{}, error), contextFields ...zap.Field) *RecoveryResult {
	result := &RecoveryResult{}

	defer func() {
		if r := recover(); r != nil {
			result.Panic = r
			result.Error = fmt.Errorf("panic recovered: %v", r)

			fields := append(contextFields,
				zap.Any("panic", r),
				zap.String("error_type", "panic"))
			logger.Error("Panic recovered in safe call", fields...)
		}
	}()

	value, err := fn()
	result.Value = value
	result.Error = err
	return result
}

type SafeExecutor struct {
	name        string
	extraFields []zap.Field
	onPanic     func(interface{})
}

type SafeExecutorOption func(*SafeExecutor)

func WithName(name string) SafeExecutorOption {
	return func(e *SafeExecutor) {
		e.name = name
	}
}

func WithFields(fields ...zap.Field) SafeExecutorOption {
	return func(e *SafeExecutor) {
		e.extraFields = fields
	}
}

func WithPanicHandler(handler func(interface{})) SafeExecutorOption {
	return func(e *SafeExecutor) {
		e.onPanic = handler
	}
}

func NewSafeExecutor(opts ...SafeExecutorOption) *SafeExecutor {
	e := &SafeExecutor{
		name: "default",
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *SafeExecutor) Execute(fn func() error) error {
	defer func() {
		if r := recover(); r != nil {
			fields := append(e.extraFields,
				zap.String("executor", e.name),
				zap.Any("panic", r))
			logger.Error("Panic recovered", fields...)

			if e.onPanic != nil {
				e.onPanic(r)
			}
		}
	}()

	return fn()
}

func (e *SafeExecutor) ExecuteWithValue(fn func() (interface{}, error)) *RecoveryResult {
	result := &RecoveryResult{}

	defer func() {
		if r := recover(); r != nil {
			result.Panic = r
			result.Error = fmt.Errorf("panic recovered: %v", r)

			fields := append(e.extraFields,
				zap.String("executor", e.name),
				zap.Any("panic", r))
			logger.Error("Panic recovered", fields...)

			if e.onPanic != nil {
				e.onPanic(r)
			}
		}
	}()

	value, err := fn()
	result.Value = value
	result.Error = err
	return result
}

type RecoverableConnection interface {
	MarkDisconnected()
	GetAccountID() string
}

func SafeConnectionCall(conn RecoverableConnection, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
			conn.MarkDisconnected()
			logger.Error("Connection panic recovered",
				zap.String("account_id", conn.GetAccountID()),
				zap.Any("panic", r))
		}
	}()

	err = fn()
	return err
}
