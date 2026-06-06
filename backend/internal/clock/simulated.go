package clock

import (
	"container/heap"
	"sync"
	"time"
)

// SimulatedClock is a deterministic clock for backtest replay.
// Time only advances when Advance() is called. All timers/tickers fire
// in deterministic order (determinism contract, M10-BASE-A5).
type SimulatedClock struct {
	mu        sync.Mutex
	now       time.Time
	seq       uint64
	events    *eventHeap
	tickers   []*simulatedTicker
	advancing bool
}

// NewSimulatedClock creates a clock starting at the given epoch.
func NewSimulatedClock(start time.Time) *SimulatedClock {
	return &SimulatedClock{
		now:    start,
		events: &eventHeap{},
	}
}

// Now returns the current simulated time.
func (c *SimulatedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// SetNow sets the current time directly.
func (c *SimulatedClock) SetNow(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

// Sleep advances the clock by d without processing events.
func (c *SimulatedClock) Sleep(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// Advance moves the clock forward to the next event time and fires all events
// scheduled up to (and including) that time. Returns the number of events fired.
func (c *SimulatedClock) Advance() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.advancing {
		return 0
	}
	c.advancing = true
	defer func() { c.advancing = false }()

	if c.events.Len() == 0 {
		return 0
	}

	target := (*c.events)[0].at
	if !target.After(c.now) && !target.Equal(c.now) {
		return 0
	}

	var fired int
	for c.events.Len() > 0 {
		ev := (*c.events)[0]
		if ev.at.After(target) {
			break
		}
		heap.Pop(c.events)
		c.now = ev.at
		fired++
		cb := ev.cb
		c.mu.Unlock()
		if cb != nil {
			cb()
		}
		c.mu.Lock()
	}
	return fired
}

// AdvanceBy moves the clock forward by d and fires all events in that window.
func (c *SimulatedClock) AdvanceBy(d time.Duration) int {
	c.mu.Lock()
	target := c.now.Add(d)
	c.mu.Unlock()
	c.fireUpTo(target)
	return 0
}

func (c *SimulatedClock) fireUpTo(target time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	var fired int
	for c.events.Len() > 0 {
		ev := (*c.events)[0]
		if ev.at.After(target) {
			break
		}
		heap.Pop(c.events)
		c.now = ev.at
		fired++
		cb := ev.cb
		c.mu.Unlock()
		if cb != nil {
			cb()
		}
		c.mu.Lock()
	}
	c.now = target
	return fired
}

func (c *SimulatedClock) schedule(at time.Time, cb func()) *simEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	ev := &simEvent{at: at, seq: c.seq, cb: cb}
	heap.Push(c.events, ev)
	return ev
}

func (c *SimulatedClock) removeEvent(ev *simEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ev.idx < 0 || ev.idx >= c.events.Len() {
		return
	}
	heap.Remove(c.events, ev.idx)
}

// NewTicker creates a simulated ticker.
func (c *SimulatedClock) NewTicker(d time.Duration) Ticker {
	t := &simulatedTicker{
		clock:    c,
		interval: d,
		ch:       make(chan time.Time, 1),
	}
	c.mu.Lock()
	c.tickers = append(c.tickers, t)
	c.mu.Unlock()
	t.start()
	return t
}

// NewTimer creates a simulated timer.
func (c *SimulatedClock) NewTimer(d time.Duration) Timer {
	at := c.Now().Add(d)
	ch := make(chan time.Time, 1)
	ev := c.schedule(at, func() {
		select {
		case ch <- at:
		default:
		}
	})
	return &simulatedTimer{clock: c, ev: ev, ch: ch}
}

// AfterFunc schedules f to run after duration d.
func (c *SimulatedClock) AfterFunc(d time.Duration, f func()) Timer {
	at := c.Now().Add(d)
	ev := c.schedule(at, f)
	return &simulatedTimer{clock: c, ev: ev, cb: f}
}
