package mql2go

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// ── Stack helpers ────────────────────────────────────────────────────

func (vm *VM) push(v interp.Value) {
	vm.stack = append(vm.stack, v)
}

func (vm *VM) pop() interp.Value {
	if len(vm.stack) == 0 {
		vm.setStackError("operand stack underflow at pc=%d", vm.pc)
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
	if n < 0 {
		vm.setStackError("negative pop count %d at pc=%d", n, vm.pc)
		return nil
	}
	if n > len(vm.stack) {
		vm.setStackError("operand stack underflow at pc=%d: need %d values, have %d", vm.pc, n, len(vm.stack))
		return nil
	}
	start := len(vm.stack) - n
	result := make([]interp.Value, n)
	copy(result, vm.stack[start:])
	vm.stack = vm.stack[:start]
	return result
}

func (vm *VM) setStackError(format string, args ...interface{}) {
	if vm.stackError == "" {
		vm.stackError = fmt.Sprintf(format, args...)
	}
}

func (vm *VM) invalidateOrderCaches() {
	vm.cachedPositions = nil
	vm.cachedOrders = nil
	vm.cachedHistory = nil
	vm.positionsLoaded = false
	vm.ordersLoaded = false
	vm.historyLoaded = false
	vm.currentPos = nil
	vm.currentOrder = nil
}

// ── Arithmetic helpers ───────────────────────────────────────────────

func (vm *VM) arith(a, b interp.Value, op string) interp.Value {
	// MQL concatenates strings with values using their display form.
	if op == "+" && (a.Kind == interp.ValString || b.Kind == interp.ValString) {
		return interp.StringVal(a.ToString() + b.ToString())
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
				vm.setStackError("division by zero")
				return interp.DecimalVal(decimal.Zero)
			}
			return interp.DecimalVal(ad.Div(bd))
		case "%":
			if bd.IsZero() {
				vm.setStackError("modulo by zero")
				return interp.DecimalVal(decimal.Zero)
			}
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
			vm.setStackError("integer division by zero")
			return interp.IntVal(0)
		}
		return interp.IntVal(ai / bi)
	case "%":
		if bi == 0 {
			vm.setStackError("integer modulo by zero")
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
			vm.setStackError("floor division by zero")
			return interp.DecimalVal(decimal.Zero)
		}
		return interp.DecimalVal(ad.Div(bd).Floor())
	}
	// Integer floor division: floor(a/b)
	ai := a.ToInt()
	bi := b.ToInt()
	if bi == 0 {
		vm.setStackError("integer floor division by zero")
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

func (vm *VM) runtimeTimeMillis() int64 {
	if vm.ctx == nil {
		return 0
	}
	return vm.ctx.ServerTime()
}

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

func (vm *VM) callBuiltin(builtinID int32, args []interp.Value) (interp.Value, error) {
	id := BuiltinID(builtinID)
	if builtinID < 0 || int(id) >= len(builtinRegistry) {
		name := fmt.Sprintf("builtin_%d", id)
		vm.recordBlindSpot(name)
		return interp.NoneVal(), fmt.Errorf("invalid builtin id: %d", builtinID)
	}
	entry := builtinRegistry[id]
	if sym, ok := interp.LookupAPI(entry.name); ok && sym.Status == interp.StatusUnsupported {
		vm.recordBlindSpot(entry.name)
		return interp.NoneVal(), fmt.Errorf("unsupported API %s: %s", entry.name, sym.Reason)
	}
	if entry.fn != nil {
		result, err := entry.fn(vm, args)
		if err != nil {
			return interp.NoneVal(), fmt.Errorf("builtin %s: %w", entry.name, err)
		}
		// Defense-in-depth: if the handler set fatalError via recordBlindSpot
		// (e.g. iADX MODE_PLUSDI), return an error immediately rather than
		// relying on the caller to check fatalError after the call.
		// This ensures fatal blind spots terminate at the current instruction
		// regardless of which opcode invoked the builtin.
		if vm.fatalError != "" {
			return interp.NoneVal(), fmt.Errorf("VM fatal: %s", vm.fatalError)
		}
		return result, nil
	}
	// No handler registered — classify severity via registry (Layer 0).
	if isFatalUnimplemented(entry.name) {
		vm.fatalError = fmt.Sprintf("unimplemented critical builtin: %s", entry.name)
		vm.recordBlindSpot(entry.name)
		return interp.NoneVal(), fmt.Errorf("VM fatal: %s", vm.fatalError)
	}
	// Non-fatal: Object/Chart/File operations — silent blind spot
	vm.recordBlindSpot(entry.name)
	return interp.NoneVal(), nil
}

func (vm *VM) recordBlindSpot(name string) {
	vm.runtimeBlindSpots[name]++
	if vm.fatalError == "" && interp.SeverityForBuiltin(name) == interp.SeverityFatal {
		vm.fatalError = fmt.Sprintf("unimplemented critical builtin: %s", name)
	}
}
