package strategy

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
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
