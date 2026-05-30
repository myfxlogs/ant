// Package execalgo — AlgoExecutor runtime (M12-A1).
// The executor consumes a Schedule produced by an Algo and submits
// child orders to the broker at their TargetTime, integrating with
// MarketState and risk checks before each submission.
package execalgo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"

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
	Ticket      int64      // broker ticket for the submitted slice
	Error       error      // non-nil on submission failure
	Timestamp   time.Time
}

// ExecutorConfig holds the dependencies needed to run an algo execution.
type ExecutorConfig struct {
	// Schedule is the execution plan produced by an Algo.
	Schedule *Schedule

	// Broker submits child orders. Must be non-nil.
	Broker mthub.BrokerExecutor

	// AccountID is the target trading account.
	AccountID string

	// MarketState is checked before each slice submission.
	// If nil, tradability checks are skipped.
	MarketState MarketStateChecker

	// EventBufferSize sets the capacity of the event channel (default 16).
	EventBufferSize int
}

// MarketStateChecker abstracts the mdgateway MarketState for executor use.
// The executor only needs to know if a symbol is tradeable.
type MarketStateChecker interface {
	IsTradeable(symbol string) (bool, string)
}

// Executor runs an algo execution schedule, submitting child orders
// to the broker at their TargetTime.
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
// The executor does not start until Start() is called.
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
// The channel is closed when the executor reaches a terminal state.
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

// Start begins executing the schedule. It returns immediately;
// execution proceeds in a background goroutine.
// The provided context controls the lifetime of the execution —
// cancelling it will cancel the executor.
func (e *Executor) Start(ctx context.Context) {
	e.mu.Lock()
	if e.state != ExecPending {
		e.mu.Unlock()
		return // already started
	}
	e.state = ExecRunning
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel

	go e.run(ctx)
}

// Cancel stops the executor. Already-submitted orders are not cancelled;
// only future slices are prevented.
func (e *Executor) Cancel() {
	e.mu.Lock()
	if e.state.IsTerminal() {
		e.mu.Unlock()
		return
	}
	e.state = ExecCancelled
	e.mu.Unlock()
	e.emit(ExecEvent{State: ExecCancelled, SliceIndex: -1, TotalSlices: len(e.cfg.Schedule.Slices), Timestamp: time.Now()})
	if e.cancel != nil {
		e.cancel()
	}
}

// Pause temporarily suspends slice submission. Running the
// executor continues from the next unsent slice.
func (e *Executor) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state == ExecRunning {
		e.state = ExecPaused
		e.emit(ExecEvent{State: ExecPaused, SliceIndex: -1, TotalSlices: len(e.cfg.Schedule.Slices), Timestamp: time.Now()})
	}
}

// Resume continues a paused executor.
func (e *Executor) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state == ExecPaused {
		e.state = ExecRunning
		e.emit(ExecEvent{State: ExecRunning, SliceIndex: -1, TotalSlices: len(e.cfg.Schedule.Slices), Timestamp: time.Now()})
	}
}

// run is the main execution loop.
func (e *Executor) run(ctx context.Context) {
	defer close(e.events)

	slices := e.cfg.Schedule.Slices
	total := len(slices)

	e.emit(ExecEvent{State: ExecRunning, SliceIndex: -1, TotalSlices: total, Timestamp: time.Now()})

	for i := 0; i < total; i++ {
		// Check if we should stop.
		select {
		case <-ctx.Done():
			e.transitionTo(ExecCancelled)
			return
		default:
		}

		e.mu.Lock()
		if e.state == ExecCancelled || e.state == ExecFailed {
			e.mu.Unlock()
			return
		}
		e.mu.Unlock()

		slice := slices[i]
		now := time.Now()

		// Wait until TargetTime, or check context.
		if slice.TargetTime.After(now) {
			waitDur := slice.TargetTime.Sub(now)
			timer := time.NewTimer(waitDur)
			select {
			case <-ctx.Done():
				timer.Stop()
				e.transitionTo(ExecCancelled)
				return
			case <-timer.C:
				// proceed
			}
		}

		// Check market state before submitting.
		if e.cfg.MarketState != nil {
			tradeable, reason := e.cfg.MarketState.IsTradeable(e.cfg.Schedule.Parent.Symbol)
			if !tradeable {
				e.emit(ExecEvent{
					State: ExecPaused, SliceIndex: i, TotalSlices: total,
					Error: fmt.Errorf("market not tradeable: %s", reason), Timestamp: time.Now(),
				})
				// Wait and retry periodically until tradeable or cancelled.
				retryTicker := time.NewTicker(5 * time.Second)
				defer retryTicker.Stop()
				for !tradeable {
					select {
					case <-ctx.Done():
						e.transitionTo(ExecCancelled)
						return
					case <-retryTicker.C:
						tradeable, reason = e.cfg.MarketState.IsTradeable(e.cfg.Schedule.Parent.Symbol)
					}
				}
				e.emit(ExecEvent{State: ExecRunning, SliceIndex: i, TotalSlices: total, Timestamp: time.Now()})
			}
		}

		// Submit the child order.
		req := &mthub.OrderRequest{
			AccountID: e.cfg.AccountID,
			Canonical: e.cfg.Schedule.Parent.Symbol,
			Side:      sideToMthub(e.cfg.Schedule.Parent.Side),
			OrderType: mthub.OrderMarket,
			Volume:    decimal.NewFromFloat(slice.Volume),
		}
		if slice.LimitPrice > 0 {
			req.OrderType = mthub.OrderLimit
			req.Price = decimal.NewFromFloat(slice.LimitPrice)
		}

		ticket, err := e.cfg.Broker.SubmitOrder(ctx, req)
		if err != nil {
			e.mu.Lock()
			e.failedCount++
			e.mu.Unlock()
			e.emit(ExecEvent{
				State: ExecRunning, SliceIndex: i, TotalSlices: total,
				Error: fmt.Errorf("slice %d submit: %w", i, err), Timestamp: time.Now(),
			})
			// Continue to next slice on failure — don't abort the whole schedule.
			continue
		}

		e.mu.Lock()
		e.submitted++
		e.nextSlice = i + 1
		e.mu.Unlock()

		e.emit(ExecEvent{
			State: ExecRunning, SliceIndex: i, TotalSlices: total,
			Ticket: ticket, Timestamp: time.Now(),
		})
	}

	e.transitionTo(ExecCompleted)
}

// transitionTo sets the state and emits an event.
func (e *Executor) transitionTo(s ExecState) {
	e.mu.Lock()
	e.state = s
	e.mu.Unlock()
	e.emit(ExecEvent{State: s, SliceIndex: -1, TotalSlices: len(e.cfg.Schedule.Slices), Timestamp: time.Now()})
}

// emit sends an event to the channel. Non-blocking: if the channel is full, the event is dropped.
func (e *Executor) emit(ev ExecEvent) {
	select {
	case e.events <- ev:
	default:
		// drop event if consumer is slow
	}
}

// sideToMthub converts execalgo side string to mthub.Side.
func sideToMthub(side string) mthub.Side {
	if side == "sell" {
		return mthub.SideSell
	}
	return mthub.SideBuy
}
