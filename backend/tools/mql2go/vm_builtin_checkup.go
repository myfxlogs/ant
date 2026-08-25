package mql2go

import "alphaforge/tools/mql2go/interp"

// MQL4/MQL5 Checkup / Platform functions — complete implementation.
// In backtest context, most of these return fixed values.
// VM-API-TRUTH-3: IsConnected/IsDemo/IsTradeAllowed read from authoritative
// AccountInfo when available; backtest defaults to true (simulated environment).

func builtinIsConnected(vm *VM, args []interp.Value) (interp.Value, error) {
	// VM-API-TRUTH-3: in backtest, always connected. In live, the host
	// process is connected if the VM is receiving events (connection is
	// a host-level concern, not a VM-level query).
	if vm.ctx == nil {
		return interp.BoolVal(true), nil
	}
	return interp.BoolVal(vm.ctx.Account().IsConnected), nil
}

func builtinIsDemo(vm *VM, args []interp.Value) (interp.Value, error) {
	// VM-API-TRUTH-3: in backtest, always demo. In live, from AccountInfo.
	if vm.ctx == nil {
		return interp.BoolVal(true), nil
	}
	return interp.BoolVal(vm.ctx.Account().IsDemo), nil
}

func builtinIsDllsAllowed(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinIsExpertEnabled(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinIsLibrariesAllowed(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinIsTradeAllowed(vm *VM, args []interp.Value) (interp.Value, error) {
	// VM-API-TRUTH-3: in backtest, always allowed. In live, actual trade
	// permission is enforced at order submission (broker rejects if not
	// allowed). The VM-level query reflects the account's trade-enabled flag.
	if vm.ctx == nil {
		return interp.BoolVal(true), nil
	}
	return interp.BoolVal(vm.ctx.Account().IsTradeAllowed), nil
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
	return interp.IntVal(int32(vm.runtimeTimeMillis() & 0x7FFFFFFF)), nil
}

func builtinGetTickCount64(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(int32(vm.runtimeTimeMillis())), nil
}

func builtinGetMicrosecondCount(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(int32(vm.runtimeTimeMillis() * 1000)), nil
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
