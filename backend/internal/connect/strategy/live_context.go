package strategy

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/mthub"
)

// buildTickContext creates a TickContext proto from a tick update.
func (s *StrategyExecutionServer) buildTickContext(ctx context.Context, cfg LiveStrategyConfig, tick *mthub.TickUpdate) *antv1.TickContext {
	tctx := &antv1.TickContext{
		Bid:          tick.Bid.String(),
		Ask:          tick.Ask.String(),
		Symbol:       cfg.Symbol,
		Timeframe:    cfg.Timeframe,
		Mode:         cfg.Mode,
		Params:       buildLiveParams(cfg.Params),
		CurrentPrice: tick.Bid.String(),
	}
	s.backfillContextStrings(ctx, cfg.AccountID, &tctx.Equity, &tctx.Balance, &tctx.Positions)
	return tctx
}

// buildTradeContext creates a TradeContext proto from a trade event.
func (s *StrategyExecutionServer) buildTradeContext(ctx context.Context, cfg LiveStrategyConfig, evt *mthub.BrokerTradeEvent) *antv1.TradeContext {
	side := "buy"
	if evt.Side == "sell" {
		side = "sell"
	}
	evtType := "fill"
	switch evt.EventType {
	case mthub.BrokerTradeFilled:
		evtType = "fill"
	case mthub.BrokerTradeClosed:
		evtType = "close"
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
	s.backfillContextStrings(ctx, cfg.AccountID, &tctx.Equity, &tctx.Balance, &tctx.Positions)
	return tctx
}

// backfillContextStrings populates equity/balance/positions from live account state.
func (s *StrategyExecutionServer) backfillContextStrings(ctx context.Context, accountID string, equity, balance *string, positions *[]*antv1.LivePosition) {
	if s.accountProvider == nil {
		*equity = "-1"
		*balance = "-1"
		return
	}
	state, err := s.accountProvider.GetAccountState(ctx, accountID)
	if err != nil || state == nil {
		*equity = "-1"
		*balance = "-1"
		return
	}
	*equity = state.Equity.String()
	*balance = state.Balance.String()
	if s.mtHub != nil {
		orders, err := s.mtHub.OpenedOrders(ctx, accountID)
		if err == nil {
			pos := make([]*antv1.LivePosition, 0, len(orders))
			for _, o := range orders {
				side := "buy"
				if o.Side == mthub.SideSell {
					side = "sell"
				}
				pos = append(pos, &antv1.LivePosition{
					Ticket:    o.Ticket,
					Side:      side,
					Volume:    o.Volume.String(),
					OpenPrice: o.OpenPrice.String(),
				})
			}
			*positions = pos
		}
	}
}

// dispatchFromBytes unmarshals a live response and dispatches signals to OMS.
func (s *StrategyExecutionServer) dispatchFromBytes(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, respBytes []byte) {
	var resp antv1.ExecuteLiveResponse
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		s.log.Error("LiveStrategyRunner: unmarshal response failed", zap.Error(err))
		return
	}
	if !resp.GetSuccess() {
		s.log.Warn("LiveStrategyRunner: strategy returned error", zap.String("error", resp.GetError()))
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
		s.dispatchLiveSignal(ctx, cfg, bar, sig)
	}
}

// getExecutor returns the WASM executor pair (wasm + go fallback).
func (s *StrategyExecutionServer) getExecutor() *execPair {
	return &execPair{wasm: s.wasmExecutor, goExec: s.goExecutor}
}

type execPair struct {
	wasm   *WasmExecutor
	goExec *GoExecutor
}

// buildLiveContext creates a full OHLCV bar context from the bar window.
func (s *StrategyExecutionServer) buildLiveContext(ctx context.Context, cfg LiveStrategyConfig, bars []liveBar) *antv1.LiveStrategyContext {
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
	s.backfillContextStrings(ctx, cfg.AccountID, &lctx.Equity, &lctx.Balance, &lctx.Positions)
	return lctx
}

// buildDeltaContext creates a delta-bar context with only the latest bar.
func (s *StrategyExecutionServer) buildDeltaContext(ctx context.Context, cfg LiveStrategyConfig, bars []liveBar) *antv1.LiveStrategyContext {
	n := len(bars)
	if n == 0 {
		return &antv1.LiveStrategyContext{Symbol: cfg.Symbol, Timeframe: cfg.Timeframe, Mode: cfg.Mode, Params: buildLiveParams(cfg.Params)}
	}
	last := bars[n-1]
	lctx := &antv1.LiveStrategyContext{
		DeltaBars: []*antv1.DeltaBar{{Open: last.open, High: last.high, Low: last.low, Close: last.close, Volume: last.volume, BarTimeMs: last.openTime}},
		Symbol:    cfg.Symbol, Timeframe: cfg.Timeframe, Mode: cfg.Mode, Params: buildLiveParams(cfg.Params),
		CurrentPrice: last.close,
	}
	s.backfillContextStrings(ctx, cfg.AccountID, &lctx.Equity, &lctx.Balance, &lctx.Positions)
	return lctx
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
