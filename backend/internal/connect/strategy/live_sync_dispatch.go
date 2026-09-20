// live_sync_dispatch.go — Synchronous VM signal dispatch
// (VM-LIVE-SYNC-DISPATCH-1, R1). Signals emitted by signal-mode trade
// builtins execute here INSIDE the VM event via the session's sync
// dispatcher — same event-loop goroutine, same barrier semantics as the
// async path. Returns real broker outcomes so builtins stop returning
// fabricated tickets/optimistic bools.

package strategy

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/strategy/runner"
	"alphaforge/strategy/sdk"
)

// dispatchSignalSync executes a VM-emitted signal synchronously inside the
// VM event (VM-LIVE-SYNC-DISPATCH-1, R1). Called from the event-loop
// goroutine via the session's sync dispatcher — same goroutine, same
// barrier semantics as the post-event async path. Returns the broker's
// real ticket for opens; error for rejections/unknowns/idle outcomes so
// the builtin can fail truthfully (-1 / false) instead of returning a
// fabricated ticket.
//
// r is the session's runner — confirmed broker facts are injected into
// its live state so MQL OrderSelect/OrdersTotal see the mutation within
// the same event (MQL4 synchronous semantics).
func (s *StrategyExecutionServer) dispatchSignalSync(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, sig *sdk.Signal, activeSess *ActiveSession, r *runner.Runner) (ticket int64, err error) {
	// Fail-closed panic containment (same as dispatchLiveSignal): a panic
	// inside the coordinator is already converged by its own recover;
	// anything reaching here is outside it. Converge to error so the
	// builtin fails instead of crashing the event-loop goroutine.
	defer func() {
		if rec := recover(); rec != nil {
			s.log.Error("dispatchSignalSync: panic recovered",
				zap.Any("panic", rec),
				zap.String("account", cfg.AccountID))
			if activeSess != nil {
				activeSess.RecordError(fmt.Sprintf("signal %v: panic recovered: %v", sig.Action, rec))
				activeSess.SetCircuitOpen(true)
				if b := activeSess.barrier; b != nil {
					if st := b.State(); st == barrierSubmitting || st == barrierAcceptedUnconfirmed {
						b.NotifyOutcomeUnknown()
					}
				}
			}
			ticket, err = 0, fmt.Errorf("signal dispatch panic: %v", rec)
		}
	}()

	psig := vmSignalToProto(sig, cfg.Symbol)
	action := psig.GetSignalType()
	s.recordSignalIntent(ctx, cfg, psig, action)

	if s.mtHub == nil {
		return 0, fmt.Errorf("no MtHubService configured, cannot dispatch live order")
	}

	res := s.dispatchByAction(ctx, cfg, bar, psig, action, activeSess)

	if r != nil {
		if res.state == barrierConfirmed {
			s.applyConfirmedToRunner(r, psig, action, res)
		}
		// Batch mutations (close_all/cancel_all): every affectedTickets
		// entry was individually coordinator-confirmed — drop them even
		// when the batch ended early on outcome_unknown.
		for _, t := range res.affectedTickets {
			switch action {
			case "close_all":
				r.RemoveConfirmedPosition(t)
			case "cancel_all":
				r.RemoveConfirmedPendingOrder(t)
			}
		}
	}

	switch res.state {
	case barrierConfirmed:
		return res.ticket, nil
	case barrierDeterministicRejected:
		return 0, fmt.Errorf("%s %s rejected by broker", action, cfg.Symbol)
	case barrierIdle:
		return 0, fmt.Errorf("%s not dispatched (barrier busy or dropped)", action)
	default:
		return 0, fmt.Errorf("%s outcome unknown — barrier locked (fail-closed)", action)
	}
}

// applyConfirmedToRunner injects broker-verified mutation facts into the
// runner's live state so the VM observes them in the same event
// (VM-LIVE-SYNC-DISPATCH-1). Only reached on barrierConfirmed — facts are
// read-after-write verified, never optimistic.
func (s *StrategyExecutionServer) applyConfirmedToRunner(r *runner.Runner, sig *antv1.StrategySignal, action string, res mutationResult) {
	switch action {
	case sideBuy, sideSell:
		if res.record != nil {
			r.ApplyConfirmedPosition(orderRecordToPosition(res.record))
		}
	case "buy_limit", "sell_limit", "buy_stop", "sell_stop",
		"buy_stop_limit", "sell_stop_limit":
		if res.record != nil {
			r.ApplyConfirmedPendingOrder(orderRecordToPending(res.record))
		}
	case string(actionClose):
		r.RemoveConfirmedPosition(sig.GetExecutedTicket())
	case "cancel":
		r.RemoveConfirmedPendingOrder(sig.GetExecutedTicket())
	case "modify":
		r.ApplyConfirmedModify(sig.GetExecutedTicket(), parseDecimal(sig.GetStopLoss()), parseDecimal(sig.GetTakeProfit()))
	}
}

// orderRecordToPosition maps a broker order record to an SDK position
// (broker facts verbatim). VM-LIVE-SYNC-DISPATCH-1.
func orderRecordToPosition(rec *mthub.OrderRecord) sdk.Position {
	side := sdk.SideBuy
	if rec.Side == mthub.SideSell {
		side = sdk.SideSell
	}
	return sdk.Position{
		Ticket:     rec.Ticket,
		Symbol:     rec.Canonical,
		Side:       side,
		Volume:     rec.Volume,
		OpenPrice:  rec.OpenPrice,
		StopLoss:   rec.StopLoss,
		TakeProfit: rec.TakeProfit,
		Profit:     rec.Profit,
		Swap:       rec.Swap,
		Commission: rec.Commission,
		Comment:    rec.Comment,
		Magic:      rec.Magic,
		OpenTime:   rec.OpenTime,
	}
}

// orderRecordToPending maps a broker order record to an SDK pending order.
// VM-LIVE-SYNC-DISPATCH-1.
func orderRecordToPending(rec *mthub.OrderRecord) sdk.PendingOrder {
	side := sdk.SideBuy
	if rec.Side == mthub.SideSell {
		side = sdk.SideSell
	}
	return sdk.PendingOrder{
		Ticket:     rec.Ticket,
		Symbol:     rec.Canonical,
		Type:       sdkOrderType(rec.OrderType),
		Side:       side,
		Volume:     rec.Volume,
		Price:      rec.OpenPrice,
		StopLoss:   rec.StopLoss,
		TakeProfit: rec.TakeProfit,
		Comment:    rec.Comment,
		Magic:      rec.Magic,
		OpenTime:   rec.OpenTime,
	}
}

func sdkOrderType(t mthub.OrderType) sdk.OrderType {
	switch t {
	case mthub.OrderLimit:
		return sdk.OrderLimit
	case mthub.OrderStop:
		return sdk.OrderStop
	case mthub.OrderStopLimit:
		return sdk.OrderStopLimit
	default:
		return sdk.OrderMarket
	}
}

// dispatchCancelAll cancels every pending order owned by this strategy's
// magic (VM-LIVE-SYNC-DISPATCH-1 — fills the cancel_all switch hole; the
// signal previously fell into default no-op). Same serial-coordination
// protocol as dispatchCloseAll: any outcome_unknown stops the batch.
func (s *StrategyExecutionServer) dispatchCancelAll(ctx context.Context, cfg LiveStrategyConfig, activeSess *ActiveSession) mutationResult {
	if s.mtHub == nil {
		s.log.Warn("LiveStrategyRunner: dispatchCancelAll: no MtHubService")
		if activeSess != nil {
			activeSess.RecordError("dispatchCancelAll: no MtHubService")
		}
		return mutationResult{state: barrierIdle}
	}
	if activeSess == nil || activeSess.barrier == nil {
		s.log.Error("dispatchCancelAll: barrier not configured — dropping (fail-closed)",
			zap.String("account", cfg.AccountID))
		return mutationResult{state: barrierIdle}
	}

	expectedMagic := strategyMagic(cfg.ScheduleID)
	bgCtx := context.WithoutCancel(ctx)
	orders, err := s.mtHub.OpenedOrders(bgCtx, cfg.AccountID)
	if err != nil {
		s.log.Error("LiveStrategyRunner: dispatchCancelAll: OpenedOrders failed",
			zap.String("account", cfg.AccountID), zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError("dispatchCancelAll: OpenedOrders: " + err.Error())
		}
		return mutationResult{state: barrierIdle}
	}

	cancelled := 0
	var cancelledTickets []int64
	for _, o := range orders {
		// Pending orders only — market positions are closed, not cancelled.
		if o.OrderType == mthub.OrderMarket {
			continue
		}
		if expectedMagic != 0 && o.Magic != expectedMagic {
			continue
		}
		result := s.coordinateMutation(ctx, cfg, activeSess, mutationSpec{
			action:         actionCancel,
			clientID:       fmt.Sprintf("cancel_all_%d_%s", o.Ticket, cfg.RunID.String()),
			expectedMagic:  expectedMagic,
			expectedTicket: o.Ticket,
			brokerCall: func(brokerCtx context.Context) (int64, error) {
				return o.Ticket, s.mtHub.DeleteOrder(brokerCtx, cfg.AccountID, o.Ticket)
			},
			verifyReadAfterWrite: verifyTicketAbsent(o.Ticket),
		}, "cancel", &antv1.StrategySignal{SignalType: "cancel"}, defaultConfirmationConfig)

		if result.state == barrierOutcomeUnknown {
			s.log.Error("LiveStrategyRunner: dispatchCancelAll: cancel outcome unknown — stopping batch",
				zap.Int64("ticket", o.Ticket),
				zap.String("account", cfg.AccountID))
			if activeSess != nil {
				activeSess.RecordError(fmt.Sprintf("cancel_all: cancel ticket=%d outcome unknown, barrier locked — remaining cancels aborted", o.Ticket))
			}
			// Individually-confirmed cancels still happened — carry their
			// tickets so the VM's live state drops them (VM-LIVE-SYNC-DISPATCH-1).
			result.affectedTickets = cancelledTickets
			return result
		}
		if result.state == barrierConfirmed {
			cancelled++
			cancelledTickets = append(cancelledTickets, o.Ticket)
		}
	}
	s.log.Info("LiveStrategyRunner: dispatchCancelAll complete",
		zap.String("account", cfg.AccountID),
		zap.Int32("magic", expectedMagic),
		zap.Int("cancelled", cancelled),
		zap.Int("total", len(orders)),
	)
	return mutationResult{state: barrierConfirmed, affectedTickets: cancelledTickets}
}
