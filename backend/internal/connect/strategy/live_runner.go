// live_runner.go — LiveStrategyRunner: subscribes to real-time data streams
// (bar, tick, trade), builds proto-native context per event, executes
// strategy via in-process Bytecode VM, and dispatches signals to OMS.
//
// Multi-model architecture:
//   BAR   → barBroker channel   → session.SendBar() → strategy.OnBar
//   TICK  → tickBroker channel  → session.SendBar() → strategy.OnTick
//   TRADE → tradeBroker channel → session.SendBar() → strategy.OnTrade
//
// Follows push-first architecture: events drive the loop; no polling.

package strategy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/mthub"
)

const maxContextBars = 500

// ExecutionModels bitmask for which callbacks the strategy supports.
type ExecutionModels int

const (
	ExecModelTick  ExecutionModels = 1 << iota // OnTick
	ExecModelTrade                              // OnTrade
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
	// Callers must pre-create the run record and set cfg.RunID.
	if cfg.RunID == uuid.Nil {
		return fmt.Errorf("live strategy runner: cfg.RunID must be set by caller")
	}
	runID := cfg.RunID

	// cleanupOrphan marks the run record as failed on early return.
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

	// Make context cancellable so StopStrategy RPC can stop this run.
	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	// Push-first: subscribe to PositionSnapshotBroker for this account.
	// backfillContextStrings reads from posCache (O(1) read, no per-bar polling).
	if s.posCache != nil && s.mtHub != nil {
		s.posCache.Subscribe(runCtx, s.mtHub, cfg.AccountID)
	}

	// Register in session registry for monitoring + control.
	// If caller pre-registered (synchronous conflict check), use that session.
	activeSess := cfg.PreRegisteredSession
	if activeSess == nil && s.sessionRegistry != nil {
		uid, _ := uuid.Parse(cfg.UserID)
		activeSess = s.sessionRegistry.Register(runID, uid, cfg.AccountID, cfg.Symbol, cfg.Timeframe, cfg.Mode, runCancel)
		if activeSess == nil {
			cleanupOrphan("another strategy is already running for this account")
			return fmt.Errorf("live strategy runner: another strategy is already running for account %s", cfg.AccountID)
		}
	}
	if activeSess != nil {
		s.log.Info("LiveStrategyRunner: session registered", zap.String("run_id", runID.String()))
	}

	// Ensure run record is closed + session deregistered on exit.
	// Stop shadow verifier if active.
	if cfg.ShadowVerifier != nil {
		cfg.ShadowVerifier.Stop()
	}

	defer func() {
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
	}()

	bars := make([]liveBar, 0, maxContextBars)
	var session Session
	var firstBar bool = true

	defer func() {
		if session != nil {
			session.Close()
		}
	}()

	for {
		select {
		case <-runCtx.Done():
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
			s.handleBar(runCtx, cfg, bar, &bars, &session, &firstBar, activeSess)

		case tick, ok := <-tickCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: tick channel closed")
				tickCh = nil
				continue
			}
			if tick.Symbol != cfg.Symbol {
				continue
			}
			s.handleTick(runCtx, cfg, tick, &session, &firstBar, activeSess)

		case evt, ok := <-tradeCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: trade channel closed")
				tradeCh = nil
				continue
			}
			if evt.Symbol != cfg.Symbol {
				continue
			}
			s.handleTrade(runCtx, cfg, evt, &session, &firstBar, activeSess)
		}
	}
}

