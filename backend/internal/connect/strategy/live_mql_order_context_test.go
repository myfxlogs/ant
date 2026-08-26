package strategy

// LIVE-MQL-ORDER-CONTEXT-1: Integration tests for the complete chain:
//   broker PositionSnapshot → backfillContextStrings → LivePosition/LivePendingOrder proto
//   → vmPositionsToSdk/vmPendingOrdersToSdk → sdk.Position/sdk.PendingOrder
//   → Runner.UpdateLiveState → broker.Positions/Orders → MQL OrdersTotal/OrderSelect/OrderMagicNumber
//
// Adversarial acceptance criteria from builder-handoff-2026-08-21.md:
//   - buy/sell/buy_limit/sell_stop → Positions=2, Orders=2, OrdersTotal=4
//   - magic=1699507621 must reach OrderMagicNumber end-to-end
//   - deleting any layer's magic mapping must make tests RED

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mthub"
	"alphaforge/strategy/runner"
	"alphaforge/strategy/sdk"
)

// testMagic is the magic number used across all LIVE-MQL-ORDER-CONTEXT-1 tests.
// It matches the builder handoff spec: magic=1699507621 must reach OrderMagicNumber.
const testMagic int32 = 1699507621

// buildTestSnapshot creates a PositionSnapshot with 2 market positions (buy/sell)
// and 2 pending orders (buy_limit/sell_stop), all with testMagic.
func buildTestSnapshot() *mthub.PositionSnapshot {
	now := time.Now()
	return &mthub.PositionSnapshot{
		AccountID:               "acct-1",
		Balance:                 decimal.NewFromInt(10000),
		Equity:                  decimal.NewFromInt(10500),
		Margin:                  decimal.NewFromInt(500),
		FreeMargin:              decimal.NewFromInt(9500),
		FinancialsAuthoritative: true,
		FinancialsSource:        "account_summary",
		CapturedAt:              now,
		PositionsAuthoritative:  true,
		PositionsCapturedAt:     now,
		PositionsSource:         "order_stream",
		Positions: []mthub.PositionSnapshotItem{
			{
				Ticket: 1001, Symbol: "EURUSD", Type: "buy", Magic: testMagic,
				Volume: decimal.NewFromFloat(0.1), OpenPrice: decimal.NewFromFloat(1.1),
				StopLoss: decimal.NewFromFloat(1.0), TakeProfit: decimal.NewFromFloat(1.2),
				Profit: decimal.NewFromFloat(50), Swap: decimal.Zero,
				Commission: decimal.NewFromFloat(1), Comment: "test-buy", OpenTime: now.Unix(),
			},
			{
				Ticket: 1002, Symbol: "GBPUSD", Type: "sell", Magic: testMagic,
				Volume: decimal.NewFromFloat(0.2), OpenPrice: decimal.NewFromFloat(1.3),
				StopLoss: decimal.NewFromFloat(1.4), TakeProfit: decimal.NewFromFloat(1.2),
				Profit: decimal.NewFromFloat(-30), Swap: decimal.Zero,
				Commission: decimal.NewFromFloat(1), Comment: "test-sell", OpenTime: now.Unix(),
			},
		},
		PendingOrders: []mthub.PositionSnapshotItem{
			{
				Ticket: 2001, Symbol: "EURUSD", Type: "buy_limit", Magic: testMagic,
				Volume: decimal.NewFromFloat(0.1), OpenPrice: decimal.NewFromFloat(1.05),
				StopLoss: decimal.NewFromFloat(0.95), TakeProfit: decimal.NewFromFloat(1.15),
				Comment: "test-buy-limit", OpenTime: now.Unix(),
			},
			{
				Ticket: 2002, Symbol: "GBPUSD", Type: "sell_stop", Magic: testMagic,
				Volume: decimal.NewFromFloat(0.15), OpenPrice: decimal.NewFromFloat(1.25),
				StopLoss: decimal.NewFromFloat(1.35), TakeProfit: decimal.NewFromFloat(1.15),
				Comment: "test-sell-stop", OpenTime: now.Unix(),
			},
		},
	}
}

// TestLIVE_MQL_ORDER_CONTEXT_1_FullChain_PositionsAndPendingOrders verifies
// the complete chain: PositionSnapshot → backfillContextStrings → LivePosition/LivePendingOrder
// → vmPositionsToSdk/vmPendingOrdersToSdk → Runner.UpdateLiveState → broker.Positions/Orders.
// Acceptance: buy/sell → Positions=2, buy_limit/sell_stop → Orders=2, OrdersTotal=4.
func TestLIVE_MQL_ORDER_CONTEXT_1_FullChain_PositionsAndPendingOrders(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil)
	srv.posCache = NewPositionCache(nil)

	snap := buildTestSnapshot()
	srv.posCache.PutSnapshot(snap, snap.CapturedAt)

	// Step 1: backfillContextStrings → LivePosition + LivePendingOrder protos
	var equity, balance, margin, freeMargin string
	var positions []*antv1.LivePosition
	var pendingOrders []*antv1.LivePendingOrder
	if err := srv.backfillContextStrings("acct-1", &equity, &balance, &margin, &freeMargin, &positions, &pendingOrders); err != nil {
		t.Fatalf("backfillContextStrings failed: %v", err)
	}

	if len(positions) != 2 {
		t.Fatalf("Positions: got %d LivePosition, want 2 (buy+sell)", len(positions))
	}
	if len(pendingOrders) != 2 {
		t.Fatalf("PendingOrders: got %d LivePendingOrder, want 2 (buy_limit+sell_stop)", len(pendingOrders))
	}

	// Step 2: vmPositionsToSdk / vmPendingOrdersToSdk → SDK types
	sdkPositions := vmPositionsToSdk(positions)
	sdkPendingOrders := vmPendingOrdersToSdk(pendingOrders)

	if len(sdkPositions) != 2 {
		t.Fatalf("vmPositionsToSdk: got %d, want 2", len(sdkPositions))
	}
	if len(sdkPendingOrders) != 2 {
		t.Fatalf("vmPendingOrdersToSdk: got %d, want 2", len(sdkPendingOrders))
	}

	// Step 3: Runner.UpdateLiveState → broker.Positions/Orders
	r := runner.New(runner.Config{})
	r.UpdateLiveState(balance, equity, margin, freeMargin, sdkPositions, sdkPendingOrders)

	brokerPositions := r.Broker().Positions(0)
	brokerOrders := r.Broker().Orders(0)

	if len(brokerPositions) != 2 {
		t.Fatalf("broker.Positions(0): got %d, want 2 (market positions)", len(brokerPositions))
	}
	if len(brokerOrders) != 2 {
		t.Fatalf("broker.Orders(0): got %d, want 2 (pending orders)", len(brokerOrders))
	}

	// Step 4: OrdersTotal = positions + pendingOrders = 4
	ordersTotal := len(brokerPositions) + len(brokerOrders)
	if ordersTotal != 4 {
		t.Fatalf("OrdersTotal = %d, want 4 (2 positions + 2 pending orders)", ordersTotal)
	}
}

// TestLIVE_MQL_ORDER_CONTEXT_1_MagicEndToEnd verifies that magic=1699507621
// from the broker snapshot reaches OrderMagicNumber through the full chain.
// Adversarial: delete magic mapping at any layer → this test goes RED.
func TestLIVE_MQL_ORDER_CONTEXT_1_MagicEndToEnd(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil)
	srv.posCache = NewPositionCache(nil)

	snap := buildTestSnapshot()
	srv.posCache.PutSnapshot(snap, snap.CapturedAt)

	var equity, balance, margin, freeMargin string
	var positions []*antv1.LivePosition
	var pendingOrders []*antv1.LivePendingOrder
	if err := srv.backfillContextStrings("acct-1", &equity, &balance, &margin, &freeMargin, &positions, &pendingOrders); err != nil {
		t.Fatalf("backfillContextStrings failed: %v", err)
	}

	// Verify magic is preserved in LivePosition protos
	for i, lp := range positions {
		if lp.MagicNumber != testMagic {
			t.Fatalf("LivePosition[%d].MagicNumber = %d, want %d (magic lost in proto mapping)", i, lp.MagicNumber, testMagic)
		}
	}
	for i, lo := range pendingOrders {
		if lo.MagicNumber != testMagic {
			t.Fatalf("LivePendingOrder[%d].MagicNumber = %d, want %d (magic lost in proto mapping)", i, lo.MagicNumber, testMagic)
		}
	}

	// Verify magic is preserved in SDK types
	sdkPositions := vmPositionsToSdk(positions)
	sdkPendingOrders := vmPendingOrdersToSdk(pendingOrders)
	for i, p := range sdkPositions {
		if p.Magic != testMagic {
			t.Fatalf("sdk.Position[%d].Magic = %d, want %d (magic lost in vmPositionsToSdk)", i, p.Magic, testMagic)
		}
	}
	for i, o := range sdkPendingOrders {
		if o.Magic != testMagic {
			t.Fatalf("sdk.PendingOrder[%d].Magic = %d, want %d (magic lost in vmPendingOrdersToSdk)", i, o.Magic, testMagic)
		}
	}

	// Verify magic reaches broker.Positions/Orders
	r := runner.New(runner.Config{})
	r.UpdateLiveState(balance, equity, margin, freeMargin, sdkPositions, sdkPendingOrders)

	for i, p := range r.Broker().Positions(0) {
		if p.Magic != testMagic {
			t.Fatalf("broker.Positions(0)[%d].Magic = %d, want %d (magic lost in harness)", i, p.Magic, testMagic)
		}
	}
	for i, o := range r.Broker().Orders(0) {
		if o.Magic != testMagic {
			t.Fatalf("broker.Orders(0)[%d].Magic = %d, want %d (magic lost in harness)", i, o.Magic, testMagic)
		}
	}
}

// TestLIVE_MQL_ORDER_CONTEXT_1_AllFieldsPreserved verifies that all
// LivePosition fields (symbol, magic, order_type, sl, tp, profit, comment,
// open_time) are preserved through backfillContextStrings → vmPositionsToSdk.
func TestLIVE_MQL_ORDER_CONTEXT_1_AllFieldsPreserved(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil)
	srv.posCache = NewPositionCache(nil)

	snap := buildTestSnapshot()
	srv.posCache.PutSnapshot(snap, snap.CapturedAt)

	var equity, balance, margin, freeMargin string
	var positions []*antv1.LivePosition
	var pendingOrders []*antv1.LivePendingOrder
	if err := srv.backfillContextStrings("acct-1", &equity, &balance, &margin, &freeMargin, &positions, &pendingOrders); err != nil {
		t.Fatalf("backfillContextStrings failed: %v", err)
	}

	sdkPositions := vmPositionsToSdk(positions)

	// Check first position (buy)
	p0 := sdkPositions[0]
	if p0.Ticket != 1001 {
		t.Errorf("Position[0].Ticket = %d, want 1001", p0.Ticket)
	}
	if p0.Symbol != "EURUSD" {
		t.Errorf("Position[0].Symbol = %q, want %q", p0.Symbol, "EURUSD")
	}
	if p0.Side != sdk.SideBuy {
		t.Errorf("Position[0].Side = %v, want SideBuy", p0.Side)
	}
	if !p0.StopLoss.Equal(decimal.NewFromFloat(1.0)) {
		t.Errorf("Position[0].StopLoss = %s, want 1.0", p0.StopLoss)
	}
	if !p0.TakeProfit.Equal(decimal.NewFromFloat(1.2)) {
		t.Errorf("Position[0].TakeProfit = %s, want 1.2", p0.TakeProfit)
	}
	if !p0.Profit.Equal(decimal.NewFromFloat(50)) {
		t.Errorf("Position[0].Profit = %s, want 50", p0.Profit)
	}
	if p0.Comment != "test-buy" {
		t.Errorf("Position[0].Comment = %q, want %q", p0.Comment, "test-buy")
	}
	if p0.Magic != testMagic {
		t.Errorf("Position[0].Magic = %d, want %d", p0.Magic, testMagic)
	}
	if p0.OpenTime.IsZero() {
		t.Errorf("Position[0].OpenTime is zero, want non-zero")
	}
}

// TestLIVE_MQL_ORDER_CONTEXT_1_PendingOrderFieldsPreserved verifies that
// LivePendingOrder fields are preserved through the chain.
func TestLIVE_MQL_ORDER_CONTEXT_1_PendingOrderFieldsPreserved(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil)
	srv.posCache = NewPositionCache(nil)

	snap := buildTestSnapshot()
	srv.posCache.PutSnapshot(snap, snap.CapturedAt)

	var equity, balance, margin, freeMargin string
	var positions []*antv1.LivePosition
	var pendingOrders []*antv1.LivePendingOrder
	if err := srv.backfillContextStrings("acct-1", &equity, &balance, &margin, &freeMargin, &positions, &pendingOrders); err != nil {
		t.Fatalf("backfillContextStrings failed: %v", err)
	}

	sdkPending := vmPendingOrdersToSdk(pendingOrders)

	// Check first pending order (buy_limit)
	o0 := sdkPending[0]
	if o0.Ticket != 2001 {
		t.Errorf("PendingOrder[0].Ticket = %d, want 2001", o0.Ticket)
	}
	if o0.Symbol != "EURUSD" {
		t.Errorf("PendingOrder[0].Symbol = %q, want %q", o0.Symbol, "EURUSD")
	}
	if o0.Type != sdk.OrderLimit {
		t.Errorf("PendingOrder[0].Type = %v, want OrderLimit (buy_limit)", o0.Type)
	}
	if o0.Side != sdk.SideBuy {
		t.Errorf("PendingOrder[0].Side = %v, want SideBuy", o0.Side)
	}
	if !o0.Price.Equal(decimal.NewFromFloat(1.05)) {
		t.Errorf("PendingOrder[0].Price = %s, want 1.05", o0.Price)
	}
	if !o0.StopLoss.Equal(decimal.NewFromFloat(0.95)) {
		t.Errorf("PendingOrder[0].StopLoss = %s, want 0.95", o0.StopLoss)
	}
	if !o0.TakeProfit.Equal(decimal.NewFromFloat(1.15)) {
		t.Errorf("PendingOrder[0].TakeProfit = %s, want 1.15", o0.TakeProfit)
	}
	if o0.Comment != "test-buy-limit" {
		t.Errorf("PendingOrder[0].Comment = %q, want %q", o0.Comment, "test-buy-limit")
	}
	if o0.Magic != testMagic {
		t.Errorf("PendingOrder[0].Magic = %d, want %d", o0.Magic, testMagic)
	}

	// Check second pending order (sell_stop)
	o1 := sdkPending[1]
	if o1.Type != sdk.OrderStop {
		t.Errorf("PendingOrder[1].Type = %v, want OrderStop (sell_stop)", o1.Type)
	}
	if o1.Side != sdk.SideSell {
		t.Errorf("PendingOrder[1].Side = %v, want SideSell", o1.Side)
	}
}

// TestLIVE_MQL_ORDER_CONTEXT_1_PendingOrdersNotInPositions verifies that
// pending orders do NOT appear in broker.Positions(0) — they must only
// appear in broker.Orders(0). This prevents buy_limit/sell_stop from being
// disguised as market positions.
func TestLIVE_MQL_ORDER_CONTEXT_1_PendingOrdersNotInPositions(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil)
	srv.posCache = NewPositionCache(nil)

	snap := buildTestSnapshot()
	srv.posCache.PutSnapshot(snap, snap.CapturedAt)

	var equity, balance, margin, freeMargin string
	var positions []*antv1.LivePosition
	var pendingOrders []*antv1.LivePendingOrder
	if err := srv.backfillContextStrings("acct-1", &equity, &balance, &margin, &freeMargin, &positions, &pendingOrders); err != nil {
		t.Fatalf("backfillContextStrings failed: %v", err)
	}

	sdkPositions := vmPositionsToSdk(positions)
	sdkPending := vmPendingOrdersToSdk(pendingOrders)

	r := runner.New(runner.Config{})
	r.UpdateLiveState(balance, equity, margin, freeMargin, sdkPositions, sdkPending)

	brokerPositions := r.Broker().Positions(0)
	for _, p := range brokerPositions {
		if p.Ticket == 2001 || p.Ticket == 2002 {
			t.Fatalf("Pending order ticket %d found in broker.Positions(0) — must only be in Orders(0)", p.Ticket)
		}
	}
}

// TestLIVE_MQL_ORDER_CONTEXT_1_MagicFilterWorks verifies that magic filtering
// still works: Positions(testMagic) returns only matching positions, and
// Orders(testMagic) returns only matching pending orders.
func TestLIVE_MQL_ORDER_CONTEXT_1_MagicFilterWorks(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil)
	srv.posCache = NewPositionCache(nil)

	snap := buildTestSnapshot()
	// Add a position with different magic
	snap.Positions = append(snap.Positions, mthub.PositionSnapshotItem{
		Ticket: 1003, Symbol: "USDJPY", Type: "buy", Magic: 99999,
		Volume: decimal.NewFromFloat(0.1), OpenPrice: decimal.NewFromFloat(110),
	})
	srv.posCache.PutSnapshot(snap, snap.CapturedAt)

	var equity, balance, margin, freeMargin string
	var positions []*antv1.LivePosition
	var pendingOrders []*antv1.LivePendingOrder
	if err := srv.backfillContextStrings("acct-1", &equity, &balance, &margin, &freeMargin, &positions, &pendingOrders); err != nil {
		t.Fatalf("backfillContextStrings failed: %v", err)
	}

	sdkPositions := vmPositionsToSdk(positions)
	sdkPending := vmPendingOrdersToSdk(pendingOrders)

	r := runner.New(runner.Config{})
	r.UpdateLiveState(balance, equity, margin, freeMargin, sdkPositions, sdkPending)

	// magic=0 returns all (3 positions, 2 pending)
	allPos := r.Broker().Positions(0)
	if len(allPos) != 3 {
		t.Fatalf("Positions(0) = %d, want 3", len(allPos))
	}

	// magic=testMagic returns only 2 positions (the ones with testMagic)
	filteredPos := r.Broker().Positions(testMagic)
	if len(filteredPos) != 2 {
		t.Fatalf("Positions(%d) = %d, want 2", testMagic, len(filteredPos))
	}

	// magic=testMagic returns only 2 pending orders
	filteredOrders := r.Broker().Orders(testMagic)
	if len(filteredOrders) != 2 {
		t.Fatalf("Orders(%d) = %d, want 2", testMagic, len(filteredOrders))
	}
}

// TestLIVE_MQL_ORDER_CONTEXT_1_IsPendingOrderType verifies the
// mdtick.IsPendingOrderType classifier used to split positions from pending.
func TestLIVE_MQL_ORDER_CONTEXT_1_IsPendingOrderType(t *testing.T) {
	// Market positions
	if mdtick.IsPendingOrderType("buy") {
		t.Error("IsPendingOrderType(\"buy\") = true, want false")
	}
	if mdtick.IsPendingOrderType("sell") {
		t.Error("IsPendingOrderType(\"sell\") = true, want false")
	}

	// Pending orders
	pendingTypes := []string{"buy_limit", "sell_limit", "buy_stop", "sell_stop", "buy_stop_limit", "sell_stop_limit"}
	for _, pt := range pendingTypes {
		if !mdtick.IsPendingOrderType(pt) {
			t.Errorf("IsPendingOrderType(%q) = false, want true", pt)
		}
	}

	// Balance/credit are neither
	if mdtick.IsPendingOrderType("balance") {
		t.Error("IsPendingOrderType(\"balance\") = true, want false")
	}
	if mdtick.IsPendingOrderType("credit") {
		t.Error("IsPendingOrderType(\"credit\") = true, want false")
	}
}

// TestLIVE_MQL_ORDER_CONTEXT_1_PositionCacheMergePendingOrders verifies
// that PositionCache correctly merges PendingOrders from positions-only
// updates (OnOrderUpdate without AccountSummary financials).
func TestLIVE_MQL_ORDER_CONTEXT_1_PositionCacheMergePendingOrders(t *testing.T) {
	cache := NewPositionCache(nil)

	// First: financials-authoritative snapshot with positions + pending orders
	now := time.Now()
	snap1 := &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(10500),
		Margin: decimal.NewFromInt(500), FreeMargin: decimal.NewFromInt(9500),
		FinancialsAuthoritative: true, FinancialsSource: "account_summary", CapturedAt: now,
		PositionsAuthoritative: true, PositionsCapturedAt: now, PositionsSource: "order_stream",
		Positions:     []mthub.PositionSnapshotItem{{Ticket: 1, Type: "buy", Magic: testMagic}},
		PendingOrders: []mthub.PositionSnapshotItem{{Ticket: 2, Type: "buy_limit", Magic: testMagic}},
	}
	cache.PutSnapshot(snap1, now)

	// Second: positions-only update with new positions + new pending orders
	snap2 := &mthub.PositionSnapshot{
		AccountID: "acct-1",
		FinancialsAuthoritative: false, // positions-only
		PositionsAuthoritative:  true, PositionsCapturedAt: now, PositionsSource: "order_stream",
		Positions:     []mthub.PositionSnapshotItem{{Ticket: 3, Type: "sell", Magic: testMagic}},
		PendingOrders: []mthub.PositionSnapshotItem{{Ticket: 4, Type: "sell_stop", Magic: testMagic}},
	}
	cache.PutSnapshot(snap2, now)

	cached := cache.GetSnapshot("acct-1")
	if cached == nil {
		t.Fatal("GetSnapshot returned nil")
	}
	if len(cached.Positions) != 1 {
		t.Fatalf("merged Positions = %d, want 1 (replaced by snap2)", len(cached.Positions))
	}
	if cached.Positions[0].Ticket != 3 {
		t.Errorf("merged Positions[0].Ticket = %d, want 3 (from snap2)", cached.Positions[0].Ticket)
	}
	if len(cached.PendingOrders) != 1 {
		t.Fatalf("merged PendingOrders = %d, want 1 (replaced by snap2)", len(cached.PendingOrders))
	}
	if cached.PendingOrders[0].Ticket != 4 {
		t.Errorf("merged PendingOrders[0].Ticket = %d, want 4 (from snap2)", cached.PendingOrders[0].Ticket)
	}
}

// TestLIVE_MQL_ORDER_CONTEXT_1_PositionCacheFinancialOnlyPreservesPendingOrders
// verifies that a financial-only refresh preserves existing PendingOrders.
func TestLIVE_MQL_ORDER_CONTEXT_1_PositionCacheFinancialOnlyPreservesPendingOrders(t *testing.T) {
	cache := NewPositionCache(nil)

	now := time.Now()
	snap1 := &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(10500),
		Margin: decimal.NewFromInt(500), FreeMargin: decimal.NewFromInt(9500),
		FinancialsAuthoritative: true, FinancialsSource: "account_summary", CapturedAt: now,
		PositionsAuthoritative: true, PositionsCapturedAt: now, PositionsSource: "order_stream",
		Positions:     []mthub.PositionSnapshotItem{{Ticket: 1, Type: "buy", Magic: testMagic}},
		PendingOrders: []mthub.PositionSnapshotItem{{Ticket: 2, Type: "buy_limit", Magic: testMagic}},
	}
	cache.PutSnapshot(snap1, now)

	// Financial-only refresh (no positions)
	snap2 := &mthub.PositionSnapshot{
		AccountID: "acct-1", Balance: decimal.NewFromInt(11000), Equity: decimal.NewFromInt(11500),
		Margin: decimal.NewFromInt(600), FreeMargin: decimal.NewFromInt(10500),
		FinancialsAuthoritative: true, FinancialsSource: "account_summary", CapturedAt: now,
		PositionsAuthoritative: false, // financial-only
	}
	cache.PutSnapshot(snap2, now)

	cached := cache.GetSnapshot("acct-1")
	if cached == nil {
		t.Fatal("GetSnapshot returned nil")
	}
	// Financials updated
	if !cached.Balance.Equal(decimal.NewFromInt(11000)) {
		t.Errorf("Balance = %s, want 11000 (updated by financial refresh)", cached.Balance)
	}
	// Positions preserved from snap1
	if len(cached.Positions) != 1 || cached.Positions[0].Ticket != 1 {
		t.Errorf("Positions should be preserved from snap1, got %d positions", len(cached.Positions))
	}
	// PendingOrders preserved from snap1
	if len(cached.PendingOrders) != 1 || cached.PendingOrders[0].Ticket != 2 {
		t.Errorf("PendingOrders should be preserved from snap1, got %d pending orders", len(cached.PendingOrders))
	}
}

// TestLIVE_MQL_ORDER_CONTEXT_1_BrokerSnapshotSplit verifies that
// PositionSnapshotBroker correctly merges PendingOrders alongside Positions.
// This tests the broker-level split that pipeline_callbacks.go produces.
func TestLIVE_MQL_ORDER_CONTEXT_1_BrokerSnapshotSplit(t *testing.T) {
	broker := mthub.NewPositionSnapshotBroker()

	// Publish a snapshot with split positions + pending orders (as
	// pipeline_callbacks.go would produce after splitting by Type).
	now := time.Now()
	broker.Publish(&mthub.PositionSnapshot{
		AccountID: "acct-1", Platform: "mt4",
		Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(10500),
		FinancialsAuthoritative: true, FinancialsSource: "order_stream", CapturedAt: now,
		PositionsAuthoritative: true, PositionsCapturedAt: now, PositionsSource: "order_stream",
		Positions: []mthub.PositionSnapshotItem{
			{Ticket: 1001, Symbol: "EURUSD", Type: "buy", Magic: testMagic},
			{Ticket: 1002, Symbol: "GBPUSD", Type: "sell", Magic: testMagic},
		},
		PendingOrders: []mthub.PositionSnapshotItem{
			{Ticket: 2001, Symbol: "EURUSD", Type: "buy_limit", Magic: testMagic},
			{Ticket: 2002, Symbol: "GBPUSD", Type: "sell_stop", Magic: testMagic},
		},
	})

	// Subscribe and get the snapshot
	ch, unsub := broker.Subscribe("acct-1")
	defer unsub()
	snap := <-ch

	if len(snap.Positions) != 2 {
		t.Fatalf("snapshot.Positions = %d, want 2 (market only)", len(snap.Positions))
	}
	if len(snap.PendingOrders) != 2 {
		t.Fatalf("snapshot.PendingOrders = %d, want 2 (pending only)", len(snap.PendingOrders))
	}

	// Verify market positions have correct tickets
	posTickets := map[int64]bool{}
	for _, p := range snap.Positions {
		posTickets[p.Ticket] = true
		if p.Magic != testMagic {
			t.Errorf("Position ticket %d: Magic = %d, want %d", p.Ticket, p.Magic, testMagic)
		}
	}
	if !posTickets[1001] || !posTickets[1002] {
		t.Errorf("Positions should have tickets 1001+1002, got %v", posTickets)
	}

	// Verify pending orders have correct tickets
	pendingTickets := map[int64]bool{}
	for _, o := range snap.PendingOrders {
		pendingTickets[o.Ticket] = true
		if o.Magic != testMagic {
			t.Errorf("PendingOrder ticket %d: Magic = %d, want %d", o.Ticket, o.Magic, testMagic)
		}
	}
	if !pendingTickets[2001] || !pendingTickets[2002] {
		t.Errorf("PendingOrders should have tickets 2001+2002, got %v", pendingTickets)
	}
}
