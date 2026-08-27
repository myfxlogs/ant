// mutation_recovery.go — Reconciliation-based recovery from barrierOutcomeUnknown
// state (LIVE-ORDER-REENTRY-1 ④-②).
//
// When a broker mutation enters outcomeUnknown (transport timeout, push lost,
// read-after-write ambiguous), the barrier stays locked to prevent duplicate
// orders. For mutations with a known ticket (close/modify/cancel), a background
// goroutine attempts reconciliation after a configurable delay:
//
//   1. Wait recoveryDelay (default 10s) for broker state to settle
//   2. Single OpenedOrders query (I6: not polling)
//   3. verify(orders) determines if the mutation took effect
//   4. barrier.Reconcile(verified) transitions to confirmed/deterministicRejected
//   5. Release barrier + clear circuit breaker → session resumes
//
// Open mutations (ticket=0) do NOT get recovery — the ticket is unknown so
// reconciliation is impossible. They stay fail-closed (manual intervention).

package strategy

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"alphaforge/internal/mthub"
)

// recoverFromOutcomeUnknown attempts reconciliation-based recovery from
// barrierOutcomeUnknown state (④-②). After a configurable delay, it performs
// a single OpenedOrders query and uses the verify function to determine
// whether the mutation took effect:
//   - verified=true → barrier.Reconcile(true) → Release + clear circuit breaker
//   - verified=false → barrier.Reconcile(false) → Release + clear circuit breaker
//   - query fails → barrier stays locked (fail-closed, manual intervention needed)
//
// The goroutine is started by coordinateMutation when outcomeUnknown is
// entered for a mutation with a known ticket. It runs independently of the
// original mutation call (which has already returned).
func (s *StrategyExecutionServer) recoverFromOutcomeUnknown(
	ctx context.Context,
	cfg LiveStrategyConfig, activeSess *ActiveSession,
	barrier *TradeBarrier, ticket int64, action mutationAction,
	verify func(orders []*mthub.OrderRecord) bool,
	conf confirmationConfig,
) {
	// VM-AUDIT-2026-08-27-7: use select+ctx.Done() so session cancellation
	// (runCtx cancel via SessionRegistry.Stop) interrupts the recovery delay
	// instead of blocking for the full recoveryDelay (default 10s).
	select {
	case <-time.After(conf.recoveryDelay):
	case <-ctx.Done():
		return
	}

	// Check if barrier is still in outcomeUnknown (may have been released
	// by session shutdown or manual intervention).
	if barrier.State() != barrierOutcomeUnknown {
		return
	}

	s.log.Info("LiveStrategyRunner: attempting reconciliation recovery from outcomeUnknown",
		zap.String("account", cfg.AccountID),
		zap.String("action", string(action)),
		zap.Int64("ticket", ticket),
	)

	rawCtx, rawCancel := context.WithTimeout(context.Background(), conf.readAfterWriteTimeout)
	orders, err := s.mtHub.OpenedOrders(rawCtx, cfg.AccountID)
	rawCancel()
	if err != nil {
		s.log.Error("LiveStrategyRunner: recovery read-after-write failed — barrier stays locked",
			zap.String("account", cfg.AccountID),
			zap.Int64("ticket", ticket),
			zap.Error(err),
		)
		if activeSess != nil {
			activeSess.RecordError(fmt.Sprintf("recovery failed for ticket=%d: %s (barrier stays locked)", ticket, err.Error()))
		}
		return
	}

	verified := verify(orders)
	barrier.Reconcile(verified)
	state := barrier.State()

	switch state {
	case barrierConfirmed:
		s.log.Info("LiveStrategyRunner: recovery succeeded — mutation confirmed via reconciliation",
			zap.String("account", cfg.AccountID),
			zap.Int64("ticket", ticket),
		)
		s.logOrderLifecycle(activeSess, cfg, "order_confirmed", "", ticket, "")
		barrier.Release()
		if activeSess != nil {
			activeSess.SetCircuitOpen(false)
		}
	case barrierDeterministicRejected:
		s.log.Info("LiveStrategyRunner: recovery succeeded — mutation confirmed NOT applied via reconciliation",
			zap.String("account", cfg.AccountID),
			zap.Int64("ticket", ticket),
		)
		s.logOrderLifecycle(activeSess, cfg, "order_rejected", "", ticket, "")
		barrier.Release()
		if activeSess != nil {
			activeSess.SetCircuitOpen(false)
		}
	default:
		// Reconcile was a no-op (barrier was no longer in outcomeUnknown).
		// Nothing to do — another path already handled it.
	}
}
