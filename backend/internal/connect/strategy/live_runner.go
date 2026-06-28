// live_runner.go — LiveStrategyRunner: subscribes to real-time data streams
// (bar, tick, trade, timer), builds proto-native context per event, executes
// strategy via WASM, and dispatches signals to OMS.
//
// Multi-model architecture:
//   BAR   → barBroker channel   → session.SendBar()   → strategy.OnBar
//   TICK  → tickBroker channel  → session.SendTick()  → strategy.OnTick
//   TRADE → tradeBroker channel → session.SendTrade() → strategy.OnTrade
//   TIMER → timer channel       → session.SendTimer() → strategy.OnTimer
//
// Follows push-first architecture: events drive the loop; no polling.

package strategy

import (
	"context"
	"fmt"
	"strconv"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/mthub"
)

const maxContextBars = 500

// ExecutionModels bitmask for which callbacks the strategy supports.
type ExecutionModels int

const (
	ExecModelBar   ExecutionModels = 1 << iota // OnBar
	ExecModelTick                               // OnTick
	ExecModelTrade                              // OnTrade
	ExecModelTimer                              // OnTimer
)

type liveBar struct {
	open, high, low, close, volume string
	openTime                       int64
}

// LiveStrategyConfig holds the parameters for running a strategy in live/paper mode.
type LiveStrategyConfig struct {
	AccountID           string
	DataSourceAccountID string
	UserID              string
	Symbol              string
	Timeframe           string
	Code                string
	Mode                string // "live" | "paper"
	Params              map[string]string
	// Models specifies which execution callbacks the strategy implements.
	// Default (0) = Bar only.
	Models ExecutionModels
}

// LiveTickSubscriber provides tick (Bid/Ask) updates for an account.
type LiveTickSubscriber interface {
	SubscribeTickUpdates(accountID string) (<-chan *mthub.TickUpdate, func())
}

// LiveTradeSubscriber provides trade events for an account.
type LiveTradeSubscriber interface {
	SubscribeTradeEvents(accountID string) (<-chan *mthub.BrokerTradeEvent, func())
}

// RunLiveStrategy subscribes to real-time data streams for the given account/symbol/timeframe,
// builds proto-native context for each event, executes the strategy via WASM,
// and dispatches the resulting signals.
//
// Blocks until ctx is cancelled. Callers should run this in a goroutine.
func (s *StrategyExecutionServer) RunLiveStrategy(ctx context.Context, cfg LiveStrategyConfig) error {
	if s.barSource == nil {
		return fmt.Errorf("live strategy runner: no BarSource configured")
	}
	if s.gate == nil {
		return fmt.Errorf("live strategy runner: risk.Gate not injected — live trading blocked per D6-A")
	}

	source, ok := s.barSource.(LiveBarSubscriber)
	if !ok {
		return fmt.Errorf("live strategy runner: BarSource does not support streaming (got %s)", s.barSource.Name())
	}

	barAccountID := cfg.AccountID
	if cfg.DataSourceAccountID != "" {
		barAccountID = cfg.DataSourceAccountID
	}

	needsTick := cfg.Models&ExecModelTick != 0
	needsTrade := cfg.Models&ExecModelTrade != 0

	s.log.Info("LiveStrategyRunner: starting",
		zap.String("trading_account", cfg.AccountID),
		zap.String("bar_source_account", barAccountID),
		zap.String("symbol", cfg.Symbol),
		zap.String("timeframe", cfg.Timeframe),
		zap.String("mode", cfg.Mode),
		zap.Bool("tick", needsTick),
		zap.Bool("trade", needsTrade),
	)

	barCh, barCancel := source.Subscribe(barAccountID)
	defer barCancel()

	var tickCh <-chan *mthub.TickUpdate
	var tickCancel func()
	if needsTick && s.mtHub != nil {
		if ts, ok := interface{}(s.mtHub).(LiveTickSubscriber); ok {
			tickCh, tickCancel = ts.SubscribeTickUpdates(cfg.AccountID)
			defer tickCancel()
		}
	}

	var tradeCh <-chan *mthub.BrokerTradeEvent
	var tradeCancel func()
	if needsTrade && s.mtHub != nil {
		if ts, ok := interface{}(s.mtHub).(LiveTradeSubscriber); ok {
			tradeCh, tradeCancel = ts.SubscribeTradeEvents(cfg.AccountID)
			defer tradeCancel()
		}
	}

	bars := make([]liveBar, 0, maxContextBars)
	var session *LiveSession
	var firstBar bool = true
	exec := s.getExecutor()

	defer func() {
		if session != nil {
			session.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("LiveStrategyRunner: context cancelled, exiting")
			return nil

		case bar, ok := <-barCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: bar channel closed, exiting")
				return nil
			}
			if bar.Symbol != cfg.Symbol || bar.Period != cfg.Timeframe {
				continue
			}
			s.handleBar(ctx, cfg, exec, bar, &bars, &session, &firstBar)

		case tick, ok := <-tickCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: tick channel closed")
				tickCh = nil
				continue
			}
			if tick.Symbol != cfg.Symbol {
				continue
			}
			s.handleTick(ctx, cfg, exec, tick, &session, &firstBar)

		case evt, ok := <-tradeCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: trade channel closed")
				tradeCh = nil
				continue
			}
			if evt.Symbol != cfg.Symbol {
				continue
			}
			s.handleTrade(ctx, cfg, exec, evt, &session, &firstBar)
		}
	}
}

func (s *StrategyExecutionServer) handleBar(
	ctx context.Context, cfg LiveStrategyConfig, exec *execPair,
	bar *mthub.BarUpdate, bars *[]liveBar,
	session **LiveSession, firstBar *bool,
) {
	*bars = append(*bars, liveBar{
		open:     bar.Open.String(),
		high:     bar.High.String(),
		low:      bar.Low.String(),
		close:    bar.Close.String(),
		volume:   strconv.FormatFloat(bar.Volume, 'f', -1, 64),
		openTime: bar.OpenTime,
	})
	if len(*bars) > maxContextBars {
		*bars = (*bars)[len(*bars)-maxContextBars:]
	}

	var lctx *antv1.LiveStrategyContext
	if *firstBar {
		lctx = s.buildLiveContext(ctx, cfg, *bars)
	} else {
		lctx = s.buildDeltaContext(ctx, cfg, *bars)
	}

	req := &antv1.ExecuteLiveRequest{
		StrategyCode: cfg.Code,
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:   lctx,
	}
	reqBytes, _ := proto.Marshal(req)

	var respBytes []byte
	var err error
	if *firstBar {
		*session = NewLiveSession(exec.wasm, cfg.Code, s.log)
		respBytes, err = (*session).Start(ctx, reqBytes)
		*firstBar = false
	} else {
		if *session == nil {
			s.log.Error("LiveStrategyRunner: session lost before bar event")
			return
		}
		respBytes, err = (*session).SendBar(reqBytes)
	}
	if err != nil {
		s.log.Error("LiveStrategyRunner: bar request failed", zap.Error(err))
		*session = nil
		*firstBar = true
		return
	}
	s.dispatchFromResponse(ctx, cfg, bar, respBytes)
}

func (s *StrategyExecutionServer) handleTick(
	ctx context.Context, cfg LiveStrategyConfig, exec *execPair,
	tick *mthub.TickUpdate, session **LiveSession, firstBar *bool,
) {
	if *session == nil {
		return // session not yet started (waiting for first bar)
	}
	// Build tick context from quote update + account state.
	tctx := s.buildTickContext(ctx, cfg, tick)

	req := &antv1.ExecuteLiveRequest{
		StrategyCode: cfg.Code,
		RequestType:  antv1.RequestType_REQUEST_TYPE_TICK,
		TickContext:  tctx,
	}
	reqBytes, _ := proto.Marshal(req)
	respBytes, err := (*session).SendBar(reqBytes)
	if err != nil {
		s.log.Warn("LiveStrategyRunner: tick request failed", zap.Error(err))
		return
	}
	s.dispatchFromBytes(ctx, cfg, nil, respBytes)
}

func (s *StrategyExecutionServer) handleTrade(
	ctx context.Context, cfg LiveStrategyConfig, exec *execPair,
	evt *mthub.BrokerTradeEvent, session **LiveSession, firstBar *bool,
) {
	if *session == nil {
		return
	}
	tctx := s.buildTradeContext(ctx, cfg, evt)

	req := &antv1.ExecuteLiveRequest{
		StrategyCode:  cfg.Code,
		RequestType:   antv1.RequestType_REQUEST_TYPE_TRADE,
		TradeContext:  tctx,
	}
	reqBytes, _ := proto.Marshal(req)
	respBytes, err := (*session).SendBar(reqBytes)
	if err != nil {
		s.log.Warn("LiveStrategyRunner: trade request failed", zap.Error(err))
		return
	}
	s.dispatchFromBytes(ctx, cfg, nil, respBytes)
}

func (s *StrategyExecutionServer) buildTickContext(ctx context.Context, cfg LiveStrategyConfig, tick *mthub.TickUpdate) *antv1.TickContext {
	tctx := &antv1.TickContext{
		Bid:        tick.Bid.String(),
		Ask:        tick.Ask.String(),
		Symbol:     cfg.Symbol,
		Timeframe:  cfg.Timeframe,
		Mode:       cfg.Mode,
		Params:     buildLiveParams(cfg.Params),
		CurrentPrice: tick.Bid.String(),
	}
	s.backfillContextStrings(ctx, cfg.AccountID, &tctx.Equity, &tctx.Balance, &tctx.Positions)
	return tctx
}

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

// ── Context builders (shared with bar path) ────────────────────────

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

// dispatchFromResponse is the legacy name kept for compatibility.
func (s *StrategyExecutionServer) dispatchFromResponse(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, respBytes []byte) {
	s.dispatchFromBytes(ctx, cfg, bar, respBytes)
}
