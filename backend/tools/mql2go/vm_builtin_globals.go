package mql2go

import (
	"sync"

	"alphaforge/tools/mql2go/interp"
)

// MQL4/MQL5 Global Variables of the Terminal.
// In backtest, these are process-local variables (no terminal persistence).
// In live mode, they would be shared across EAs via the terminal's global variable store.

var (
	gvStore   = map[string]float64{}
	gvStoreMu sync.RWMutex
)

func builtinGlobalVariableSet(vm *VM, args []interp.Value) (interp.Value, error) {
	name := argS(args, 0)
	val := argD(args, 1).InexactFloat64()
	gvStoreMu.Lock()
	gvStore[name] = val
	gvStoreMu.Unlock()
	return interp.IntVal(int32(len(args))), nil
}

func builtinGlobalVariableGet(vm *VM, args []interp.Value) (interp.Value, error) {
	name := argS(args, 0)
	gvStoreMu.RLock()
	v, ok := gvStore[name]
	gvStoreMu.RUnlock()
	if !ok {
		return interp.DecimalVal(decimalZero), nil
	}
	return interp.DecimalVal(safeDecimalFromFloat(v)), nil
}

func builtinGlobalVariableDel(vm *VM, args []interp.Value) (interp.Value, error) {
	name := argS(args, 0)
	gvStoreMu.Lock()
	delete(gvStore, name)
	gvStoreMu.Unlock()
	return interp.BoolVal(true), nil
}

func builtinGlobalVariableCheck(vm *VM, args []interp.Value) (interp.Value, error) {
	name := argS(args, 0)
	gvStoreMu.RLock()
	_, ok := gvStore[name]
	gvStoreMu.RUnlock()
	return interp.BoolVal(ok), nil
}

func builtinGlobalVariableTemp(vm *VM, args []interp.Value) (interp.Value, error) {
	name := argS(args, 0)
	val := argD(args, 1).InexactFloat64()
	gvStoreMu.Lock()
	if _, ok := gvStore[name]; !ok {
		gvStore[name] = val
	}
	gvStoreMu.Unlock()
	return interp.BoolVal(true), nil
}

func builtinGlobalVariableFlush(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinGlobalVariablesDeleteAll(vm *VM, args []interp.Value) (interp.Value, error) {
	gvStoreMu.Lock()
	n := len(gvStore)
	gvStore = map[string]float64{}
	gvStoreMu.Unlock()
	return interp.IntVal(int32(n)), nil
}

func builtinGlobalVariablesTotal(vm *VM, args []interp.Value) (interp.Value, error) {
	gvStoreMu.RLock()
	n := len(gvStore)
	gvStoreMu.RUnlock()
	return interp.IntVal(int32(n)), nil
}

func builtinGlobalVariableName(vm *VM, args []interp.Value) (interp.Value, error) {
	idx := int(argI(args, 0))
	gvStoreMu.RLock()
	defer gvStoreMu.RUnlock()
	i := 0
	for name := range gvStore {
		if i == idx {
			return interp.StringVal(name), nil
		}
		i++
	}
	return interp.StringVal(""), nil
}
