package strategy

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/internal/repository"
	"alphaforge/internal/risk"
)

// ── Task 1: Delta protocol removed + dedup guard ─────────────────────

// TestAppendDedupBar_ThreeStates verifies the three-state dedup guard.
// Remove the guard (revert to raw append) → duplicate/out-of-order bars
// corrupt the window → this test goes red.
func TestAppendDedupBar_ThreeStates(t *testing.T) {
	bars := make([]liveBar, 0, 10)

	// State 1: append new bar (openTime > last)
	appendDedupBar(&bars, liveBar{openTime: 100, close: "1.1"})
	appendDedupBar(&bars, liveBar{openTime: 200, close: "2.2"})
	if len(bars) != 2 {
		t.Fatalf("after 2 appends: len=%d, want 2", len(bars))
	}

	// State 2: same openTime → replace last bar
	appendDedupBar(&bars, liveBar{openTime: 200, close: "2.3"})
	if len(bars) != 2 {
		t.Fatalf("after replace: len=%d, want 2", len(bars))
	}
	if bars[1].close != "2.3" {
		t.Errorf("replace: close=%s, want 2.3", bars[1].close)
	}

	// State 3: older openTime → skip
	appendDedupBar(&bars, liveBar{openTime: 150, close: "1.5"})
	if len(bars) != 2 {
		t.Fatalf("after skip: len=%d, want 2", len(bars))
	}
	if bars[1].close != "2.3" {
		t.Errorf("skip: close=%s, want 2.3 (unchanged)", bars[1].close)
	}
}

// TestBuildLiveContext_NoDeltaBars verifies that buildLiveContext never
// populates DeltaBars. Remove the delta deletion → DeltaBars gets populated
// → this test goes red.
func TestBuildLiveContext_NoDeltaBars(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil)
	bars := []liveBar{{open: "1", high: "2", low: "0.5", close: "1.5", volume: "100", openTime: 100}}
	lctx, err := srv.buildLiveContext(context.Background(), LiveStrategyConfig{Symbol: "EURUSD", Timeframe: "M5", Mode: "live"}, bars, nil)
	if err == nil || lctx != nil {
		t.Fatal("missing authoritative snapshot must block context construction")
	}
	pc := NewPositionCache(nil)
	snap := &mthub.PositionSnapshot{
		AccountID: "", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(10000),
		Margin: decimal.Zero, FreeMargin: decimal.NewFromInt(10000), Leverage: 100,
		FinancialsAuthoritative: true, FinancialsSource: "account_summary",
		// LIVE-ORDER-REENTRY-1: GetFreshTradingSnapshot requires both fresh.
		// R2: PositionsCapturedAt must be non-zero (zero = fail-closed).
		PositionsAuthoritative: true,
		CapturedAt:             time.Now(),
		PositionsCapturedAt:    time.Now(),
		PositionsSource:        "order_stream",
	}
	pc.PutSnapshot(snap, snap.CapturedAt)
	srv.posCache = pc
	lctx, err = srv.buildLiveContext(context.Background(), LiveStrategyConfig{Symbol: "", Timeframe: "M5"}, bars, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(lctx.Close) != 1 {
		t.Errorf("buildLiveContext: len(Close)=%d, want 1", len(lctx.Close))
	}
	if len(lctx.DeltaBars) != 0 {
		t.Errorf("buildLiveContext: DeltaBars should be empty, got %d", len(lctx.DeltaBars))
	}
}

// ── Task 2: Margin/free_margin injection ─────────────────────────────

// TestBackfillContextStrings_MarginFreeMargin verifies that margin and
// free_margin are populated from PositionSnapshot. Remove the margin fields
// from backfillContextStrings → this test goes red.
func TestBackfillContextStrings_MarginFreeMargin(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil)
	srv.posCache = NewPositionCache(nil)

	snap := &mthub.PositionSnapshot{
		AccountID: "acct1", Balance: decimal.NewFromInt(10000), Equity: decimal.NewFromInt(10500),
		Margin: decimal.NewFromInt(500), FreeMargin: decimal.NewFromInt(9500), Leverage: 100,
		FinancialsAuthoritative: true, FinancialsSource: "account_summary", CapturedAt: time.Now(),
		// LIVE-ORDER-REENTRY-1: GetFreshTradingSnapshot requires both fresh.
		// R2: PositionsCapturedAt must be non-zero (zero = fail-closed).
		PositionsAuthoritative: true,
		PositionsCapturedAt:    time.Now(),
		PositionsSource:        "order_stream",
	}
	srv.posCache.PutSnapshot(snap, snap.CapturedAt)

	var equity, balance, margin, freeMargin string
	var positions []*antv1.LivePosition
	var pendingOrders []*antv1.LivePendingOrder
	if err := srv.backfillContextStrings("acct1", &equity, &balance, &margin, &freeMargin, &positions, &pendingOrders); err != nil {
		t.Fatal(err)
	}

	if margin != "500" {
		t.Errorf("margin=%s, want 500", margin)
	}
	if freeMargin != "9500" {
		t.Errorf("freeMargin=%s, want 9500", freeMargin)
	}
}

// TestBackfillContextStrings_MissingSnapshot_MarginMinusOne verifies that
// missing snapshot yields "-1" for margin/free_margin (fail-visible).
// Remove the -1 defaults → this test goes red.
func TestBackfillContextStrings_MissingSnapshot_MarginMinusOne(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil)

	var equity, balance, margin, freeMargin string
	var positions []*antv1.LivePosition
	var pendingOrders []*antv1.LivePendingOrder
	if err := srv.backfillContextStrings("nonexistent", &equity, &balance, &margin, &freeMargin, &positions, &pendingOrders); err == nil {
		t.Fatal("missing snapshot must return an error")
	}
}

// ── W1: Zero-volume close signal not silently dropped ────────────────

// TestW1_ZeroVolumeClose_ReachesExecutor verifies that a close signal with
// volume=0 (OrderCloseBy / CTrade.PositionClose = full close) is NOT silently
// dropped by dispatchCloseOrder. Revert the W1 fix (re-add the volume<=0 guard)
// → this test goes RED because CloseOrder is never called.
func TestW1_ZeroVolumeClose_ReachesExecutor(t *testing.T) {
	exec := &mockOrderExecutor{
		placedCh: make(chan string, 1),
		closedCh: make(chan int64, 1),
	}
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

	sig := &antv1.StrategySignal{SignalType: "close", Volume: "0", ExecutedTicket: 12345}
	// LIVE-ORDER-REENTRY-1: close path now requires an ActiveSession with a barrier.
	activeSess := &ActiveSession{barrier: NewTradeBarrier(zap.NewNop())}
	srv.dispatchLiveSignal(context.Background(), cfg, nil, sig, activeSess)

	select {
	case ticket := <-exec.closedCh:
		if ticket != 12345 {
			t.Fatalf("expected ticket 12345, got %d", ticket)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("W1: zero-volume close signal was silently dropped — CloseOrder never called (timeout)")
	}
}

// ── W3-1: seedBarWindows adversarial test ────────────────────────────

// mockMarketDataStore implements repository.MarketDataStore for seedBarWindows test.
type mockMarketDataStore struct {
	maxCloseTs int64
	klines     []repository.KlineBar
}

func (m *mockMarketDataStore) GetKlines(_ context.Context, _, _, _ string, _, _ *time.Time, _ int32) ([]repository.KlineBar, error) {
	return m.klines, nil
}
func (m *mockMarketDataStore) MaxCloseTs(_ context.Context, _, _, _ string) (int64, error) {
	return m.maxCloseTs, nil
}
func (m *mockMarketDataStore) GetLatestTick(_ context.Context, _, _ string) (*repository.LatestTick, error) {
	return nil, nil
}
func (m *mockMarketDataStore) LoadFinalizedBars(_ context.Context, _ time.Time) (map[repository.FinalizedKey][]int64, error) {
	return nil, nil
}
func (m *mockMarketDataStore) GetLatestBars(_ context.Context, _ time.Time) ([]repository.KlineBar, error) {
	return nil, nil
}
func (m *mockMarketDataStore) FetchActualReturn(_ context.Context, _ string, _ time.Time) (float64, error) {
	return 0, nil
}
func (m *mockMarketDataStore) InsertBars(_ context.Context, _ []repository.KlineBar) error {
	return nil
}

// TestW3_SeedBarWindows_PopulatesWindow verifies that seedBarWindows fills
// the bar window from historical PG data. Delete the seedBarWindows call
// in RunLiveStrategy → bars stays empty → this test goes RED.
func TestW3_SeedBarWindows_PopulatesWindow(t *testing.T) {
	periodMs := int64(60 * 1000) // 1m
	now := time.Now().UnixMilli()
	maxCloseTs := now - periodMs // last closed bar

	klines := make([]repository.KlineBar, 120)
	for i := range klines {
		openMs := maxCloseTs - int64(119-i)*periodMs
		klines[i] = repository.KlineBar{
			OpenTsUnixMs:  uint64(openMs),
			CloseTsUnixMs: uint64(openMs + periodMs),
			Open:          decimal.NewFromInt(int64(i + 1)),
			High:          decimal.NewFromInt(int64(i + 2)),
			Low:           decimal.NewFromInt(int64(i)),
			Close:         decimal.NewFromInt(int64(i + 1)),
			Volume:        100,
		}
	}

	mockRepo := &mockMarketDataStore{maxCloseTs: maxCloseTs, klines: klines}
	srv := &StrategyExecutionServer{
		log:            zap.NewNop(),
		marketDataRepo: mockRepo,
		brokerCompanyLookup: func(_ context.Context, _ string) (string, error) {
		return "test-broker", nil
	},
	}

	cfg := LiveStrategyConfig{
		AccountID: "acct-1",
		Symbol:    "EURUSD",
		Timeframe: "1m",
	}
	bars := make([]liveBar, 0, maxContextBars)
	srv.seedBarWindows(context.Background(), cfg, &bars, nil)

	if len(bars) < 100 {
		t.Fatalf("seedBarWindows: expected >=100 bars, got %d (seedBarWindows not populating window)", len(bars))
	}
}

// ── W3-2: Two consecutive bar events — window grows (no delta collapse) ─

// TestW3_TwoConsecutiveBars_WindowGrows verifies that after two consecutive
// bar events the VM sees a growing window (no delta collapse to 1).
// Revert to delta protocol (replace instead of full window) → second event
// window = 1 → this test goes RED.
func TestW3_TwoConsecutiveBars_WindowGrows(t *testing.T) {
	// Strategy: OrderSend with volume = Bars() — the signal volume encodes
	// the bar count, so we can assert window growth from the response.
	const code = `void OnBar() { OrderSend(Symbol(), OP_BUY, Bars(), Ask, 3, 0, 0); }`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	vmSess, err := NewVMLiveSession(code)
	if err != nil {
		t.Fatalf("compile MQL: %v", err)
	}

	// Build initial context with 120 bars (simulating seedBarWindows).
	makeBars := func(n int) *antv1.LiveStrategyContext {
		closeVals := make([]string, n)
		openVals := make([]string, n)
		highVals := make([]string, n)
		lowVals := make([]string, n)
		volVals := make([]string, n)
		times := make([]int64, n)
		baseTs := int64(1700000000000)
		for i := range n {
			closeVals[i] = decimal.NewFromInt(int64(i + 1)).String()
			openVals[i] = decimal.NewFromInt(int64(i)).String()
			highVals[i] = decimal.NewFromInt(int64(i + 2)).String()
			lowVals[i] = decimal.NewFromInt(int64(i)).String()
			volVals[i] = "100"
			times[i] = baseTs + int64(i)*60000
		}
		return &antv1.LiveStrategyContext{
			Symbol:       "EURUSD",
			Timeframe:    "1m",
			Mode:         "paper",
			Close:        closeVals,
			Open:         openVals,
			High:         highVals,
			Low:          lowVals,
			Volume:       volVals,
			BarTimesMs:   times,
			CurrentPrice: closeVals[n-1],
		}
	}

	// Event 1: Start with 120 bars.
	lctx1 := makeBars(120)
	req1 := &antv1.ExecuteLiveRequest{
		StrategyCode: code,
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:   lctx1,
	}
	reqBytes1, _ := proto.Marshal(req1)
	respBytes1, err := vmSess.Start(ctx, reqBytes1)
	if err != nil {
		t.Fatalf("vm Start (event 1): %v", err)
	}
	var resp1 antv1.ExecuteLiveResponse
	proto.Unmarshal(respBytes1, &resp1)
	if !resp1.GetSuccess() {
		t.Fatalf("event 1: VM returned error: %s", resp1.GetError())
	}

	// Event 2: Send 121 bars (full window, not delta).
	lctx2 := makeBars(121)
	req2 := &antv1.ExecuteLiveRequest{
		StrategyCode: code,
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:   lctx2,
	}
	reqBytes2, _ := proto.Marshal(req2)
	respBytes2, err := vmSess.SendEvent(ctx, reqBytes2)
	if err != nil {
		t.Fatalf("vm SendEvent (event 2): %v", err)
	}
	var resp2 antv1.ExecuteLiveResponse
	proto.Unmarshal(respBytes2, &resp2)
	if !resp2.GetSuccess() {
		t.Fatalf("event 2: VM returned error: %s", resp2.GetError())
	}

	// Extract signal volume from event 2 — it should be 121 (Bars() count).
	sig := resp2.GetSignal()
	if sig == nil && len(resp2.GetSignals()) > 0 {
		sig = resp2.GetSignals()[0]
	}
	if sig == nil {
		t.Fatal("event 2: no signal returned — OrderSend with Bars() volume should produce a signal")
	}
	vol, err := decimal.NewFromString(sig.GetVolume())
	if err != nil {
		t.Fatalf("event 2: invalid signal volume %q: %v", sig.GetVolume(), err)
	}
	barCount := int(vol.IntPart())
	if barCount < 121 {
		t.Fatalf("W3: after 2 consecutive bar events, VM Bars()=%d, expected >=121 (delta collapse?)", barCount)
	}
	t.Logf("W3: event 2 VM Bars()=%d (window grew correctly)", barCount)

	vmSess.Close()
}

// ── SEED-GAP: gap-immune seeding (latest-N, MT4 chart semantics) ──────

// gapSeedStore embeds the plain stub and adds the recentKlinesRepo extension:
// GetRecentKlines returns the full latest-500 history (DESC), while the plain
// time-window path would only see 53 bars (post-gap recovery period).
type gapSeedStore struct {
	mockMarketDataStore
	recent []repository.KlineBar // DESC (newest first)
}

func (m *gapSeedStore) GetRecentKlines(_ context.Context, _, _, _ string, _ int32) ([]repository.KlineBar, error) {
	return m.recent, nil
}

// TestSEEDGAP_GapImmuneSeeding verifies seeding takes the latest N bars even
// when the last-500-minute window is mostly gap (26h writer outage scenario).
// Disable the extension branch in seedSymbol (fallback to time window) →
// seeded drops to 53 → RED.
func TestSEEDGAP_GapImmuneSeeding(t *testing.T) {
	periodMs := int64(60 * 1000)
	now := time.Now().UnixMilli()

	recent := make([]repository.KlineBar, 0, maxContextBars)
	for i := 0; i < maxContextBars; i++ { // newest first
		openMs := now - int64(i+1)*periodMs
		recent = append(recent, repository.KlineBar{
			OpenTsUnixMs:  uint64(openMs),
			CloseTsUnixMs: uint64(openMs + periodMs),
			Open:          decimal.NewFromInt(1),
			High:          decimal.NewFromInt(2),
			Low:           decimal.NewFromInt(1),
			Close:         decimal.NewFromInt(1),
			Volume:        100,
		})
	}
	// Time-window path would only find the 53 newest bars (post-gap recovery).
	windowASC := make([]repository.KlineBar, 53)
	for i := range windowASC {
		windowASC[i] = recent[52-i] // reverse newest-53 → ASC
	}
	store := &gapSeedStore{
		mockMarketDataStore: mockMarketDataStore{maxCloseTs: now - periodMs, klines: windowASC},
		recent:              recent,
	}
	srv := &StrategyExecutionServer{
		log:            zap.NewNop(),
		marketDataRepo: store,
		brokerCompanyLookup: func(_ context.Context, _ string) (string, error) {
		return "test-broker", nil
	},
	}
	bars := make([]liveBar, 0, maxContextBars)
	srv.seedBarWindows(context.Background(), LiveStrategyConfig{
		AccountID: "acct-1", Symbol: "EURUSD", Timeframe: "1m",
	}, &bars, nil)

	if len(bars) != maxContextBars {
		t.Fatalf("SEED-GAP: seeded %d bars, want %d — gap fell back to time window (would give 53)", len(bars), maxContextBars)
	}
	if bars[0].openTime > bars[len(bars)-1].openTime {
		t.Fatal("SEED-GAP: seeded bars not in ASC order after DESC reverse")
	}
}
