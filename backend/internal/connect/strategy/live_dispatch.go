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
	action := sig.GetSignalType()
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

	if cfg.Mode == "paper" {
		s.dispatchPaperSignal(ctx, cfg, bar, sig)
		return
	}

	if s.mtHub == nil {
		s.log.Warn("LiveStrategyRunner: no MtHubService configured, cannot dispatch live order")
		return
	}

	// T3.1: dispatch based on expanded action set.
	switch action {
	case sideBuy, sideSell:
		if activeSess != nil && activeSess.IsCircuitOpen() {
			s.log.Warn("LiveStrategyRunner: suppressing order — circuit breaker open",
				zap.String("account", cfg.AccountID),
				zap.String("symbol", cfg.Symbol),
				zap.String("action", action),
			)
			return
		}
		s.dispatchMarketOrder(ctx, cfg, barOpenTimeForSignal(bar, cfg), sig, activeSess)
	case "buy_limit", "sell_limit", "buy_stop", "sell_stop",
		"buy_stop_limit", "sell_stop_limit":
		if activeSess != nil && activeSess.IsCircuitOpen() {
			s.log.Warn("LiveStrategyRunner: suppressing pending order — circuit breaker open",
				zap.String("account", cfg.AccountID),
				zap.String("symbol", cfg.Symbol),
				zap.String("action", action),
			)
			return
		}
		s.dispatchPendingOrder(ctx, cfg, barOpenTimeForSignal(bar, cfg), sig, activeSess)
	case string(actionClose):
		s.dispatchCloseOrder(ctx, cfg, sig, activeSess)
	case "close_all":
		s.dispatchCloseAll(ctx, cfg, activeSess)
	case "modify":
		s.dispatchModifyOrder(ctx, cfg, sig, activeSess)
	case "cancel":
		s.dispatchCancelOrder(ctx, cfg, sig, activeSess)
	default:
		// hold, unknown — no-op.
	}
}

// ── T3.1 action dispatchers ──────────────────────────────────────────

func (s *StrategyExecutionServer) dispatchMarketOrder(ctx context.Context, cfg LiveStrategyConfig, barOpenTime int64, sig *antv1.StrategySignal, activeSess *ActiveSession) {
	side := signalToSide(sig.GetSignalType())
	if side == 0 {
		return
	}
	s.submitOrder(ctx, cfg, side, mthub.OrderMarket, barOpenTime, sig, activeSess)
}

func (s *StrategyExecutionServer) dispatchPendingOrder(ctx context.Context, cfg LiveStrategyConfig, barOpenTime int64, sig *antv1.StrategySignal, activeSess *ActiveSession) {
	side := signalToSide(sig.GetSignalType())
	if side == 0 {
		return
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
	s.submitOrder(ctx, cfg, side, orderType, barOpenTime, sig, activeSess)
}

// dispatchCloseAll closes all open positions for the account matching this strategy.
// ARCH-4: When ScheduleID is set, positions are filtered by the strategy's magic
// number to avoid closing positions opened by other strategies on the same account.
// When ScheduleID is zero (legacy callers), falls back to symbol-only matching.
//
// LIVE-ORDER-REENTRY-1 B2: close_all takes the authoritative OpenedOrders list
// first, then serially closes each matching position through the same coordinator.
// Any outcome_unknown stops subsequent closes and keeps the barrier locked.
func (s *StrategyExecutionServer) dispatchCloseAll(ctx context.Context, cfg LiveStrategyConfig, activeSess *ActiveSession) {
	if s.mtHub == nil {
		s.log.Warn("LiveStrategyRunner: dispatchCloseAll: no MtHubService")
		if activeSess != nil {
			activeSess.RecordError("dispatchCloseAll: no MtHubService")
		}
		return
	}
	if activeSess == nil || activeSess.barrier == nil {
		s.log.Error("dispatchCloseAll: barrier not configured — dropping (fail-closed)",
			zap.String("account", cfg.AccountID))
		return
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
		return
	}

	closed := 0
	skipped := 0
	for _, o := range orders {
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
			return
		}
		// R7b: only count confirmed closes. A deterministic rejection
		// means the close did NOT happen — counting it as "closed" would
		// inflate the success count and mask failures.
		if result.state == barrierConfirmed {
			closed++
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
}

func (s *StrategyExecutionServer) dispatchCloseOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal, activeSess *ActiveSession) {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: close order without ticket")
		if activeSess != nil {
			activeSess.RecordError("close order without ticket")
		}
		return
	}
	// B2: close goes through the shared coordinator with full confirmation.
	s.coordinateMutation(ctx, cfg, activeSess, mutationSpec{
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

func (s *StrategyExecutionServer) dispatchModifyOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal, activeSess *ActiveSession) {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: modify order without ticket")
		if activeSess != nil {
			activeSess.RecordError("modify order without ticket")
		}
		return
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
	s.coordinateMutation(ctx, cfg, activeSess, mutationSpec{
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

func (s *StrategyExecutionServer) dispatchCancelOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal, activeSess *ActiveSession) {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: cancel order without ticket")
		if activeSess != nil {
			activeSess.RecordError("cancel order without ticket")
		}
		return
	}
	// B2: cancel goes through the shared coordinator with full confirmation.
	s.coordinateMutation(ctx, cfg, activeSess, mutationSpec{
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
func (s *StrategyExecutionServer) submitOrder(ctx context.Context, cfg LiveStrategyConfig, side mthub.Side, orderType mthub.OrderType, barOpenTime int64, sig *antv1.StrategySignal, activeSess *ActiveSession) {
	req := &mthub.OrderRequest{
		AccountID: cfg.AccountID,
		Canonical: cfg.Symbol,
		Side:      side,
		OrderType: orderType,
		Volume:    parseDecimal(sig.GetVolume()),
		Magic:     strategyMagic(cfg.ScheduleID),
		ClientID:  strategyOrderClientID(cfg.RunID, barOpenTime, sig.GetSignalType()),
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
	s.coordinateMutation(ctx, cfg, activeSess, mutationSpec{
		action:        actionOpen,
		clientID:      req.ClientID,
		expectedMagic: req.Magic,
		brokerCall: func(brokerCtx context.Context) (int64, error) {
			record, err := s.mtHub.PlaceOrder(brokerCtx, req)
			if err != nil {
				return 0, err
			}
			return record.Ticket, nil
		},
		verifyReadAfterWrite: nil, // set after ticket is known — see below
	}, sideStr, sig, defaultConfirmationConfig)
}
