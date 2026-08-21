package strategy

import (
	"context"
	"strconv"
	"time"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/runner"
	"alphaforge/strategy/sdk"
)

// vmHandleBar processes a bar event.
func vmHandleBar(ctx context.Context, r *runner.Runner, lctx *antv1.LiveStrategyContext) *antv1.ExecuteLiveResponse {
	if lctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "bar_context missing"}
	}
	r.UpdateLiveState(lctx.Balance, lctx.Equity, lctx.Margin, lctx.FreeMargin, vmPositionsToSdk(lctx.Positions), vmPendingOrdersToSdk(lctx.PendingOrders))
	r.UpdateSymbolInfo(lctx.Point, lctx.Digits, lctx.ContractSize, strconv.FormatInt(int64(lctx.StopsLevel), 10))

	n := len(lctx.Close)
	barWindow := make([]sdk.Bar, n)
	for i := 0; i < n; i++ {
		barWindow[i] = sdk.Bar{
			Open:      parseDecimal(lctx.Open[i]),
			High:      parseDecimal(lctx.High[i]),
			Low:       parseDecimal(lctx.Low[i]),
			Close:     parseDecimal(lctx.Close[i]),
			Volume:    parseInt64(lctx.Volume[i]),
			Timestamp: lctx.BarTimesMs[i],
		}
	}

	barSeries := sdk.BarsToSlice(barWindow)

	// Convert multi-symbol series from the live context.
	if len(lctx.Symbols) > 0 {
		extra := make(map[string][]sdk.Bar, len(lctx.Symbols))
		for _, ss := range lctx.Symbols {
			n := len(ss.Close)
			if n == 0 {
				continue
			}
			bars := make([]sdk.Bar, n)
			for i := 0; i < n; i++ {
				bars[i] = sdk.Bar{
					Open:   parseDecimal(ss.Open[i]),
					High:   parseDecimal(ss.High[i]),
					Low:    parseDecimal(ss.Low[i]),
					Close:  parseDecimal(ss.Close[i]),
					Volume: parseInt64(ss.Volume[i]),
				}
			}
			extra[ss.Symbol] = bars
		}
		r.UpdateExtraBars(extra)
	}

	sig, err := r.OnBar(ctx, barSeries, lctx.Timeframe)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
	}
	return vmSignalResponse(sig, lctx.Symbol)
}

// vmHandleTick processes a tick event.
func vmHandleTick(ctx context.Context, r *runner.Runner, tctx *antv1.TickContext) *antv1.ExecuteLiveResponse {
	if tctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "tick_context missing"}
	}
	r.UpdateLiveState(tctx.Balance, tctx.Equity, tctx.Margin, tctx.FreeMargin, vmPositionsToSdk(tctx.Positions), vmPendingOrdersToSdk(tctx.PendingOrders))
	r.UpdateSymbolInfo(tctx.Point, tctx.Digits, tctx.ContractSize, strconv.FormatInt(int64(tctx.StopsLevel), 10))
	bid := parseDecimal(tctx.Bid)
	ask := parseDecimal(tctx.Ask)
	r.UpdateTickState(bid, ask)
	sig, err := r.OnTick(ctx, bid, ask)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
	}
	return vmSignalResponse(sig, tctx.Symbol)
}

// vmHandleTrade processes a trade event.
// If the strategy also implements OnTradeTransaction (MQL5), it is dispatched
// immediately after OnTrade, and signals from both are combined.
func vmHandleTrade(ctx context.Context, r *runner.Runner, evctx *antv1.TradeContext) *antv1.ExecuteLiveResponse {
	if evctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "trade_context missing"}
	}
	r.UpdateLiveState(evctx.Balance, evctx.Equity, evctx.Margin, evctx.FreeMargin, vmPositionsToSdk(evctx.Positions), vmPendingOrdersToSdk(evctx.PendingOrders))

	side := sdk.SideBuy
	if evctx.Side == "sell" {
		side = sdk.SideSell
	}
	event := sdk.TradeEvent{
		Ticket:     evctx.Ticket,
		Symbol:     evctx.Symbol,
		EventType:  vmTradeEventType(evctx.EventType),
		Side:       side,
		Volume:     parseDecimal(evctx.Volume),
		Price:      parseDecimal(evctx.Price),
		StopLoss:   parseDecimal(evctx.StopLoss),
		TakeProfit: parseDecimal(evctx.TakeProfit),
		Profit:     parseDecimal(evctx.Profit),
		Commission: parseDecimal(evctx.Commission),
		Swap:       parseDecimal(evctx.Swap),
	}
	sig, err := r.OnTrade(ctx, event)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
	}
	resp := vmSignalResponse(sig, evctx.Symbol)

	// Dispatch OnTradeTransaction (MQL5) if the strategy implements it.
	if r.HasOnTradeTransaction() {
		ttSig, ttErr := r.OnTradeTransaction(ctx)
		if ttErr != nil {
			return resp
		}
		if ttSig != nil {
			ttSignalProto := vmSignalToProto(ttSig, evctx.Symbol)
			if ttSignalProto != nil {
				resp.Signals = append(resp.Signals, ttSignalProto)
			}
		}
	}

	return resp
}

// vmHandleTimer processes a timer event.
func vmHandleTimer(ctx context.Context, r *runner.Runner, tmctx *antv1.TimerContext) *antv1.ExecuteLiveResponse {
	if tmctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "timer_context missing"}
	}
	r.UpdateLiveState(tmctx.Balance, tmctx.Equity, tmctx.Margin, tmctx.FreeMargin, vmPositionsToSdk(tmctx.Positions), vmPendingOrdersToSdk(tmctx.PendingOrders))
	sig, err := r.OnTimerTick(ctx)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
	}
	return vmSignalResponse(sig, tmctx.Symbol)
}

// vmPositionsToSdk converts LivePosition protos to SDK Positions.
// LIVE-MQL-ORDER-CONTEXT-1: now preserves ALL fields (symbol, magic, sl, tp,
// profit, swap, commission, comment, open_time) so MQL OrderSelect/OrderMagicNumber/
// OrderSymbol/OrderStopLoss/OrderTakeProfit/OrderComment return broker-original values.
func vmPositionsToSdk(lps []*antv1.LivePosition) []sdk.Position {
	positions := make([]sdk.Position, 0, len(lps))
	for _, lp := range lps {
		side := sdk.SideBuy
		if lp.Side == "sell" {
			side = sdk.SideSell
		}
		positions = append(positions, sdk.Position{
			Ticket:     lp.Ticket,
			Symbol:     lp.Symbol,
			Side:       side,
			Volume:     parseDecimal(lp.Volume),
			OpenPrice:  parseDecimal(lp.OpenPrice),
			StopLoss:   parseDecimal(lp.Sl),
			TakeProfit: parseDecimal(lp.Tp),
			Profit:     parseDecimal(lp.Profit),
			Swap:       parseDecimal(lp.Swap),
			Commission: parseDecimal(lp.Commission),
			Comment:    lp.Comment,
			Magic:      lp.MagicNumber,
			OpenTime:   time.Unix(lp.OpenTime, 0),
		})
	}
	return positions
}

// vmPendingOrdersToSdk converts LivePendingOrder protos to SDK PendingOrders.
// LIVE-MQL-ORDER-CONTEXT-1: pending orders (limit/stop) are separate from
// market positions so MQL OrdersTotal/OrderSelect can distinguish them.
func vmPendingOrdersToSdk(lpos []*antv1.LivePendingOrder) []sdk.PendingOrder {
	orders := make([]sdk.PendingOrder, 0, len(lpos))
	for _, lo := range lpos {
		side := sdk.SideBuy
		if lo.Side == "sell" {
			side = sdk.SideSell
		}
		orders = append(orders, sdk.PendingOrder{
			Ticket:     lo.Ticket,
			Symbol:     lo.Symbol,
			Type:       vmPendingOrderType(lo.OrderType),
			Side:       side,
			Volume:     parseDecimal(lo.Volume),
			Price:      parseDecimal(lo.Price),
			StopLoss:   parseDecimal(lo.Sl),
			TakeProfit: parseDecimal(lo.Tp),
			Comment:    lo.Comment,
			Magic:      lo.MagicNumber,
			OpenTime:   time.Unix(lo.OpenTime, 0),
		})
	}
	return orders
}

// vmPendingOrderType converts a pending order type string to sdk.OrderType.
func vmPendingOrderType(s string) sdk.OrderType {
	switch s {
	case "buy_limit", "sell_limit":
		return sdk.OrderLimit
	case "buy_stop", "sell_stop":
		return sdk.OrderStop
	case "buy_stop_limit", "sell_stop_limit":
		return sdk.OrderStopLimit
	default:
		return sdk.OrderMarket
	}
}

func vmTradeEventType(s string) sdk.TradeEventType {
	switch s {
	case "fill":
		return sdk.TradeFilled
	case "close":
		return sdk.TradeClosed
	case "modify":
		return sdk.TradeModified
	case "cancel":
		return sdk.TradeCancelled
	}
	return sdk.TradeFilled
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
	}
}
