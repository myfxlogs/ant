package mql2go

import (
	"anttrader/tools/mql2go/interp"
)

// MQL4/MQL5 Array functions — additions to existing implementations.

func builtinArrayBsearch(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 2 || args[0].Kind != interp.ValArray {
		return interp.IntVal(-1), nil
	}
	arr := args[0].Array
	target := argD(args, 1)
	lo, hi := 0, len(arr)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		v := arr[mid].ToDecimal()
		if v.Equal(target) {
			return interp.IntVal(int32(mid)), nil
		}
		if v.LessThan(target) {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return interp.IntVal(int32(lo)), nil
}

func builtinArrayCompare(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 2 || args[0].Kind != interp.ValArray || args[1].Kind != interp.ValArray {
		return interp.IntVal(-2), nil
	}
	a, b := args[0].Array, args[1].Array
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	for i := 0; i < minLen; i++ {
		av, bv := a[i].ToDecimal(), b[i].ToDecimal()
		switch {
		case av.LessThan(bv):
			return interp.IntVal(-1), nil
		case av.GreaterThan(bv):
			return interp.IntVal(1), nil
		}
	}
	switch {
	case len(a) < len(b):
		return interp.IntVal(-1), nil
	case len(a) > len(b):
		return interp.IntVal(1), nil
	default:
		return interp.IntVal(0), nil
	}
}

func builtinArrayInsert(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 3 || args[0].Kind != interp.ValArray || args[1].Kind != interp.ValArray {
		return interp.IntVal(0), nil
	}
	dst := args[0].Array
	src := args[1].Array
	pos := int(argI(args, 2))
	if pos < 0 {
		pos = 0
	}
	if pos > len(dst) {
		pos = len(dst)
	}
	result := make([]interp.Value, 0, len(dst)+len(src))
	result = append(result, dst[:pos]...)
	result = append(result, src...)
	result = append(result, dst[pos:]...)
	args[0].Array = result
	return interp.IntVal(int32(len(result))), nil
}

func builtinArrayRemove(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 2 || args[0].Kind != interp.ValArray {
		return interp.IntVal(0), nil
	}
	arr := args[0].Array
	pos := int(argI(args, 1))
	if pos < 0 || pos >= len(arr) {
		return interp.IntVal(0), nil
	}
	count := 1
	if len(args) > 2 {
		count = int(argI(args, 2))
	}
	if pos+count > len(arr) {
		count = len(arr) - pos
	}
	result := make([]interp.Value, 0, len(arr)-count)
	result = append(result, arr[:pos]...)
	result = append(result, arr[pos+count:]...)
	args[0].Array = result
	return interp.IntVal(int32(count)), nil
}

func builtinArrayReverse(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 1 || args[0].Kind != interp.ValArray {
		return interp.BoolVal(false), nil
	}
	arr := args[0].Array
	start := 0
	count := len(arr)
	if len(args) > 1 {
		start = int(argI(args, 1))
	}
	if len(args) > 2 {
		count = int(argI(args, 2))
	}
	if start < 0 {
		start = 0
	}
	if start+count > len(arr) {
		count = len(arr) - start
	}
	for i, j := start, start+count-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
	return interp.BoolVal(true), nil
}

func builtinArraySwap(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 2 || args[0].Kind != interp.ValArray || args[1].Kind != interp.ValArray {
		return interp.BoolVal(false), nil
	}
	args[0].Array, args[1].Array = args[1].Array, args[0].Array
	return interp.BoolVal(true), nil
}

func builtinArrayPrint(vm *VM, args []interp.Value) (interp.Value, error) {
	// No-op in backtest — no console output for array printing
	return interp.NoneVal(), nil
}

func builtinArrayGetAsSeries(vm *VM, args []interp.Value) (interp.Value, error) {
	// In our VM, arrays are always series-indexed (newest first)
	return interp.BoolVal(true), nil
}

func builtinArrayIsDynamic(vm *VM, args []interp.Value) (interp.Value, error) {
	// All arrays in our VM are dynamic
	return interp.BoolVal(true), nil
}
