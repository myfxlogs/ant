// live_dispatch_paper.go — Paper signal dispatch and order submission extracted from live_dispatch.go.
package strategy

import (
	"context"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

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
	// VM-TRADE-CONTEXT-4: Use EA-configured magic from signal if non-zero,
	// falling back to schedule magic for legacy callers.
	magic := sig.GetMagic()
	if magic == 0 {
		magic = strategyMagic(cfg.ScheduleID)
	}
	req := &mthub.OrderRequest{
		AccountID: cfg.AccountID,
		Canonical: cfg.Symbol,
		Side:      side,
		OrderType: orderType,
		Volume:    parseDecimal(sig.GetVolume()),
		Magic:     magic,
		Deviation: sig.GetDeviation(), // VM-TRADE-CONTEXT-5: EA-configured deviation
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
