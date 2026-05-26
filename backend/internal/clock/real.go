package clock

import (
	"sync"
	"time"
)

// RealClock is the production Clock backed by the system wall clock.
type RealClock struct{}

// NewRealClock creates a real wall-clock instance.
func NewRealClock() *RealClock { return &RealClock{} }

func (RealClock) Now() time.Time                     { return time.Now() }
func (RealClock) Sleep(d time.Duration)               { time.Sleep(d) }
func (RealClock) NewTicker(d time.Duration) Ticker    { return &realTicker{T: time.NewTicker(d)} }
func (RealClock) NewTimer(d time.Duration) Timer      { return &realTimer{T: time.NewTimer(d)} }
func (RealClock) AfterFunc(d time.Duration, f func()) Timer {
	return &realTimer{T: time.AfterFunc(d, f)}
}

type realTicker struct {
	T    *time.Ticker
	mu   sync.Mutex
	done bool
}

func (t *realTicker) C() <-chan time.Time { return t.T.C }
func (t *realTicker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.done {
		t.T.Stop()
		t.done = true
	}
}
func (t *realTicker) Reset(d time.Duration) { t.T.Reset(d) }

type realTimer struct {
	T    *time.Timer
	mu   sync.Mutex
	done bool
	fn   func() // for AfterFunc
}

func (t *realTimer) C() <-chan time.Time { return t.T.C }
func (t *realTimer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.done {
		t.done = true
		return t.T.Stop()
	}
	return false
}
func (t *realTimer) Reset(d time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.done {
		return t.T.Reset(d)
	}
	return false
}
