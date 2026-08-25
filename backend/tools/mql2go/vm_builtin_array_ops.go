// vm_builtin_array.go — Array builtins extracted from vm_builtin_string.go.
package mql2go

import (
	"fmt"

	"alphaforge/tools/mql2go/interp"
)

func builtinArrayInitialize(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 2 || args[0].Kind != interp.ValArray {
		return interp.IntVal(0), nil
	}
	arr := args[0].ArrayData()
	fillVal := args[1]
	for i := range arr {
		arr[i] = fillVal
	}
	return interp.IntVal(int32(len(arr))), nil
}

func builtinArrayFree(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) > 0 && args[0].Kind == interp.ValArray {
		args[0].SetArrayData(nil)
	}
	return interp.NoneVal(), nil
}

func builtinArrayRange(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 2 || args[0].Kind != interp.ValArray || argI(args, 1) != 0 {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(1), nil
}

func builtinArrayIsSeries(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(len(args) > 0 && args[0].Kind == interp.ValArray), nil
}

func builtinArrayResize(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 2 || args[0].Kind != interp.ValArray {
		return interp.IntVal(-1), nil
	}
	newSize := int(argI(args, 1))
	if newSize < 0 {
		return interp.IntVal(-1), nil
	}
	arr := args[0].ArrayData()
	if newSize <= len(arr) {
		args[0].SetArrayData(arr[:newSize])
		return interp.IntVal(int32(newSize)), nil
	}
	resized := make([]interp.Value, newSize)
	copy(resized, arr)
	for i := len(arr); i < newSize; i++ {
		resized[i] = interp.NoneVal()
	}
	args[0].SetArrayData(resized)
	return interp.IntVal(int32(newSize)), nil
}

func builtinArrayCopy(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 2 || args[0].Kind != interp.ValArray || args[1].Kind != interp.ValArray {
		return interp.IntVal(0), nil
	}
	dst := args[0].ArrayData()
	src := args[1].ArrayData()
	dstStart := 0
	srcStart := 0
	count := len(src)
	if len(args) > 2 {
		dstStart = int(argI(args, 2))
	}
	if len(args) > 3 {
		srcStart = int(argI(args, 3))
	}
	if len(args) > 4 {
		count = int(argI(args, 4))
	}
	if srcStart >= len(src) || count <= 0 {
		return interp.IntVal(0), nil
	}
	avail := len(src) - srcStart
	if count > avail {
		count = avail
	}
	if dstStart+count > len(dst) {
		count = len(dst) - dstStart
	}
	if count <= 0 {
		return interp.IntVal(0), nil
	}
	copy(dst[dstStart:], src[srcStart:srcStart+count])
	return interp.IntVal(int32(count)), nil
}

func builtinArrayMaximum(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 1 || args[0].Kind != interp.ValArray {
		return interp.IntVal(-1), nil
	}
	arr := args[0].ArrayData()
	if len(arr) == 0 {
		return interp.IntVal(-1), nil
	}
	maxIdx := 0
	maxVal := arr[0]
	for i := 1; i < len(arr); i++ {
		if arr[i].ToDecimal().GreaterThan(maxVal.ToDecimal()) {
			maxVal = arr[i]
			maxIdx = i
		}
	}
	return interp.IntVal(int32(maxIdx)), nil
}

func builtinArrayMinimum(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 1 || args[0].Kind != interp.ValArray {
		return interp.IntVal(-1), nil
	}
	arr := args[0].ArrayData()
	if len(arr) == 0 {
		return interp.IntVal(-1), nil
	}
	minIdx := 0
	minVal := arr[0]
	for i := 1; i < len(arr); i++ {
		if arr[i].ToDecimal().LessThan(minVal.ToDecimal()) {
			minVal = arr[i]
			minIdx = i
		}
	}
	return interp.IntVal(int32(minIdx)), nil
}

func builtinArraySort(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 1 || args[0].Kind != interp.ValArray {
		return interp.IntVal(0), nil
	}
	arr := args[0].ArrayData()
	// Simple ascending sort by decimal value
	for i := 1; i < len(arr); i++ {
		for j := i; j > 0 && arr[j].ToDecimal().LessThan(arr[j-1].ToDecimal()); j-- {
			arr[j], arr[j-1] = arr[j-1], arr[j]
		}
	}
	return interp.IntVal(int32(len(arr))), nil
}

func builtinArrayFill(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 3 || args[0].Kind != interp.ValArray {
		return interp.IntVal(0), nil
	}
	arr := args[0].ArrayData()
	start := int(argI(args, 1))
	count := int(argI(args, 2))
	fillVal := interp.NoneVal()
	if len(args) > 3 {
		fillVal = args[3]
	}
	for i := start; i < start+count && i < len(arr); i++ {
		if i >= 0 {
			arr[i] = fillVal
		}
	}
	return interp.IntVal(int32(count)), nil
}

// Ensure fmt is used
var _ = fmt.Sprintf
