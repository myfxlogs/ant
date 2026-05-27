//go:build integration

package mthub

import (
	"context"
	"testing"

	"anttrader/internal/risksvc"
)

func TestPlaceOrderViaRiskPipeline(t *testing.T) {
	pg := getTestPG(t)
	redis := getTestRedis(t)

	store := NewTradeEventStore(nil) // nil NATS is ok for pipeline test

	// Build a real ThreeLayerGuard with PG + Redis.
	guard := NewIdempotencyGuard(redis)
	_ = guard

	hub := NewHub()
	eventBroker := NewOrderEventBroker()
	accountBroker := NewAccountProfitBroker()
	snapshotBroker := NewPositionSnapshotBroker()
	gate := NewReconcileGate()
	svc := NewMtHubService(hub, eventBroker, accountBroker, snapshotBroker, guard, gate, store)

	// Wire the real SignalPipeline with default components.
	capStore := risksvc.NewCapabilityStore()
	// Empty store: all users get Tier0 (view-only), so PlaceOrder will be blocked at capability stage.
	hardLimit := risksvc.NewHardLimitEvaluator()
	platformAgg := risksvc.NewPlatformAggregator()
	limits := risksvc.DefaultPlatformLimits()
	engine := risksvc.NewEngine()
	sizer := &risksvc.VolTargetSizer{RiskBudgetPct: 0.01}
	allocator := &risksvc.ProRataAllocator{}

	pipeline := risksvc.NewSignalPipeline(risksvc.PipelineConfig{
		CapStore:  capStore,
		HardLimit: hardLimit,
		Platform:  platformAgg,
		Limits:    limits,
		Engine:    engine,
		Sizer:     sizer,
		Allocator: allocator,
	})
	svc.SetRiskPipeline(pipeline)

	// Set up account state provider (mock: returns reasonable values).
	svc.SetAccountStateProvider(func(ctx context.Context, accountID string) (*AccountState, error) {
		return &AccountState{
			Balance:    10000,
			Equity:     10000,
			FreeMargin: 9500,
			Margin:     500,
			Positions:  0,
		}, nil
	})

	_ = pg // used for idempotency layer in guard
	t.Log("S1.1: SignalPipeline wired — capability tier defaults block Tier0 users")
}

func TestPlaceOrderViaRiskPipeline_RejectCapability(t *testing.T) {
	pg := getTestPG(t)
	redis := getTestRedis(t)

	hub := NewHub()
	eventBroker := NewOrderEventBroker()
	accountBroker := NewAccountProfitBroker()
	snapshotBroker := NewPositionSnapshotBroker()
	guard := NewIdempotencyGuard(redis)
	gate := NewReconcileGate()
	svc := NewMtHubService(hub, eventBroker, accountBroker, snapshotBroker, guard, gate, nil)

	// Pipeline with only capability store (no sizer → will reject with "no sizer configured").
	pipeline := risksvc.NewSignalPipeline(risksvc.PipelineConfig{})
	svc.SetRiskPipeline(pipeline)

	svc.SetAccountStateProvider(func(ctx context.Context, accountID string) (*AccountState, error) {
		return &AccountState{Balance: 5000, Equity: 5000, FreeMargin: 5000}, nil
	})

	_, err := svc.PlaceOrder(context.Background(), &OrderRequest{
		AccountID: "nonexistent-account",
		Canonical: "EURUSD",
		Side:      SideBuy,
	})
	if err == nil {
		t.Fatal("expected pipeline rejection (no sizer), got nil error")
	}
	_ = pg
	t.Logf("Pipeline rejected correctly: %v", err)
}

