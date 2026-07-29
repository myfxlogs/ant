package strategy

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
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
	s.backfillContextStrings(cfg.AccountID, &tctx.Equity, &tctx.Balance, &tctx.Positions)
	return tctx
}

// buildTradeContext creates a TradeContext proto from a trade event.
func (s *StrategyExecutionServer) buildTradeContext(ctx context.Context, cfg LiveStrategyConfig, evt *mthub.BrokerTradeEvent) *antv1.TradeContext {
	side := "buy"
	if evt.Side == sideSell {
		side = sideSell
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
	s.backfillContextStrings(cfg.AccountID, &tctx.Equity, &tctx.Balance, &tctx.Positions)
	return tctx
}

// backfillContextStrings populates equity/balance/positions from the push-based
// PositionCache (subscribed to PositionSnapshotBroker). No polling — O(1) read.
func (s *StrategyExecutionServer) backfillContextStrings(accountID string, equity, balance *string, positions *[]*antv1.LivePosition) {
	if s.posCache == nil {
		*equity = "-1"
		*balance = "-1"
		return
	}
	snap := s.posCache.GetSnapshot(accountID)
	if snap == nil {
		*equity = "-1"
		*balance = "-1"
		return
	}
	*equity = snap.Equity.String()
	*balance = snap.Balance.String()
	pos := make([]*antv1.LivePosition, 0, len(snap.Positions))
	for _, p := range snap.Positions {
		side := "buy"
		if p.Type == sideSell {
			side = sideSell
		}
		pos = append(pos, &antv1.LivePosition{
			Ticket:    p.Ticket,
			Side:      side,
			Volume:    p.Volume.String(),
			OpenPrice: p.OpenPrice.String(),
		})
	}
	*positions = pos
}

// dispatchFromBytes unmarshals a live response and dispatches signals to OMS.
func (s *StrategyExecutionServer) dispatchFromBytes(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, respBytes []byte, activeSess *ActiveSession) {
	var resp antv1.ExecuteLiveResponse
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		s.log.Error("LiveStrategyRunner: unmarshal response failed", zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError(err.Error())
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
func (s *StrategyExecutionServer) buildLiveContext(ctx context.Context, cfg LiveStrategyConfig, bars []liveBar, extraBars map[string][]liveBar) *antv1.LiveStrategyContext {
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
	s.backfillContextStrings(cfg.AccountID, &lctx.Equity, &lctx.Balance, &lctx.Positions)
	lctx.Symbols = buildSymbolSeries(extraBars)
	return lctx
}

// buildDeltaContext creates a delta-bar context with only the latest bar.
func (s *StrategyExecutionServer) buildDeltaContext(ctx context.Context, cfg LiveStrategyConfig, bars []liveBar, extraBars map[string][]liveBar) *antv1.LiveStrategyContext {
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
	s.backfillContextStrings(cfg.AccountID, &lctx.Equity, &lctx.Balance, &lctx.Positions)
	lctx.Symbols = buildSymbolSeries(extraBars)
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
