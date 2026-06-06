package clock

import (
	"time"
)

// --- simulated timer ---

type simulatedTimer struct {
	clock *SimulatedClock
	ev    *simEvent
	ch    chan time.Time
	cb    func()
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
	cb := t.cb
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
	idx int
}

type eventHeap []*simEvent

func (h eventHeap) Len() int      { return len(h) }
func (h eventHeap) Less(i, j int) bool {
	if h[i].at.Equal(h[j].at) {
		return h[i].seq < h[j].seq
	}
	return h[i].at.Before(h[j].at)
}
func (h eventHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i]; h[i].idx = i; h[j].idx = j }
func (h *eventHeap) Push(x any)   { ev := x.(*simEvent); ev.idx = len(*h); *h = append(*h, ev) }
func (h *eventHeap) Pop() any     { old := *h; n := len(old); ev := old[n-1]; old[n-1] = nil; ev.idx = -1; *h = old[:n-1]; return ev }
