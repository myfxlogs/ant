// Package execalgo — Executor lifecycle methods (Start/Cancel/Pause/Resume).
package execalgo

import (
	"context"
	"time"
)

// Start begins executing the schedule. It returns immediately;
// execution proceeds in a background goroutine.
func (e *Executor) Start(ctx context.Context) {
	e.mu.Lock()
	if e.state != ExecPending {
		e.mu.Unlock()
		return
	}
	e.state = ExecRunning
	e.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	e.cancel = cancel

	// #nosec G118 — executor runs for algo execution lifetime
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

// Pause temporarily suspends slice submission.
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
