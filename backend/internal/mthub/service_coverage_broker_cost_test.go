package mthub

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"alphaforge/internal/costsvc"
	"alphaforge/internal/risk"
)

func TestSubmitToBroker_MarginPrecheckPass(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockMarginExecutor{
		mockExecutor:   mockExecutor{platform: "MT5"},
		marginRequired: dec(100),
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{
			Balance:    dec(10000),
			Equity:     dec(10000),
			FreeMargin: dec(5000),
			UsedMargin: dec(100),
		}, nil
	})
	rec, err := svc.submitToBroker(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, "ord-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", rec.Ticket)
	}
}

func TestSubmitToBroker_MarginPrecheckReject(t *testing.T) {
	t.Parallel()
	// D6-A: margin precheck is now handled by the Gate (MarginPreCheck rule),
	// not in submitToBroker. submitToBroker just resolves executor and places order.
	// Verify submitToBroker succeeds even with tight margin — Gate rejection is tested separately.
	svc := newTestService()
	exec := &mockMarginExecutor{
		mockExecutor:   mockExecutor{platform: "MT5"},
		marginRequired: dec(99999),
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{
			Balance:    dec(100),
			Equity:     dec(100),
			FreeMargin: dec(50),
			UsedMargin: dec(50),
		}, nil
	})
	rec, err := svc.submitToBroker(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, "ord-1")
	if err != nil {
		t.Fatalf("submitToBroker should not do margin precheck (D6-A Gate handles it): %v", err)
	}
	if rec.Ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", rec.Ticket)
	}
}

func TestSubmitToBroker_MarginRPCError_Skips(t *testing.T) {
	t.Parallel()
	// D6-A: submitToBroker no longer calls RequiredMargin RPC — that was part of
	// the old risksvc.PreCheck path. submitToBroker just resolves executor and places order.
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockMarginExecutor{
		mockExecutor: mockExecutor{platform: "MT5"},
		marginErr:    context.DeadlineExceeded,
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return &risk.AccountState{Balance: dec(10000), Equity: dec(10000)}, nil
	})
	rec, err := svc.submitToBroker(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, "ord-1")
	if err != nil {
		t.Fatalf("expected success (no margin RPC in submitToBroker), got %v", err)
	}
	if rec.Ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", rec.Ticket)
	}
}

func TestSubmitToBroker_StateProviderError_Skips(t *testing.T) {
	t.Parallel()
	// D6-A: submitToBroker no longer fetches account state — that's done in evaluatePlaceGate.
	// submitToBroker just resolves executor and places order regardless of state provider.
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	exec := &mockMarginExecutor{
		mockExecutor:   mockExecutor{platform: "MT5"},
		marginRequired: dec(100),
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	svc.SetAccountStateProvider(func(_ context.Context, _ string) (*risk.AccountState, error) {
		return nil, context.DeadlineExceeded
	})
	rec, err := svc.submitToBroker(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, "ord-1")
	if err != nil {
		t.Fatalf("expected success (no state fetch in submitToBroker), got %v", err)
	}
	if rec.Ticket != 99999 {
		t.Fatalf("expected ticket 99999, got %d", rec.Ticket)
	}
}

// --- PlaceOrder coverage ---

func TestHubCostEstimator_RefreshCache(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	est := NewHubCostEstimator(hub, &costsvc.CostModel{Symbol: "DEFAULT"}, nil)
	est.Refresh("EURUSD")
}

// --- HubCostEstimator with active account but nil executor ---

func TestHubCostEstimator_FetchSymbolModel_NilExecutor(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now()}, nil)
	est := NewHubCostEstimator(hub, &costsvc.CostModel{Symbol: "DEFAULT", SpreadPips: dec(2)}, nil)
	result := est.Estimate(context.Background(), costsvc.EstimateParams{
		Symbol: "EURUSD", Side: "buy", Lots: dec(1), Price: dec(1.085),
		ContractSize: dec(100000),
	})
	_ = result
}

// --- estimateOrderCost ---

func TestEstimateOrderCost(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetCostEstimator(NewHubCostEstimator(NewHub(), &costsvc.CostModel{
		Symbol: "DEFAULT", SpreadPips: dec(1.5), PipSize: dec(0.0001), PipValue: dec(10),
	}, nil))
	ce := svc.estimateOrderCost(context.Background(), &OrderRequest{
		Canonical: "EURUSD", Side: SideBuy, Volume: dec(0.1), Price: dec(1.085),
	})
	if ce == nil {
		t.Fatal("expected non-nil cost estimate")
	}
}

// --- SubscribeAccountStatus without broker ---

func TestHubCostEstimator_CacheHit(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(_ context.Context, _ []string) ([]*SymbolParam, error) {
			return []*SymbolParam{
				{Canonical: "EURUSD", Digits: 5, PointValue: dec(1)},
			}, nil
		},
	}
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	est := NewHubCostEstimator(hub, &costsvc.CostModel{Symbol: "DEFAULT"}, nil)
	params := costsvc.EstimateParams{
		Symbol: "EURUSD", Side: "buy", Lots: dec(1), Price: dec(1.085),
		ContractSize: dec(100000),
	}
	est.Estimate(context.Background(), params)
	est.Estimate(context.Background(), params)
}

// --- HubCostEstimator fetch error path ---

func TestHubCostEstimator_FetchError(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(_ context.Context, _ []string) ([]*SymbolParam, error) {
			return nil, ErrSessionNotFound
		},
	}
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	est := NewHubCostEstimator(hub, &costsvc.CostModel{Symbol: "DEFAULT", SpreadPips: dec(2)}, nil)
	result := est.Estimate(context.Background(), costsvc.EstimateParams{
		Symbol: "EURUSD", Side: "buy", Lots: dec(1), Price: dec(1.085),
		ContractSize: dec(100000),
	})
	_ = result
}

// --- PlaceOrder with guard (covers preTradeChecks guard path) ---

func TestHubCostEstimator_DoubleCheckedLockingCacheHit(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	fetchCh := make(chan struct{})
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(_ context.Context, _ []string) ([]*SymbolParam, error) {
			<-fetchCh // block until signaled
			return []*SymbolParam{
				{Canonical: "EURUSD", Digits: 5, PointValue: dec(1)},
			}, nil
		},
	}
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	est := NewHubCostEstimator(hub, &costsvc.CostModel{Symbol: "DEFAULT"}, nil)
	params := costsvc.EstimateParams{
		Symbol: "EURUSD", Side: "buy", Lots: dec(1), Price: dec(1.085),
		ContractSize: dec(100000),
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// G1: will fetch (slow), then cache the result.
	go func() {
		defer wg.Done()
		est.Estimate(context.Background(), params)
	}()

	// Give G1 time to acquire the write lock and start fetching.
	time.Sleep(10 * time.Millisecond)

	// G2: will miss RLock cache, wait for write lock. When G1 finishes and unlocks,
	// G2 acquires lock and hits the double-checked locking cache hit (line 62).
	go func() {
		defer wg.Done()
		est.Estimate(context.Background(), params)
	}()

	// Give G2 time to hit the RLock cache miss and wait for the write lock.
	time.Sleep(10 * time.Millisecond)

	// Release G1's fetch — G1 caches and unlocks, G2 acquires lock and hits cache.
	close(fetchCh)
	wg.Wait()
}

// --- Idempotency guard error paths (using failing Redis client) ---
