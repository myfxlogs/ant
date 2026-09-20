package strategy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

// dispatchLiveSignal routes the strategy signal to the appropriate destination.
// LIVE mode: signal → OMS → broker (via mthub)
// PAPER mode: signal → log (paper portfolio coming later)
//
// T3.1 (hard injury ②): expanded action set from buy/sell only to full broker semantics:
//
//	Market:    buy, sell         → PlaceOrder(market)
//	Pending:   buy_limit, sell_limit, buy_stop, sell_stop,
//	           buy_stop_limit, sell_stop_limit → PlaceOrder(limit/stop/stop_limit)
//	Close:     close, close_all  → CloseOrder
//	Modify:    modify            → ModifyOrder
//	Cancel:    cancel            → CancelPending
//
// LIVE-ORDER-REENTRY-1: all 5 mutation types share the same execution protocol
// via coordinateMutation — synchronous barrier with confirmation, no fire-and-forget.
func (s *StrategyExecutionServer) dispatchLiveSignal(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, sig *antv1.StrategySignal, activeSess *ActiveSession) {
	// Fail-closed panic containment: this runs on the VM event-loop
	// goroutine — an uncaught panic crashes the whole process. Panics
	// inside coordinateMutation are already converged by its own recover;
	// anything reaching here happened outside the coordinator (persist,
	// logging, sub-dispatcher frames). The barrier is only converged to
	// outcomeUnknown when it is already in-flight — an idle barrier must
	// never be locked, or all subsequent mutations would be rejected.
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("dispatchLiveSignal: panic recovered",
				zap.Any("panic", r),
				zap.String("account", cfg.AccountID),
				zap.String("signal_type", sig.GetSignalType()))
			if activeSess != nil {
				activeSess.RecordError(fmt.Sprintf("signal %s: panic recovered: %v", sig.GetSignalType(), r))
				activeSess.SetCircuitOpen(true)
				if b := activeSess.barrier; b != nil {
					if st := b.State(); st == barrierSubmitting || st == barrierAcceptedUnconfirmed {
						b.NotifyOutcomeUnknown()
					}
				}
			}
		}
	}()
	action := sig.GetSignalType()
	s.recordSignalIntent(ctx, cfg, sig, action)

	if cfg.Mode == "paper" {
		s.dispatchPaperSignal(ctx, cfg, bar, sig)
		return
	}

	if s.mtHub == nil {
		s.log.Warn("LiveStrategyRunner: no MtHubService configured, cannot dispatch live order")
		return
	}

	// T3.1: dispatch based on expanded action set.
	s.dispatchByAction(ctx, cfg, bar, sig, action, activeSess)
}

// recordSignalIntent performs the pre-dispatch audit preamble shared by the
// async path (dispatchLiveSignal) and the synchronous path
// (dispatchSignalSync): schedule-run log + info log + DB persist.
// VM-LIVE-SYNC-DISPATCH-1 (R1).
func (s *StrategyExecutionServer) recordSignalIntent(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal, action string) {
	uid, _ := uuid.Parse(cfg.UserID)
	if s.sessionRegistry != nil {
		s.sessionRegistry.InsertScheduleRunLog(ctx, uid, cfg.ScheduleID,
			"signal", action, "received", "", action, parseDecimal(sig.GetVolume()))
	}
	s.log.Info("LiveStrategyRunner: signal",
		zap.String("account", cfg.AccountID),
		zap.String("symbol", cfg.Symbol),
		zap.String("type", action),
		zap.String("volume", sig.GetVolume()),
		zap.String("sl", sig.GetStopLoss()),
		zap.String("tp", sig.GetTakeProfit()),
	)

	// Persist signal to DB before dispatching.
	s.persistSignal(ctx, cfg, sig)
}

// dispatchByAction routes a signal to the matching broker-mutation
// dispatcher and returns the coordinated outcome. Opens are suppressed
// when the circuit breaker is open. Shared by the async dispatch path and
// dispatchSignalSync (VM-LIVE-SYNC-DISPATCH-1).
func (s *StrategyExecutionServer) dispatchByAction(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, sig *antv1.StrategySignal, action string, activeSess *ActiveSession) mutationResult {
	switch action {
	case sideBuy, sideSell:
		if activeSess != nil && activeSess.IsCircuitOpen() {
			s.log.Warn("LiveStrategyRunner: suppressing order — circuit breaker open",
				zap.String("account", cfg.AccountID),
				zap.String("symbol", cfg.Symbol),
				zap.String("action", action),
			)
			return mutationResult{state: barrierIdle}
		}
		return s.dispatchMarketOrder(ctx, cfg, barOpenTimeForSignal(bar, cfg), sig, activeSess)
	case "buy_limit", "sell_limit", "buy_stop", "sell_stop",
		"buy_stop_limit", "sell_stop_limit":
		if activeSess != nil && activeSess.IsCircuitOpen() {
			s.log.Warn("LiveStrategyRunner: suppressing pending order — circuit breaker open",
				zap.String("account", cfg.AccountID),
				zap.String("symbol", cfg.Symbol),
				zap.String("action", action),
			)
			return mutationResult{state: barrierIdle}
		}
		return s.dispatchPendingOrder(ctx, cfg, barOpenTimeForSignal(bar, cfg), sig, activeSess)
	case string(actionClose):
		return s.dispatchCloseOrder(ctx, cfg, sig, activeSess)
	case "close_all":
		return s.dispatchCloseAll(ctx, cfg, activeSess)
	case "modify":
		return s.dispatchModifyOrder(ctx, cfg, sig, activeSess)
	case "cancel":
		return s.dispatchCancelOrder(ctx, cfg, sig, activeSess)
	case "cancel_all":
		return s.dispatchCancelAll(ctx, cfg, activeSess)
	default:
		// hold, unknown — no-op.
		return mutationResult{state: barrierIdle}
	}
}

// ── T3.1 action dispatchers ──────────────────────────────────────────

func (s *StrategyExecutionServer) dispatchPaperSignal(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, sig *antv1.StrategySignal) {
	if s.paperEngine == nil {
		s.log.Warn("LiveStrategyRunner: no PaperEngine, dropping paper signal")
		return
	}

	action := sig.GetSignalType()

	switch action {
	case "close", "close_all":
		if err := s.paperEngine.ClosePaperOrder(ctx, cfg.AccountID, cfg.Symbol); err != nil {
			s.log.Error("LiveStrategyRunner: paper close failed",
				zap.String("run", cfg.RunID.String()),
				zap.String("symbol", cfg.Symbol), zap.String("action", action),
				zap.Error(err))
			return
		}
		if s.sessionRegistry != nil {
			if sess, ok := s.sessionRegistry.Get(cfg.RunID); ok {
				sess.SetPnL("0")
			}
		}
		return
	case "modify":
		sl := parseDecimal(sig.GetStopLoss())
		tp := parseDecimal(sig.GetTakeProfit())
		if err := s.paperEngine.ModifyPaperOrder(ctx, cfg.AccountID, cfg.Symbol, sl, tp); err != nil {
			s.log.Error("LiveStrategyRunner: paper modify failed",
				zap.String("run", cfg.RunID.String()),
				zap.String("symbol", cfg.Symbol), zap.String("action", action),
				zap.Error(err))
		}
		return
	case "cancel":
		if err := s.paperEngine.CancelPaperOrder(ctx, cfg.AccountID, cfg.Symbol); err != nil {
			s.log.Error("LiveStrategyRunner: paper cancel failed",
				zap.String("run", cfg.RunID.String()),
				zap.String("symbol", cfg.Symbol), zap.String("action", action),
				zap.Error(err))
		}
		return
	}

	var bid, ask decimal.Decimal
	if bar != nil {
		bid = bar.Bid
		ask = bar.Ask
	}
	if err := s.paperEngine.PlacePaperOrder(ctx, cfg.AccountID, cfg.Symbol,
		action, parseDecimal(sig.GetVolume()), bid, ask); err != nil {
		s.log.Error("LiveStrategyRunner: paper order failed",
			zap.String("run", cfg.RunID.String()),
			zap.String("symbol", cfg.Symbol), zap.String("action", action),
			zap.String("volume", sig.GetVolume()), zap.String("price", sig.GetPrice()),
			zap.Error(err))
		return
	}
	// Update running PnL for the paper session after each fill.
	if s.sessionRegistry != nil {
		if sess, ok := s.sessionRegistry.Get(cfg.RunID); ok {
			pnl, _ := s.paperEngine.PaperPnl(ctx, cfg.AccountID, cfg.Symbol, bid, ask)
			sess.SetPnL(pnl.String())
		}
	}
}

// submitOrder is the common order submission helper (T3.1 / D6-A / LIVE-ORDER-REENTRY-1).
// LIVE-ORDER-REENTRY-1: submission is synchronous via coordinateMutation,
// restoring MT4 EA single-threaded OrderSend semantics. The event loop blocks
// until the broker mutation reaches a deterministic outcome.
