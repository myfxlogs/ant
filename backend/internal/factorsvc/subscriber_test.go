package factorsvc

import (
	"context"
	"sync"
	"testing"

	"go.uber.org/zap"
)

func TestNewSubscriber(t *testing.T) {
	s := NewSubscriber(zap.NewNop())
	if s == nil {
		t.Fatal("NewSubscriber returned nil")
	}
	if s.HandlerCount() != 0 {
		t.Fatalf("expected 0 handlers, got %d", s.HandlerCount())
	}
}

func TestSubscriber_SubscribeAndDispatch(t *testing.T) {
	s := NewSubscriber(zap.NewNop())

	var mu sync.Mutex
	var received []FactorValue

	id := s.Subscribe(func(ctx context.Context, fv FactorValue) {
		mu.Lock()
		received = append(received, fv)
		mu.Unlock()
	})

	if id == "" {
		t.Fatal("expected non-empty subscription id")
	}
	if s.HandlerCount() != 1 {
		t.Fatalf("expected 1 handler, got %d", s.HandlerCount())
	}

	fv := FactorValue{
		UserID:   "t1",
		Factor:   "sma10",
		Symbol:   "EURUSD",
		TsUnixMs: 1000,
		Value:    1.2345,
	}
	s.Dispatch(context.Background(), fv)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 received, got %d", len(received))
	}
	if received[0].Value != 1.2345 {
		t.Fatalf("expected value 1.2345, got %f", received[0].Value)
	}
}

func TestSubscriber_Unsubscribe(t *testing.T) {
	s := NewSubscriber(zap.NewNop())

	var received []FactorValue
	id := s.Subscribe(func(ctx context.Context, fv FactorValue) {
		received = append(received, fv)
	})

	s.Unsubscribe(id)
	if s.HandlerCount() != 0 {
		t.Fatalf("expected 0 handlers after unsubscribe, got %d", s.HandlerCount())
	}

	s.Dispatch(context.Background(), FactorValue{Factor: "test"})
	if len(received) != 0 {
		t.Fatal("expected no deliveries after unsubscribe")
	}
}

func TestSubscriber_MultipleHandlers(t *testing.T) {
	s := NewSubscriber(zap.NewNop())

	var mu sync.Mutex
	var count int

	for i := 0; i < 5; i++ {
		s.Subscribe(func(ctx context.Context, fv FactorValue) {
			mu.Lock()
			count++
			mu.Unlock()
		})
	}

	s.Dispatch(context.Background(), FactorValue{Factor: "test"})

	mu.Lock()
	defer mu.Unlock()
	if count != 5 {
		t.Fatalf("expected 5 deliveries, got %d", count)
	}
}

func TestSubscriber_DispatchBatch(t *testing.T) {
	s := NewSubscriber(zap.NewNop())

	var mu sync.Mutex
	var count int

	s.Subscribe(func(ctx context.Context, fv FactorValue) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	fvs := []FactorValue{
		{Factor: "a", Value: 1},
		{Factor: "b", Value: 2},
		{Factor: "c", Value: 3},
	}
	s.DispatchBatch(context.Background(), fvs)

	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Fatalf("expected 3 deliveries from batch, got %d", count)
	}
}
