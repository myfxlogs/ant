package mql2go

import (
	"fmt"
	"strings"
	"time"

	"anttrader/tools/mql2go/interp"
)

// MQL4/MQL5 Conversion functions — complete implementation.

func builtinCharToString(vm *VM, args []interp.Value) (interp.Value, error) {
	ch := argI(args, 0)
	return interp.StringVal(string(rune(ch))), nil
}

func builtinCharArrayToString(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 1 || args[0].Kind != interp.ValArray {
		return interp.StringVal(""), nil
	}
	arr := args[0].Array
	start := 0
	count := -1
	if len(args) > 1 {
		start = int(argI(args, 1))
	}
	if len(args) > 2 {
		count = int(argI(args, 2))
	}
	if start < 0 {
		start = 0
	}
	if start >= len(arr) {
		return interp.StringVal(""), nil
	}
	end := len(arr)
	if count >= 0 {
		end = start + count
		if end > len(arr) {
			end = len(arr)
		}
	}
	var sb strings.Builder
	for i := start; i < end; i++ {
		sb.WriteRune(rune(arr[i].ToInt()))
	}
	return interp.StringVal(sb.String()), nil
}

func builtinShortToString(vm *VM, args []interp.Value) (interp.Value, error) {
	ch := argI(args, 0)
	return interp.StringVal(string(rune(ch))), nil
}

func builtinShortArrayToString(vm *VM, args []interp.Value) (interp.Value, error) {
	return builtinCharArrayToString(vm, args)
}

func builtinStringToColor(vm *VM, args []interp.Value) (interp.Value, error) {
	s := argS(args, 0)
	s = strings.TrimPrefix(s, "clr")
	var r, g, b int
	if _, err := fmt.Sscanf(s, "%d,%d,%d", &r, &g, &b); err == nil {
		return interp.IntVal(int32(r | g<<8 | b<<16)), nil
	}
	return interp.IntVal(0), nil
}

func builtinStringToCharArray(vm *VM, args []interp.Value) (interp.Value, error) {
	s := argS(args, 0)
	arr := make([]interp.Value, len(s))
	for i, c := range s {
		arr[i] = interp.IntVal(int32(c))
	}
	return interp.Value{Kind: interp.ValArray, Array: arr}, nil
}

func builtinStringToShortArray(vm *VM, args []interp.Value) (interp.Value, error) {
	return builtinStringToCharArray(vm, args)
}

func builtinEnumToString(vm *VM, args []interp.Value) (interp.Value, error) {
	// In MQL, EnumToString converts an enum value to its name.
	// We don't have the enum mapping at runtime, so return the integer as string.
	return interp.StringVal(fmt.Sprintf("%d", argI(args, 0))), nil
}

func builtinTimeToString(vm *VM, args []interp.Value) (interp.Value, error) {
	ts := int64(argI(args, 0))
	mode := int32(0)
	if len(args) > 1 {
		mode = argI(args, 1)
	}
	t := time.Unix(ts, 0).UTC()
	switch mode {
	case 1: // TIME_DATE
		return interp.StringVal(t.Format("2006.01.02")), nil
	case 2: // TIME_MINUTES
		return interp.StringVal(t.Format("15:04")), nil
	case 4: // TIME_SECONDS
		return interp.StringVal(t.Format("15:04:05")), nil
	default: // TIME_DATE|TIME_MINUTES
		return interp.StringVal(t.Format("2006.01.02 15:04")), nil
	}
}
