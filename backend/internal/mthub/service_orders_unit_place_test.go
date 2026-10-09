package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/internal/costsvc"
)

func TestSubmitToBroker_NoSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	_, err := svc.submitToBroker(context.Background(), &OrderRequest{
		AccountID: "acc-1", Side: SideBuy, OrderType: OrderMarket,
	}, "ord-1")
	if err == nil {
		t.Fatal("expected error for no session")
	}
}

func TestPlaceOrder_KillSwitch(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetKillSwitch(&mockKillSwitch{engaged: true})
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Side: SideBuy, OrderType: OrderMarket,
	})
	if err == nil || err.Error() != ErrKillSwitchEngaged.Error() {
		t.Fatalf("expected ErrKillSwitchEngaged, got %v", err)
	}
}

func TestPlaceOrder_NoSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Side: SideBuy, OrderType: OrderMarket, Canonical: "EURUSD",
	})
	if err == nil {
		t.Fatal("expected error for no session")
	}
}

func TestPlaceOrder_Success(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	// Without OMS writer, cost estimator, or event store — should still succeed.
	record, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.Ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", record.Ticket)
	}
	// VM-LIVE-PARITY-F1: PlaceOrder passes the adapter receipt through —
	// state now comes from the adapter's explicit derivation (mock returns
	// Open for a market fill), not the old service-side Pending fabrication.
	if record.State != OrderStateOpen {
		t.Fatalf("expected adapter-derived state Open, got %d", record.State)
	}
}

func TestPlaceOrder_ExecutorError(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{
		platform: "MT5",
		placeOrderFn: func(ctx context.Context, req *OrderRequest) (*OrderRecord, error) {
			return nil, ErrSessionNotFound
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
	})
	if err == nil {
		t.Fatal("expected executor error")
	}
}

func TestPlaceOrder_OwnershipCheck_Unauthenticated(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, exec)
	svc.SetAccountOwnerVerifier(func(ctx context.Context, userID, accountID string) (bool, error) {
		return false, nil
	})
	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Side: SideBuy, OrderType: OrderMarket,
	})
	if err == nil {
		t.Fatal("expected ownership error")
	}
}

func TestPlaceOrder_WithEventStore(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	// Attach an event store — should publish order created event without panic.
	svc.eventStore = NewTradeEventStore(nil) // nil NATS conn, Publish will be no-op
	record, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.Ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", record.Ticket)
	}
}

func TestPlaceOrder_WithCostEstimator(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(ctx context.Context, canonicals []string) ([]*SymbolParam, error) {
			return []*SymbolParam{
				{Canonical: "EURUSD", Digits: 5, PointValue: decimal.NewFromFloat(0.00001)},
			}, nil
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	def := &costsvc.CostModel{Symbol: "DEFAULT", SpreadPips: decimal.Zero}
	svc.SetCostEstimator(NewHubCostEstimator(svc.hub, def, nil))
	record, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.Ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", record.Ticket)
	}
}
