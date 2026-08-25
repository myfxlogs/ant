package strategy

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
)

// buildTickContext creates a TickContext proto from a tick update.
func (s *StrategyExecutionServer) buildTickContext(ctx context.Context, cfg LiveStrategyConfig, tick *mthub.TickUpdate) (*antv1.TickContext, error) {
	tctx := &antv1.TickContext{
		Bid:          tick.Bid.String(),
		Ask:          tick.Ask.String(),
		Symbol:       cfg.Symbol,
		Timeframe:    cfg.Timeframe,
		Mode:         cfg.Mode,
		Params:       buildLiveParams(cfg.Params),
		CurrentPrice: tick.Bid.String(),
	}
	if err := s.backfillContextStrings(cfg.AccountID, &tctx.Equity, &tctx.Balance, &tctx.Margin, &tctx.FreeMargin, &tctx.Positions, &tctx.PendingOrders); err != nil && cfg.Mode == "live" {
		return nil, err
	}
	s.backfillTickSymbolInfo(cfg, tctx)
	return tctx, nil
}

// buildTradeContext creates a TradeContext proto from a trade event.
func (s *StrategyExecutionServer) buildTradeContext(ctx context.Context, cfg LiveStrategyConfig, evt *mthub.BrokerTradeEvent) (*antv1.TradeContext, error) {
	// VM-TRADE-CONTEXT-6 round 5: unknown side/event type must fail-closed,
	// not default to buy/fill (which would silently change trade semantics).
	side, err := brokerSideFromString(evt.Side)
	if err != nil {
		return nil, fmt.Errorf("buildTradeContext: %w", err)
	}
	evtType, err := brokerTradeEventTypeString(evt.EventType)
	if err != nil {
		return nil, fmt.Errorf("buildTradeContext: %w", err)
	}
	tctx := &antv1.TradeContext{
		Ticket:     evt.Ticket,
		Symbol:     evt.Symbol,
		EventType:  evtType,
		Side:       side,
		Volume:     evt.Volume.String(),
		Price:      evt.Price.String(),
		StopLoss:   evt.StopLoss.String(),
		TakeProfit: evt.TakeProfit.String(),
		Profit:     evt.Profit.String(),
		Commission: evt.Commission.String(),
		Swap:       evt.Swap.String(),
		Mode:       cfg.Mode, // VM-TRADE-CONTEXT-6 round 5: mode for financial validation
	}
	if err := s.backfillContextStrings(cfg.AccountID, &tctx.Equity, &tctx.Balance, &tctx.Margin, &tctx.FreeMargin, &tctx.Positions, &tctx.PendingOrders); err != nil && cfg.Mode == "live" {
		return nil, err
	}
	return tctx, nil
}

// backfillContextStrings populates equity/balance/margin/free_margin/positions/pending_orders
// from the push-based PositionCache (subscribed to PositionSnapshotBroker). No polling — O(1) read.
// Missing or stale authoritative snapshots return an error and block execution.
// LIVE-ORDER-REENTRY-1: uses GetFreshTradingSnapshot — both financials AND
// positions must be fresh for VM evaluation. A financial-only refresh must
// not make stale positions usable.
// LIVE-MQL-ORDER-CONTEXT-1: now also populates pending orders and all
// LivePosition fields (symbol, magic, order_type, sl, tp, profit, comment,
// open_time) so MQL OrdersTotal/OrderSelect/OrderMagicNumber preserve
// broker-original account-level semantics.
func (s *StrategyExecutionServer) backfillContextStrings(accountID string, equity, balance, margin, freeMargin *string, positions *[]*antv1.LivePosition, pendingOrders *[]*antv1.LivePendingOrder) error {
	if s.posCache == nil {
		return fmt.Errorf("authoritative account snapshot unavailable: position cache not configured")
	}
	snap, ok := s.posCache.GetFreshTradingSnapshot(accountID, time.Now())
	if !ok {
		return fmt.Errorf("authoritative account snapshot unavailable or stale: account=%s", accountID)
	}
	*equity = snap.Equity.String()
	*balance = snap.Balance.String()
	*margin = snap.Margin.String()
	*freeMargin = snap.FreeMargin.String()
	pos := make([]*antv1.LivePosition, 0, len(snap.Positions))
	for _, p := range snap.Positions {
		// VM-TRADE-CONTEXT-6 round 5: unknown side must fail-closed,
		// not default to buy (which would silently change trade semantics).
		side, err := brokerSideFromString(p.Type)
		if err != nil {
			return fmt.Errorf("position %d: %w", p.Ticket, err)
		}
		pos = append(pos, &antv1.LivePosition{
			Ticket:      p.Ticket,
			Side:        side,
			Volume:      p.Volume.String(),
			OpenPrice:   p.OpenPrice.String(),
			Sl:          p.StopLoss.String(),
			Tp:          p.TakeProfit.String(),
			Swap:        p.Swap.String(),
			Commission:  p.Commission.String(),
			Symbol:      p.Symbol,
			MagicNumber: p.Magic,
			OrderType:   p.Type,
			Profit:      p.Profit.String(),
			Comment:     p.Comment,
			OpenTime:    p.OpenTime,
		})
	}
	*positions = pos

	// LIVE-MQL-ORDER-CONTEXT-1: populate pending orders with full fields.
	pending := make([]*antv1.LivePendingOrder, 0, len(snap.PendingOrders))
	for _, o := range snap.PendingOrders {
		// VM-TRADE-CONTEXT-6 round 5: unknown order type must fail-closed.
		pSide, err := pendingOrderSide(o.Type)
		if err != nil {
			return fmt.Errorf("pending order %d: %w", o.Ticket, err)
		}
		pending = append(pending, &antv1.LivePendingOrder{
			Ticket:      o.Ticket,
			Symbol:      o.Symbol,
			OrderType:   o.Type,
			Side:        pSide,
			Volume:      o.Volume.String(),
			Price:       o.OpenPrice.String(),
			Sl:          o.StopLoss.String(),
			Tp:          o.TakeProfit.String(),
			Comment:     o.Comment,
			MagicNumber: o.Magic,
			OpenTime:    o.OpenTime,
		})
	}
	*pendingOrders = pending
	return nil
}

// dispatchFromBytes unmarshals a live response and dispatches signals to OMS.
func (s *StrategyExecutionServer) dispatchFromBytes(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, respBytes []byte, activeSess *ActiveSession) {
	var resp antv1.ExecuteLiveResponse
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		s.log.Error("LiveStrategyRunner: unmarshal response failed", zap.Error(err))
		if activeSess != nil {
			activeSess.RecordError(err.Error())
		}
		return
	}
	if !resp.GetSuccess() {
		s.log.Warn("LiveStrategyRunner: strategy returned error", zap.String("error", resp.GetError()))
		if activeSess != nil {
			activeSess.RecordError(resp.GetError())
		}
		return
	}
	signals := resp.GetSignals()
	if len(signals) == 0 {
		sig := resp.GetSignal()
		if sig != nil && sig.GetSignalType() != "" && sig.GetSignalType() != "hold" {
			signals = []*antv1.StrategySignal{sig}
		}
	}
	for _, sig := range signals {
		if sig.GetSignalType() == "hold" || sig.GetSignalType() == "" {
			continue
		}
		if activeSess != nil {
			activeSess.RecordSignal(&SignalEvent{
				RunID:      cfg.RunID,
				AccountID:  cfg.AccountID,
				Symbol:     cfg.Symbol,
				SignalType: sig.GetSignalType(),
				Volume:     sig.GetVolume(),
				Price:      sig.GetPrice(),
				StopLoss:   sig.GetStopLoss(),
				TakeProfit: sig.GetTakeProfit(),
				Reason:     sig.GetReason(),
				Timestamp:  time.Now(),
			})
		}
		if cfg.ShadowVerifier != nil {
			barTime := int64(0)
			if bar != nil {
				barTime = bar.OpenTime
			}
			cfg.ShadowVerifier.RecordLiveSignal(barTime, sig.GetSignalType(), sig.GetVolume(), sig.GetPrice())
		}
		s.dispatchLiveSignal(ctx, cfg, bar, sig, activeSess)
	}
}

// buildLiveContext creates a full OHLCV bar context from the bar window.
