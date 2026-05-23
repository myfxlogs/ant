package mthub

import (
	"sync"
	"testing"
	"time"
)

// ── Subscribe/Unsubscribe lifecycle ──

func TestNewOrderEventBroker(t *testing.T) {
	b := NewOrderEventBroker()
	if b == nil {
		t.Fatal("NewOrderEventBroker returned nil")
	}
	if b.SubscriberCount() != 0 {
		t.Fatalf("expected 0 subscribers, got %d", b.SubscriberCount())
	}
}

func TestOrderEventBroker_SubscribeUnsubscribe(t *testing.T) {
	b := NewOrderEventBroker()

	ch := b.Subscribe("acc-1")
	if ch == nil {
		t.Fatal("expected non-nil channel from Subscribe")
	}
	if b.SubscriberCount() != 1 {
		t.Errorf("SubscriberCount = %d, want 1", b.SubscriberCount())
	}

	b.Unsubscribe("acc-1")
	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount = %d after unsubscribe, want 0", b.SubscriberCount())
	}

	// Channel should be closed after unsubscribe.
	_, ok := <-ch
	if ok {
		t.Error("expected channel to be closed after unsubscribe")
	}
}

// ── Publish: success + silent drop without subscriber ──

func TestOrderEventBroker_Publish(t *testing.T) {
	b := NewOrderEventBroker()
	ch := b.Subscribe("acc-1")

	ev := &OrderEvent{AccountId: "acc-1", Type: "order_update"}
	b.Publish(ev)

	select {
	case received := <-ch:
		if received.AccountId != "acc-1" {
			t.Errorf("AccountId = %s, want acc-1", received.AccountId)
		}
		if received.Type != "order_update" {
			t.Errorf("Type = %s, want order_update", received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected to receive event, but timed out")
	}

	b.Unsubscribe("acc-1")
}

func TestOrderEventBroker_PublishNoSubscriber(t *testing.T) {
	b := NewOrderEventBroker()

	// Publish to an account with no subscriber — should not panic or block.
	ev := &OrderEvent{AccountId: "no-such-account", Type: "order_update"}
	b.Publish(ev) // silent drop
}

func TestOrderEventBroker_PublishFullChannel(t *testing.T) {
	b := NewOrderEventBroker()
	_ = b.Subscribe("acc-1")

	// Fill the channel (capacity 64).
	for i := 0; i < 64; i++ {
		b.Publish(&OrderEvent{AccountId: "acc-1", Type: "fill"})
	}

	// One more — should be dropped, not block.
	done := make(chan struct{})
	go func() {
		b.Publish(&OrderEvent{AccountId: "acc-1", Type: "overflow"})
		close(done)
	}()

	select {
	case <-done:
		// ok, didn't block
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Publish blocked on full channel — expected silent drop")
	}

	b.Unsubscribe("acc-1")
}

// ── 3-account event fan-in test (M7.1-3 acceptance) ──

func TestOrderEventBroker_FanIn(t *testing.T) {
	b := NewOrderEventBroker()

	// Subscribe 3 accounts.
	ch1 := b.Subscribe("acc-1")
	ch2 := b.Subscribe("acc-2")
	ch3 := b.Subscribe("acc-3")

	if b.SubscriberCount() != 3 {
		t.Fatalf("SubscriberCount = %d, want 3", b.SubscriberCount())
	}

	// Publish one event per account.
	b.Publish(&OrderEvent{AccountId: "acc-1", Type: "ev1"})
	b.Publish(&OrderEvent{AccountId: "acc-2", Type: "ev2"})
	b.Publish(&OrderEvent{AccountId: "acc-3", Type: "ev3"})

	// Each subscriber must receive exactly its own event.
	expect := func(ch <-chan *OrderEvent, wantType string) {
		select {
		case ev := <-ch:
			if ev.Type != wantType {
				t.Errorf("expected %s, got %s", wantType, ev.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("timed out waiting for %s", wantType)
		}
	}

	expect(ch1, "ev1")
	expect(ch2, "ev2")
	expect(ch3, "ev3")

	// Unsubscribe all.
	b.Unsubscribe("acc-1")
	b.Unsubscribe("acc-2")
	b.Unsubscribe("acc-3")

	if b.SubscriberCount() != 0 {
		t.Errorf("SubscriberCount = %d after unsubscribe all, want 0", b.SubscriberCount())
	}
}

// ── Concurrent fan-in (send events from multiple goroutines) ──

func TestOrderEventBroker_ConcurrentFanIn(t *testing.T) {
	b := NewOrderEventBroker()

	const numAccounts = 10
	chs := make([]<-chan *OrderEvent, numAccounts)
	for i := 0; i < numAccounts; i++ {
		accountID := "acc-" + string(rune('0'+i))
		chs[i] = b.Subscribe(accountID)
	}

	if b.SubscriberCount() != numAccounts {
		t.Fatalf("SubscriberCount = %d, want %d", b.SubscriberCount(), numAccounts)
	}

	// Publish from multiple goroutines concurrently.
	var wg sync.WaitGroup
	for i := 0; i < numAccounts; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			accountID := "acc-" + string(rune('0'+idx))
			b.Publish(&OrderEvent{AccountId: accountID, Type: "concurrent"})
		}(i)
	}
	wg.Wait()

	// Each subscriber must receive exactly one event.
	for i := 0; i < numAccounts; i++ {
		select {
		case ev := <-chs[i]:
			if ev.Type != "concurrent" {
				t.Errorf("account %d: expected 'concurrent', got %s", i, ev.Type)
			}
		case <-time.After(200 * time.Millisecond):
			t.Errorf("account %d: timed out waiting for event", i)
		}
	}

	// Clean up.
	for i := 0; i < numAccounts; i++ {
		accountID := "acc-" + string(rune('0'+i))
		b.Unsubscribe(accountID)
	}
}
