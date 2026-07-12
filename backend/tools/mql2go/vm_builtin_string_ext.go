package mql2go

import (
	"strings"

	"alphaforge/tools/mql2go/interp"
)

// MQL4/MQL5 String functions — complete implementation.

func builtinStringAdd(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 2 {
		return interp.BoolVal(false), nil
	}
	// StringAdd(str_var, add_str) — appends add_str to str_var
	// In VM, strings are passed by value, so we return the concatenated string.
	return interp.StringVal(argS(args, 0) + argS(args, 1)), nil
}

func builtinStringCompare(vm *VM, args []interp.Value) (interp.Value, error) {
	a := argS(args, 0)
	b := argS(args, 1)
	caseSensitive := true
	if len(args) > 2 {
		caseSensitive = argI(args, 2) != 0
	}
	if !caseSensitive {
		a = strings.ToLower(a)
		b = strings.ToLower(b)
	}
	switch {
	case a < b:
		return interp.IntVal(-1), nil
	case a > b:
		return interp.IntVal(1), nil
	default:
		return interp.IntVal(0), nil
	}
}

func builtinStringGetCharacter(vm *VM, args []interp.Value) (interp.Value, error) {
	s := argS(args, 0)
	pos := int(argI(args, 1))
	if pos < 0 || pos >= len(s) {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(s[pos])), nil
}

func builtinStringSetCharacter(vm *VM, args []interp.Value) (interp.Value, error) {
	s := argS(args, 0)
	pos := int(argI(args, 1))
	ch := argI(args, 2)
	if pos < 0 || pos >= len(s) {
		return interp.StringVal(s), nil
	}
	return interp.StringVal(s[:pos] + string(rune(ch)) + s[pos+1:]), nil
}

func builtinStringToLower(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(strings.ToLower(argS(args, 0))), nil
}

func builtinStringToUpper(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(strings.ToUpper(argS(args, 0))), nil
}

func builtinStringBufferLen(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(int32(len(argS(args, 0)))), nil
}

func builtinStringInit(vm *VM, args []interp.Value) (interp.Value, error) {
	size := int(argI(args, 0))
	if size < 0 {
		size = 0
	}
	fill := byte(' ')
	if len(args) > 1 {
		fill = byte(argI(args, 1))
	}
	return interp.StringVal(strings.Repeat(string(fill), size)), nil
}

func builtinStringFill(vm *VM, args []interp.Value) (interp.Value, error) {
	s := argS(args, 0)
	fill := byte(' ')
	if len(args) > 1 {
		fill = byte(argI(args, 1))
	}
	if len(s) == 0 {
		return interp.StringVal(s), nil
	}
	return interp.StringVal(strings.Repeat(string(fill), len(s))), nil
}
