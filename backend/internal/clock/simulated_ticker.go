package clock

import (
	"sync"
	"time"
)

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
