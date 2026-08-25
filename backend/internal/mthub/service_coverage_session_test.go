package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/costsvc"
)

// --- SessionState coverage ---

func TestSessionState_WithExecutor(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	state := svc.SessionState(context.Background(), "acc-1")
	if state != "connected" {
		t.Fatalf("expected connected, got %s", state)
	}
}

// --- Platform with session ---

func TestPlatform_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	if svc.Platform("acc-1") != "MT5" {
		t.Fatal("expected MT5")
	}
}

func TestPlatformFunc_WithSession(t *testing.T) {
	t.Parallel()
	hub := NewHub()
	exec := &mockExecutor{platform: "MT4"}
	hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	if platform("acc-1", hub) != "MT4" {
		t.Fatal("expected MT4")
	}
}

// --- OpenedOrders / OrderHistory / SymbolParams / PriceHistory / SymbolList / SubscribeSymbols with session ---

func TestOpenedOrders_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	orders, err := svc.OpenedOrders(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orders != nil {
		t.Fatalf("expected nil orders from mock, got %d", len(orders))
	}
}

func TestOrderHistory_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	orders, err := svc.OrderHistory(context.Background(), "acc-1", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if orders != nil {
		t.Fatalf("expected nil orders from mock, got %d", len(orders))
	}
}

func TestSymbolParams_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{
		platform: "MT5",
		fetchSymbolParamsFn: func(_ context.Context, _ []string) ([]*SymbolParam, error) {
			return nil, nil
		},
	}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	params, err := svc.SymbolParams(context.Background(), "acc-1", []string{"EURUSD"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params != nil {
		t.Fatalf("expected nil params from mock, got %d", len(params))
	}
}

func TestPriceHistory_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	bars, err := svc.PriceHistory(context.Background(), "acc-1", "EURUSD", "M1", 0, 0, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bars != nil {
		t.Fatalf("expected nil bars from mock, got %d", len(bars))
	}
}

func TestSymbolList_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	symbols, err := svc.SymbolList(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if symbols != nil {
		t.Fatalf("expected nil symbols from mock, got %d", len(symbols))
	}
}

func TestSubscribeSymbols_WithSession(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	exec := &mockExecutor{platform: "MT5"}
	svc.hub.Register("acc-1", &Session{AccountID: "acc-1", CreatedAt: time.Now(), MaxAge: 4 * time.Hour}, exec)
	err := svc.SubscribeSymbols(context.Background(), "acc-1", []string{"EURUSD"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- postCloseSuccess with event store ---

func TestPostCloseSuccess_WithEventStore(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.eventStore = NewTradeEventStore(nil)
	svc.postCloseSuccess(context.Background(), "close-1", "acc-1", 123)
}

// --- postCloseFailure with logger ---

func TestPostCloseFailure_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.postCloseFailure(context.Background(), "close-1", "acc-1", 123, ErrSessionNotFound)
}

// --- publishOrderCreatedEvent with event store ---

func TestPublishOrderCreatedEvent_WithEventStore(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.eventStore = NewTradeEventStore(nil)
	svc.publishOrderCreatedEvent(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, 12345, nil)
}

// --- publishOrderCreatedEvent with logger and event store ---

func TestPublishOrderCreatedEvent_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.eventStore = NewTradeEventStore(nil)
	svc.publishOrderCreatedEvent(context.Background(), &OrderRequest{
		AccountID: "acc-1", Canonical: "EURUSD",
		Side: SideBuy, OrderType: OrderMarket,
		Volume: dec(0.1), Price: dec(1.085),
	}, 12345, &antv1.CostEstimate{})
}

// --- omsTransition with logger (nil writer still) ---

func TestOmsTransition_WithLogger(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.SetLogger(zap.NewNop())
	svc.omsTransition(context.Background(), "ord-1", "acc-1", OMSStateNew, OMSStateValidated)
}

// --- DerivedState recalculate with data ---

func TestDerivedState_Recalculate_WithData(t *testing.T) {
	t.Parallel()
	cache := NewStateCache(nil, testLogger())
	cache.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "BUY", Volume: dec(0.1), Price: dec(1.085),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	computer := NewDerivedComputer(cache, 1*time.Minute)
	computer.recalculate()
	ds := computer.State()
	acc := ds.GetAccount("acc-1")
	if acc == nil {
		t.Fatal("expected acc-1 derived state")
	}
	if !acc.Exposure.GreaterThan(decimal.Zero) {
		t.Fatal("expected positive exposure")
	}
}

func TestIsTerminalOrderState(t *testing.T) {
	t.Parallel()
	terminals := []string{"CLOSED", "FILLED", "CANCELLED", "EXPIRED", "FAILED", "REJECTED"}
	for _, s := range terminals {
		if !isTerminalOrderState(s) {
			t.Errorf("%s should be terminal", s)
		}
	}
	nonTerminals := []string{"WORKING", "SUBMITTED", "NEW", "VALIDATED", "RISK_APPROVED"}
	for _, s := range nonTerminals {
		if isTerminalOrderState(s) {
			t.Errorf("%s should not be terminal", s)
		}
	}
}

func TestCostToProto(t *testing.T) {
	t.Parallel()
	est := costsvc.CostBreakdown{
		SpreadCost:   dec(1.5),
		Commission:   dec(7),
		SlippageCost: dec(0.5),
		SwapCost:     dec(0),
		TotalCost:    dec(9),
	}
	p := costToProto(&est)
	if p.SpreadCost != "1.5" {
		t.Fatalf("expected 1.5, got %s", p.SpreadCost)
	}
}

// --- HubCostEstimator Refresh ---

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

func TestSubscribeAccountStatus_NoBroker(t *testing.T) {
	t.Parallel()
	svc := &MtHubService{}
	ch, cancel := svc.SubscribeAccountStatus("acc-1")
	defer cancel()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel should be closed when no broker")
		}
	default:
		t.Fatal("channel should be closed when no broker")
	}
}

// --- SubscribeUserOrderEvents ---

func TestSubscribeUserOrderEvents(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	ch, cancel := svc.SubscribeUserOrderEvents(context.Background(), "user-1")
	defer cancel()
	_ = ch
}

// --- PublishPositionSnapshot ---

func TestPublishPositionSnapshot(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.PublishPositionSnapshot(&PositionSnapshot{AccountID: "acc-1"})
}

// --- SubscribePositionSnapshots ---

func TestSubscribePositionSnapshots(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	ch, cancel := svc.SubscribePositionSnapshots(context.Background(), "acc-1")
	defer cancel()
	_ = ch
}

// --- PublishAccountProfit ---

func TestPublishAccountProfit(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	svc.PublishAccountProfit(&AccountProfitEvent{AccountID: "acc-1"})
}

// --- SubscribeAccountProfit ---

func TestSubscribeAccountProfit(t *testing.T) {
	t.Parallel()
	svc := newTestService()
	ch, cancel := svc.SubscribeAccountProfit(context.Background(), "acc-1")
	defer cancel()
	_ = ch
}

// --- HubCostEstimator cache hit (second call hits cache) ---

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

// --- StateCache: position increase weighted average ---

func TestStateCache_PositionIncrease_WeightedAvg(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	// Buy 0.1 at 1.0850
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "BUY", Volume: dec(0.1), Price: dec(1.0850),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos := c.GetPosition("acc-1", "EURUSD")
	if !pos.AvgPrice.Equal(dec(1.0850)) {
		t.Fatalf("expected avg price 1.0850, got %s", pos.AvgPrice)
	}

	// Buy 0.05 more at 1.0900 → position increases, weighted avg
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2,
		Canonical: "EURUSD", Side: "BUY", Volume: dec(0.05), Price: dec(1.0900),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos = c.GetPosition("acc-1", "EURUSD")
	// Weighted: (1.0850*0.1 + 1.0900*0.05) / 0.15 = (0.10850 + 0.05450) / 0.15 = 0.16300 / 0.15 = 1.08667...
	expected := dec(1.0850).Mul(dec(0.1)).Add(dec(1.0900).Mul(dec(0.05))).Div(dec(0.15))
	if !pos.AvgPrice.Equal(expected) {
		t.Fatalf("expected weighted avg %s, got %s", expected, pos.AvgPrice)
	}
}

// --- StateCache: short position increase weighted average ---

func TestStateCache_ShortPositionIncrease_WeightedAvg(t *testing.T) {
	t.Parallel()
	c := NewStateCache(nil, testLogger())

	// Sell 0.1 at 1.0850 → short
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 1,
		Canonical: "EURUSD", Side: "SELL", Volume: dec(0.1), Price: dec(1.0850),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos := c.GetPosition("acc-1", "EURUSD")
	if !pos.NetVolume.Equal(dec(-0.1)) {
		t.Fatalf("expected -0.1, got %s", pos.NetVolume)
	}

	// Sell 0.05 more at 1.0900 → short increases
	c.ApplyEvent(&TradeEvent{
		EventType: TradeEventOrderFilled, AccountID: "acc-1", Ticket: 2,
		Canonical: "EURUSD", Side: "SELL", Volume: dec(0.05), Price: dec(1.0900),
		ToState: "FILLED", Timestamp: time.Now(),
	})
	pos = c.GetPosition("acc-1", "EURUSD")
	if !pos.NetVolume.Equal(dec(-0.15)) {
		t.Fatalf("expected -0.15, got %s", pos.NetVolume)
	}
	expected := dec(1.0850).Mul(dec(0.1)).Add(dec(1.0900).Mul(dec(0.05))).Div(dec(0.15))
	if !pos.AvgPrice.Equal(expected) {
		t.Fatalf("expected weighted avg %s, got %s", expected, pos.AvgPrice)
	}
}

// --- Pure function: isValidOMSTransition ---

func TestIsValidOMSTransition(t *testing.T) {
	t.Parallel()
	valid := []struct{ from, to OMSState }{
		{OMSStateNew, OMSStateValidated},
		{OMSStateValidated, OMSStateRiskApproved},
		{OMSStateValidated, OMSStateRejected},
		{OMSStateRiskApproved, OMSStateSubmitted},
		{OMSStateRiskApproved, OMSStateFailed},
		{OMSStateSubmitted, OMSStateWorking},
		{OMSStateSubmitted, OMSStateFilled},
		{OMSStateSubmitted, OMSStateCancelled},
		{OMSStateWorking, OMSStatePartiallyFilled},
		{OMSStateWorking, OMSStateFilled},
		{OMSStateUnknown, OMSStateReconciling},
		{OMSStateReconciling, OMSStateWorking},
		{OMSStateRequoted, OMSStateRiskApproved},
		{OMSStateMarginCall, OMSStateCancelled},
	}
	for _, tr := range valid {
		if !isValidOMSTransition(tr.from, tr.to) {
			t.Errorf("%s → %s should be valid", tr.from, tr.to)
		}
	}
	invalid := []struct{ from, to OMSState }{
		{OMSStateNew, OMSStateSubmitted},
		{OMSStateFilled, OMSStateWorking},
		{OMSStateCancelled, OMSStateWorking},
		{OMSStateRejected, OMSStateNew},
		{OMSStateExpired, OMSStateWorking},
	}
	for _, tr := range invalid {
		if isValidOMSTransition(tr.from, tr.to) {
			t.Errorf("%s → %s should be invalid", tr.from, tr.to)
		}
	}
}

// --- Pure function: hashToNegative ---

func TestHashToNegative_Coverage(t *testing.T) {
	t.Parallel()
	v := hashToNegative("test-order-id")
	if v >= 0 {
		t.Fatalf("expected negative value, got %d", v)
	}
	v2 := hashToNegative("test-order-id")
	if v != v2 {
		t.Fatal("hashToNegative should be deterministic")
	}
	v3 := hashToNegative("different-id")
	if v == v3 {
		t.Fatal("different inputs should produce different hashes")
	}
}

// --- Pure function: advisoryLockKey ---

func TestAdvisoryLockKey_Coverage(t *testing.T) {
	t.Parallel()
	k1, k2 := advisoryLockKey("acc-1", "client-1")
	_ = k1
	_ = k2
	k3, k4 := advisoryLockKey("acc-1", "client-1")
	if k1 != k3 || k2 != k4 {
		t.Fatal("advisoryLockKey should be deterministic")
	}
	k5, k6 := advisoryLockKey("acc-2", "client-1")
	if k1 == k5 && k2 == k6 {
		t.Fatal("different accountID should produce different keys")
	}
}

// --- Broker drop paths (fill buffer then publish to hit default case) ---

func TestPositionSnapshotBroker_Publish_DropPath(t *testing.T) {
	t.Parallel()
	b := NewPositionSnapshotBroker()
	ch, cancel := b.Subscribe("acc-1")
	// Buffer size is 8 — fill it.
	for i := 0; i < 8; i++ {
		b.Publish(&PositionSnapshot{AccountID: "acc-1"})
	}
	// This publish should hit the default (drop) case.
	b.Publish(&PositionSnapshot{AccountID: "acc-1"})
	cancel()
	_ = ch
}

func TestAccountStatusBroker_Publish_DropPath(t *testing.T) {
	t.Parallel()
	b := NewAccountStatusBroker()
	ch, cancel := b.Subscribe("acc-1")
	// Buffer size is 8 — fill it.
	for i := 0; i < 8; i++ {
		b.Publish(&AccountStatusEvent{AccountID: "acc-1"})
	}
	// This publish should hit the default (drop) case.
	b.Publish(&AccountStatusEvent{AccountID: "acc-1"})
	cancel()
	_ = ch
}

func TestOrderEventBroker_PublishEvent_DropPath(t *testing.T) {
	t.Parallel()
	b := NewOrderEventBroker()
	ch, cancel := b.Subscribe("user-1")
	// Buffer size is 64 — fill it.
	for i := 0; i < 64; i++ {
		b.PublishEvent("user-1", &OrderEvent{AccountID: "acc-1"})
	}
	// This publish should hit the default (drop) case.
	b.PublishEvent("user-1", &OrderEvent{AccountID: "acc-1"})
	cancel()
	_ = ch
}

func TestAccountProfitBroker_Publish_DropPath(t *testing.T) {
	t.Parallel()
	b := NewAccountProfitBroker()
	ch, cancel := b.Subscribe("acc-1")
	// Buffer size is 64 — fill it.
	for i := 0; i < 64; i++ {
		b.Publish(&AccountProfitEvent{AccountID: "acc-1"})
	}
	// This publish should hit the default (drop) case.
	b.Publish(&AccountProfitEvent{AccountID: "acc-1"})
	cancel()
	_ = ch
}
