package mql2go

import (
	"time"

	"alphaforge/tools/mql2go/interp"
)

// MQL4/MQL5 Checkup / Platform functions — real implementations only.
// VM-API-TRUTH-1 batch 2a: 12 fixed-value stubs removed (reclassified
// StatusUnsupported in interp/api_registry.go; the compiler now rejects
// them instead of letting strategies run on fake platform data).
// Remaining: IsConnected/IsDemo/IsTradeAllowed (VM-API-TRUTH-3 real),
// GetLastError/ResetLastError/SetUserError (lastError state machine),
// CurTime/GetTickCount* (real time sources).

// builtinIsConnected returns the authoritative connection status from the
// SDK context. VM-API-TRUTH-3: was hardcoded true, now reads vm.ctx.Account().
// IsConnected. When vm.ctx is nil (backtest without live context), defaults
// to true (backtest simulation is always "connected").
func builtinIsConnected(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.BoolVal(true), nil
	}
	return interp.BoolVal(vm.ctx.Account().IsConnected), nil
}

// builtinIsDemo returns the authoritative demo-account flag from the SDK
// context. VM-API-TRUTH-3: was hardcoded true, now reads vm.ctx.Account().
// IsDemo. When vm.ctx is nil (backtest without live context), defaults to
// true (backtest simulation runs on a demo account).
func builtinIsDemo(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.BoolVal(true), nil
	}
	return interp.BoolVal(vm.ctx.Account().IsDemo), nil
}

// builtinIsTradeAllowed returns the authoritative trade-permission flag from
// the SDK context. VM-API-TRUTH-3: was hardcoded true, now reads vm.ctx.
// Account().IsTradeAllowed. Investor accounts have IsTradeAllowed=false
// (set by Batch 2 injectAccountTruth). When vm.ctx is nil (backtest without
// live context), defaults to true (backtest simulation allows trading).
func builtinIsTradeAllowed(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.BoolVal(true), nil
	}
	return interp.BoolVal(vm.ctx.Account().IsTradeAllowed), nil
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

// builtinGetLastError implements MQL4 GetLastError: returns the current
// _LastError value and clears it (read-then-reset semantics).
func builtinGetLastError(vm *VM, args []interp.Value) (interp.Value, error) {
	old := vm.lastError
	vm.lastError = 0
	return interp.IntVal(old), nil
}

// builtinResetLastError implements MQL4 ResetLastError: clears _LastError.
func builtinResetLastError(vm *VM, args []interp.Value) (interp.Value, error) {
	vm.lastError = 0
	return interp.NoneVal(), nil
}

func builtinSetUserError(vm *VM, args []interp.Value) (interp.Value, error) {
	vm.lastError = 65536 + argI(args, 0) // ERR_USER_ERROR_FIRST + code
	return interp.NoneVal(), nil
}

func builtinCurTime(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(int32(vm.ctx.ServerTime() / 1000)), nil
}
