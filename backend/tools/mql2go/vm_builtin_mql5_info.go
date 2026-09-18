package mql2go

import (
	"fmt"

	"alphaforge/strategy/sdk"
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
		"time":     interp.IntVal(int32(vm.ctx.ServerTime() / 1000)),
		"bid":      interp.DecimalVal(vm.ctx.Bid()),
		"ask":      interp.DecimalVal(vm.ctx.Ask()),
		"last":     interp.DecimalVal(vm.ctx.Bid()),
		nodeVolume: interp.IntVal(0),
	}
	return interp.Value{Kind: interp.ValClass, Class: &interp.ClassInstance{Fields: fields}}, nil
}

func builtinSymbolName(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(vm.ctx.Symbol()), nil
}

func builtinSymbolSelect(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinSymbolsTotal(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(1), nil
}

func builtinSymbolIsSynchronized(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

// MQL5 AccountInfo* functions. Prop numbers follow the real MQL5 enums
// (ENUM_ACCOUNT_INFO_DOUBLE/INTEGER/STRING); props without an authoritative
// source in the VM return an error (fail-closed) instead of a fake value.
func builtinAccountInfoDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	prop := argI(args, 0)
	switch prop {
	case 0: // ACCOUNT_BALANCE
		return interp.DecimalVal(vm.ctx.Account().Balance), nil
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
	case 1, 7, 8, 9, 10, 11, 12, 13: // known props without a VM data source
		return interp.DecimalVal(decimalZero), fmt.Errorf("AccountInfoDouble: prop %d (%s) has no authoritative source in the VM", prop, accountInfoDoubleNoSourceName(prop))
	default:
		return interp.DecimalVal(decimalZero), fmt.Errorf("AccountInfoDouble: unknown prop %d", prop)
	}
}

// accountInfoDoubleNoSourceName returns the ENUM_ACCOUNT_INFO_DOUBLE constant
// name for known props without a VM data source (error message context).
func accountInfoDoubleNoSourceName(prop int32) string {
	switch prop {
	case 1:
		return "ACCOUNT_CREDIT"
	case 7:
		return "ACCOUNT_MARGIN_SO_CALL"
	case 8:
		return "ACCOUNT_MARGIN_SO_SO"
	case 9:
		return "ACCOUNT_MARGIN_INITIAL"
	case 10:
		return "ACCOUNT_MARGIN_MAINTENANCE"
	case 11:
		return "ACCOUNT_ASSETS"
	case 12:
		return "ACCOUNT_LIABILITIES"
	case 13:
		return "ACCOUNT_COMMISSION_BLOCKED"
	}
	return "unknown"
}

func builtinAccountInfoInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	prop := argI(args, 0)
	switch prop {
	case 0: // ACCOUNT_LOGIN (SimBroker leaves Login=0 = not reported, honest value)
		return interp.IntVal(int32(vm.ctx.Account().Login)), nil
	case 1: // ACCOUNT_TRADE_MODE (contest folded into demo — IsDemo field definition)
		if vm.ctx.Account().IsDemo {
			return interp.IntVal(0), nil // ACCOUNT_TRADE_MODE_DEMO
		}
		return interp.IntVal(2), nil // ACCOUNT_TRADE_MODE_REAL
	case 2: // ACCOUNT_LEVERAGE
		if vm.ctx != nil {
			return interp.IntVal(vm.ctx.Account().Leverage), nil
		}
		return interp.IntVal(100), nil
	case 5, 6: // ACCOUNT_TRADE_ALLOWED / ACCOUNT_TRADE_EXPERT (single mtapi flag; EA ≡ account in this VM)
		if vm.ctx.Account().IsTradeAllowed {
			return interp.IntVal(1), nil
		}
		return interp.IntVal(0), nil
	case 7: // ACCOUNT_MARGIN_MODE
		switch vm.ctx.Mode() {
		case sdk.ModeNetting:
			return interp.IntVal(0), nil // ACCOUNT_MARGIN_MODE_RETAIL_NETTING
		case sdk.ModeHedging:
			return interp.IntVal(2), nil // ACCOUNT_MARGIN_MODE_RETAIL_HEDGING
		default:
			return interp.IntVal(-1), fmt.Errorf("AccountInfoInteger: prop 7 (ACCOUNT_MARGIN_MODE) has no authoritative source in the VM")
		}
	case 10: // ACCOUNT_HEDGE_ALLOWED
		switch vm.ctx.Mode() {
		case sdk.ModeHedging:
			return interp.IntVal(1), nil
		case sdk.ModeNetting:
			return interp.IntVal(0), nil
		default:
			return interp.IntVal(-1), fmt.Errorf("AccountInfoInteger: prop 10 (ACCOUNT_HEDGE_ALLOWED) has no authoritative source in the VM")
		}
	case 3, 4, 8, 9: // known props without a VM data source
		return interp.IntVal(-1), fmt.Errorf("AccountInfoInteger: prop %d (%s) has no authoritative source in the VM", prop, accountInfoIntegerNoSourceName(prop))
	default:
		return interp.IntVal(-1), fmt.Errorf("AccountInfoInteger: unknown prop %d", prop)
	}
}

// accountInfoIntegerNoSourceName returns the ENUM_ACCOUNT_INFO_INTEGER
// constant name for known props without a VM data source (error message
// context).
func accountInfoIntegerNoSourceName(prop int32) string {
	switch prop {
	case 3:
		return "ACCOUNT_LIMIT_ORDERS"
	case 4:
		return "ACCOUNT_MARGIN_SO_MODE"
	case 8:
		return "ACCOUNT_CURRENCY_DIGITS"
	case 9:
		return "ACCOUNT_FIFO_CLOSE"
	}
	return "unknown"
}

func builtinAccountInfoString(vm *VM, args []interp.Value) (interp.Value, error) {
	prop := argI(args, 0)
	switch prop {
	case 0: // ACCOUNT_NAME — no field on sdk.AccountInfo
		return interp.StringVal(""), fmt.Errorf("AccountInfoString: prop 0 (ACCOUNT_NAME) has no authoritative source in the VM")
	case 1: // ACCOUNT_SERVER — no field on sdk.AccountInfo
		return interp.StringVal(""), fmt.Errorf("AccountInfoString: prop 1 (ACCOUNT_SERVER) has no authoritative source in the VM")
	case 2: // ACCOUNT_CURRENCY (empty = missing fact, fail-closed)
		if c := vm.ctx.Account().Currency; c != "" {
			return interp.StringVal(c), nil
		}
		return interp.StringVal(""), fmt.Errorf("AccountInfoString: prop 2 (ACCOUNT_CURRENCY) has no authoritative source in the VM")
	case 3: // ACCOUNT_COMPANY (empty = missing fact, fail-closed)
		if c := vm.ctx.Account().Company; c != "" {
			return interp.StringVal(c), nil
		}
		return interp.StringVal(""), fmt.Errorf("AccountInfoString: prop 3 (ACCOUNT_COMPANY) has no authoritative source in the VM")
	default:
		return interp.StringVal(""), fmt.Errorf("AccountInfoString: unknown prop %d", prop)
	}
}
