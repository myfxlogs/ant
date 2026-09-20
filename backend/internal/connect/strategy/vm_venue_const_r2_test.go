package strategy

// VM-LIVE-VENUE-R2 T2: venue trade enum backfill — param absent → -1 unknown
// sentinel on both contexts; param present → verbatim passthrough (0 =
// disabled / no-freeze is a true value, never bleached to -1).
// Mutations: drop the -1 defaults → sentinel test RED; bleach 0 →
// passthrough test RED.

import (
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

func TestBackfill_TradeEnums_SentinelWhenParamAbsent(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil) // mtHub nil → no fallback fetch

	lctx := &antv1.LiveStrategyContext{}
	srv.backfillSymbolInfo(LiveStrategyConfig{}, lctx)
	if lctx.TradeMode != -1 || lctx.FreezeLevel != -1 || lctx.TradeExemode != -1 {
		t.Errorf("param==nil: LSC trade enums = %d/%d/%d, want -1/-1/-1 (unknown sentinel)",
			lctx.TradeMode, lctx.FreezeLevel, lctx.TradeExemode)
	}

	tctx := &antv1.TickContext{}
	srv.backfillTickSymbolInfo(LiveStrategyConfig{}, tctx)
	if tctx.TradeMode != -1 || tctx.FreezeLevel != -1 || tctx.TradeExemode != -1 {
		t.Errorf("param==nil: Tick trade enums = %d/%d/%d, want -1/-1/-1 (unknown sentinel)",
			tctx.TradeMode, tctx.FreezeLevel, tctx.TradeExemode)
	}
}

func TestBackfill_TradeEnums_VerbatimPassthrough(t *testing.T) {
	srv := NewStrategyExecutionServer(nil, nil)
	// mt5 disabled symbol: TradeMode=0 and FreezeLevel=0 are REAL values.
	param := &mthub.SymbolParam{Digits: 2, TradeMode: 0, FreezeLevel: 0, TradeExemode: 2}

	lctx := &antv1.LiveStrategyContext{}
	srv.backfillSymbolInfo(LiveStrategyConfig{SymbolParam: param}, lctx)
	if lctx.TradeMode != 0 || lctx.FreezeLevel != 0 {
		t.Errorf("param 0/0 bleached to %d/%d — 0 is a true value (disabled/no-freeze), never -1",
			lctx.TradeMode, lctx.FreezeLevel)
	}
	if lctx.TradeExemode != 2 {
		t.Errorf("LSC TradeExemode = %d, want param verbatim 2", lctx.TradeExemode)
	}

	tctx := &antv1.TickContext{}
	srv.backfillTickSymbolInfo(LiveStrategyConfig{SymbolParam: param}, tctx)
	if tctx.TradeMode != 0 || tctx.FreezeLevel != 0 || tctx.TradeExemode != 2 {
		t.Errorf("Tick passthrough = %d/%d/%d, want 0/0/2 verbatim",
			tctx.TradeMode, tctx.FreezeLevel, tctx.TradeExemode)
	}
}
