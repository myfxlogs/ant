package strategy

import (
	"context"
	"fmt"
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

	// VM-TRADE-CONTEXT-6 S2: OHLCV array length validation — mismatched
	// lengths cause index-out-of-range panics. Fail-closed with explicit error.
	if err := validateOHLCVLengths(lctx.Open, lctx.High, lctx.Low, lctx.Close, lctx.Volume, lctx.BarTimesMs); err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "OHLCV array length mismatch: " + err.Error()}
	}

	// VM-TRADE-CONTEXT-6 S4: nil repeated message rejection in live mode.
	if resp := rejectNilRepeatedInLive(lctx.Mode, lctx.Positions, lctx.PendingOrders); resp != nil {
		return resp
	}

	r.UpdateLiveState(lctx.Balance, lctx.Equity, lctx.Margin, lctx.FreeMargin, vmPositionsToSdk(lctx.Positions), vmPendingOrdersToSdk(lctx.PendingOrders))
	// VM-TRADE-CONTEXT-6 S6: propagate authoritative Login to VM's AccountNumber().
	r.SetLogin(lctx.Login)
	// VM-API-TRUTH-3: propagate authoritative account status to VM builtins.
	r.SetAccountStatus(lctx.IsDemo, lctx.IsConnected, lctx.IsTradeAllowed)
	r.UpdateSymbolInfo(lctx.Point, lctx.Digits, lctx.ContractSize, strconv.FormatInt(int64(lctx.StopsLevel), 10))

	// VM-TRADE-CONTEXT-6 S3: strict parse bars — invalid decimals fail-closed.
	barWindow, resp := parseBarsStrict(lctx.Open, lctx.High, lctx.Low, lctx.Close, lctx.Volume, lctx.BarTimesMs, "")
	if resp != nil {
		return resp
	}
	barSeries := sdk.BarsToSlice(barWindow)

	// Convert multi-symbol series from the live context.
	if len(lctx.Symbols) > 0 {
		extra, errResp := parseExtraSymbolsStrict(lctx.Symbols)
		if errResp != nil {
			return errResp
		}
		r.UpdateExtraBars(extra)
	}

	sig, err := r.OnBar(ctx, barSeries, lctx.Timeframe)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
	}
	return vmSignalResponse(sig, lctx.Symbol)
}

// rejectNilRepeatedInLive returns an error response if mode is live and
// positions or pendingOrders are nil (data missing). Returns nil if OK.
// VM-TRADE-CONTEXT-6 S4.
func rejectNilRepeatedInLive(mode string, positions []*antv1.LivePosition, pendingOrders []*antv1.LivePendingOrder) *antv1.ExecuteLiveResponse {
	if mode != modeLive {
		return nil
	}
	if positions == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "live mode requires positions (nil = data missing)"}
	}
	if pendingOrders == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "live mode requires pending_orders (nil = data missing)"}
	}
	return nil
}

// parseBarsStrict parses OHLCV arrays into sdk.Bar slice using strict parsers.
// Returns (bars, nil) on success or (nil, errorResponse) on failure.
// symbolPrefix is "" for primary symbol or "symbol <name> " for extra symbols.
// VM-TRADE-CONTEXT-6 S3.
func parseBarsStrict(open, high, low, close, volume []string, barTimesMs []int64, symbolPrefix string) ([]sdk.Bar, *antv1.ExecuteLiveResponse) {
	n := len(close)
	bars := make([]sdk.Bar, n)
	for i := 0; i < n; i++ {
		o, err := parseDecimalStrict(open[i])
		if err != nil {
			return nil, &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in " + symbolPrefix + "Open[" + strconv.Itoa(i) + "]: " + err.Error()}
		}
		h, err := parseDecimalStrict(high[i])
		if err != nil {
			return nil, &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in " + symbolPrefix + "High[" + strconv.Itoa(i) + "]: " + err.Error()}
		}
		l, err := parseDecimalStrict(low[i])
		if err != nil {
			return nil, &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in " + symbolPrefix + "Low[" + strconv.Itoa(i) + "]: " + err.Error()}
		}
		c, err := parseDecimalStrict(close[i])
		if err != nil {
			return nil, &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in " + symbolPrefix + "Close[" + strconv.Itoa(i) + "]: " + err.Error()}
		}
		v, err := parseInt64Strict(volume[i])
		if err != nil {
			return nil, &antv1.ExecuteLiveResponse{Success: false, Error: "invalid int in " + symbolPrefix + "Volume[" + strconv.Itoa(i) + "]: " + err.Error()}
		}
		bars[i] = sdk.Bar{Open: o, High: h, Low: l, Close: c, Volume: v}
		if barTimesMs != nil {
			bars[i].Timestamp = barTimesMs[i]
		}
	}
	return bars, nil
}

// parseExtraSymbolsStrict parses multi-symbol OHLCV series with strict validation.
// Returns (extraBars, nil) on success or (nil, errorResponse) on failure.
// VM-TRADE-CONTEXT-6 S2+S3.
func parseExtraSymbolsStrict(symbols []*antv1.LiveSymbolSeries) (map[string][]sdk.Bar, *antv1.ExecuteLiveResponse) {
	extra := make(map[string][]sdk.Bar, len(symbols))
	for _, ss := range symbols {
		if err := validateOHLCVLengths(ss.Open, ss.High, ss.Low, ss.Close, ss.Volume, nil); err != nil {
			return nil, &antv1.ExecuteLiveResponse{Success: false, Error: "OHLCV array length mismatch for symbol " + ss.Symbol + ": " + err.Error()}
		}
		if len(ss.Close) == 0 {
			continue
		}
		bars, errResp := parseBarsStrict(ss.Open, ss.High, ss.Low, ss.Close, ss.Volume, nil, "symbol "+ss.Symbol+" ")
		if errResp != nil {
			return nil, errResp
		}
		extra[ss.Symbol] = bars
	}
	return extra, nil
}

// vmHandleTick processes a tick event.
func vmHandleTick(ctx context.Context, r *runner.Runner, tctx *antv1.TickContext) *antv1.ExecuteLiveResponse {
	if tctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "tick_context missing"}
	}
	// VM-TRADE-CONTEXT-6 S4: nil repeated message rejection in live mode.
	if tctx.Mode == modeLive {
		if tctx.Positions == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: "live mode requires positions (nil = data missing)"}
		}
		if tctx.PendingOrders == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: "live mode requires pending_orders (nil = data missing)"}
		}
	}
	r.UpdateLiveState(tctx.Balance, tctx.Equity, tctx.Margin, tctx.FreeMargin, vmPositionsToSdk(tctx.Positions), vmPendingOrdersToSdk(tctx.PendingOrders))
	r.UpdateSymbolInfo(tctx.Point, tctx.Digits, tctx.ContractSize, strconv.FormatInt(int64(tctx.StopsLevel), 10))
	// VM-TRADE-CONTEXT-6 S3: strict parse in live path.
	bid, err := parseDecimalStrict(tctx.Bid)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in Bid: " + err.Error()}
	}
	ask, err := parseDecimalStrict(tctx.Ask)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in Ask: " + err.Error()}
	}
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
	// VM-TRADE-CONTEXT-6 S4: nil repeated message rejection in live mode.
	if evctx.Mode == modeLive {
		if evctx.Positions == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: "live mode requires positions (nil = data missing)"}
		}
		if evctx.PendingOrders == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: "live mode requires pending_orders (nil = data missing)"}
		}
	}
	r.UpdateLiveState(evctx.Balance, evctx.Equity, evctx.Margin, evctx.FreeMargin, vmPositionsToSdk(evctx.Positions), vmPendingOrdersToSdk(evctx.PendingOrders))

	side := sdk.SideBuy
	if evctx.Side == "sell" {
		side = sdk.SideSell
	}
	// VM-TRADE-CONTEXT-6 S3: strict parse in live path.
	vol, err := parseDecimalStrict(evctx.Volume)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in Volume: " + err.Error()}
	}
	price, err := parseDecimalStrict(evctx.Price)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in Price: " + err.Error()}
	}
	sl, err := parseDecimalStrict(evctx.StopLoss)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in StopLoss: " + err.Error()}
	}
	tp, err := parseDecimalStrict(evctx.TakeProfit)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in TakeProfit: " + err.Error()}
	}
	profit, err := parseDecimalStrict(evctx.Profit)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in Profit: " + err.Error()}
	}
	commission, err := parseDecimalStrict(evctx.Commission)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in Commission: " + err.Error()}
	}
	swap, err := parseDecimalStrict(evctx.Swap)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "invalid decimal in Swap: " + err.Error()}
	}
	event := sdk.TradeEvent{
		Ticket:     evctx.Ticket,
		Symbol:     evctx.Symbol,
		EventType:  vmTradeEventType(evctx.EventType),
		Side:       side,
		Volume:     vol,
		Price:      price,
		StopLoss:   sl,
		TakeProfit: tp,
		Profit:     profit,
		Commission: commission,
		Swap:       swap,
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
	// VM-TRADE-CONTEXT-6 S4: nil repeated message rejection in live mode.
	if tmctx.Mode == modeLive {
		if tmctx.Positions == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: "live mode requires positions (nil = data missing)"}
		}
		if tmctx.PendingOrders == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: "live mode requires pending_orders (nil = data missing)"}
		}
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

// validateOHLCVLengths checks that all OHLCV arrays have the same length.
// barTimesMs may be nil (multi-symbol series don't carry timestamps).
// VM-TRADE-CONTEXT-6 S2: mismatched lengths cause index-out-of-range panics.
func validateOHLCVLengths(open, high, low, close, volume []string, barTimesMs []int64) error {
	n := len(close)
	if len(open) != n {
		return fmt.Errorf("open length %d != close length %d", len(open), n)
	}
	if len(high) != n {
		return fmt.Errorf("high length %d != close length %d", len(high), n)
	}
	if len(low) != n {
		return fmt.Errorf("low length %d != close length %d", len(low), n)
	}
	if len(volume) != n {
		return fmt.Errorf("volume length %d != close length %d", len(volume), n)
	}
	if barTimesMs != nil && len(barTimesMs) != n {
		return fmt.Errorf("barTimesMs length %d != close length %d", len(barTimesMs), n)
	}
	return nil
}
