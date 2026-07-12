package mql2go

import (
	"alphaforge/tools/mql2go/interp"
)

// MQL5 trade helpers, order/deal/history functions.
// Most are stubs returning safe defaults — they require MQL5 broker integration
// that is not available in the backtest VM.

func builtinOrderCalcMargin(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinOrderCalcProfit(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinOrderCheck(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinPositionSelect(vm *VM, args []interp.Value) (interp.Value, error) {
	// Delegate to PositionSelectByTicket
	return builtinPositionSelectByTicket(vm, args)
}

// MQL5 pending order functions — stubs (backtest tracks positions, not pending orders)
func builtinOrderGetTicket(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinOrderGetDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimalZero), nil
}

func builtinOrderGetInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinOrderGetString(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(""), nil
}

func builtinOrdersTotalMQL5(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

// MQL5 history/deal functions — stubs (backtest doesn't expose deal history)
func builtinHistorySelect(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinHistorySelectByPosition(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinHistoryDealsTotal(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinHistoryDealSelect(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(false), nil
}

func builtinHistoryDealGetTicket(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinHistoryDealGetDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimalZero), nil
}

func builtinHistoryDealGetInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinHistoryDealGetString(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(""), nil
}

func builtinHistoryOrdersTotal(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinHistoryOrderSelect(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(false), nil
}

func builtinHistoryOrderGetTicket(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinHistoryOrderGetDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimalZero), nil
}

func builtinHistoryOrderGetInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinHistoryOrderGetString(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(""), nil
}
