package strategy

// VM-LIVE-PARITY-F2: live symbol-info backfill — populates the full
// SymbolParam fact set (point/digits/contract/stops + lots/tick/swap) onto
// LiveStrategyContext and TickContext from pre-fetched symbol params
// (W2: no per-event RPC). Extracted from live_context.go to keep file sizes
// within the line budget.

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

// backfillSymbolInfo populates symbol facts on LiveStrategyContext from the
// pre-fetched symbol params (W2: no per-event RPC). Falls back to a one-shot
// 5s-timeout fetch if startup pre-fetch failed.
func (s *StrategyExecutionServer) backfillSymbolInfo(cfg LiveStrategyConfig, lctx *antv1.LiveStrategyContext) {
	param := cfg.SymbolParam
	if param == nil && s.mtHub != nil && cfg.AccountID != "" && cfg.Symbol != "" {
		fetchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		param, _ = s.mtHub.CachedSymbolParam(fetchCtx, cfg.AccountID, cfg.Symbol)
		cancel()
	}
	if param == nil {
		return
	}
	lctx.Point = pointOrDerived(param)
	lctx.Digits = param.Digits
	lctx.ContractSize = param.ContractSize.String()
	lctx.StopsLevel = param.StopLevel
	// VM-LIVE-PARITY-F2: live VM sees the full SymbolParam fact set.
	lctx.LotMin = param.LotMin.String()
	lctx.LotMax = param.LotMax.String()
	lctx.LotStep = param.LotStep.String()
	lctx.TickValue = param.TickValue.String()
	lctx.TickSize = param.TickSize.String()
	lctx.SwapLong = param.SwapLong.String()
	lctx.SwapShort = param.SwapShort.String()
}

// backfillTickSymbolInfo populates symbol facts on TickContext from the
// pre-fetched symbol params (W2: no per-event RPC). Falls back to a one-shot
// 5s-timeout fetch if startup pre-fetch failed.
func (s *StrategyExecutionServer) backfillTickSymbolInfo(cfg LiveStrategyConfig, tctx *antv1.TickContext) {
	param := cfg.SymbolParam
	if param == nil && s.mtHub != nil && cfg.AccountID != "" && cfg.Symbol != "" {
		fetchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		param, _ = s.mtHub.CachedSymbolParam(fetchCtx, cfg.AccountID, cfg.Symbol)
		cancel()
	}
	if param == nil {
		return
	}
	tctx.Point = pointOrDerived(param)
	tctx.Digits = param.Digits
	tctx.ContractSize = param.ContractSize.String()
	tctx.StopsLevel = param.StopLevel
	// VM-LIVE-PARITY-F2: live VM sees the full SymbolParam fact set.
	tctx.LotMin = param.LotMin.String()
	tctx.LotMax = param.LotMax.String()
	tctx.LotStep = param.LotStep.String()
	tctx.TickValue = param.TickValue.String()
	tctx.TickSize = param.TickSize.String()
	tctx.SwapLong = param.SwapLong.String()
	tctx.SwapShort = param.SwapShort.String()
}

// pointOrDerived returns the broker-reported point size, or — only when the
// broker gave none and the symbol has digits — the definitional derivation
// 10^-digits (e.g. Digits=2 → "0.01"). This is the sole allowed derivation
// in the parity model: it is a quoting convention, not a broker fact, and is
// labeled as such. Adapter outputs stay raw (0 = unknown); derivation happens
// only at the VM boundary.
func pointOrDerived(param *mthub.SymbolParam) string {
	if !param.PointValue.IsZero() {
		return param.PointValue.String()
	}
	if param.Digits > 0 {
		return decimal.New(1, -int32(param.Digits)).String()
	}
	return param.PointValue.String()
}
