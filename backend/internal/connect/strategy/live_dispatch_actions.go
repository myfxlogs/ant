package strategy

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

func (s *StrategyExecutionServer) dispatchMarketOrder(ctx context.Context, cfg LiveStrategyConfig, barOpenTime int64, sig *antv1.StrategySignal, activeSess *ActiveSession) mutationResult {
	side := signalToSide(sig.GetSignalType())
	if side == 0 {
		return mutationResult{state: barrierIdle}
	}
	return s.submitOrder(ctx, cfg, side, mthub.OrderMarket, barOpenTime, sig, activeSess)
}

func (s *StrategyExecutionServer) dispatchPendingOrder(ctx context.Context, cfg LiveStrategyConfig, barOpenTime int64, sig *antv1.StrategySignal, activeSess *ActiveSession) mutationResult {
	side := signalToSide(sig.GetSignalType())
	if side == 0 {
		return mutationResult{state: barrierIdle}
	}
	var orderType mthub.OrderType
	switch sig.GetSignalType() {
	case "buy_limit", "sell_limit":
		orderType = mthub.OrderLimit
	case "buy_stop", "sell_stop":
		orderType = mthub.OrderStop
	case "buy_stop_limit", "sell_stop_limit":
		orderType = mthub.OrderStopLimit
	default:
		orderType = mthub.OrderLimit
	}
	return s.submitOrder(ctx, cfg, side, orderType, barOpenTime, sig, activeSess)
}

// dispatchCloseAll closes all open positions for the account matching this strategy.
// ARCH-4: When ScheduleID is set, positions are filtered by the strategy's magic
// number to avoid closing positions opened by other strategies on the same account.
// When ScheduleID is zero (legacy callers), falls back to symbol-only matching.
//
// LIVE-ORDER-REENTRY-1 B2: close_all takes the authoritative OpenedOrders list
// first, then serially closes each matching position through the same coordinator.
// Any outcome_unknown stops subsequent closes and keeps the barrier locked.
func (s *StrategyExecutionServer) dispatchCloseAll(ctx context.Context, cfg LiveStrategyConfig, activeSess *ActiveSession) mutationResult {
	if s.mtHub == nil {
		s.log.Warn("LiveStrategyRunner: dispatchCloseAll: no MtHubService")
		if activeSess != nil {
			activeSess.RecordError("dispatchCloseAll: no MtHubService")
		}
		return mutationResult{state: barrierIdle}
	}
	if activeSess == nil || activeSess.barrier == nil {
		s.log.Error("dispatchCloseAll: barrier not configured — dropping (fail-closed)",
			zap.String("account", cfg.AccountID))
		return mutationResult{state: barrierIdle}
	}

	expectedMagic := strategyMagic(cfg.ScheduleID)
	bgCtx := context.WithoutCancel(ctx)
	orders, err := s.mtHub.OpenedOrders(bgCtx, cfg.AccountID)
	if err != nil {
		s.log.Error("LiveStrategyRunner: dispatchCloseAll: OpenedOrders failed",
			zap.String("account", cfg.AccountID), zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError("dispatchCloseAll: OpenedOrders: " + err.Error())
		}
		return mutationResult{state: barrierIdle}
	}

	closed := 0
	skipped := 0
	var closedTickets []int64
	for _, o := range orders {
		// Market positions only — pendings require DeleteOrder (cancel_all);
		// CloseOrder on a pending is a deterministic rejection and its
		// confirmation wait would poison the batch with outcome_unknown.
		if o.OrderType != mthub.OrderMarket {
			skipped++
			continue
		}
		// ARCH-4: filter by magic or symbol.
		if expectedMagic != 0 {
			if o.Magic != expectedMagic {
				skipped++
				continue
			}
		} else {
			if o.Canonical != cfg.Symbol && o.SymbolRaw != cfg.Symbol {
				skipped++
				continue
			}
		}
		// B2: each close goes through the shared coordinator with full
		// confirmation. Any outcome_unknown stops subsequent closes.
		result := s.coordinateMutation(ctx, cfg, activeSess, mutationSpec{
			action:         actionClose,
			clientID:       fmt.Sprintf("close_all_%d_%s", o.Ticket, cfg.RunID.String()),
			expectedMagic:  expectedMagic,
			expectedTicket: o.Ticket,
			brokerCall: func(brokerCtx context.Context) (int64, error) {
				return o.Ticket, s.mtHub.CloseOrder(brokerCtx, cfg.AccountID, o.Ticket, o.Volume)
			},
			verifyReadAfterWrite: verifyTicketAbsent(o.Ticket),
		}, "close", &antv1.StrategySignal{SignalType: "close", Volume: o.Volume.String()}, defaultConfirmationConfig)

		if result.state == barrierOutcomeUnknown {
			s.log.Error("LiveStrategyRunner: dispatchCloseAll: close outcome unknown — stopping subsequent closes",
				zap.Int64("ticket", o.Ticket),
				zap.String("account", cfg.AccountID))
			if activeSess != nil {
				activeSess.RecordError(fmt.Sprintf("close_all: close ticket=%d outcome unknown, barrier locked — remaining closes aborted", o.Ticket))
			}
			// Individually-confirmed closes still happened — carry their
			// tickets so the VM's live state drops them (VM-LIVE-SYNC-DISPATCH-1).
			result.affectedTickets = closedTickets
			return result
		}
		// R7b: only count confirmed closes. A deterministic rejection
		// means the close did NOT happen — counting it as "closed" would
		// inflate the success count and mask failures.
		if result.state == barrierConfirmed {
			closed++
			closedTickets = append(closedTickets, o.Ticket)
		}
	}
	s.log.Info("LiveStrategyRunner: dispatchCloseAll complete",
		zap.String("account", cfg.AccountID),
		zap.String("symbol", cfg.Symbol),
		zap.Int32("magic", expectedMagic),
		zap.Int("closed", closed),
		zap.Int("skipped", skipped),
		zap.Int("total", len(orders)),
	)
	return mutationResult{state: barrierConfirmed, affectedTickets: closedTickets}
}

func (s *StrategyExecutionServer) dispatchCloseOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal, activeSess *ActiveSession) mutationResult {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: close order without ticket")
		if activeSess != nil {
			activeSess.RecordError("close order without ticket")
		}
		return mutationResult{state: barrierIdle}
	}
	// B2: close goes through the shared coordinator with full confirmation.
	return s.coordinateMutation(ctx, cfg, activeSess, mutationSpec{
		action:         actionClose,
		clientID:       fmt.Sprintf("close_%d", ticket),
		expectedMagic:  strategyMagic(cfg.ScheduleID),
		expectedTicket: ticket,
		brokerCall: func(brokerCtx context.Context) (int64, error) {
			// W1: volume=0 is valid for close signals (full close).
			return ticket, s.mtHub.CloseOrder(brokerCtx, cfg.AccountID, ticket, parseDecimal(sig.GetVolume()))
		},
		verifyReadAfterWrite: verifyTicketAbsent(ticket),
	}, "close", sig, defaultConfirmationConfig)
}

func (s *StrategyExecutionServer) dispatchModifyOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal, activeSess *ActiveSession) mutationResult {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: modify order without ticket")
		if activeSess != nil {
			activeSess.RecordError("modify order without ticket")
		}
		return mutationResult{state: barrierIdle}
	}
	// B2: modify goes through the shared coordinator with full confirmation.
	// R5: read-after-write verifies SL/TP/price actually changed, not just
	// ticket presence. R5-⑤: parseDecimalPtr distinguishes "not provided"
	// (nil, don't check) from "explicitly zero" (clear SL/TP to 0).
	sl := parseDecimal(sig.GetStopLoss())
	tp := parseDecimal(sig.GetTakeProfit())
	px := parseDecimal(sig.GetPrice())
	slPtr := parseDecimalPtr(sig.GetStopLoss())
	tpPtr := parseDecimalPtr(sig.GetTakeProfit())
	pxPtr := parseDecimalPtr(sig.GetPrice())
	return s.coordinateMutation(ctx, cfg, activeSess, mutationSpec{
		action:         actionModify,
		clientID:       fmt.Sprintf("modify_%d", ticket),
		expectedMagic:  strategyMagic(cfg.ScheduleID),
		expectedTicket: ticket,
		brokerCall: func(brokerCtx context.Context) (int64, error) {
			return ticket, s.mtHub.ModifyOrder(brokerCtx, cfg.AccountID, ticket, sl, tp, px)
		},
		verifyReadAfterWrite: verifyTicketModified(ticket, slPtr, tpPtr, pxPtr),
	}, "modify", sig, defaultConfirmationConfig)
}

func (s *StrategyExecutionServer) dispatchCancelOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal, activeSess *ActiveSession) mutationResult {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: cancel order without ticket")
		if activeSess != nil {
			activeSess.RecordError("cancel order without ticket")
		}
		return mutationResult{state: barrierIdle}
	}
	// B2: cancel goes through the shared coordinator with full confirmation.
	return s.coordinateMutation(ctx, cfg, activeSess, mutationSpec{
		action:         actionCancel,
		clientID:       fmt.Sprintf("cancel_%d", ticket),
		expectedMagic:  strategyMagic(cfg.ScheduleID),
		expectedTicket: ticket,
		brokerCall: func(brokerCtx context.Context) (int64, error) {
			return ticket, s.mtHub.DeleteOrder(brokerCtx, cfg.AccountID, ticket)
		},
		verifyReadAfterWrite: verifyTicketAbsent(ticket),
	}, "cancel", sig, defaultConfirmationConfig)
}

func (s *StrategyExecutionServer) submitOrder(ctx context.Context, cfg LiveStrategyConfig, side mthub.Side, orderType mthub.OrderType, barOpenTime int64, sig *antv1.StrategySignal, activeSess *ActiveSession) mutationResult {
	req := &mthub.OrderRequest{
		AccountID: cfg.AccountID,
		Canonical: cfg.Symbol,
		Side:      side,
		OrderType: orderType,
		Volume:    parseDecimal(sig.GetVolume()),
		Magic:     strategyMagic(cfg.ScheduleID),
		ClientID:  strategyOrderClientID(cfg.RunID, barOpenTime, sig.GetSignalType()),
		// VM-LIVE-PARITY-F3/F1: strategy comment + deviation reach the broker.
		Comment:   sig.GetComment(),
		Deviation: sig.GetDeviation(),
	}
	sl := parseDecimal(sig.GetStopLoss())
	if sl.GreaterThan(decimal.Zero) {
		req.StopLoss = sl
	}
	tp := parseDecimal(sig.GetTakeProfit())
	if tp.GreaterThan(decimal.Zero) {
		req.TakeProfit = tp
	}
	px := parseDecimal(sig.GetPrice())
	if px.GreaterThan(decimal.Zero) {
		req.Price = px
	}

	sideStr := sideToString(side)
	var rec *mthub.OrderRecord
	res := s.coordinateMutation(ctx, cfg, activeSess, mutationSpec{
		action:        actionOpen,
		clientID:      req.ClientID,
		expectedMagic: req.Magic,
		brokerCall: func(brokerCtx context.Context) (int64, error) {
			record, err := s.mtHub.PlaceOrder(brokerCtx, req)
			if err != nil {
				return 0, err
			}
			rec = record
			return record.Ticket, nil
		},
		verifyReadAfterWrite: nil, // set after ticket is known — see below
	}, sideStr, sig, defaultConfirmationConfig)
	// VM-LIVE-SYNC-DISPATCH-1: broker record rides the result so the caller
	// can inject broker facts into the VM's live state.
	res.record = rec
	return res
}
