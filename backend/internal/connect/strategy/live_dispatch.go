package strategy

import (
	"context"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
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
func (s *StrategyExecutionServer) dispatchLiveSignal(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, sig *antv1.StrategySignal) {
	action := sig.GetSignalType()
	s.log.Info("LiveStrategyRunner: signal",
		zap.String("account", cfg.AccountID),
		zap.String("symbol", cfg.Symbol),
		zap.String("type", action),
		zap.String("volume", sig.GetVolume()),
		zap.String("sl", sig.GetStopLoss()),
		zap.String("tp", sig.GetTakeProfit()),
	)

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
	case "buy", "sell":
		s.dispatchMarketOrder(ctx, cfg, sig)
	case "buy_limit", "sell_limit", "buy_stop", "sell_stop",
		"buy_stop_limit", "sell_stop_limit":
		s.dispatchPendingOrder(ctx, cfg, sig)
	case "close":
		s.dispatchCloseOrder(ctx, cfg, sig)
	case "close_all":
		s.dispatchCloseAll(ctx, cfg)
	case "modify":
		s.dispatchModifyOrder(ctx, cfg, sig)
	case "cancel":
		s.dispatchCancelOrder(ctx, cfg, sig)
	default:
		// hold, unknown — no-op.
	}
}

// ── T3.1 action dispatchers ──────────────────────────────────────────

func (s *StrategyExecutionServer) dispatchMarketOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
	side := signalToSide(sig.GetSignalType())
	if side == 0 {
		return
	}
	s.submitOrder(ctx, cfg, side, mthub.OrderMarket, sig)
}

func (s *StrategyExecutionServer) dispatchPendingOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
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
	s.submitOrder(ctx, cfg, side, orderType, sig)
}

// dispatchCloseAll closes all open positions for the account.
func (s *StrategyExecutionServer) dispatchCloseAll(ctx context.Context, cfg LiveStrategyConfig) {
	if s.mtHub == nil {
		s.log.Warn("LiveStrategyRunner: dispatchCloseAll: no MtHubService")
		return
	}
	// Detach from parent cancellation but preserve values (userID, auth).
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		orders, err := s.mtHub.OpenedOrders(bgCtx, cfg.AccountID)
		if err != nil {
			s.log.Error("LiveStrategyRunner: dispatchCloseAll: OpenedOrders failed",
				zap.String("account", cfg.AccountID), zap.Error(err))
			return
		}
		closed := 0
		for _, o := range orders {
			if err := s.mtHub.CloseOrder(bgCtx, cfg.AccountID, o.Ticket, o.Volume); err != nil {
				s.log.Warn("LiveStrategyRunner: dispatchCloseAll: CloseOrder failed",
					zap.Int64("ticket", o.Ticket), zap.Error(err))
				continue
			}
			closed++
		}
		s.log.Info("LiveStrategyRunner: dispatchCloseAll complete",
			zap.String("account", cfg.AccountID),
			zap.Int("closed", closed),
			zap.Int("total", len(orders)),
		)
	}()
}

func (s *StrategyExecutionServer) dispatchCloseOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: close order without ticket")
		return
	}
	// D6-A: gate moved to MtHubService.CloseOrder (single chokepoint).
	volume := parseDecimal(sig.GetVolume())
	go func() {
		if err := s.mtHub.CloseOrder(context.WithoutCancel(ctx), cfg.AccountID, ticket, volume); err != nil {
			s.log.Error("LiveStrategyRunner: CloseOrder failed",
				zap.Int64("ticket", ticket), zap.Error(err))
			return
		}
		s.log.Info("LiveStrategyRunner: position closed", zap.Int64("ticket", ticket))
	}()
}

func (s *StrategyExecutionServer) dispatchModifyOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: modify order without ticket")
		return
	}
	sl := parseDecimal(sig.GetStopLoss())
	tp := parseDecimal(sig.GetTakeProfit())
	price := parseDecimal(sig.GetPrice())

	go func() {
		if err := s.mtHub.ModifyOrder(context.WithoutCancel(ctx), cfg.AccountID, ticket, sl, tp, price); err != nil {
			s.log.Error("LiveStrategyRunner: ModifyOrder failed",
				zap.Int64("ticket", ticket), zap.Error(err))
		}
	}()
}

func (s *StrategyExecutionServer) dispatchCancelOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: cancel order without ticket")
		return
	}
	// MT5 OrderClose handles both market positions AND pending orders.
	// With lots=decimal.Zero (no volume), the gateway cancels the pending
	// order instead of closing a position. See mt5.proto OrderClose docs.
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		if err := s.mtHub.CloseOrder(bgCtx, cfg.AccountID, ticket, decimal.Zero); err != nil {
			s.log.Error("LiveStrategyRunner: CancelOrder failed",
				zap.Int64("ticket", ticket), zap.Error(err))
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
func (s *StrategyExecutionServer) submitOrder(ctx context.Context, cfg LiveStrategyConfig, side mthub.Side, orderType mthub.OrderType, sig *antv1.StrategySignal) {
	req := &mthub.OrderRequest{
		AccountID: cfg.AccountID,
		Canonical: cfg.Symbol,
		Side:      side,
		OrderType: orderType,
		Volume:    parseDecimal(sig.GetVolume()),
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
		record, err := s.mtHub.PlaceOrder(placeCtx, req)
		if err != nil {
			s.log.Error("LiveStrategyRunner: order submission failed",
				zap.String("symbol", cfg.Symbol),
				zap.String("side", sideStr),
				zap.Error(err),
			)
			return
		}
		s.log.Info("LiveStrategyRunner: order submitted",
			zap.Int64("ticket", record.Ticket),
			zap.String("symbol", cfg.Symbol),
			zap.String("side", sideStr),
		)
	}()
}

// signalToSide maps a strategy signal action to mthub.Side. Returns 0 for non-directional signals.
// T3.1: expanded to cover all buy_* / sell_* variants (hard injury ②).
func signalToSide(action string) mthub.Side {
	switch action {
	case "buy", "buy_limit", "buy_stop", "buy_stop_limit":
		return mthub.SideBuy
	case "sell", "sell_limit", "sell_stop", "sell_stop_limit":
		return mthub.SideSell
	default:
		return 0
	}
}

// sideToString returns a human-readable Side string for logging.
func sideToString(side mthub.Side) string {
	switch side {
	case mthub.SideBuy:
		return "buy"
	case mthub.SideSell:
		return "sell"
	default:
		return "unknown"
	}
}
