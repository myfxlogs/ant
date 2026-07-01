package mql2go

import (
	"time"

	"anttrader/tools/mql2go/interp"
)

// MQL4/MQL5 Checkup / Platform functions — complete implementation.
// In backtest context, most of these return fixed values.

func builtinIsConnected(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinIsDemo(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(false), nil
}

func builtinIsDllsAllowed(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(false), nil
}

func builtinIsExpertEnabled(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinIsLibrariesAllowed(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(false), nil
}

func builtinIsTradeAllowed(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinIsTradeContextBusy(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(false), nil
}

func builtinIsStopped(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(false), nil
}

func builtinUninitializeReason(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinMQLInfoInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinMQLInfoString(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(""), nil
}

func builtinTerminalInfoDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimalZero), nil
}

func builtinTerminalInfoInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinTerminalInfoString(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(""), nil
}

func builtinGetTickCount(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(int32(time.Now().UnixMilli() & 0x7FFFFFFF)), nil
}

func builtinGetTickCount64(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(int32(time.Now().UnixMilli())), nil
}

func builtinGetMicrosecondCount(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(int32(time.Now().UnixMicro())), nil
}

func builtinSetUserError(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.NoneVal(), nil
}

func builtinSetReturnError(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.NoneVal(), nil
}

func builtinCurTime(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(vm.ctx.ServerTime() / 1000)), nil
}
