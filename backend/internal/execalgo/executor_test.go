package execalgo

import (
	"context"
	"sync"
	"testing"
	"time"

	"anttrader/internal/mthub"
)

// mockBroker records submitted orders for test assertions.
type mockBroker struct {
	mu      sync.Mutex
	orders  []*mthub.OrderRequest
	tickets []int64
	nextID  int64
}

func (m *mockBroker) SubmitOrder(_ context.Context, req *mthub.OrderRequest) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.orders = append(m.orders, req)
	m.nextID++
	ticket := m.nextID
	m.tickets = append(m.tickets, ticket)
	return ticket, nil
}

func (m *mockBroker) Platform() string { return "mock" }

func (m *mockBroker) submittedOrders() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.orders)
}

// mockMarketState always reports tradeable.
type mockMarketState struct{}

func (mockMarketState) IsTradeable(_ string) (bool, string) { return true, "" }

func TestExecutorFullSchedule(t *testing.T) {
	broker := &mockBroker{}
	schedule := &Schedule{
		Algo: "twap",
		Parent: ParentOrder{
			Symbol:      "EURUSD",
			Side:        "buy",
			TotalVolume: 1.0,
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(10 * time.Second),
		},
		Slices: []ChildOrder{
			{Sequence: 0, Volume: 0.25, TargetTime: time.Now().Add(100 * time.Millisecond)},
			{Sequence: 1, Volume: 0.25, TargetTime: time.Now().Add(200 * time.Millisecond)},
			{Sequence: 2, Volume: 0.25, TargetTime: time.Now().Add(300 * time.Millisecond)},
			{Sequence: 3, Volume: 0.25, TargetTime: time.Now().Add(400 * time.Millisecond)},
		},
	}

	exec := NewExecutor(ExecutorConfig{
		Schedule:     schedule,
		Broker:       broker,
		AccountID:    "test-account",
		MarketState:  mockMarketState{},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exec.Start(ctx)

	// Collect events until terminal.
	var events []ExecEvent
	for ev := range exec.Events() {
		events = append(events, ev)
		if ev.State.IsTerminal() {
			break
		}
	}

	if exec.State() != ExecCompleted {
		t.Fatalf("expected ExecCompleted, got %v", exec.State())
	}

	if broker.submittedOrders() != 4 {
		t.Errorf("expected 4 submitted orders, got %d", broker.submittedOrders())
	}

	submitted, total := exec.Progress()
	if submitted != 4 || total != 4 {
		t.Errorf("progress: submitted=%d total=%d, want 4/4", submitted, total)
	}
}

func TestExecutorCancel(t *testing.T) {
	broker := &mockBroker{}
	schedule := &Schedule{
		Algo: "twap",
		Parent: ParentOrder{
			Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
			StartTime: time.Now(), EndTime: time.Now().Add(30 * time.Second),
		},
		Slices: []ChildOrder{
			{Sequence: 0, Volume: 0.5, TargetTime: time.Now().Add(50 * time.Millisecond)},
			{Sequence: 1, Volume: 0.5, TargetTime: time.Now().Add(10 * time.Second)}, // far future
		},
	}

	exec := NewExecutor(ExecutorConfig{
		Schedule: schedule, Broker: broker, AccountID: "test-account",
	})

	ctx, cancel := context.WithCancel(context.Background())
	exec.Start(ctx)

	// Wait for first slice, then cancel.
	time.Sleep(200 * time.Millisecond)
	exec.Cancel()

	// Drain events.
	for range exec.Events() {
	}

	if exec.State() != ExecCancelled {
		t.Fatalf("expected ExecCancelled, got %v", exec.State())
	}

	// Only the first near-future slice should have been submitted.
	if submitted := broker.submittedOrders(); submitted != 1 {
		t.Errorf("expected 1 submitted order before cancel, got %d", submitted)
	}

	cancel() // clean up context
}

func TestExecutorPauseResume(t *testing.T) {
	broker := &mockBroker{}
	schedule := &Schedule{
		Algo: "twap",
		Parent: ParentOrder{
			Symbol: "EURUSD", Side: "sell", TotalVolume: 1.0,
			StartTime: time.Now(), EndTime: time.Now().Add(5 * time.Second),
		},
		Slices: []ChildOrder{
			{Sequence: 0, Volume: 0.5, TargetTime: time.Now().Add(100 * time.Millisecond)},
			{Sequence: 1, Volume: 0.5, TargetTime: time.Now().Add(200 * time.Millisecond)},
		},
	}

	exec := NewExecutor(ExecutorConfig{
		Schedule: schedule, Broker: broker, AccountID: "test-account",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	exec.Start(ctx)

	// Pause after first slice should submit.
	time.Sleep(150 * time.Millisecond)
	exec.Pause()
	if exec.State() != ExecPaused {
		t.Fatalf("expected ExecPaused, got %v", exec.State())
	}

	// Resume.
	exec.Resume()
	if exec.State() != ExecRunning {
		t.Fatalf("expected ExecRunning after resume, got %v", exec.State())
	}

	// Drain events.
	for range exec.Events() {
	}

	if exec.State() != ExecCompleted {
		t.Fatalf("expected ExecCompleted, got %v", exec.State())
	}

	if submitted := broker.submittedOrders(); submitted != 2 {
		t.Errorf("expected 2 submitted orders, got %d", submitted)
	}
}

func TestExecutorLimitOrderSlice(t *testing.T) {
	broker := &mockBroker{}
	schedule := &Schedule{
		Algo: "twap",
		Parent: ParentOrder{
			Symbol: "EURUSD", Side: "buy", TotalVolume: 1.0,
			StartTime: time.Now(), EndTime: time.Now().Add(2 * time.Second),
			LimitPrice: 1.1050,
		},
		Slices: []ChildOrder{
			{Sequence: 0, Volume: 1.0, TargetTime: time.Now().Add(100 * time.Millisecond), LimitPrice: 1.1050},
		},
	}

	exec := NewExecutor(ExecutorConfig{
		Schedule: schedule, Broker: broker, AccountID: "test-account",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	exec.Start(ctx)
	for range exec.Events() {
	}

	if exec.State() != ExecCompleted {
		t.Fatalf("expected ExecCompleted, got %v", exec.State())
	}

	// Verify the order was submitted as a limit order.
	broker.mu.Lock()
	order := broker.orders[0]
	broker.mu.Unlock()

	if order.OrderType != mthub.OrderLimit {
		t.Errorf("expected OrderLimit, got %v", order.OrderType)
	}
	if order.Price.InexactFloat64() != 1.1050 {
		t.Errorf("expected price 1.1050, got %v", order.Price)
	}
}
