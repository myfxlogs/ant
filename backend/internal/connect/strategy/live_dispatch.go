package strategy

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
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
func (s *StrategyExecutionServer) dispatchLiveSignal(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, sig *antv1.StrategySignal, activeSess *ActiveSession) {
	action := sig.GetSignalType()
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
	case "buy", sideSell:
		if activeSess != nil && activeSess.IsCircuitOpen() {
			s.log.Warn("LiveStrategyRunner: suppressing order — circuit breaker open",
				zap.String("account", cfg.AccountID),
				zap.String("symbol", cfg.Symbol),
				zap.String("action", action),
			)
			return
		}
		s.dispatchMarketOrder(ctx, cfg, bar.OpenTime, sig, activeSess)
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
		s.dispatchPendingOrder(ctx, cfg, bar.OpenTime, sig, activeSess)
	case "close":
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
func (s *StrategyExecutionServer) dispatchCloseAll(ctx context.Context, cfg LiveStrategyConfig, activeSess *ActiveSession) {
	if s.mtHub == nil {
		s.log.Warn("LiveStrategyRunner: dispatchCloseAll: no MtHubService")
		if activeSess != nil {
			activeSess.RecordError("dispatchCloseAll: no MtHubService")
		}
		return
	}
	expectedMagic := strategyMagic(cfg.ScheduleID)
	// Detach from parent cancellation but preserve values (userID, auth).
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error("panic in dispatchCloseAll", zap.Any("panic", r))
			}
		}()
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
			// ARCH-4: When magic is set, filter by magic number (strategy attribution).
			// When magic is 0 (legacy/no ScheduleID), fall back to symbol-only matching.
			if expectedMagic != 0 {
				if o.Magic != expectedMagic {
					skipped++
					continue
				}
			} else {
				// L1: Only close positions matching this strategy's symbol.
				if o.Canonical != cfg.Symbol && o.SymbolRaw != cfg.Symbol {
					skipped++
					continue
				}
			}
			if err := s.mtHub.CloseOrder(bgCtx, cfg.AccountID, o.Ticket, o.Volume); err != nil {
				s.log.Warn("LiveStrategyRunner: dispatchCloseAll: CloseOrder failed",
					zap.Int64("ticket", o.Ticket), zap.Error(err))
				if activeSess != nil {
					activeSess.RecordError(fmt.Sprintf("CloseOrder ticket=%d: %s", o.Ticket, err.Error()))
				}
				continue
			}
			closed++
		}
		s.log.Info("LiveStrategyRunner: dispatchCloseAll complete",
			zap.String("account", cfg.AccountID),
			zap.String("symbol", cfg.Symbol),
			zap.Int32("magic", expectedMagic),
			zap.Int("closed", closed),
			zap.Int("skipped", skipped),
			zap.Int("total", len(orders)),
		)
	}()
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
	// D6-A: gate moved to MtHubService.CloseOrder (single chokepoint).
	volume := parseDecimal(sig.GetVolume())
	if volume.LessThanOrEqual(decimal.Zero) {
		s.log.Warn("LiveStrategyRunner: close order with zero volume, skipping",
			zap.Int64("ticket", ticket))
		if activeSess != nil {
			activeSess.RecordError(fmt.Sprintf("close order ticket=%d: zero volume", ticket))
		}
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error("panic in dispatchCloseOrder", zap.Any("panic", r))
			}
		}()
		if err := s.mtHub.CloseOrder(context.WithoutCancel(ctx), cfg.AccountID, ticket, volume); err != nil {
			s.log.Error("LiveStrategyRunner: CloseOrder failed",
				zap.Int64("ticket", ticket), zap.Error(err))
			if activeSess != nil {
				activeSess.RecordError(fmt.Sprintf("CloseOrder ticket=%d: %s", ticket, err.Error()))
			}
			return
		}
		s.log.Info("LiveStrategyRunner: position closed", zap.Int64("ticket", ticket))
	}()
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
	sl := parseDecimal(sig.GetStopLoss())
	tp := parseDecimal(sig.GetTakeProfit())
	price := parseDecimal(sig.GetPrice())

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error("panic in dispatchModifyOrder", zap.Any("panic", r))
			}
		}()
		if err := s.mtHub.ModifyOrder(context.WithoutCancel(ctx), cfg.AccountID, ticket, sl, tp, price); err != nil {
			s.log.Error("LiveStrategyRunner: ModifyOrder failed",
				zap.Int64("ticket", ticket), zap.Error(err))
			if activeSess != nil {
				activeSess.RecordError(fmt.Sprintf("ModifyOrder ticket=%d: %s", ticket, err.Error()))
			}
		}
	}()
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
	// L10: Use DeleteOrder — MT4 adapter calls OrderDelete, MT5 adapter
	// calls OrderClose(lots=0). Both are platform-correct cancel paths.
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error("panic in dispatchCancelOrder", zap.Any("panic", r))
			}
		}()
		if err := s.mtHub.DeleteOrder(bgCtx, cfg.AccountID, ticket); err != nil {
			s.log.Error("LiveStrategyRunner: DeleteOrder failed",
				zap.Int64("ticket", ticket), zap.Error(err))
			if activeSess != nil {
				activeSess.RecordError(fmt.Sprintf("DeleteOrder ticket=%d: %s", ticket, err.Error()))
			}
			return
		}
		s.log.Info("LiveStrategyRunner: pending order cancelled", zap.Int64("ticket", ticket))
	}()
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
			s.log.Warn("LiveStrategyRunner: paper close failed", zap.Error(err))
		}
		return
	case "modify":
		sl := parseDecimal(sig.GetStopLoss())
		tp := parseDecimal(sig.GetTakeProfit())
		if err := s.paperEngine.ModifyPaperOrder(ctx, cfg.AccountID, cfg.Symbol, sl, tp); err != nil {
			s.log.Warn("LiveStrategyRunner: paper modify failed", zap.Error(err))
		}
		return
	case "cancel":
		if err := s.paperEngine.CancelPaperOrder(ctx, cfg.AccountID, cfg.Symbol); err != nil {
			s.log.Warn("LiveStrategyRunner: paper cancel failed", zap.Error(err))
		}
		return
	}

	bid := bar.Bid
	ask := bar.Ask
	if err := s.paperEngine.PlacePaperOrder(ctx, cfg.AccountID, cfg.Symbol,
		action, parseDecimal(sig.GetVolume()), bid, ask); err != nil {
		s.log.Warn("LiveStrategyRunner: paper order failed", zap.Error(err))
	}
}

// submitOrder is the common order submission helper (T3.1 / D6-A).
// Every order MUST pass through Gate.Evaluate() before reaching mthub.
func (s *StrategyExecutionServer) submitOrder(ctx context.Context, cfg LiveStrategyConfig, side mthub.Side, orderType mthub.OrderType, barOpenTime int64, sig *antv1.StrategySignal, activeSess *ActiveSession) {
	req := &mthub.OrderRequest{
		AccountID: cfg.AccountID,
		Canonical: cfg.Symbol,
		Side:      side,
		OrderType: orderType,
		Volume:    parseDecimal(sig.GetVolume()),
		// ARCH-4: Magic Number attribution — each strategy schedule gets a
		// deterministic magic derived from ScheduleID, so positions can be
		// attributed to the correct strategy when multiple share an account.
		Magic: strategyMagic(cfg.ScheduleID),
		// LIVE-2: deterministic ClientID for idempotency — same bar + same signal type
		// within the same strategy run produces the same key, so duplicate dispatches
		// (bar replay, VM retry, network glitch) are deduplicated by the idempotency guard.
		ClientID: strategyOrderClientID(cfg.RunID, barOpenTime, sig.GetSignalType()),
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

	// D6-A: gate moved to MtHubService.PlaceOrder (single chokepoint).
	// Pass user ID in context for mthub capability check.
	placeCtx := context.WithoutCancel(ctx)
	if cfg.UserID != "" {
		placeCtx = context.WithValue(placeCtx, interceptor.UserIDKey, cfg.UserID)
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error("panic in submitOrder", zap.Any("panic", r))
			}
		}()
		record, err := s.mtHub.PlaceOrder(placeCtx, req)
		if err != nil {
			if errors.Is(err, mthub.ErrCircuitOpen) {
				if activeSess != nil {
					activeSess.SetCircuitOpen(true)
				}
				s.log.Warn("LiveStrategyRunner: order rejected — circuit breaker open",
					zap.String("account", cfg.AccountID),
					zap.String("symbol", cfg.Symbol),
					zap.String("side", sideStr),
				)
				return
			}
			s.log.Error("LiveStrategyRunner: order submission failed",
				zap.String("symbol", cfg.Symbol),
				zap.String("side", sideStr),
				zap.Error(err),
			)
			if activeSess != nil {
				activeSess.RecordError(fmt.Sprintf("order %s %s: %s", sideStr, cfg.Symbol, err.Error()))
			}
			return
		}
		// Order succeeded — clear circuit open flag if it was set.
		if activeSess != nil {
			activeSess.SetCircuitOpen(false)
		}
		s.log.Info("LiveStrategyRunner: order submitted",
			zap.Int64("ticket", record.Ticket),
			zap.String("symbol", cfg.Symbol),
			zap.String("side", sideStr),
		)
	}()
}
