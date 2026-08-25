package mql2go

import (
	"fmt"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// ── Account builtins ─────────────────────────────────────────────────

func builtinAccountBalance(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(vm.ctx.Account().Balance), nil
}

func builtinAccountEquity(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(vm.ctx.Account().Equity), nil
}

func builtinAccountFreeMargin(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(vm.ctx.Account().FreeMargin), nil
}

func builtinAccountMargin(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(vm.ctx.Account().Margin), nil
}

func builtinAccountLeverage(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(vm.ctx.Account().Leverage), nil
}

func builtinAccountCurrency(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.StringVal(""), nil
	}
	return interp.StringVal(vm.ctx.Account().Currency), nil
}

// builtinAccountCompany returns the broker company name from the
// authoritative account context. VM-API-TRUTH-2: previously this was a
// noop returning "" — a silent blind spot. Now reads from Account().Company.
func builtinAccountCompany(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.StringVal(""), nil
	}
	return interp.StringVal(vm.ctx.Account().Company), nil
}

// ── Symbol info builtins ─────────────────────────────────────────────

func builtinSymbolInfoDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	sym := argS(args, 0)
	if sym == "" {
		sym = vm.ctx.Symbol()
	}
	info, err := vm.ctx.Broker().SymbolInfo(sym)
	if err != nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	prop := argI(args, 1)
	switch prop {
	case 0: // SYMBOL_POINT
		return interp.DecimalVal(info.Point), nil
	case 1: // SYMBOL_VOLUME_MIN
		return interp.DecimalVal(info.VolumeMin), nil
	case 2: // SYMBOL_VOLUME_MAX
		return interp.DecimalVal(info.VolumeMax), nil
	case 3: // SYMBOL_VOLUME_STEP
		return interp.DecimalVal(info.VolumeStep), nil
	case 4: // SYMBOL_TICK_VALUE
		return interp.DecimalVal(info.TickValue), nil
	case 5: // SYMBOL_TICK_SIZE
		return interp.DecimalVal(info.TickSize), nil
	case 6: // SYMBOL_SWAP_LONG
		return interp.DecimalVal(info.SwapLong), nil
	case 7: // SYMBOL_SWAP_SHORT
		return interp.DecimalVal(info.SwapShort), nil
	default:
		return interp.DecimalVal(decimal.Zero), nil
	}
}

func builtinSymbolInfoInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.IntVal(0), nil
	}
	sym := argS(args, 0)
	if sym == "" {
		sym = vm.ctx.Symbol()
	}
	info, err := vm.ctx.Broker().SymbolInfo(sym)
	if err != nil {
		return interp.IntVal(0), nil
	}
	prop := argI(args, 1)
	switch prop {
	case 0: // SYMBOL_DIGITS
		return interp.IntVal(info.Digits), nil
	case 1: // SYMBOL_SPREAD
		return interp.IntVal(info.Spread), nil
	case 2: // SYMBOL_TRADE_STOPS_LEVEL
		return interp.IntVal(info.StopsLevel), nil
	default:
		return interp.IntVal(0), nil
	}
}

func builtinSymbolInfoString(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.StringVal(""), nil
	}
	sym := argS(args, 0)
	if sym == "" {
		sym = vm.ctx.Symbol()
	}
	info, err := vm.ctx.Broker().SymbolInfo(sym)
	if err != nil {
		return interp.StringVal(""), nil
	}
	return interp.StringVal(info.Name), nil
}

func builtinMarketInfo(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil || vm.ctx.Broker() == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	sym := argS(args, 0)
	if sym == "" {
		sym = vm.ctx.Symbol()
	}
	info, err := vm.ctx.Broker().SymbolInfo(sym)
	if err != nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	mode := argI(args, 1)
	switch mode {
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
	case 17: // MODE_TICKVALUE
		return interp.DecimalVal(info.TickValue), nil
	case 18: // MODE_TICKSIZE
		return interp.DecimalVal(info.TickSize), nil
	case 20: // MODE_MINLOT
		return interp.DecimalVal(info.VolumeMin), nil
	case 21: // MODE_MAXLOT
		return interp.DecimalVal(info.VolumeMax), nil
	case 22: // MODE_LOTSTEP
		return interp.DecimalVal(info.VolumeStep), nil
	default:
		return interp.DecimalVal(decimal.Zero), nil
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
		return interp.IntVal(int32(len(args[0].ArrayData()))), nil
	}
	return interp.IntVal(0), nil
}
