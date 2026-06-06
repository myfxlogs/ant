// Package execalgo — AlgoExecutor runtime (M12-A1).
// The executor consumes a Schedule produced by an Algo and submits
// child orders to the broker at their TargetTime, integrating with
// MarketState and risk checks before each submission.
package execalgo

import (
	"context"
	"sync"
	"time"

	"anttrader/internal/mthub"
)

// ExecState represents the current state of an algo execution.
type ExecState int

const (
	ExecPending  ExecState = iota // not yet started
	ExecRunning                   // actively submitting slices
	ExecPaused                    // temporarily paused (market not tradeable)
	ExecCompleted                 // all slices submitted
	ExecCancelled                 // cancelled by user
	ExecFailed                    // unrecoverable error
)

func (s ExecState) String() string {
	switch s {
	case ExecPending:
		return "pending"
	case ExecRunning:
		return "running"
	case ExecPaused:
		return "paused"
	case ExecCompleted:
		return "completed"
	case ExecCancelled:
		return "cancelled"
	case ExecFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// IsTerminal returns true for states that will not transition further.
func (s ExecState) IsTerminal() bool {
	return s == ExecCompleted || s == ExecCancelled || s == ExecFailed
}

// ExecEvent is emitted when the executor's state or progress changes.
type ExecEvent struct {
	State       ExecState
	SliceIndex  int // -1 for state-only events
	TotalSlices int
	Ticket      int64 // broker ticket for the submitted slice
	Error       error // non-nil on submission failure
	Timestamp   time.Time
}

// ExecutorConfig holds dependencies for an algo execution.
type ExecutorConfig struct {
	Schedule    *Schedule
	Broker      mthub.BrokerExecutor
	AccountID   string
	MarketState MarketStateChecker

	// EventBufferSize sets the capacity of the event channel (default 16).
	EventBufferSize int
}

// MarketStateChecker abstracts the mdgateway MarketState for executor use.
type MarketStateChecker interface {
	IsTradeable(symbol string) (bool, string)
}

// Executor runs an algo execution schedule.
type Executor struct {
	cfg    ExecutorConfig
	state  ExecState
	mu     sync.Mutex
	events chan ExecEvent
	cancel context.CancelFunc

	// Progress tracking
	nextSlice   int
	submitted   int
	failedCount int
}

// NewExecutor creates an Executor with the given configuration.
func NewExecutor(cfg ExecutorConfig) *Executor {
	bufSize := cfg.EventBufferSize
	if bufSize <= 0 {
		bufSize = 16
	}
	return &Executor{
		cfg:       cfg,
		state:     ExecPending,
		events:    make(chan ExecEvent, bufSize),
		nextSlice: 0,
	}
}

// Events returns a read-only channel of execution events.
func (e *Executor) Events() <-chan ExecEvent { return e.events }

// State returns the current execution state (concurrency-safe).
func (e *Executor) State() ExecState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

// Progress returns (submitted, total) slice counts.
func (e *Executor) Progress() (submitted, total int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.submitted, len(e.cfg.Schedule.Slices)
}
