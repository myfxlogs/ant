package clock

import (
	"container/heap"
	"sync"
	"time"
)

// SimulatedClock is a deterministic clock for backtest replay.
// Time only advances when Advance() is called. All timers/tickers fire
// in deterministic order — the same sequence of events always produces
// the same output (determinism contract, M10-BASE-A5).
type SimulatedClock struct {
	mu       sync.Mutex
	now      time.Time
	seq      uint64             // tiebreaker for deterministic ordering
	events   *eventHeap          // pending timer events
	tickers  []*simulatedTicker // active tickers
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

// SetNow sets the current time directly (for rewinding/snapshot restore).
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
		return 0 // prevent re-entrant advance from timer callbacks
	}
	c.advancing = true
	defer func() { c.advancing = false }()

	if c.events.Len() == 0 {
		return 0
	}

	// Find the earliest event time.
	target := (*c.events)[0].at
	if !target.After(c.now) && !target.Equal(c.now) {
		return 0
	}

	// Fire all events at or before the target time.
	var fired int
	for c.events.Len() > 0 {
		ev := (*c.events)[0]
		if ev.at.After(target) {
			break
		}
		heap.Pop(c.events)
		c.now = ev.at
		fired++

		// Fire the callback (without holding lock to avoid deadlock).
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

// --- simulated ticker ---

type simulatedTicker struct {
	clock    *SimulatedClock
	interval time.Duration
	ch       chan time.Time
	ev       *simEvent
	stopped  bool
	mu       sync.Mutex
}

func (t *simulatedTicker) start() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return
	}
	at := t.clock.Now().Add(t.interval)
	t.ev = t.clock.schedule(at, func() {
		select {
		case t.ch <- at:
		default:
		}
		// Re-schedule for next tick.
		t.mu.Lock()
		if !t.stopped {
			next := at.Add(t.interval)
			t.ev = t.clock.schedule(next, func() {
				select {
				case t.ch <- next:
				default:
				}
			})
		}
		t.mu.Unlock()
	})
}

func (t *simulatedTicker) C() <-chan time.Time { return t.ch }
func (t *simulatedTicker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopped = true
	if t.ev != nil {
		t.clock.removeEvent(t.ev)
		t.ev = nil
	}
}
func (t *simulatedTicker) Reset(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ev != nil {
		t.clock.removeEvent(t.ev)
	}
	t.interval = d
	if !t.stopped {
		at := t.clock.Now().Add(d)
		t.ev = t.clock.schedule(at, func() {
			select {
			case t.ch <- at:
			default:
			}
		})
	}
}

// --- simulated timer ---

type simulatedTimer struct {
	clock *SimulatedClock
	ev    *simEvent
	ch    chan time.Time
	cb    func() // preserved callback for AfterFunc (nil for NewTimer)
}

func (t *simulatedTimer) C() <-chan time.Time {
	if t.ch == nil {
		return nil
	}
	return t.ch
}
func (t *simulatedTimer) Stop() bool {
	if t.ev == nil {
		return false
	}
	t.clock.removeEvent(t.ev)
	t.ev = nil
	return true
}
func (t *simulatedTimer) Reset(d time.Duration) bool {
	if t.ev != nil {
		t.clock.removeEvent(t.ev)
	}
	at := t.clock.Now().Add(d)
	cb := t.cb // preserve original AfterFunc callback
	if cb == nil && t.ch != nil {
		ch := t.ch
		cb = func() {
			select {
			case ch <- at:
			default:
			}
		}
	}
	t.ev = t.clock.schedule(at, cb)
	return true
}

// --- event priority queue ---

type simEvent struct {
	at  time.Time
	seq uint64
	cb  func()
	idx int // index in heap
}

type eventHeap []*simEvent

func (h eventHeap) Len() int            { return len(h) }
func (h eventHeap) Less(i, j int) bool   {
	if h[i].at.Equal(h[j].at) {
		return h[i].seq < h[j].seq
	}
	return h[i].at.Before(h[j].at)
}
func (h eventHeap) Swap(i, j int)        { h[i], h[j] = h[j], h[i]; h[i].idx = i; h[j].idx = j }
func (h *eventHeap) Push(x any)          { ev := x.(*simEvent); ev.idx = len(*h); *h = append(*h, ev) }
func (h *eventHeap) Pop() any            { old := *h; n := len(old); ev := old[n-1]; old[n-1] = nil; ev.idx = -1; *h = old[:n-1]; return ev }
