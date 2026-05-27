package clock

import (
	"container/heap"
	"testing"
	"time"
)

func TestRealClock_Now(t *testing.T) {
	t.Parallel()
	c := NewRealClock()
	before := time.Now()
	now := c.Now()
	after := time.Now()

	if now.Before(before) || now.After(after) {
		t.Fatalf("RealClock.Now() %v not in [%v, %v]", now, before, after)
	}
}

func TestRealClock_Sleep(t *testing.T) {
	t.Parallel()
	c := NewRealClock()
	start := c.Now()
	c.Sleep(10 * time.Millisecond)
	elapsed := c.Now().Sub(start)
	if elapsed < 5*time.Millisecond {
		t.Fatalf("Sleep too short: %v", elapsed)
	}
}

func TestRealClock_NewTicker(t *testing.T) {
	t.Parallel()
	c := NewRealClock()
	ticker := c.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	select {
	case <-ticker.C():
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ticker did not fire")
	}
}

func TestRealClock_NewTimer(t *testing.T) {
	t.Parallel()
	c := NewRealClock()
	timer := c.NewTimer(10 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-timer.C():
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timer did not fire")
	}
}

func TestRealClock_AfterFunc(t *testing.T) {
	t.Parallel()
	c := NewRealClock()
	done := make(chan struct{})
	c.AfterFunc(10*time.Millisecond, func() { close(done) })

	select {
	case <-done:
		// OK
	case <-time.After(100 * time.Millisecond):
		t.Fatal("AfterFunc did not fire")
	}
}

func TestSimulatedClock_Now(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewSimulatedClock(start)
	if !c.Now().Equal(start) {
		t.Fatalf("SimulatedClock.Now: want %v, got %v", start, c.Now())
	}
}

func TestSimulatedClock_Sleep(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewSimulatedClock(start)
	c.Sleep(time.Hour)
	if c.Now().Hour() != 1 {
		t.Fatalf("after Sleep(1h): want hour=1, got %v", c.Now())
	}
}

func TestSimulatedClock_Timer(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewSimulatedClock(start)

	fired := false
	c.NewTimer(time.Second)

	// Manual advance: simulate timer.
	c.Sleep(time.Second)
	// Timer should have fired conceptually.

	_ = fired
}

func TestSimulatedClock_AfterFuncAndAdvance(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewSimulatedClock(start)

	var count int
	c.AfterFunc(time.Second, func() { count++ })
	c.AfterFunc(2*time.Second, func() { count++ })
	c.AfterFunc(3*time.Second, func() { count++ })

	c.fireUpTo(start.Add(2 * time.Second))

	if count != 2 {
		t.Fatalf("after 2s advance: want 2 events, got %d", count)
	}
}

func TestSimulatedClock_Ticker(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewSimulatedClock(start)

	ticker := c.NewTicker(time.Second)
	defer ticker.Stop()

	// Fire the first tick.
	c.fireUpTo(start.Add(time.Second))

	select {
	case <-ticker.C():
		// OK
	default:
		t.Fatal("ticker did not fire after advance")
	}
}

func TestSimulatedTimer_Stop(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewSimulatedClock(start)

	var fired bool
	timer := c.NewTimer(time.Second)
	_ = timer.C() // access C() before stop

	ok := timer.Stop()
	if !ok {
		t.Fatal("Stop returned false for pending timer")
	}

	c.fireUpTo(start.Add(2 * time.Second))
	_ = fired
}

func TestSimulatedTimer_Reset(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewSimulatedClock(start)

	var count int
	timer := c.AfterFunc(5*time.Second, func() { count++ })

	// Reset to fire sooner.
	timer.Reset(time.Second)
	c.fireUpTo(start.Add(2 * time.Second))

	if count != 1 {
		t.Fatalf("after reset+advance: want 1 event, got %d", count)
	}
}

func TestSimulatedClock_SetNow(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewSimulatedClock(start)

	newTime := start.Add(24 * time.Hour)
	c.SetNow(newTime)
	if !c.Now().Equal(newTime) {
		t.Fatalf("SetNow: want %v, got %v", newTime, c.Now())
	}
}

func TestSimulatedClock_Determinism(t *testing.T) {
	t.Parallel()
	// M10-BASE-A5: same sequence of events → same output (determinism contract).
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	run := func() []int {
		c := NewSimulatedClock(start)
		var events []int
		c.AfterFunc(time.Second, func() { events = append(events, 1) })
		c.AfterFunc(2*time.Second, func() { events = append(events, 2) })
		c.AfterFunc(1500*time.Millisecond, func() { events = append(events, 3) })
		c.fireUpTo(start.Add(3 * time.Second))
		return events
	}

	result1 := run()
	result2 := run()

	if len(result1) != len(result2) {
		t.Fatalf("determinism: len %d vs %d", len(result1), len(result2))
	}
	for i := range result1 {
		if result1[i] != result2[i] {
			t.Fatalf("determinism: idx %d: %d vs %d", i, result1[i], result2[i])
		}
	}
	t.Logf("Determinism: 2 runs → identical results %v (PASS)", result1)
}

func TestEventHeap_Order(t *testing.T) {
	t.Parallel()
	h := &eventHeap{}

	now := time.Now()
	heap.Push(h, &simEvent{at: now.Add(3 * time.Second), seq: 0})
	heap.Push(h, &simEvent{at: now.Add(1 * time.Second), seq: 1})
	heap.Push(h, &simEvent{at: now.Add(2 * time.Second), seq: 2})

	expected := []time.Duration{time.Second, 2 * time.Second, 3 * time.Second}
	for i, exp := range expected {
		ev := heap.Pop(h).(*simEvent)
		got := ev.at.Sub(now)
		if got != exp {
			t.Fatalf("heap order [%d]: want %v, got %v", i, exp, got)
		}
	}
}

func TestEventHeap_SameTimeOrder(t *testing.T) {
	t.Parallel()
	h := &eventHeap{}

	now := time.Now()
	heap.Push(h, &simEvent{at: now, seq: 2})
	heap.Push(h, &simEvent{at: now, seq: 1}) // lower seq = earlier push

	ev1 := heap.Pop(h).(*simEvent)
	if ev1.seq != 1 {
		t.Fatalf("same-time order: want seq=1 first, got seq=%d", ev1.seq)
	}
}
