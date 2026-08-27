package strategy

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

// buildTickContext creates a TickContext proto from a tick update.
func (s *StrategyExecutionServer) buildTickContext(ctx context.Context, cfg LiveStrategyConfig, tick *mthub.TickUpdate) (*antv1.TickContext, error) {
	tctx := &antv1.TickContext{
		Bid:          tick.Bid.String(),
		Ask:          tick.Ask.String(),
		Symbol:       cfg.Symbol,
		Timeframe:    cfg.Timeframe,
		Mode:         cfg.Mode,
		Params:       buildLiveParams(cfg.Params),
		CurrentPrice: tick.Bid.String(),
	}
	if err := s.backfillContextStrings(cfg.AccountID, &tctx.Equity, &tctx.Balance, &tctx.Margin, &tctx.FreeMargin, &tctx.Positions, &tctx.PendingOrders); err != nil && cfg.Mode == modeLive {
		return nil, err
	}
	s.backfillTickSymbolInfo(cfg, tctx)
	return tctx, nil
}

// buildTradeContext creates a TradeContext proto from a trade event.
func (s *StrategyExecutionServer) buildTradeContext(ctx context.Context, cfg LiveStrategyConfig, evt *mthub.BrokerTradeEvent) (*antv1.TradeContext, error) {
	side := sideBuy
	if evt.Side == sideSell {
		side = sideSell
	}
	evtType := "fill"
	switch evt.EventType {
	case mthub.BrokerTradeFilled:
		evtType = "fill"
	case mthub.BrokerTradeClosed:
		evtType = string(actionClose)
	case mthub.BrokerTradeModified:
		evtType = "modify"
	case mthub.BrokerTradeCancelled:
		evtType = "cancel"
	}
	tctx := &antv1.TradeContext{
		Ticket:     evt.Ticket,
		Symbol:     evt.Symbol,
		EventType:  evtType,
		Side:       side,
		Volume:     evt.Volume.String(),
		Price:      evt.Price.String(),
		StopLoss:   evt.StopLoss.String(),
		TakeProfit: evt.TakeProfit.String(),
		Profit:     evt.Profit.String(),
		Commission: evt.Commission.String(),
		Swap:       evt.Swap.String(),
	}
	if err := s.backfillContextStrings(cfg.AccountID, &tctx.Equity, &tctx.Balance, &tctx.Margin, &tctx.FreeMargin, &tctx.Positions, &tctx.PendingOrders); err != nil && cfg.Mode == modeLive {
		return nil, err
	}
	return tctx, nil
}

// backfillContextStrings populates equity/balance/margin/free_margin/positions/pending_orders
// from the push-based PositionCache (subscribed to PositionSnapshotBroker). No polling — O(1) read.
// Missing or stale authoritative snapshots return an error and block execution.
// LIVE-ORDER-REENTRY-1: uses GetFreshTradingSnapshot — both financials AND
// positions must be fresh for VM evaluation. A financial-only refresh must
// not make stale positions usable.
// LIVE-MQL-ORDER-CONTEXT-1: now also populates pending orders and all
// LivePosition fields (symbol, magic, order_type, sl, tp, profit, comment,
// open_time) so MQL OrdersTotal/OrderSelect/OrderMagicNumber preserve
// broker-original account-level semantics.
func (s *StrategyExecutionServer) backfillContextStrings(accountID string, equity, balance, margin, freeMargin *string, positions *[]*antv1.LivePosition, pendingOrders *[]*antv1.LivePendingOrder) error {
	if s.posCache == nil {
		return fmt.Errorf("authoritative account snapshot unavailable: position cache not configured")
	}
	snap, ok := s.posCache.GetFreshTradingSnapshot(accountID, time.Now())
	if !ok {
		return fmt.Errorf("authoritative account snapshot unavailable or stale: account=%s", accountID)
	}
	*equity = snap.Equity.String()
	*balance = snap.Balance.String()
	*margin = snap.Margin.String()
	*freeMargin = snap.FreeMargin.String()
	pos := make([]*antv1.LivePosition, 0, len(snap.Positions))
	for _, p := range snap.Positions {
		side := sideBuy
		if p.Type == sideSell {
			side = sideSell
		}
		pos = append(pos, &antv1.LivePosition{
			Ticket:      p.Ticket,
			Side:        side,
			Volume:      p.Volume.String(),
			OpenPrice:   p.OpenPrice.String(),
			Sl:          p.StopLoss.String(),
			Tp:          p.TakeProfit.String(),
			Swap:        p.Swap.String(),
			Commission:  p.Commission.String(),
			Symbol:      p.Symbol,
			MagicNumber: p.Magic,
			OrderType:   p.Type,
			Profit:      p.Profit.String(),
			Comment:     p.Comment,
			OpenTime:    p.OpenTime,
		})
	}
	*positions = pos

	// LIVE-MQL-ORDER-CONTEXT-1: populate pending orders with full fields.
	pending := make([]*antv1.LivePendingOrder, 0, len(snap.PendingOrders))
	for _, o := range snap.PendingOrders {
		pending = append(pending, &antv1.LivePendingOrder{
			Ticket:      o.Ticket,
			Symbol:      o.Symbol,
			OrderType:   o.Type,
			Side:        pendingOrderSide(o.Type),
			Volume:      o.Volume.String(),
			Price:       o.OpenPrice.String(),
			Sl:          o.StopLoss.String(),
			Tp:          o.TakeProfit.String(),
			Comment:     o.Comment,
			MagicNumber: o.Magic,
			OpenTime:    o.OpenTime,
		})
	}
	*pendingOrders = pending
	return nil
}

// pendingOrderSide derives the buy/sell side from a pending order type string.
// "buy_limit" → "buy", "sell_stop" → "sell", etc.
func pendingOrderSide(orderType string) string {
	if len(orderType) >= 3 && orderType[:3] == sideBuy {
		return sideBuy
	}
	return "sell"
}

// dispatchResponse dispatches a live response's signals to the OMS.
// FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP: receives *antv1.ExecuteLiveResponse
// directly (no proto unmarshal) — the in-process Session returns the struct
// pointer, preserving empty-slice semantics for repeated fields.
func (s *StrategyExecutionServer) dispatchResponse(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, resp *antv1.ExecuteLiveResponse, activeSess *ActiveSession) {
	if resp == nil {
		s.log.Error("LiveStrategyRunner: nil response from VM")
		if activeSess != nil {
			activeSess.RecordError("nil response from VM")
		}
		return
	}
	if !resp.GetSuccess() {
		s.log.Warn("LiveStrategyRunner: strategy returned error", zap.String("error", resp.GetError()))
		if activeSess != nil {
			activeSess.RecordError(resp.GetError())
		}
		return
	}
	signals := resp.GetSignals()
	if len(signals) == 0 {
		sig := resp.GetSignal()
		if sig != nil && sig.GetSignalType() != "" && sig.GetSignalType() != "hold" {
			signals = []*antv1.StrategySignal{sig}
		}
	}
	for _, sig := range signals {
		if sig.GetSignalType() == "hold" || sig.GetSignalType() == "" {
			continue
		}
		if activeSess != nil {
			activeSess.RecordSignal(&SignalEvent{
				RunID:      cfg.RunID,
				AccountID:  cfg.AccountID,
				Symbol:     cfg.Symbol,
				SignalType: sig.GetSignalType(),
				Volume:     sig.GetVolume(),
				Price:      sig.GetPrice(),
				StopLoss:   sig.GetStopLoss(),
				TakeProfit: sig.GetTakeProfit(),
				Reason:     sig.GetReason(),
				Timestamp:  time.Now(),
			})
		}
		if cfg.ShadowVerifier != nil {
			barTime := int64(0)
			if bar != nil {
				barTime = bar.OpenTime
			}
			cfg.ShadowVerifier.RecordLiveSignal(barTime, sig.GetSignalType(), sig.GetVolume(), sig.GetPrice())
		}
		s.dispatchLiveSignal(ctx, cfg, bar, sig, activeSess)
	}
}

// buildLiveContext creates a full OHLCV bar context from the bar window.
func (s *StrategyExecutionServer) buildLiveContext(ctx context.Context, cfg LiveStrategyConfig, bars []liveBar, extraBars map[string][]liveBar) (*antv1.LiveStrategyContext, error) {
	n := len(bars)
	closeVals := make([]string, n)
	openVals := make([]string, n)
	highVals := make([]string, n)
	lowVals := make([]string, n)
	volVals := make([]string, n)
	times := make([]int64, n)
	for i, b := range bars {
		closeVals[i] = b.close
		openVals[i] = b.open
		highVals[i] = b.high
		lowVals[i] = b.low
		volVals[i] = b.volume
		times[i] = b.openTime
	}
	lctx := &antv1.LiveStrategyContext{
		Close:      closeVals,
		Open:       openVals,
		High:       highVals,
		Low:        lowVals,
		Volume:     volVals,
		BarTimesMs: times,
		Symbol:     cfg.Symbol,
		Timeframe:  cfg.Timeframe,
		Mode:       cfg.Mode,
		Params:     buildLiveParams(cfg.Params),
	}
	if n > 0 {
		lctx.CurrentPrice = closeVals[n-1]
	}
	if err := s.backfillContextStrings(cfg.AccountID, &lctx.Equity, &lctx.Balance, &lctx.Margin, &lctx.FreeMargin, &lctx.Positions, &lctx.PendingOrders); err != nil && cfg.Mode == modeLive {
		return nil, err
	}
	s.backfillSymbolInfo(cfg, lctx)
	lctx.Symbols = buildSymbolSeries(extraBars)

	// VM-TRADE-CONTEXT-6 S6: inject server-side account truth.
	// Login/Company/IsDemo/IsConnected/IsTradeAllowed come from mt_accounts
	// (authoritative source), not from the client request. Live mode requires
	// all lookups to succeed (fail-closed); paper mode tolerates errors.
	if err := s.injectAccountTruth(ctx, cfg, lctx); err != nil {
		return nil, err
	}

	return lctx, nil
}

// injectAccountTruth populates Login/Company/IsDemo/IsConnected/IsTradeAllowed
// from server-side mt_accounts lookups. VM-TRADE-CONTEXT-6 S6.
//
// Live mode: all lookups must succeed — DB errors fail-closed (return error).
// Paper mode: lookup errors are non-fatal (fail-open for simulation).
// Investor accounts get IsTradeAllowed=false even when connected (investor
// accounts can view but not trade).
func (s *StrategyExecutionServer) injectAccountTruth(ctx context.Context, cfg LiveStrategyConfig, lctx *antv1.LiveStrategyContext) error {
	if cfg.AccountID == "" {
		return nil
	}

	// Login
	if s.accountLoginLookup != nil {
		login, err := s.accountLoginLookup(ctx, cfg.AccountID)
		if err != nil && cfg.Mode == modeLive {
			return fmt.Errorf("account login lookup failed: %w", err)
		} else if err == nil {
			lctx.Login = login
		}
	}

	// Company (broker) — reuse existing brokerCompanyLookup if available.
	if s.brokerCompanyLookup != nil {
		lctx.Company = s.brokerCompanyLookup(ctx, cfg.AccountID)
	}

	// IsDemo
	if err := s.lookupBool(ctx, cfg, s.accountIsDemoLookup, &lctx.IsDemo, "is_demo"); err != nil {
		return err
	}

	// IsConnected
	if err := s.lookupBool(ctx, cfg, s.accountConnectedLookup, &lctx.IsConnected, "connected"); err != nil {
		return err
	}

	// IsTradeAllowed
	if err := s.lookupBool(ctx, cfg, s.accountTradeAllowedLookup, &lctx.IsTradeAllowed, "trade_allowed"); err != nil {
		return err
	}

	// Investor gating: if account is investor, IsTradeAllowed must be false
	// even when connected and trade_allowed status is true.
	var isInvestor bool
	if err := s.lookupBool(ctx, cfg, s.accountIsInvestorLookup, &isInvestor, "is_investor"); err != nil {
		return err
	}
	if isInvestor {
		lctx.IsTradeAllowed = false
	}

	return nil
}

// lookupBool is a shared helper for bool lookups with live-mode fail-closed.
// VM-TRADE-CONTEXT-6 S6.
func (s *StrategyExecutionServer) lookupBool(ctx context.Context, cfg LiveStrategyConfig, fn func(ctx context.Context, accountID string) (bool, error), dst *bool, name string) error {
	if fn == nil {
		return nil
	}
	val, err := fn(ctx, cfg.AccountID)
	if err != nil {
		if cfg.Mode == modeLive {
			return fmt.Errorf("account %s lookup failed: %w", name, err)
		}
		return nil
	}
	*dst = val
	return nil
}

func buildLiveParams(params map[string]string) []*antv1.LiveParam {
	if len(params) == 0 {
		return nil
	}
	out := make([]*antv1.LiveParam, 0, len(params))
	for k, v := range params {
		out = append(out, &antv1.LiveParam{Key: k, Value: v})
	}
	return out
}

func buildSymbolSeries(extraBars map[string][]liveBar) []*antv1.LiveSymbolSeries {
	if len(extraBars) == 0 {
		return nil
	}
	out := make([]*antv1.LiveSymbolSeries, 0, len(extraBars))
	for sym, bars := range extraBars {
		if len(bars) == 0 {
			continue
		}
		n := len(bars)
		closeVals := make([]string, n)
		openVals := make([]string, n)
		highVals := make([]string, n)
		lowVals := make([]string, n)
		volVals := make([]string, n)
		for i, b := range bars {
			closeVals[i] = b.close
			openVals[i] = b.open
			highVals[i] = b.high
			lowVals[i] = b.low
			volVals[i] = b.volume
		}
		out = append(out, &antv1.LiveSymbolSeries{
			Symbol: sym,
			Close:  closeVals,
			Open:   openVals,
			High:   highVals,
			Low:    lowVals,
			Volume: volVals,
		})
	}
	return out
}

// backfillSymbolInfo populates Point/Digits/ContractSize/StopsLevel on
// LiveStrategyContext from the pre-fetched symbol params (W2: no per-event RPC).
// Falls back to a one-shot 5s-timeout fetch if startup pre-fetch failed.
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
	lctx.Point = param.PointValue.String()
	lctx.Digits = param.Digits
	lctx.ContractSize = param.ContractSize.String()
	lctx.StopsLevel = param.StopLevel
}

// backfillTickSymbolInfo populates Point/Digits/ContractSize/StopsLevel on
// TickContext from the pre-fetched symbol params (W2: no per-event RPC).
// Falls back to a one-shot 5s-timeout fetch if startup pre-fetch failed.
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
	tctx.Point = param.PointValue.String()
	tctx.Digits = param.Digits
	tctx.ContractSize = param.ContractSize.String()
	tctx.StopsLevel = param.StopLevel
}
