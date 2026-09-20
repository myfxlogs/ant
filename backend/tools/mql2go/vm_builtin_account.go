package mql2go

import (
	"fmt"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// ── Account builtins ─────────────────────────────────────────────────

func builtinAccountBalance(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(vm.ctx.Account().Balance), nil
}

func builtinAccountEquity(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(vm.ctx.Account().Equity), nil
}

func builtinAccountFreeMargin(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(vm.ctx.Account().FreeMargin), nil
}

func builtinAccountMargin(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(vm.ctx.Account().Margin), nil
}

func builtinAccountLeverage(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(vm.ctx.Account().Leverage), nil
}

// VM-API-TRUTH-1 batch 2e: real implementations replacing noop stubs.

func builtinAccountProfit(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(vm.ctx.Account().Equity.Sub(vm.ctx.Account().Balance)), nil
}

func builtinAccountCurrency(vm *VM, args []interp.Value) (interp.Value, error) {
	c := vm.ctx.Account().Currency
	if c == "" {
		return interp.StringVal(""), fmt.Errorf("AccountCurrency: no authoritative currency in the VM")
	}
	return interp.StringVal(c), nil
}

func builtinAccountCompany(vm *VM, args []interp.Value) (interp.Value, error) {
	c := vm.ctx.Account().Company
	if c == "" {
		return interp.StringVal(""), fmt.Errorf("AccountCompany: no authoritative company in the VM")
	}
	return interp.StringVal(c), nil
}

func builtinAccountFreeMarginCheck(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx.Broker() == nil {
		return interp.DecimalVal(decimal.Zero), fmt.Errorf("AccountFreeMarginCheck: no broker in the VM")
	}
	sym := argS(args, 0)
	if sym == "" {
		sym = vm.ctx.Symbol()
	}
	cmd := argI(args, 1)
	volume := argD(args, 2) // out-of-range → decimal.Zero (volume=0 → required=0 → FreeMargin)
	info, err := vm.ctx.Broker().SymbolInfo(sym)
	if err != nil {
		return interp.DecimalVal(decimal.Zero), fmt.Errorf("AccountFreeMarginCheck: %w", err)
	}
	var price decimal.Decimal
	switch cmd {
	case 0: // OP_BUY
		price = vm.ctx.Ask()
	case 1: // OP_SELL
		price = vm.ctx.Bid()
	default:
		return interp.DecimalVal(decimal.Zero), fmt.Errorf("AccountFreeMarginCheck: invalid cmd %d (must be 0/1)", cmd)
	}
	lev := decimal.NewFromInt(int64(vm.ctx.Account().Leverage))
	if lev.IsZero() {
		return interp.DecimalVal(decimal.Zero), fmt.Errorf("AccountFreeMarginCheck: leverage is zero")
	}
	required := volume.Mul(info.ContractSize).Mul(price).Div(lev)
	return interp.DecimalVal(vm.ctx.Account().FreeMargin.Sub(required)), nil
}

// ── Symbol info builtins ─────────────────────────────────────────────

func builtinSymbolInfoDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx.Broker() == nil {
		return interp.DecimalVal(decimal.Zero), fmt.Errorf("SymbolInfoDouble: no broker in the VM")
	}
	sym := argS(args, 0)
	if sym == "" {
		sym = vm.ctx.Symbol()
	}
	info, err := vm.ctx.Broker().SymbolInfo(sym)
	if err != nil {
		return interp.DecimalVal(decimal.Zero), fmt.Errorf("SymbolInfoDouble: %w", err)
	}
	prop := argI(args, 1)
	// VM-ENUM-NUMBERING-1: prop numbers are the real MQL5
	// ENUM_SYMBOL_INFO_DOUBLE values (current doc order).
	switch prop {
	case 0: // SYMBOL_BID
		if sym != vm.ctx.Symbol() || vm.ctx.Bid().IsZero() {
			return interp.DecimalVal(decimal.Zero), fmt.Errorf("SymbolInfoDouble: no authoritative bid for %q in the VM", sym)
		}
		return interp.DecimalVal(vm.ctx.Bid()), nil
	case 3: // SYMBOL_ASK
		if sym != vm.ctx.Symbol() || vm.ctx.Ask().IsZero() {
			return interp.DecimalVal(decimal.Zero), fmt.Errorf("SymbolInfoDouble: no authoritative ask for %q in the VM", sym)
		}
		return interp.DecimalVal(vm.ctx.Ask()), nil
	case 6, 7, 8: // SYMBOL_LAST/LASTHIGH/LASTLOW — venue 0: forex has no last-deal price (real MT5 returns 0 too)
		return interp.DecimalVal(decimal.Zero), nil
	case 9, 10, 11: // SYMBOL_VOLUME_REAL/VOLUMEHIGH_REAL/VOLUMELOW_REAL — venue 0: no centralized volume in backtest
		return interp.DecimalVal(decimal.Zero), nil
	case 13: // SYMBOL_POINT
		return interp.DecimalVal(info.Point), nil
	case 14: // SYMBOL_TRADE_TICK_VALUE
		return interp.DecimalVal(info.TickValue), nil
	case 17: // SYMBOL_TRADE_TICK_SIZE
		return interp.DecimalVal(info.TickSize), nil
	case 18: // SYMBOL_TRADE_CONTRACT_SIZE
		return interp.DecimalVal(info.ContractSize), nil
	case 22: // SYMBOL_VOLUME_MIN
		return interp.DecimalVal(info.VolumeMin), nil
	case 23: // SYMBOL_VOLUME_MAX
		return interp.DecimalVal(info.VolumeMax), nil
	case 24: // SYMBOL_VOLUME_STEP
		return interp.DecimalVal(info.VolumeStep), nil
	case 26: // SYMBOL_SWAP_LONG
		return interp.DecimalVal(info.SwapLong), nil
	case 27: // SYMBOL_SWAP_SHORT
		return interp.DecimalVal(info.SwapShort), nil
	default:
		return interp.DecimalVal(decimal.Zero), fmt.Errorf("SymbolInfoDouble: unsupported prop %d", prop)
	}
}

func builtinSymbolInfoInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx.Broker() == nil {
		return interp.IntVal(0), fmt.Errorf("SymbolInfoInteger: no broker in the VM")
	}
	sym := argS(args, 0)
	if sym == "" {
		sym = vm.ctx.Symbol()
	}
	info, err := vm.ctx.Broker().SymbolInfo(sym)
	if err != nil {
		return interp.IntVal(0), fmt.Errorf("SymbolInfoInteger: %w", err)
	}
	prop := argI(args, 1)
	// VM-ENUM-NUMBERING-1: prop numbers are the real MQL5
	// ENUM_SYMBOL_INFO_INTEGER values (current doc order).
	switch prop {
	case 3: // SYMBOL_CUSTOM — venue 0: symbols come from the broker, not synthetic
		return interp.IntVal(0), nil
	case 5: // SYMBOL_CHART_MODE — venue SYMBOL_CHART_MODE_BID: backtest models on Bid
		return interp.IntVal(0), nil
	case 6: // SYMBOL_EXIST — reaching here means the broker resolved the symbol
		return interp.IntVal(1), nil
	case 7: // SYMBOL_SELECT — same venue semantics as SymbolSelect→true
		return interp.IntVal(1), nil
	case 8: // SYMBOL_VISIBLE — same
		return interp.IntVal(1), nil
	case 15: // SYMBOL_TIME
		if sym != vm.ctx.Symbol() || vm.ctx.ServerTime() == 0 {
			return interp.IntVal(0), fmt.Errorf("SymbolInfoInteger: no authoritative server time for %q in the VM", sym)
		}
		return interp.IntVal(int32(vm.ctx.ServerTime() / 1000)), nil
	case 17: // SYMBOL_DIGITS
		return interp.IntVal(info.Digits), nil
	case 18: // SYMBOL_SPREAD_FLOAT — venue 0: fixed-spread backtest model (live re-check via LIVE-ACCOUNT-FIELDS-1)
		return interp.IntVal(0), nil
	case 19: // SYMBOL_SPREAD
		return interp.IntVal(info.Spread), nil
	case 20: // SYMBOL_TICKS_BOOKDEPTH — venue 0: no DOM queue
		return interp.IntVal(0), nil
	case 21: // SYMBOL_TRADE_CALC_MODE — venue SYMBOL_CALC_MODE_FOREX
		return interp.IntVal(0), nil
	case 22: // SYMBOL_TRADE_MODE — venue fact verbatim (backtest model FULL; live: broker enum, -1 = unknown)
		return interp.IntVal(info.TradeMode), nil
	case 23, 24: // SYMBOL_START_TIME/SYMBOL_EXPIRATION_TIME — venue 0: perpetual symbols (real MT5 too for non-futures)
		return interp.IntVal(0), nil
	case 25: // SYMBOL_TRADE_STOPS_LEVEL
		return interp.IntVal(info.StopsLevel), nil
	case 26: // SYMBOL_TRADE_FREEZE_LEVEL — venue fact verbatim (backtest model 0; live: broker, -1 = unknown)
		return interp.IntVal(info.FreezeLevel), nil
	case 27: // SYMBOL_TRADE_EXEMODE — venue fact verbatim (backtest model MARKET; live: mt4 Exemode, -1 = unknown/mt5)
		return interp.IntVal(info.TradeExemode), nil
	case 28: // SYMBOL_SWAP_MODE — venue SYMBOL_SWAP_MODE_POINTS: swap modeled in points
		return interp.IntVal(1), nil
	case 33: // SYMBOL_ORDER_MODE — venue mask: market|limit|stop|stop-limit|SL|TP (OrderSend support)
		return interp.IntVal(63), nil
	case 34: // SYMBOL_ORDER_GTC_MODE — venue SYMBOL_ORDERS_GTC: no order-expiry model
		return interp.IntVal(0), nil
	default:
		return interp.IntVal(0), fmt.Errorf("SymbolInfoInteger: unsupported prop %d", prop)
	}
}

func builtinMarketInfo(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx.Broker() == nil {
		return interp.DecimalVal(decimal.Zero), fmt.Errorf("MarketInfo: no broker in the VM")
	}
	sym := argS(args, 0)
	if sym == "" {
		sym = vm.ctx.Symbol()
	}
	info, err := vm.ctx.Broker().SymbolInfo(sym)
	if err != nil {
		return interp.DecimalVal(decimal.Zero), fmt.Errorf("MarketInfo: %w", err)
	}
	mode := argI(args, 1)
	// VM-ENUM-NUMBERING-1: mode numbers are the real MQL4 MarketInfo values.
	switch mode {
	case 5: // MODE_TIME
		if sym != vm.ctx.Symbol() || vm.ctx.ServerTime() == 0 {
			return interp.DecimalVal(decimal.Zero), fmt.Errorf("MarketInfo: no authoritative server time for %q in the VM", sym)
		}
		return interp.DecimalVal(decimal.NewFromInt(vm.ctx.ServerTime() / 1000)), nil
	case 9: // MODE_BID
		if sym != vm.ctx.Symbol() || vm.ctx.Bid().IsZero() {
			return interp.DecimalVal(decimal.Zero), fmt.Errorf("MarketInfo: no authoritative bid for %q in the VM", sym)
		}
		return interp.DecimalVal(vm.ctx.Bid()), nil
	case 10: // MODE_ASK
		if sym != vm.ctx.Symbol() || vm.ctx.Ask().IsZero() {
			return interp.DecimalVal(decimal.Zero), fmt.Errorf("MarketInfo: no authoritative ask for %q in the VM", sym)
		}
		return interp.DecimalVal(vm.ctx.Ask()), nil
	case 11: // MODE_POINT
		return interp.DecimalVal(info.Point), nil
	case 12: // MODE_DIGITS
		return interp.DecimalVal(decimal.NewFromInt(int64(info.Digits))), nil
	case 13: // MODE_SPREAD
		return interp.DecimalVal(decimal.NewFromInt(int64(info.Spread))), nil
	case 14: // MODE_STOPLEVEL
		return interp.DecimalVal(decimal.NewFromInt(int64(info.StopsLevel))), nil
	case 15: // MODE_LOTSIZE
		return interp.DecimalVal(info.ContractSize), nil
	case 16: // MODE_TICKVALUE
		return interp.DecimalVal(info.TickValue), nil
	case 17: // MODE_TICKSIZE
		return interp.DecimalVal(info.TickSize), nil
	case 18: // MODE_SWAPLONG
		return interp.DecimalVal(info.SwapLong), nil
	case 19: // MODE_SWAPSHORT
		return interp.DecimalVal(info.SwapShort), nil
	case 20: // MODE_STARTING — venue 0: perpetual symbols
		return interp.DecimalVal(decimal.Zero), nil
	case 21: // MODE_EXPIRATION — venue 0
		return interp.DecimalVal(decimal.Zero), nil
	case 22: // MODE_TRADEALLOWED — VM-LIVE-VENUE-R2: symbol axis "may open"
		// ∧ account axis. long_only/short_only/full (1/2/4) may open;
		// disabled(0)/close_only(3)/unknown(-1) fail closed to 0. The account
		// axis stays mandatory (builtinIsTradeAllowed carries it alone).
		if vm.ctx.Account().IsTradeAllowed && (info.TradeMode == 1 || info.TradeMode == 2 || info.TradeMode == 4) {
			return interp.DecimalVal(decimal.NewFromInt(1)), nil
		}
		return interp.DecimalVal(decimal.Zero), nil
	case 23: // MODE_MINLOT
		return interp.DecimalVal(info.VolumeMin), nil
	case 24: // MODE_LOTSTEP
		return interp.DecimalVal(info.VolumeStep), nil
	case 25: // MODE_MAXLOT
		return interp.DecimalVal(info.VolumeMax), nil
	case 28, 31: // MODE_MARGININIT / MODE_MARGINREQUIRED — same initial=margin model as AccountFreeMarginCheck (volume=1)
		if sym != vm.ctx.Symbol() || vm.ctx.Ask().IsZero() {
			return interp.DecimalVal(decimal.Zero), fmt.Errorf("MarketInfo: no authoritative ask for %q in the VM", sym)
		}
		lev := decimal.NewFromInt(int64(vm.ctx.Account().Leverage))
		if lev.IsZero() {
			return interp.DecimalVal(decimal.Zero), fmt.Errorf("MarketInfo: leverage is zero")
		}
		return interp.DecimalVal(info.ContractSize.Mul(vm.ctx.Ask()).Div(lev)), nil
	case 32: // MODE_FREEZELEVEL — venue fact verbatim (backtest model 0; live: broker, -1 = unknown)
		return interp.DecimalVal(decimal.NewFromInt(int64(info.FreezeLevel))), nil
	case 33: // MODE_CLOSEBY_ALLOWED — venue 0: close-by not supported
		return interp.DecimalVal(decimal.Zero), nil
	default:
		return interp.DecimalVal(decimal.Zero), fmt.Errorf("MarketInfo: unsupported mode %d", mode)
	}
}

// ── String format builtin ────────────────────────────────────────────

func builtinStringFormat(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) == 0 {
		return interp.StringVal(""), nil
	}
	format := argS(args, 0)
	rest := make([]interface{}, len(args)-1)
	for i := 1; i < len(args); i++ {
		switch args[i].Kind {
		case interp.ValInt:
			rest[i-1] = args[i].Int
		case interp.ValDecimal:
			rest[i-1] = args[i].Decimal.InexactFloat64()
		case interp.ValString:
			rest[i-1] = args[i].Str
		case interp.ValBool:
			rest[i-1] = args[i].Bool
		default:
			rest[i-1] = args[i].ToString()
		}
	}
	return interp.StringVal(fmt.Sprintf(format, rest...)), nil
}

// ── Array builtins ───────────────────────────────────────────────────

func builtinArraySize(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) > 0 && args[0].Kind == interp.ValArray {
		return interp.IntVal(int32(len(args[0].Array))), nil
	}
	return interp.IntVal(0), nil
}
