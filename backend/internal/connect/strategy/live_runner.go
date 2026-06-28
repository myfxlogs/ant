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
	"anttrader/tools/mql2go"
	"anttrader/tools/mql2go/interp"
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
		if isMQLStrategy(cfg.Code) {
			ir, irErr := mql2go.CompileToIR(cfg.Code)
			if irErr != nil {
				s.log.Error("LiveStrategyRunner: compile MQL to IR failed", zap.Error(irErr))
				return
			}
			*session = NewInterpLiveSession(exec.wasm, interp.SerializeIR(ir), s.log)
		} else {
			*session = NewLiveSession(exec.wasm, cfg.Code, s.log)
		}
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

