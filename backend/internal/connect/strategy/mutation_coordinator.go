// mutation_coordinator.go — Shared orchestration for all broker mutation
// types (open, close, modify, cancel, close_all) (LIVE-ORDER-REENTRY-1 B1/B2/B5).
//
// All five mutation types share the same execution protocol:
//
//   1. Acquire barrier (CAS idle→submitting)
//   2. Pre-listen OnOrderUpdate (listener covers ENTIRE mutation cycle)
//   3. Call broker RPC with injectable deadline (B5: no WithoutCancel without timeout)
//   4. Classify error via mthub.ClassifyMutationError (B4: typed, no string guessing)
//      - pre-broker rejection → deterministic rejected → release
//      - broker-phase error → outcome unknown → locked (fail-closed)
//      - confirmed push + RPC error → confirmed (B4: don't lock on confirmed)
//   5. NotifyBrokerAccepted (for open: ticket from RPC; for close/modify/cancel: known ticket)
//   6. Wait for push confirmation (bounded pushWait)
//   7. If no push → single read-after-write OpenedOrders (I6: not polling)
//   8. Classify confirmation → confirmed/rejected/unknown
//   9. Unsubscribe listener (defer — covers entire cycle, B1)

package strategy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/mthub"
)

// mutationAction identifies the type of broker mutation for read-after-write
// verification logic.
type mutationAction string

const (
	actionOpen   mutationAction = "open"
	actionClose  mutationAction = "close"
	actionModify mutationAction = "modify"
	actionCancel mutationAction = "cancel"
)

// mutationSpec describes a single broker mutation for the coordinator.
// Business dispatchers construct this; the coordinator runs the shared protocol.
type mutationSpec struct {
	action        mutationAction
	clientID      string
	expectedMagic int32
	// expectedTicket is the ticket to confirm against. For open: 0 (filled
	// after RPC returns). For close/modify/cancel: the known ticket.
	expectedTicket int64
	// brokerCall performs the actual broker RPC. Returns (ticket, error).
	// For open: ticket is the broker-assigned ticket. For others: ticket is
	// the same expectedTicket (passed through for uniformity).
	brokerCall func(ctx context.Context) (int64, error)
	// verifyReadAfterWrite checks whether the OpenedOrders result confirms
	// the mutation. For open/modify: ticket present = confirmed. For close/
	// cancel: ticket absent = confirmed.
	verifyReadAfterWrite func(orders []*mthub.OrderRecord) bool
}

// mutationResult is the outcome of a coordinated mutation.
type mutationResult struct {
	state  tradeBarrierState
	ticket int64
}

// coordinateMutation runs the shared mutation protocol for all 5 types.
// The caller (event loop) blocks until a deterministic outcome is reached.
// The barrier is released only for confirmed and deterministicRejected.
// For outcomeUnknown, the barrier stays locked (fail-closed).
func (s *StrategyExecutionServer) coordinateMutation(
	ctx context.Context, cfg LiveStrategyConfig, activeSess *ActiveSession,
	spec mutationSpec, sideStr string, sig *antv1.StrategySignal,
	conf confirmationConfig,
) mutationResult {
	if activeSess == nil || activeSess.barrier == nil {
		s.log.Error("coordinateMutation: barrier not configured — dropping (fail-closed)",
			zap.String("account", cfg.AccountID))
		return mutationResult{state: barrierIdle}
	}

	barrier := activeSess.barrier
	if !barrier.Acquire(spec.clientID, spec.expectedMagic, string(spec.action)) {
		s.log.Warn("LiveStrategyRunner: mutation dropped — barrier busy",
			zap.String("account", cfg.AccountID),
			zap.String("action", string(spec.action)),
			zap.String("barrier_state", barrier.State().String()),
		)
		activeSess.RecordError(fmt.Sprintf("%s dropped: previous broker mutation still in flight", spec.action))
		return mutationResult{state: barrierIdle}
	}

	s.logOrderLifecycle(activeSess, cfg, "order_submitting", sideStr, 0, "")
	if s.sessionRegistry != nil {
		uid, _ := uuid.Parse(cfg.UserID)
		s.sessionRegistry.InsertScheduleRunLog(ctx, uid, cfg.ScheduleID,
			"order", sideStr, "submitting", "", sig.GetSignalType(), parseDecimal(sig.GetVolume()))
	}

	// B1: Pre-listen for confirmation BEFORE the broker call. The listener
	// covers the ENTIRE mutation cycle — it is only unsubscribed after the
	// final outcome is determined (defer).
	confirmCh, confirmUnsub := s.mtHub.SubscribePositionSnapshots(ctx, cfg.AccountID)
	confirmDone := make(chan struct{})
	go func() {
		defer close(confirmDone)
		for snap := range confirmCh {
			if snap.UpdateTicket != 0 {
				barrier.NotifyConfirmationEvent(snap.UpdateTicket, snap.UpdateMagic, snap.UpdateType)
			}
		}
	}()
	defer func() {
		confirmUnsub()
		<-confirmDone
	}()

	// B5: broker RPC with injectable deadline. No WithoutCancel without timeout.
	brokerCtx := context.WithoutCancel(ctx)
	if cfg.UserID != "" {
		brokerCtx = context.WithValue(brokerCtx, interceptor.UserIDKey, cfg.UserID)
	}
	var brokerCancel context.CancelFunc
	brokerCtx, brokerCancel = context.WithTimeout(brokerCtx, conf.mutationRPCTimeout)
	defer brokerCancel()

	ticket, err := spec.brokerCall(brokerCtx)

	if err != nil {
		outcome := mthub.ClassifyMutationError(err)
		switch outcome {
		case "deterministic_rejected":
			barrier.NotifyDeterministicRejected()
			s.log.Warn("LiveStrategyRunner: mutation rejected (pre-broker)",
				zap.String("account", cfg.AccountID),
				zap.String("action", string(spec.action)),
				zap.String("side", sideStr),
				zap.Error(err),
			)
			s.logOrderLifecycle(activeSess, cfg, "order_rejected", sideStr, 0, err.Error())
			if activeSess != nil {
				activeSess.RecordError(fmt.Sprintf("%s %s: %s", spec.action, cfg.Symbol, err.Error()))
			}
			barrier.Release()
			return mutationResult{state: barrierDeterministicRejected}
		case "outcome_unknown":
			// R1: For close/modify/cancel where we know the ticket, a push
			// may have already arrived and been cached, OR may arrive in
			// the listener goroutine's queue but not yet processed.
			// Call NotifyBrokerAccepted with the known ticket so cached
			// events can match, then wait with a bounded timeout for the
			// barrier to reach a terminal state. This eliminates the
			// scheduling race between listener goroutine processing and
			// our state check — no time.Sleep needed.
			if spec.expectedTicket != 0 {
				barrier.NotifyBrokerAccepted(spec.expectedTicket)
				// Wait for push confirmation with a bounded timeout. If
				// the push was already cached, NotifyBrokerAccepted
				// transitioned to confirmed and WaitConfirmed returns
				// instantly. If the push is in the listener's queue,
				// WaitConfirmed blocks until the listener processes it.
				convCtx, convCancel := context.WithTimeout(ctx, conf.pushWait)
				convState := barrier.WaitConfirmed(convCtx)
				convCancel()
				if convState == barrierConfirmed {
					s.log.Info("LiveStrategyRunner: RPC error but push confirmed — converging to confirmed",
						zap.String("account", cfg.AccountID),
						zap.String("action", string(spec.action)),
						zap.Error(err),
					)
					s.logOrderLifecycle(activeSess, cfg, "order_confirmed", sideStr, spec.expectedTicket, "")
					barrier.Release()
					return mutationResult{state: barrierConfirmed, ticket: spec.expectedTicket}
				}
			}
			barrier.NotifyOutcomeUnknown()
			if activeSess != nil {
				activeSess.SetCircuitOpen(true)
				activeSess.RecordError(fmt.Sprintf("%s %s: outcome unknown, barrier locked: %s", spec.action, cfg.Symbol, err.Error()))
			}
			s.log.Error("LiveStrategyRunner: mutation outcome unknown — barrier locked",
				zap.String("account", cfg.AccountID),
				zap.String("action", string(spec.action)),
				zap.String("side", sideStr),
				zap.String("client_id", spec.clientID),
				zap.Error(err),
			)
			s.logOrderLifecycle(activeSess, cfg, "order_outcome_unknown", sideStr, spec.expectedTicket, err.Error())
			// ④-②: For known-ticket mutations (close/modify/cancel), start a
			// background reconciliation goroutine that may recover the barrier
			// after a delay. Open mutations (ticket=0) stay fail-closed.
			if spec.expectedTicket != 0 {
				verify := spec.verifyReadAfterWrite
				if verify == nil {
					verify = verifyTicketPresent(spec.expectedTicket)
				}
				go s.recoverFromOutcomeUnknown(cfg, activeSess, barrier, spec.expectedTicket, spec.expectedMagic, spec.action, verify, conf)
			}
			return mutationResult{state: barrierOutcomeUnknown}
		}
	}

	// Broker accepted — clear circuit open.
	if activeSess != nil {
		activeSess.SetCircuitOpen(false)
	}

	// For open: ticket comes from RPC. For close/modify/cancel: ticket is known.
	effectiveTicket := ticket
	if effectiveTicket == 0 {
		effectiveTicket = spec.expectedTicket
	}

	s.log.Info("LiveStrategyRunner: mutation submitted",
		zap.Int64("ticket", effectiveTicket),
		zap.String("action", string(spec.action)),
		zap.String("symbol", cfg.Symbol),
		zap.String("side", sideStr),
	)
	s.logOrderLifecycle(activeSess, cfg, "order_submitted", sideStr, effectiveTicket, "")
	if s.sessionRegistry != nil {
		uid, _ := uuid.Parse(cfg.UserID)
		s.sessionRegistry.InsertScheduleRunLog(ctx, uid, cfg.ScheduleID,
			"order", sideStr, "submitted", "", sig.GetSignalType(), parseDecimal(sig.GetVolume()))
	}

	// Notify accepted and wait for confirmation.
	barrier.NotifyBrokerAccepted(effectiveTicket)

	// For open: verifyReadAfterWrite is nil (ticket unknown at spec creation).
	// Construct it now that we have the effective ticket.
	verify := spec.verifyReadAfterWrite
	if verify == nil {
		verify = verifyTicketPresent(effectiveTicket)
	}

	confirmState := s.waitForConfirmation(ctx, cfg, activeSess, barrier, effectiveTicket, spec.expectedMagic, spec.action, verify, conf)

	switch confirmState {
	case barrierConfirmed:
		s.log.Info("LiveStrategyRunner: mutation confirmed",
			zap.Int64("ticket", effectiveTicket),
			zap.String("action", string(spec.action)),
			zap.String("symbol", cfg.Symbol),
		)
		s.logOrderLifecycle(activeSess, cfg, "order_confirmed", sideStr, effectiveTicket, "")
		barrier.Release()
	case barrierDeterministicRejected:
		s.logOrderLifecycle(activeSess, cfg, "order_rejected", sideStr, effectiveTicket, "confirmation_failed")
		barrier.Release()
	case barrierOutcomeUnknown:
		if activeSess != nil {
			activeSess.SetCircuitOpen(true)
			activeSess.RecordError(fmt.Sprintf("mutation ticket=%d: confirmation outcome unknown, barrier locked", effectiveTicket))
		}
		s.logOrderLifecycle(activeSess, cfg, "order_outcome_unknown", sideStr, effectiveTicket, "confirmation_timeout")
		// ④-②: For known-ticket mutations, start background reconciliation.
		// Open mutations (ticket=0 at spec creation, but effectiveTicket is
		// now known from RPC) also get recovery since we have the ticket.
		verify := spec.verifyReadAfterWrite
		if verify == nil {
			verify = verifyTicketPresent(effectiveTicket)
		}
		go s.recoverFromOutcomeUnknown(cfg, activeSess, barrier, effectiveTicket, spec.expectedMagic, spec.action, verify, conf)
		// Do NOT release.
	default:
		// Context cancelled — release to avoid deadlock on shutdown.
		if ctx.Err() != nil {
			barrier.Release()
		}
	}

	return mutationResult{state: confirmState, ticket: effectiveTicket}
}

// waitForConfirmation waits for push confirmation with a bounded fallback
// to a single read-after-write OpenedOrders query (I6: not polling).
func (s *StrategyExecutionServer) waitForConfirmation(
	ctx context.Context, cfg LiveStrategyConfig, activeSess *ActiveSession,
	barrier *TradeBarrier, ticket int64, magic int32, action mutationAction,
	verify func(orders []*mthub.OrderRecord) bool,
	conf confirmationConfig,
) tradeBarrierState {
	// Phase 1: wait for push confirmation with a bounded timeout.
	waitCtx, waitCancel := context.WithTimeout(ctx, conf.pushWait)
	state := barrier.WaitConfirmed(waitCtx)
	waitCancel()
	if state.isTerminal() {
		return state
	}

	// Phase 2: push not received — single read-after-write (I6: exactly once).
	s.log.Info("LiveStrategyRunner: push not received, performing read-after-write",
		zap.Int64("ticket", ticket),
		zap.String("account", cfg.AccountID),
		zap.String("action", string(action)),
	)
	rawCtx, rawCancel := context.WithTimeout(context.WithoutCancel(ctx), conf.readAfterWriteTimeout)
	orders, err := s.mtHub.OpenedOrders(rawCtx, cfg.AccountID)
	rawCancel()
	if err != nil {
		s.log.Error("LiveStrategyRunner: read-after-write failed — outcome unknown",
			zap.Int64("ticket", ticket),
			zap.String("account", cfg.AccountID),
			zap.Error(err),
		)
		barrier.NotifyOutcomeUnknown()
		return barrier.WaitConfirmed(ctx)
	}

	confirmed := verify(orders)
	if confirmed {
		// Publish the read-after-write result into the existing position
		// pipeline so PositionCache and all subscribers see the broker's
		// authoritative state.
		s.publishReadAfterWriteSnapshot(cfg, orders)
		barrier.NotifyConfirmationEvent(ticket, magic, string(action))
		// Wait briefly for the barrier to process the notification.
		quickCtx, quickCancel := context.WithTimeout(ctx, time.Second)
		state := barrier.WaitConfirmed(quickCtx)
		quickCancel()
		if state == barrierConfirmed {
			return barrierConfirmed
		}
		// Force-confirm based on the authoritative read.
		return barrierConfirmed
	}

	// Read-after-write shows order NOT in expected state — outcome unknown.
	s.log.Error("LiveStrategyRunner: read-after-write confirms NOT in expected state — outcome unknown",
		zap.Int64("ticket", ticket),
		zap.String("account", cfg.AccountID),
		zap.String("action", string(action)),
	)
	barrier.NotifyOutcomeUnknown()
	return barrier.WaitConfirmed(ctx)
}
