package connect

import (
	"sync"
	"testing"
	"time"
)

func TestAccountEnabledChangePubSub_DeliversToSubscriber(t *testing.T) {
	s := &StreamService{}
	userID := "u1"
	accountID := "a1"

	_, ch, unsubscribe := s.subscribeAccountEnabledChanges(userID)
	defer unsubscribe()

	s.NotifyAccountEnabledState(userID, accountID, true)

	select {
	case msg := <-ch:
		if msg.accountID != accountID {
			t.Fatalf("expected accountID=%s, got=%s", accountID, msg.accountID)
		}
		if !msg.enabled {
			t.Fatalf("expected enabled=true")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected to receive pubsub message")
	}
}

func TestAccountEnabledChangePubSub_UnsubscribeStopsDelivery(t *testing.T) {
	s := &StreamService{}
	userID := "u1"

	_, ch, unsubscribe := s.subscribeAccountEnabledChanges(userID)
	unsubscribe()

	s.NotifyAccountEnabledState(userID, "a1", true)

	select {
	case <-ch:
		t.Fatalf("did not expect message after unsubscribe")
	case <-time.After(50 * time.Millisecond):
		// ok
	}
}

func TestAccountEnabledChangePubSub_ConcurrentNotifyAndUnsubscribe_NoPanic(t *testing.T) {
	s := &StreamService{}
	userID := "u1"

	// Create many subscribers and concurrently unsubscribe while notifying.
	unsubs := make([]func(), 0, 64)
	for i := 0; i < 64; i++ {
		_, _, un := s.subscribeAccountEnabledChanges(userID)
		unsubs = append(unsubs, un)
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				s.NotifyAccountEnabledState(userID, "a1", true)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < len(unsubs); i++ {
			unsubs[i]()
		}
	}()

	time.Sleep(50 * time.Millisecond)
	close(stop)
	wg.Wait()
}
