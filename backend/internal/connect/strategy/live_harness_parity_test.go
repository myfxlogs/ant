package strategy

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
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
	lctx := srv.buildLiveContext(context.Background(), LiveStrategyConfig{Symbol: "EURUSD", Timeframe: "M5"}, bars, nil)
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

	srv.posCache.snapshots["acct1"] = &mthub.PositionSnapshot{
		Balance:    decimal.NewFromInt(10000),
		Equity:     decimal.NewFromInt(10500),
		Margin:     decimal.NewFromInt(500),
		FreeMargin: decimal.NewFromInt(9500),
	}

	var equity, balance, margin, freeMargin string
	var positions []*antv1.LivePosition
	srv.backfillContextStrings("acct1", &equity, &balance, &margin, &freeMargin, &positions)

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
	srv.backfillContextStrings("nonexistent", &equity, &balance, &margin, &freeMargin, &positions)

	if margin != "-1" {
		t.Errorf("margin=%s, want -1 (fail-visible)", margin)
	}
	if freeMargin != "-1" {
		t.Errorf("freeMargin=%s, want -1 (fail-visible)", freeMargin)
	}
}
