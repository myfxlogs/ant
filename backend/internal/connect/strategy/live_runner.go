// live_runner.go — LiveStrategyRunner: subscribes to real-time data streams
// (bar, tick, trade), builds proto-native context per event, executes
// strategy via in-process Bytecode VM, and dispatches signals to OMS.
//
// Multi-model architecture:
//   BAR   → barBroker channel   → session.SendEvent() → strategy.OnBar
//   TICK  → tickBroker channel  → session.SendEvent() → strategy.OnTick
//   TRADE → tradeBroker channel → session.SendEvent() → strategy.OnTrade
//
// Follows push-first architecture: events drive the loop; no polling.

package strategy

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/mthub"
	"alphaforge/strategy/backtest"
	"alphaforge/tools/mql2go"
)

const maxContextBars = 500

// ExecutionModels bitmask for which callbacks the strategy supports.
type ExecutionModels int

const (
	ExecModelTick  ExecutionModels = 1 << iota // OnTick
	ExecModelTrade                             // OnTrade
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
	StrategyID          string // imported strategy ID for bytecode cache (optional)
	// Models specifies which execution callbacks the strategy implements.
	// Default (0) = Bar only.
	Models ExecutionModels

	// ExtraSymbols are secondary symbols whose bars are fetched and exposed
	// to the strategy via BarsForSymbol. Trading still targets Symbol.
	ExtraSymbols []string

	// RunID must be pre-set by caller (run record pre-created in DB).
	RunID uuid.UUID

	// PreRegisteredSession is set by callers that synchronously register
	// the session before launching RunLiveStrategy. If set, RunLiveStrategy
	// skips registration.
	PreRegisteredSession *ActiveSession

	// ShadowVerifier runs a background consistency check comparing live
	// signals with shadow backtest results. Nil = disabled.
	ShadowVerifier *ShadowVerifier
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
// builds proto-native context for each event, executes the strategy via Bytecode VM,
// and dispatches the resulting signals.
//
// Blocks until ctx is cancelled. Callers should run this in a goroutine.
func (s *StrategyExecutionServer) RunLiveStrategy(ctx context.Context, cfg LiveStrategyConfig) error {
	if cfg.RunID == uuid.Nil {
		return fmt.Errorf("live strategy runner: cfg.RunID must be set by caller")
	}
	runID := cfg.RunID

	cleanupOrphan := func(errMsg string) {
		if s.runRepo != nil {
			_ = s.runRepo.UpdateStopped(context.Background(), runID, "error", errMsg)
		}
	}

	if s.barSource == nil {
		cleanupOrphan("no BarSource configured")
		return fmt.Errorf("live strategy runner: no BarSource configured")
	}
	if s.gate == nil {
		cleanupOrphan("risk.Gate not injected")
		return fmt.Errorf("live strategy runner: risk.Gate not injected — live trading blocked per D6-A")
	}

	source, ok := s.barSource.(LiveBarSubscriber)
	if !ok {
		cleanupOrphan("BarSource does not support streaming")
		return fmt.Errorf("live strategy runner: BarSource does not support streaming (got %s)", s.barSource.Name())
	}

	barAccountID := cfg.AccountID
	if cfg.DataSourceAccountID != "" {
		barAccountID = cfg.DataSourceAccountID
	}

	needsTick, needsTrade := s.detectExecModels(ctx, cfg)

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

	tickCh, tickCancel := s.subscribeTickUpdates(cfg.AccountID, needsTick)
	if tickCancel != nil {
		defer tickCancel()
	}

	tradeCh, tradeCancel := s.subscribeTradeEvents(cfg.AccountID, needsTrade)
	if tradeCancel != nil {
		defer tradeCancel()
	}

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	if s.posCache != nil && s.mtHub != nil {
		s.posCache.Subscribe(runCtx, s.mtHub, cfg.AccountID)
	}

	activeSess := s.registerLiveSession(&cfg, runID, runCancel, cleanupOrphan)
	if activeSess == nil && cfg.PreRegisteredSession == nil && s.sessionRegistry != nil {
		return fmt.Errorf("live strategy runner: another strategy is already running for account %s", cfg.AccountID)
	}

	s.setupShadowVerifier(runCtx, &cfg)
	defer s.cleanupLiveSession(runID, cfg, activeSess)

	bars := make([]liveBar, 0, maxContextBars)
	var session Session
	var firstBar = true

	extraBars, extraSymbolSet := initExtraBars(cfg)

	defer func() {
		if session != nil {
			_ = session.Close()
		}
	}()

	s.runLiveEventLoop(liveEventLoopParams{
		runCtx:         runCtx,
		cfg:            cfg,
		barCh:          barCh,
		tickCh:         tickCh,
		tradeCh:        tradeCh,
		bars:           &bars,
		session:        &session,
		firstBar:       &firstBar,
		activeSess:     activeSess,
		extraBars:      extraBars,
		extraSymbolSet: extraSymbolSet,
	})
	return nil
}

type liveEventLoopParams struct {
	runCtx         context.Context
	cfg            LiveStrategyConfig
	barCh          <-chan *mthub.BarUpdate
	tickCh         <-chan *mthub.TickUpdate
	tradeCh        <-chan *mthub.BrokerTradeEvent
	bars           *[]liveBar
	session        *Session
	firstBar       *bool
	activeSess     *ActiveSession
	extraBars      map[string][]liveBar
	extraSymbolSet map[string]bool
}

func (s *StrategyExecutionServer) runLiveEventLoop(p liveEventLoopParams) {
	for {
		select {
		case <-p.runCtx.Done():
			s.log.Info("LiveStrategyRunner: context cancelled, exiting")
			return

		case bar, ok := <-p.barCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: bar channel closed, exiting")
				return
			}
			if p.extraSymbolSet[bar.Symbol] && bar.Period == p.cfg.Timeframe {
				handleExtraSymbolBar(bar, p.extraBars)
				continue
			}
			if bar.Symbol != p.cfg.Symbol || bar.Period != p.cfg.Timeframe {
				continue
			}
			s.handleBar(p.runCtx, p.cfg, bar, p.bars, p.session, p.firstBar, p.activeSess, p.extraBars)

		case tick, ok := <-p.tickCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: tick channel closed")
				p.tickCh = nil
				continue
			}
			if tick.Symbol != p.cfg.Symbol {
				continue
			}
			s.handleTick(p.runCtx, p.cfg, tick, p.session, p.firstBar, p.activeSess)

		case evt, ok := <-p.tradeCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: trade channel closed")
				p.tradeCh = nil
				continue
			}
			if evt.Symbol != p.cfg.Symbol {
				continue
			}
			s.handleTrade(p.runCtx, p.cfg, evt, p.session, p.firstBar, p.activeSess)
		}
	}
}

func (s *StrategyExecutionServer) detectExecModels(ctx context.Context, cfg LiveStrategyConfig) (needsTick, needsTrade bool) {
	needsTick = cfg.Models&ExecModelTick != 0
	needsTrade = cfg.Models&ExecModelTrade != 0
	if cfg.Models == 0 && cfg.Code != "" {
		var cachedBytecode []byte
		if cfg.StrategyID != "" && s.importedRepo != nil {
			if sid, parseErr := uuid.Parse(cfg.StrategyID); parseErr == nil {
				cachedBytecode, _ = s.importedRepo.GetBytecode(ctx, sid)
			}
		}
		probe, _, probeErr := mql2go.CompileMQLCached(cfg.Code, cachedBytecode)
		if probeErr == nil {
			needsTick = probe.HasOnTick()
			needsTrade = probe.Bytecode().OnTrade >= 0
		}
	}
	return
}

func (s *StrategyExecutionServer) subscribeTickUpdates(accountID string, needsTick bool) (<-chan *mthub.TickUpdate, func()) {
	if !needsTick || s.mtHub == nil {
		return nil, nil
	}
	ts, ok := interface{}(s.mtHub).(LiveTickSubscriber)
	if !ok {
		return nil, nil
	}
	return ts.SubscribeTickUpdates(accountID)
}

func (s *StrategyExecutionServer) subscribeTradeEvents(accountID string, needsTrade bool) (<-chan *mthub.BrokerTradeEvent, func()) {
	if !needsTrade || s.mtHub == nil {
		return nil, nil
	}
	ts, ok := interface{}(s.mtHub).(LiveTradeSubscriber)
	if !ok {
		return nil, nil
	}
	return ts.SubscribeTradeEvents(accountID)
}

func initExtraBars(cfg LiveStrategyConfig) (map[string][]liveBar, map[string]bool) {
	extraBars := make(map[string][]liveBar, len(cfg.ExtraSymbols))
	extraSymbolSet := make(map[string]bool, len(cfg.ExtraSymbols))
	for _, sym := range cfg.ExtraSymbols {
		if sym != "" && sym != cfg.Symbol {
			extraSymbolSet[sym] = true
			extraBars[sym] = make([]liveBar, 0, maxContextBars)
		}
	}
	return extraBars, extraSymbolSet
}

func handleExtraSymbolBar(bar *mthub.BarUpdate, extraBars map[string][]liveBar) {
	ew := extraBars[bar.Symbol]
	ew = append(ew, liveBar{
		open:     bar.Open.String(),
		high:     bar.High.String(),
		low:      bar.Low.String(),
		close:    bar.Close.String(),
		volume:   strconv.FormatFloat(bar.Volume, 'f', -1, 64),
		openTime: bar.OpenTime,
	})
	if len(ew) > maxContextBars {
		ew = ew[len(ew)-maxContextBars:]
	}
	extraBars[bar.Symbol] = ew
}

func (s *StrategyExecutionServer) registerLiveSession(cfg *LiveStrategyConfig, runID uuid.UUID, runCancel func(), cleanupOrphan func(string)) *ActiveSession {
	activeSess := cfg.PreRegisteredSession
	if activeSess == nil && s.sessionRegistry != nil {
		uid, _ := uuid.Parse(cfg.UserID)
		activeSess = s.sessionRegistry.Register(runID, uid, cfg.AccountID, cfg.Symbol, cfg.Timeframe, cfg.Mode, runCancel)
		if activeSess == nil {
			cleanupOrphan("another strategy is already running for this account")
			return nil
		}
	}
	if activeSess != nil {
		s.log.Info("LiveStrategyRunner: session registered", zap.String("run_id", runID.String()))
	}
	return activeSess
}

func (s *StrategyExecutionServer) setupShadowVerifier(runCtx context.Context, cfg *LiveStrategyConfig) {
	if cfg.ShadowVerifier == nil && cfg.Code != "" {
		btCfg := backtest.Config{
			Symbol:         cfg.Symbol,
			Timeframe:      cfg.Timeframe,
			Params:         cfg.Params,
			InitialCapital: decimal.NewFromInt(10000),
		}
		cfg.ShadowVerifier = NewShadowVerifier(cfg.Code, btCfg, s.log)
	}
	if cfg.ShadowVerifier != nil {
		cfg.ShadowVerifier.Start(runCtx)
	}
}

func (s *StrategyExecutionServer) cleanupLiveSession(runID uuid.UUID, cfg LiveStrategyConfig, activeSess *ActiveSession) {
	if cfg.ShadowVerifier != nil {
		cfg.ShadowVerifier.Stop()
	}
	if s.sessionRegistry != nil && runID != uuid.Nil {
		s.sessionRegistry.Deregister(runID)
	}
	if s.runRepo != nil && runID != uuid.Nil {
		status := "stopped"
		if activeSess != nil && activeSess.ErrorCount > 0 {
			status = "error"
		}
		errMsg := ""
		if activeSess != nil {
			errMsg = activeSess.LastError
		}
		if err := s.runRepo.UpdateStopped(context.Background(), runID, status, errMsg); err != nil {
			s.log.Warn("LiveStrategyRunner: failed to update run record on stop", zap.Error(err))
		}
	}
}
