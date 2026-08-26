package mql2go

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// ── Stack helpers ────────────────────────────────────────────────────

// invalidateOrderCaches clears all lazy-loaded order/position caches and
// selection state. Called after every successful mutation builtin and at
// the top of runEvent (VM-TRADE-CONTEXT-1).
func (vm *VM) invalidateOrderCaches() {
	vm.cachedPositions = nil
	vm.cachedOrders = nil
	vm.cachedHistory = nil
	vm.currentPos = nil
	vm.currentOrder = nil
}

func (vm *VM) push(v interp.Value) {
	vm.stack = append(vm.stack, v)
}

func (vm *VM) pop() interp.Value {
	if len(vm.stack) == 0 {
		return interp.NoneVal()
	}
	n := len(vm.stack)
	v := vm.stack[n-1]
	vm.stack = vm.stack[:n-1]
	return v
}

func (vm *VM) pop2() (interp.Value, interp.Value) {
	b := vm.pop()
	a := vm.pop()
	return b, a
}

func (vm *VM) popN(n int) []interp.Value {
	if n > len(vm.stack) {
		n = len(vm.stack)
	}
	start := len(vm.stack) - n
	result := make([]interp.Value, n)
	copy(result, vm.stack[start:])
	vm.stack = vm.stack[:start]
	return result
}

// ── Arithmetic helpers ───────────────────────────────────────────────

func (vm *VM) arith(a, b interp.Value, op string) interp.Value {
	// String concatenation: "a" + "b" → "ab"
	if op == "+" && a.Kind == interp.ValString && b.Kind == interp.ValString {
		return interp.StringVal(a.Str + b.Str)
	}

	// If either is decimal, use decimal arithmetic
	if a.Kind == interp.ValDecimal || b.Kind == interp.ValDecimal {
		ad := a.ToDecimal()
		bd := b.ToDecimal()
		switch op {
		case "+":
			return interp.DecimalVal(ad.Add(bd))
		case "-":
			return interp.DecimalVal(ad.Sub(bd))
		case "*":
			return interp.DecimalVal(ad.Mul(bd))
		case "/":
			if bd.IsZero() {
				return interp.DecimalVal(decimal.Zero)
			}
			return interp.DecimalVal(ad.Div(bd))
		case "%":
			return interp.DecimalVal(ad.Mod(bd))
		}
	}
	// Integer arithmetic
	ai := a.ToInt()
	bi := b.ToInt()
	switch op {
	case "+":
		return interp.IntVal(ai + bi)
	case "-":
		return interp.IntVal(ai - bi)
	case "*":
		return interp.IntVal(ai * bi)
	case "/":
		if bi == 0 {
			return interp.IntVal(0)
		}
		return interp.IntVal(ai / bi)
	case "%":
		if bi == 0 {
			return interp.IntVal(0)
		}
		return interp.IntVal(ai % bi)
	}
	return interp.NoneVal()
}

func (vm *VM) compare(a, b interp.Value) int {
	ad := a.ToDecimal()
	bd := b.ToDecimal()
	return ad.Cmp(bd)
}

// floorDiv implements Python floor division (//): result = floor(a / b).
// Unlike Go's integer division (truncation toward zero), Python's // floors toward negative infinity.
// -5 // 2 = -3 (Go: -5 / 2 = -2)
func (vm *VM) floorDiv(a, b interp.Value) interp.Value {
	// If either is decimal, use decimal arithmetic with Floor
	if a.Kind == interp.ValDecimal || b.Kind == interp.ValDecimal {
		ad := a.ToDecimal()
		bd := b.ToDecimal()
		if bd.IsZero() {
			return interp.DecimalVal(decimal.Zero)
		}
		return interp.DecimalVal(ad.Div(bd).Floor())
	}
	// Integer floor division: floor(a/b)
	ai := a.ToInt()
	bi := b.ToInt()
	if bi == 0 {
		return interp.IntVal(0)
	}
	q := ai / bi
	// If signs differ and there's a remainder, floor down by 1
	if (ai < 0) != (bi < 0) && ai%bi != 0 {
		q--
	}
	return interp.IntVal(q)
}

// ── Series access ────────────────────────────────────────────────────

func (vm *VM) getSeries(name string, shift int32) interp.Value {
	if vm.ctx == nil || vm.ctx.Bars() == nil {
		return interp.DecimalVal(decimal.Zero)
	}
	bars := vm.ctx.Bars()
	switch name {
	case "Close":
		return interp.DecimalVal(bars.Close(int(shift)))
	case "Open":
		return interp.DecimalVal(bars.Open(int(shift)))
	case "High":
		return interp.DecimalVal(bars.High(int(shift)))
	case "Low":
		return interp.DecimalVal(bars.Low(int(shift)))
	case "Volume":
		return interp.IntVal(int32(bars.Volume(int(shift))))
	case "Time":
		return interp.IntVal(int32(bars.Time(int(shift)) / 1000))
	default:
		vm.recordBlindSpot("series: " + name)
		return interp.DecimalVal(decimal.Zero)
	}
}

// ── Field access ─────────────────────────────────────────────────────

func (vm *VM) getField(obj interp.Value, fieldName string) interp.Value {
	if obj.Kind != interp.ValClass || obj.Class == nil {
		return interp.NoneVal()
	}
	if v, ok := obj.Class.Fields[fieldName]; ok {
		return v
	}
	return interp.NoneVal()
}

func (vm *VM) setField(obj interp.Value, fieldName string, val interp.Value) {
	if obj.Kind != interp.ValClass || obj.Class == nil {
		return
	}
	obj.Class.Fields[fieldName] = val
}

// ── Builtin dispatch ─────────────────────────────────────────────────

// isFatalUnimplemented checks the API registry to determine if an unimplemented
// builtin should cause execution to stop. Registry-driven per Layer 0.
func isFatalUnimplemented(name string) bool {
	sym, ok := interp.LookupAPI(name)
	if !ok {
		for _, p := range []string{"Order", "Position", "MarketInfo", "iClose", "iOpen", "iHigh", "iLow", "iTime", "iVolume"} {
			if strings.HasPrefix(name, p) {
				return true
			}
		}
		return false
	}
	switch sym.Status {
	case interp.StatusImplemented:
		return true
	case interp.StatusUnsupported:
		return true
	default:
		return false
	}
}

func (vm *VM) callBuiltin(builtinID int32, args []interp.Value) interp.Value {
	id := BuiltinID(builtinID)
	if int(id) >= len(builtinRegistry) {
		vm.recordBlindSpot(fmt.Sprintf("builtin_%d", id))
		return interp.NoneVal()
	}
	entry := builtinRegistry[id]
	if entry.fn != nil {
		result, err := entry.fn(vm, args)
		if err != nil {
			vm.recordBlindSpot(entry.name)
			return interp.NoneVal()
		}
		return result
	}
	// No handler registered — classify severity via registry (Layer 0).
	if isFatalUnimplemented(entry.name) {
		vm.fatalError = fmt.Sprintf("unimplemented critical builtin: %s", entry.name)
		vm.recordBlindSpot(entry.name)
		return interp.NoneVal()
	}
	// Non-fatal: Object/Chart/File operations — silent blind spot
	vm.recordBlindSpot(entry.name)
	return interp.NoneVal()
}

func (vm *VM) recordBlindSpot(name string) {
	vm.runtimeBlindSpots[name]++
}
