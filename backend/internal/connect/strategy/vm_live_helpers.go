package strategy

import (
	"fmt"
	"time"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/sdk"
)

// vmLiveStateToSdk converts positions and pending orders in one call,
// returning an error if any strict parse fails. VM-TRADE-CONTEXT-6.
func vmLiveStateToSdk(lps []*antv1.LivePosition, lpos []*antv1.LivePendingOrder) ([]sdk.Position, []sdk.PendingOrder, error) {
	positions, err := vmPositionsToSdk(lps)
	if err != nil {
		return nil, nil, err
	}
	orders, err := vmPendingOrdersToSdk(lpos)
	if err != nil {
		return nil, nil, err
	}
	return positions, orders, nil
}

// vmPositionsToSdk converts LivePosition protos to SDK Positions.
// LIVE-MQL-ORDER-CONTEXT-1: now preserves ALL fields (symbol, magic, sl, tp,
// profit, swap, commission, comment, open_time) so MQL OrderSelect/OrderMagicNumber/
// OrderSymbol/OrderStopLoss/OrderTakeProfit/OrderComment return broker-original values.
// VM-TRADE-CONTEXT-6: strict parsing — invalid decimal fails closed.
func vmPositionsToSdk(lps []*antv1.LivePosition) ([]sdk.Position, error) {
	positions := make([]sdk.Position, 0, len(lps))
	for i, lp := range lps {
		side, err := vmSideFromString(lp.Side)
		if err != nil {
			return nil, fmt.Errorf("positions[%d].side: %w", i, err)
		}
		vol, err := parseDecimalStrict(lp.Volume)
		if err != nil {
			return nil, fmt.Errorf("positions[%d].volume: %w", i, err)
		}
		if vol.IsNegative() {
			return nil, fmt.Errorf("positions[%d].volume: negative value %s", i, vol)
		}
		openPrice, err := parseDecimalStrict(lp.OpenPrice)
		if err != nil {
			return nil, fmt.Errorf("positions[%d].open_price: %w", i, err)
		}
		if openPrice.IsNegative() {
			return nil, fmt.Errorf("positions[%d].open_price: negative value %s", i, openPrice)
		}
		sl, err := parseDecimalStrict(lp.Sl)
		if err != nil {
			return nil, fmt.Errorf("positions[%d].sl: %w", i, err)
		}
		tp, err := parseDecimalStrict(lp.Tp)
		if err != nil {
			return nil, fmt.Errorf("positions[%d].tp: %w", i, err)
		}
		profit, err := parseDecimalStrict(lp.Profit)
		if err != nil {
			return nil, fmt.Errorf("positions[%d].profit: %w", i, err)
		}
		swap, err := parseDecimalStrict(lp.Swap)
		if err != nil {
			return nil, fmt.Errorf("positions[%d].swap: %w", i, err)
		}
		commission, err := parseDecimalStrict(lp.Commission)
		if err != nil {
			return nil, fmt.Errorf("positions[%d].commission: %w", i, err)
		}
		positions = append(positions, sdk.Position{
			Ticket:     lp.Ticket,
			Symbol:     lp.Symbol,
			Side:       side,
			Volume:     vol,
			OpenPrice:  openPrice,
			StopLoss:   sl,
			TakeProfit: tp,
			Profit:     profit,
			Swap:       swap,
			Commission: commission,
			Comment:    lp.Comment,
			Magic:      lp.MagicNumber,
			OpenTime:   time.Unix(lp.OpenTime, 0),
		})
	}
	return positions, nil
}

// vmPendingOrdersToSdk converts LivePendingOrder protos to SDK PendingOrders.
// LIVE-MQL-ORDER-CONTEXT-1: pending orders (limit/stop) are separate from
// market positions so MQL OrdersTotal/OrderSelect can distinguish them.
// VM-TRADE-CONTEXT-6: strict parsing — invalid decimal fails closed.
func vmPendingOrdersToSdk(lpos []*antv1.LivePendingOrder) ([]sdk.PendingOrder, error) {
	orders := make([]sdk.PendingOrder, 0, len(lpos))
	for i, lo := range lpos {
		side, err := vmSideFromString(lo.Side)
		if err != nil {
			return nil, fmt.Errorf("pending_orders[%d].side: %w", i, err)
		}
		ot, err := vmPendingOrderType(lo.OrderType)
		if err != nil {
			return nil, fmt.Errorf("pending_orders[%d].order_type: %w", i, err)
		}
		vol, err := parseDecimalStrict(lo.Volume)
		if err != nil {
			return nil, fmt.Errorf("pending_orders[%d].volume: %w", i, err)
		}
		if vol.IsNegative() {
			return nil, fmt.Errorf("pending_orders[%d].volume: negative value %s", i, vol)
		}
		price, err := parseDecimalStrict(lo.Price)
		if err != nil {
			return nil, fmt.Errorf("pending_orders[%d].price: %w", i, err)
		}
		if price.IsNegative() {
			return nil, fmt.Errorf("pending_orders[%d].price: negative value %s", i, price)
		}
		sl, err := parseDecimalStrict(lo.Sl)
		if err != nil {
			return nil, fmt.Errorf("pending_orders[%d].sl: %w", i, err)
		}
		tp, err := parseDecimalStrict(lo.Tp)
		if err != nil {
			return nil, fmt.Errorf("pending_orders[%d].tp: %w", i, err)
		}
		orders = append(orders, sdk.PendingOrder{
			Ticket:     lo.Ticket,
			Symbol:     lo.Symbol,
			Type:       ot,
			Side:       side,
			Volume:     vol,
			Price:      price,
			StopLoss:   sl,
			TakeProfit: tp,
			Comment:    lo.Comment,
			Magic:      lo.MagicNumber,
			OpenTime:   time.Unix(lo.OpenTime, 0),
		})
	}
	return orders, nil
}

// vmPendingOrderType converts a pending order type string to sdk.OrderType.
// VM-TRADE-CONTEXT-6 round 4: unknown order type must fail-closed, not default
// to OrderMarket (which would silently change trade semantics).
func vmPendingOrderType(s string) (sdk.OrderType, error) {
	switch s {
	case "buy_limit", "sell_limit":
		return sdk.OrderLimit, nil
	case "buy_stop", "sell_stop":
		return sdk.OrderStop, nil
	case "buy_stop_limit", "sell_stop_limit":
		return sdk.OrderStopLimit, nil
	case "buy_market", "sell_market", "market":
		return sdk.OrderMarket, nil
	case "":
		return 0, fmt.Errorf("empty order type")
	default:
		return 0, fmt.Errorf("unknown order type %q", s)
	}
}

// vmTradeEventType converts a trade event type string to sdk.TradeEventType.
// VM-TRADE-CONTEXT-6 round 4: unknown event type must fail-closed, not default
// to TradeFilled (which would silently change trade semantics).
func vmTradeEventType(s string) (sdk.TradeEventType, error) {
	switch s {
	case "fill":
		return sdk.TradeFilled, nil
	case "close":
		return sdk.TradeClosed, nil
	case "modify":
		return sdk.TradeModified, nil
	case "cancel":
		return sdk.TradeCancelled, nil
	case "":
		return 0, fmt.Errorf("empty trade event type")
	default:
		return 0, fmt.Errorf("unknown trade event type %q", s)
	}
}

// vmSideFromString converts a side string to sdk.Side.
// VM-TRADE-CONTEXT-6 round 4: unknown side must fail-closed, not default to buy.
func vmSideFromString(s string) (sdk.PositionSide, error) {
	switch s {
	case "buy":
		return sdk.SideBuy, nil
	case "sell":
		return sdk.SideSell, nil
	case "":
		return 0, fmt.Errorf("empty side")
	default:
		return 0, fmt.Errorf("unknown side %q", s)
	}
}

func vmSignalResponse(sig *sdk.Signal, symbol string) *antv1.ExecuteLiveResponse {
	resp := &antv1.ExecuteLiveResponse{Success: true}
	if sig != nil {
		ss := vmSignalToProto(sig, symbol)
		resp.Signal = ss
		resp.Signals = []*antv1.StrategySignal{ss}
	}
	return resp
}

func vmSignalToProto(sig *sdk.Signal, symbol string) *antv1.StrategySignal {
	if sig == nil {
		return nil
	}
	signalType := "hold"
	switch sig.Action {
	case sdk.ActionBuy:
		signalType = "buy"
	case sdk.ActionSell:
		signalType = "sell"
	case sdk.ActionBuyLimit:
		signalType = "buy_limit"
	case sdk.ActionSellLimit:
		signalType = "sell_limit"
	case sdk.ActionBuyStop:
		signalType = "buy_stop"
	case sdk.ActionSellStop:
		signalType = "sell_stop"
	case sdk.ActionClose:
		signalType = "close"
	case sdk.ActionModify:
		signalType = "modify"
	case sdk.ActionCancel:
		signalType = "cancel"
	case sdk.ActionCloseAll:
		signalType = "close_all"
	case sdk.ActionCancelAll:
		signalType = "cancel_all"
	}
	sym := sig.Symbol
	if sym == "" {
		sym = symbol
	}
	return &antv1.StrategySignal{
		Symbol:         sym,
		SignalType:     signalType,
		Volume:         sig.Volume.String(),
		Price:          sig.Price.String(),
		StopLoss:       sig.StopLoss.String(),
		TakeProfit:     sig.TakeProfit.String(),
		ExecutedTicket: sig.OrderTicket,
		Magic:          sig.Magic,          // VM-TRADE-CONTEXT-4
		Deviation:      sig.Deviation,      // VM-TRADE-CONTEXT-4
		OppositeTicket: sig.OppositeTicket, // VM-TRADE-CONTEXT-3
	}
}
