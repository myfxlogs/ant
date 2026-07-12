package mql2go

import (
	"alphaforge/tools/mql2go/interp"
)

// MQL5 market info additions and account info functions.

func builtinSymbolInfoTick(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.BoolVal(false), nil
	}
	// SymbolInfoTick(symbol, tick) fills an MqlTick struct.
	// Return true with a struct containing bid/ask.
	fields := map[string]interp.Value{
		"time":  interp.IntVal(int32(vm.ctx.ServerTime() / 1000)),
		"bid":   interp.DecimalVal(vm.ctx.Bid()),
		"ask":   interp.DecimalVal(vm.ctx.Ask()),
		"last":  interp.DecimalVal(vm.ctx.Bid()),
		"volume": interp.IntVal(0),
	}
	return interp.Value{Kind: interp.ValClass, Class: &interp.ClassInstance{Fields: fields}}, nil
}

func builtinSymbolName(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.StringVal(""), nil
	}
	return interp.StringVal(vm.ctx.Symbol()), nil
}

func builtinSymbolSelect(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinSymbolsTotal(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(1), nil
}

func builtinSymbolInfoMarginRate(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinSymbolInfoSessionQuote(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinSymbolInfoSessionTrade(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinSymbolIsSynchronized(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

// MQL5 AccountInfo* functions
func builtinAccountInfoDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	prop := argI(args, 0)
	switch prop {
	case 0: // ACCOUNT_BALANCE
		return interp.DecimalVal(vm.ctx.Account().Balance), nil
	case 1: // ACCOUNT_CREDIT
		return interp.DecimalVal(decimalZero), nil
	case 2: // ACCOUNT_PROFIT
		return interp.DecimalVal(vm.ctx.Account().Equity.Sub(vm.ctx.Account().Balance)), nil
	case 3: // ACCOUNT_EQUITY
		return interp.DecimalVal(vm.ctx.Account().Equity), nil
	case 4: // ACCOUNT_MARGIN
		return interp.DecimalVal(vm.ctx.Account().Margin), nil
	case 5: // ACCOUNT_MARGIN_FREE
		return interp.DecimalVal(vm.ctx.Account().FreeMargin), nil
	case 6: // ACCOUNT_MARGIN_LEVEL
		if vm.ctx.Account().Margin.IsZero() {
			return interp.DecimalVal(decimalZero), nil
		}
		return interp.DecimalVal(vm.ctx.Account().Equity.Div(vm.ctx.Account().Margin)), nil
	default:
		return interp.DecimalVal(decimalZero), nil
	}
}

func builtinAccountInfoInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	prop := argI(args, 0)
	switch prop {
	case 32: // ACCOUNT_LEVERAGE
		if vm.ctx != nil {
			return interp.IntVal(vm.ctx.Account().Leverage), nil
		}
		return interp.IntVal(100), nil
	case 35: // ACCOUNT_TRADE_ALLOWED
		return interp.BoolVal(true), nil
	case 36: // ACCOUNT_TRADE_EXPERT
		return interp.BoolVal(true), nil
	default:
		return interp.IntVal(0), nil
	}
}

func builtinAccountInfoString(vm *VM, args []interp.Value) (interp.Value, error) {
	prop := argI(args, 0)
	switch prop {
	case 0: // ACCOUNT_CURRENCY
		return interp.StringVal("USD"), nil
	case 1: // ACCOUNT_NAME
		return interp.StringVal("Backtest"), nil
	case 2: // ACCOUNT_SERVER
		return interp.StringVal("SimBroker"), nil
	case 3: // ACCOUNT_COMPANY
		return interp.StringVal("SimBroker"), nil
	default:
		return interp.StringVal(""), nil
	}
}

func builtinAccountStopoutMode(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinAccountCredit(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimalZero), nil
}
