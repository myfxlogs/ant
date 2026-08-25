package strategy

import (
	"context"
	"fmt"
	"strconv"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/runner"
	"alphaforge/strategy/sdk"
)

// vmHandleBar processes a bar event.
func vmHandleBar(ctx context.Context, r *runner.Runner, lctx *antv1.LiveStrategyContext) *antv1.ExecuteLiveResponse {
	if lctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "bar_context missing"}
	}
	// VM-TRADE-CONTEXT-6: validate OHLCV array lengths before indexing.
	// Mismatched lengths would cause out-of-bounds panic.
	n := len(lctx.Close)
	if len(lctx.Open) != n || len(lctx.High) != n || len(lctx.Low) != n ||
		len(lctx.Volume) != n || len(lctx.BarTimesMs) != n {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf(
			"bar_context: OHLCV array length mismatch: close=%d open=%d high=%d low=%d volume=%d times=%d",
			n, len(lctx.Open), len(lctx.High), len(lctx.Low), len(lctx.Volume), len(lctx.BarTimesMs))}
	}
	// VM-TRADE-CONTEXT-6: reject nil repeated messages (symbol/position/order).
	for i, lp := range lctx.Positions {
		if lp == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf(
				"bar_context: positions[%d] is nil", i)}
		}
	}
	for i, lo := range lctx.PendingOrders {
		if lo == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf(
				"bar_context: pending_orders[%d] is nil", i)}
		}
	}
	for i, ss := range lctx.Symbols {
		if ss == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf(
				"bar_context: symbols[%d] is nil", i)}
		}
	}
	// VM-TRADE-CONTEXT-6: strict parsing of positions/orders — fail closed.
	// All validation/parsing happens BEFORE any runner method call so that
	// a bad request cannot partially mutate runner state then fail.
	positions, orders, perr := vmLiveStateToSdk(lctx.Positions, lctx.PendingOrders)
	if perr != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("bar_context: %v", perr)}
	}
	// VM-TRADE-CONTEXT-6: strict-parse all OHLCV bars before touching runner.
	barWindow := make([]sdk.Bar, n)
	for i := 0; i < n; i++ {
		open, err := parseDecimalStrict(lctx.Open[i])
		if err != nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("bar[%d].open: %v", i, err)}
		}
		high, err := parseDecimalStrict(lctx.High[i])
		if err != nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("bar[%d].high: %v", i, err)}
		}
		low, err := parseDecimalStrict(lctx.Low[i])
		if err != nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("bar[%d].low: %v", i, err)}
		}
		closeVal, err := parseDecimalStrict(lctx.Close[i])
		if err != nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("bar[%d].close: %v", i, err)}
		}
		vol, err := parseInt64Strict(lctx.Volume[i])
		if err != nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("bar[%d].volume: %v", i, err)}
		}
		barWindow[i] = sdk.Bar{
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closeVal,
			Volume:    vol,
			Timestamp: lctx.BarTimesMs[i],
		}
	}
	// VM-TRADE-CONTEXT-6: strict-parse multi-symbol series before touching runner.
	var extra map[string][]sdk.Bar
	if len(lctx.Symbols) > 0 {
		extra = make(map[string][]sdk.Bar, len(lctx.Symbols))
		for _, ss := range lctx.Symbols {
			sn := len(ss.Close)
			if sn == 0 {
				continue
			}
			if len(ss.Open) != sn || len(ss.High) != sn || len(ss.Low) != sn || len(ss.Volume) != sn {
				return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf(
					"symbol %s: OHLCV array length mismatch: close=%d open=%d high=%d low=%d volume=%d",
					ss.Symbol, sn, len(ss.Open), len(ss.High), len(ss.Low), len(ss.Volume))}
			}
			bars := make([]sdk.Bar, sn)
			for i := 0; i < sn; i++ {
				so, err := parseDecimalStrict(ss.Open[i])
				if err != nil {
					return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("symbol %s bar[%d].open: %v", ss.Symbol, i, err)}
				}
				sh, err := parseDecimalStrict(ss.High[i])
				if err != nil {
					return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("symbol %s bar[%d].high: %v", ss.Symbol, i, err)}
				}
				sl, err := parseDecimalStrict(ss.Low[i])
				if err != nil {
					return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("symbol %s bar[%d].low: %v", ss.Symbol, i, err)}
				}
				sc, err := parseDecimalStrict(ss.Close[i])
				if err != nil {
					return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("symbol %s bar[%d].close: %v", ss.Symbol, i, err)}
				}
				sv, err := parseInt64Strict(ss.Volume[i])
				if err != nil {
					return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("symbol %s bar[%d].volume: %v", ss.Symbol, i, err)}
				}
				bars[i] = sdk.Bar{Open: so, High: sh, Low: sl, Close: sc, Volume: sv}
			}
			extra[ss.Symbol] = bars
		}
	}
	// VM-TRADE-CONTEXT-6 round 5: mode-aware financial validation.
	// Live mode requires non-empty authoritative financial fields;
	// paper/backtest allows empty (simulation may not have real account data).
	if err := validateFinancialFieldsForMode(lctx.Balance, lctx.Equity, lctx.Margin, lctx.FreeMargin, lctx.Mode); err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("bar_context: %v", err)}
	}
	// All validation/parsing complete — now update runner state.
	r.UpdateLiveState(lctx.Balance, lctx.Equity, lctx.Margin, lctx.FreeMargin, positions, orders)
	r.UpdateSymbolInfo(lctx.Point, lctx.Digits, lctx.ContractSize, strconv.FormatInt(int64(lctx.StopsLevel), 10))
	r.UpdateAccountIdentity(lctx.Login, lctx.Company)                         // VM-TRADE-CONTEXT-3
	r.UpdateAccountStatus(lctx.IsDemo, lctx.IsConnected, lctx.IsTradeAllowed) // VM-API-TRUTH-3
	if extra != nil {
		r.UpdateExtraBars(extra)
	}

	barSeries := sdk.BarsToSlice(barWindow)
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
	// VM-TRADE-CONTEXT-6: reject nil repeated messages.
	for i, lp := range tctx.Positions {
		if lp == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("tick_context: positions[%d] is nil", i)}
		}
	}
	for i, lo := range tctx.PendingOrders {
		if lo == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("tick_context: pending_orders[%d] is nil", i)}
		}
	}
	// VM-TRADE-CONTEXT-6: strict parsing of positions/orders — fail closed.
	// All validation/parsing happens BEFORE any runner method call.
	positions, orders, perr := vmLiveStateToSdk(tctx.Positions, tctx.PendingOrders)
	if perr != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("tick_context: %v", perr)}
	}
	// VM-TRADE-CONTEXT-6: strict parsing — invalid bid/ask fails closed.
	bid, err := parseDecimalStrict(tctx.Bid)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("tick bid: %v", err)}
	}
	ask, err := parseDecimalStrict(tctx.Ask)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("tick ask: %v", err)}
	}
	// VM-TRADE-CONTEXT-6 round 5: mode-aware financial validation.
	if err := validateFinancialFieldsForMode(tctx.Balance, tctx.Equity, tctx.Margin, tctx.FreeMargin, tctx.Mode); err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("tick_context: %v", err)}
	}
	// All validation/parsing complete — now update runner state.
	r.UpdateLiveState(tctx.Balance, tctx.Equity, tctx.Margin, tctx.FreeMargin, positions, orders)
	r.UpdateSymbolInfo(tctx.Point, tctx.Digits, tctx.ContractSize, strconv.FormatInt(int64(tctx.StopsLevel), 10))
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
	// VM-TRADE-CONTEXT-6: reject nil repeated messages.
	for i, lp := range evctx.Positions {
		if lp == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade_context: positions[%d] is nil", i)}
		}
	}
	for i, lo := range evctx.PendingOrders {
		if lo == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade_context: pending_orders[%d] is nil", i)}
		}
	}
	// VM-TRADE-CONTEXT-6: strict parsing of positions/orders — fail closed.
	// All validation/parsing happens BEFORE any runner method call.
	positions, orders, perr := vmLiveStateToSdk(evctx.Positions, evctx.PendingOrders)
	if perr != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade_context: %v", perr)}
	}

	// VM-TRADE-CONTEXT-6 round 4: unknown side/event type must fail-closed.
	side, err := vmSideFromString(evctx.Side)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade side: %v", err)}
	}
	eventType, err := vmTradeEventType(evctx.EventType)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade event_type: %v", err)}
	}
	// VM-TRADE-CONTEXT-6: strict parsing — invalid trade event fields fail closed.
	vol, err := parseDecimalStrict(evctx.Volume)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade volume: %v", err)}
	}
	if vol.IsNegative() {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade volume: negative value %s", vol)}
	}
	price, err := parseDecimalStrict(evctx.Price)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade price: %v", err)}
	}
	if price.IsNegative() {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade price: negative value %s", price)}
	}
	sl, err := parseDecimalStrict(evctx.StopLoss)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade stop_loss: %v", err)}
	}
	tp, err := parseDecimalStrict(evctx.TakeProfit)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade take_profit: %v", err)}
	}
	profit, err := parseDecimalStrict(evctx.Profit)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade profit: %v", err)}
	}
	commission, err := parseDecimalStrict(evctx.Commission)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade commission: %v", err)}
	}
	swap, err := parseDecimalStrict(evctx.Swap)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade swap: %v", err)}
	}
	// VM-TRADE-CONTEXT-6 round 5: mode-aware financial validation.
	if err := validateFinancialFieldsForMode(evctx.Balance, evctx.Equity, evctx.Margin, evctx.FreeMargin, evctx.Mode); err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("trade_context: %v", err)}
	}
	// All validation/parsing complete — now update runner state.
	r.UpdateLiveState(evctx.Balance, evctx.Equity, evctx.Margin, evctx.FreeMargin, positions, orders)
	event := sdk.TradeEvent{
		Ticket:     evctx.Ticket,
		Symbol:     evctx.Symbol,
		EventType:  eventType,
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
			// VM-TRADE-CONTEXT-3: do NOT swallow OnTradeTransaction error —
			// return an error response instead of the previous success response.
			return &antv1.ExecuteLiveResponse{Success: false, Error: ttErr.Error()}
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
	// VM-TRADE-CONTEXT-6: reject nil repeated messages.
	for i, lp := range tmctx.Positions {
		if lp == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("timer_context: positions[%d] is nil", i)}
		}
	}
	for i, lo := range tmctx.PendingOrders {
		if lo == nil {
			return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("timer_context: pending_orders[%d] is nil", i)}
		}
	}
	// VM-TRADE-CONTEXT-6: strict parsing of positions/orders — fail closed.
	positions, orders, perr := vmLiveStateToSdk(tmctx.Positions, tmctx.PendingOrders)
	if perr != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("timer_context: %v", perr)}
	}
	// VM-TRADE-CONTEXT-6 round 5: mode-aware financial validation.
	if err := validateFinancialFieldsForMode(tmctx.Balance, tmctx.Equity, tmctx.Margin, tmctx.FreeMargin, tmctx.Mode); err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("timer_context: %v", err)}
	}
	r.UpdateLiveState(tmctx.Balance, tmctx.Equity, tmctx.Margin, tmctx.FreeMargin, positions, orders)
	sig, err := r.OnTimerTick(ctx)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
	}
	return vmSignalResponse(sig, tmctx.Symbol)
}
