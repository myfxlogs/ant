package strategy

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/internal/risk"
)

// mockPaperEngine is a no-op PaperOrderExecutor for testing.
type mockPaperEngine struct{}

func (m *mockPaperEngine) PlacePaperOrder(ctx context.Context, accountID, symbol, side string, volume, bid, ask decimal.Decimal) error {
	return nil
}
func (m *mockPaperEngine) ClosePaperOrder(ctx context.Context, accountID, symbol string) error {
	return nil
}
func (m *mockPaperEngine) ModifyPaperOrder(ctx context.Context, accountID, symbol string, sl, tp decimal.Decimal) error {
	return nil
}
func (m *mockPaperEngine) CancelPaperOrder(ctx context.Context, accountID, symbol string) error {
	return nil
}
func (m *mockPaperEngine) PaperPnl(ctx context.Context, accountID, symbol string, bid, ask decimal.Decimal) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

// mockOrderExecutor implements mthub.OrderExecutor for live path tests.
// Records the ClientID of each PlaceOrder call via a channel for synchronization.
type mockOrderExecutor struct {
	placedCh chan string
	closedCh chan int64
}

func (m *mockOrderExecutor) Platform() string { return "mock" }
func (m *mockOrderExecutor) PlaceOrder(_ context.Context, req *mthub.OrderRequest) (int64, error) {
	m.placedCh <- req.ClientID
	return 1, nil
}
func (m *mockOrderExecutor) CloseOrder(_ context.Context, ticket int64, _ decimal.Decimal) error {
	if m.closedCh != nil {
		m.closedCh <- ticket
	}
	return nil
}
func (m *mockOrderExecutor) DeleteOrder(_ context.Context, _ int64) error { return nil }
func (m *mockOrderExecutor) ModifyOrder(_ context.Context, _ int64, _, _, _ decimal.Decimal) error {
	return nil
}
func (m *mockOrderExecutor) FetchOpenedOrders(_ context.Context) ([]*mthub.OrderRecord, error) {
	return nil, nil
}
func (m *mockOrderExecutor) FetchOrderHistory(_ context.Context, _, _ time.Time) ([]*mthub.OrderRecord, error) {
	return nil, nil
}
func (m *mockOrderExecutor) FetchSymbolParams(_ context.Context, canonicals []string) ([]*mthub.SymbolParam, error) {
	if len(canonicals) == 0 {
		return nil, nil
	}
	return []*mthub.SymbolParam{{
		Canonical:    canonicals[0],
		ContractSize: decimal.NewFromInt(100000),
		LotSize:      decimal.NewFromInt(100000),
	}}, nil
}
func (m *mockOrderExecutor) FetchAllSymbols(_ context.Context) ([]string, error) { return nil, nil }
func (m *mockOrderExecutor) FetchPriceHistory(_ context.Context, _ string, _ string, _, _ int64, _ int) ([]*mthub.Bar, error) {
	return nil, nil
}
func (m *mockOrderExecutor) AddSymbols(_ context.Context, _ []string) error { return nil }
func (m *mockOrderExecutor) SubscribeOrderEvents(_ context.Context, _ mthub.OrderEventHandler) error {
	return nil
}

// DEPLOY-LIVE-1 adversarial proof: tick signal (bar=nil) with buy action
// must NOT panic. Before fix, bar.OpenTime nil dereference crashed the process.
// After fix, barOpenTimeForSignal returns 0 (or TickSeq counter) and order
// is dispatched without panic.
// Uses paper mode with mockPaperEngine to reach dispatchPaperSignal which
// accesses bar.Bid/bar.Ask — fixed to nil-safe.
func TestDeployLive1_NilBarTickSignalNoPanic(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop(), paperEngine: &mockPaperEngine{}}
	cfg := LiveStrategyConfig{
		AccountID: "acct-1",
		UserID:    "user-1",
		Symbol:    "EURUSD",
		Mode:      "paper",
		RunID:     uuid.New(),
		TickSeq:   new(atomic.Int64),
	}
	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("dispatchLiveSignal panicked with nil bar: %v", r)
		}
	}()
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, nil)
}

// DEPLOY-LIVE-1 adversarial proof: tick signal with sell action + nil bar.
func TestDeployLive1_NilBarTickSignalSellNoPanic(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop(), paperEngine: &mockPaperEngine{}}
	cfg := LiveStrategyConfig{
		AccountID: "acct-1",
		UserID:    "user-1",
		Symbol:    "EURUSD",
		Mode:      "paper",
		RunID:     uuid.New(),
		TickSeq:   new(atomic.Int64),
	}
	sig := &antv1.StrategySignal{SignalType: "sell", Volume: "0.1"}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("dispatchLiveSignal panicked with nil bar: %v", r)
		}
	}()
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, nil)
}

// DEPLOY-LIVE-1 adversarial proof: pending order with nil bar.
func TestDeployLive1_NilBarPendingOrderNoPanic(t *testing.T) {
	srv := &StrategyExecutionServer{log: zap.NewNop(), paperEngine: &mockPaperEngine{}}
	cfg := LiveStrategyConfig{
		AccountID: "acct-1",
		UserID:    "user-1",
		Symbol:    "EURUSD",
		Mode:      "paper",
		RunID:     uuid.New(),
		TickSeq:   new(atomic.Int64),
	}
	sig := &antv1.StrategySignal{SignalType: "buy_limit", Volume: "0.1", Price: "1.1000"}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("dispatchLiveSignal panicked with nil bar: %v", r)
		}
	}()
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, nil)
}

// DEPLOY-LIVE-1 附带: barOpenTimeForSignal with nil bar uses TickSeq counter.
// Two consecutive tick signals must produce different barOpenTime values
// (preventing ClientID collision).
func TestDeployLive1_TickSeqUniqueness(t *testing.T) {
	cfg := LiveStrategyConfig{
		TickSeq: new(atomic.Int64),
	}

	t1 := barOpenTimeForSignal(nil, cfg)
	t2 := barOpenTimeForSignal(nil, cfg)
	if t1 == t2 {
		t.Fatalf("consecutive tick signals have same barOpenTime: %d == %d (ClientID collision)", t1, t2)
	}
}

// DEPLOY-LIVE-1 附带: barOpenTimeForSignal with non-nil bar returns bar.OpenTime.
func TestDeployLive1_BarSignalUsesBarOpenTime(t *testing.T) {
	cfg := LiveStrategyConfig{
		TickSeq: new(atomic.Int64),
	}
	bar := &mthub.BarUpdate{OpenTime: 12345}

	got := barOpenTimeForSignal(bar, cfg)
	if got != 12345 {
		t.Fatalf("barOpenTimeForSignal with non-nil bar = %d, want 12345", got)
	}
}

// DEPLOY-LIVE-1 附带: barOpenTimeForSignal with nil bar and nil TickSeq returns 0.
func TestDeployLive1_NilBarNilTickSeqReturnsZero(t *testing.T) {
	cfg := LiveStrategyConfig{}

	got := barOpenTimeForSignal(nil, cfg)
	if got != 0 {
		t.Fatalf("barOpenTimeForSignal with nil bar and nil TickSeq = %d, want 0", got)
	}
}

// DEPLOY-LIVE-1-COVERAGE: live path test — Mode="live" + non-nil mtHub.
// Exercises the actual live dispatch call site (live_dispatch.go:63) where
// barOpenTimeForSignal(bar, cfg) is called. With nil bar, the fix returns
// TickSeq counter; reverting to bar.OpenTime causes nil pointer panic.
// The mock executor's PlaceOrder records the ClientID so we can verify
// two consecutive tick signals get different ClientIDs (no collision).
func TestDeployLive1_LivePathNilBarNoPanic(t *testing.T) {
	exec := &mockOrderExecutor{placedCh: make(chan string, 2)}
	hub := mthub.NewHub()
	svc := mthub.NewMtHubService(hub, mthub.NewOrderEventBroker(), mthub.NewAccountProfitBroker(), mthub.NewPositionSnapshotBroker(), nil, nil, nil)
	svc.SetLogger(zap.NewNop())
	svc.SetGate(risk.NewDefaultGate())
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: decimal.NewFromInt(100000), Equity: decimal.NewFromInt(100000)}, nil
	})
	hub.Register("acct-1", &mthub.Session{AccountID: "acct-1", CreatedAt: time.Now()}, exec)

	srv := &StrategyExecutionServer{log: zap.NewNop(), mtHub: svc}
	cfg := LiveStrategyConfig{
		AccountID:  "acct-1",
		UserID:     "user-1",
		Symbol:     "EURUSD",
		Mode:       "live",
		RunID:      uuid.New(),
		TickSeq:    new(atomic.Int64),
		ScheduleID: uuid.New(),
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("dispatchLiveSignal panicked with nil bar on live path: %v", r)
		}
	}()

	sig := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, nil)

	var clientID1 string
	select {
	case clientID1 = <-exec.placedCh:
		if clientID1 == "" {
			t.Fatal("expected non-empty ClientID from PlaceOrder")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("PlaceOrder was not called within 2s — live path not reached")
	}

	sig2 := &antv1.StrategySignal{SignalType: "buy", Volume: "0.1"}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig2, nil)

	select {
	case clientID2 := <-exec.placedCh:
		if clientID2 == clientID1 {
			t.Fatalf("two consecutive tick signals have same ClientID: %s (collision)", clientID1)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second PlaceOrder was not called within 2s")
	}
}
