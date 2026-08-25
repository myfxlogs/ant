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
	"sync/atomic"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/mthub"
)

const maxContextBars = 500

const modeLive = "live"

const modePaper = "paper"

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

	// ScheduleID identifies the strategy schedule this run belongs to.
	// Used for Magic Number attribution when multiple strategies share an account.
	ScheduleID uuid.UUID

	// RunID must be pre-set by caller (run record pre-created in DB).
	RunID uuid.UUID

	// PreRegisteredSession is set by callers that synchronously register
	// the session before launching RunLiveStrategy. If set, RunLiveStrategy
	// skips registration.
	PreRegisteredSession *ActiveSession

	// ShadowVerifier runs a background consistency check comparing live
	// signals with shadow backtest results. Nil = disabled.
	ShadowVerifier *ShadowVerifier

	// TickSeq is an atomic counter for tick/trade signals, ensuring unique
	// ClientIDs when bar is nil (OnTick/OnTrade paths). Bar signals use
	// bar.OpenTime for determinism; tick signals use this counter.
	TickSeq *atomic.Int64

	// EntitlementCheck, if non-nil, is called on every finalized bar to verify
	// the user still holds an active entitlement (subscription/trial/ownership).
	// If it returns false, the session self-terminates (no new signals).
	// Nil = no check (user's own strategy). Push-first: rides on bar events, not a timer.
	EntitlementCheck func(ctx context.Context) bool

	// SymbolParam is pre-fetched at run startup from broker (CachedSymbolParam).
	// Stored here so bar/tick builders do O(1) string fill — no per-event RPC.
	// W2: moved out of tick/bar hot path to avoid per-event broker RPC on cache miss.
	SymbolParam *mthub.SymbolParam
}

// LiveTickSubscriber provides tick (Bid/Ask) updates for an account.
type LiveTickSubscriber interface {
	SubscribeTickUpdates(accountID string) (<-chan *mthub.TickUpdate, func())
}

// LiveTradeSubscriber provides trade events for an account.
type LiveTradeSubscriber interface {
	SubscribeTradeEvents(accountID string) (<-chan *mthub.BrokerTradeEvent, func())
}

// preflightLiveChecks runs LEAKAGE-1 bound account check and T5 coverage gate.
// All live-launch paths converge on RunLiveStrategy — these checks are non-bypassable.
func (s *StrategyExecutionServer) preflightLiveChecks(ctx context.Context, cfg LiveStrategyConfig, cleanupOrphan func(string)) error {
	if cfg.Mode == modeLive && cfg.AccountID != "" {
		if accountUUID, parseErr := uuid.Parse(cfg.AccountID); parseErr == nil && accountUUID != uuid.Nil {
			uid, _ := uuid.Parse(cfg.UserID)
			if err := s.checkBoundAccount(ctx, uid, accountUUID); err != nil {
				cleanupOrphan(fmt.Sprintf("bound account check failed: %v", err))
				return fmt.Errorf("live strategy runner: bound account check: %w", err)
			}
		}
	}
	if cfg.Mode == modeLive {
		if s.coverageChecker != nil {
			if err := s.coverageChecker.CheckLiveCoverage(ctx, cfg.StrategyID, cfg.Code); err != nil {
				cleanupOrphan(fmt.Sprintf("fatal coverage check failed: %v", err))
				return fmt.Errorf("live strategy runner: fatal coverage gate: %w", err)
			}
		} else {
			s.log.Warn("coverage checker not injected, live coverage gate skipped",
				zap.String("strategy_id", cfg.StrategyID),
				zap.String("account_id", cfg.AccountID))
		}
	}
	return nil
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

	if err := s.preflightLiveChecks(ctx, cfg, cleanupOrphan); err != nil {
		return err
	}

	if cfg.Symbol == "" {
		cleanupOrphan("no symbol specified")
		return fmt.Errorf("live strategy runner: symbol is required — specify which instrument to trade")
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

	// Demand-driven: ensure gateway subscribes to this strategy's symbol.
	// Retries until gateway session is available (handles startup race).
	subscribeSymbolsWithRetry(ctx, s.mtHub, barAccountID, cfg.Symbol, s.log)

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

	s.seedBarWindows(ctx, cfg, &bars, extraBars)

	s.prefetchSymbolParam(ctx, &cfg)

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

// shouldRunOnBar reports whether a finalized bar for the strategy's primary
// symbol/timeframe should trigger OnBar. Open (in-progress) bars are excluded
// (LIVE-1): they are chart-feed snapshots, not strategy events — feeding them
// re-triggers OnBar mid-formation, corrupts the bar window, and diverges live
// from closed-bar backtest. Intra-bar updates belong on the tick channel.
func shouldRunOnBar(bar *mthub.BarUpdate, symbol, timeframe string) bool {
	return bar.Closed && bar.Symbol == symbol && bar.Period == timeframe
}
