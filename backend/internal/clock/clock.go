// Package clock provides a Clock interface for deterministic time control (M10-BASE-A5).
//
// All production code must use Clock instead of direct time.Now/time.Sleep calls.
// This enables deterministic backtest replay — a simulated clock can fast-forward
// through historical data while producing identical results on every run.
package clock

import "time"

// Clock is the single time source for all ant subsystems.
// Implementations: RealClock (wall clock), SimulatedClock (backtest replay).
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// Sleep pauses the current goroutine for the given duration.
	Sleep(d time.Duration)

	// NewTicker creates a Ticker that fires at intervals.
	NewTicker(d time.Duration) Ticker

	// NewTimer creates a Timer that fires once after d.
	NewTimer(d time.Duration) Timer

	// AfterFunc calls f in a new goroutine after duration d.
	AfterFunc(d time.Duration, f func()) Timer
}

// Ticker is the Clock equivalent of time.Ticker.
type Ticker interface {
	C() <-chan time.Time
	Stop()
	Reset(d time.Duration)
}

// Timer is the Clock equivalent of time.Timer.
type Timer interface {
	C() <-chan time.Time
	Stop() bool
	Reset(d time.Duration) bool
}
