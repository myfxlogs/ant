package mql2go

import (
	"fmt"

	"alphaforge/tools/mql2go/interp"
)

// This file contains opcode handler implementations extracted from
// vm_execute.go for file-size compliance.

func (vm *VM) executeStack(ins Instruction) {
	switch ins.Op {
	case OP_PUSH_CONST:
		vm.push(constToValue(vm.bc.Consts[ins.A]))
	case OP_PUSH_VAR:
		if int(ins.A) < len(vm.locals) {
			vm.push(vm.locals[ins.A])
		} else {
			vm.setStackError("OP_PUSH_VAR: local slot %d out of range (len=%d)", ins.A, len(vm.locals))
			vm.push(interp.NoneVal())
		}
	case OP_PUSH_GLOBAL:
		if int(ins.A) < len(vm.globals) {
			vm.push(vm.globals[ins.A])
		} else {
			vm.setStackError("OP_PUSH_GLOBAL: global slot %d out of range (len=%d)", ins.A, len(vm.globals))
			vm.push(interp.NoneVal())
		}
	case OP_STORE_VAR:
		if int(ins.A) < len(vm.locals) {
			vm.locals[ins.A] = vm.pop()
		} else {
			vm.setStackError("OP_STORE_VAR: local slot %d out of range (len=%d)", ins.A, len(vm.locals))
			vm.pop()
		}
	case OP_STORE_GLOBAL:
		if int(ins.A) < len(vm.globals) {
			vm.globals[ins.A] = vm.pop()
		} else {
			vm.setStackError("OP_STORE_GLOBAL: global slot %d out of range (len=%d)", ins.A, len(vm.globals))
			vm.pop()
		}
	case OP_POP:
		vm.pop()
	case OP_DUP:
		if len(vm.stack) > 0 {
			vm.push(vm.stack[len(vm.stack)-1])
		} else {
			vm.setStackError("OP_DUP: stack underflow")
		}
	case OP_SWAP:
		if len(vm.stack) >= 2 {
			n := len(vm.stack)
			vm.stack[n-1], vm.stack[n-2] = vm.stack[n-2], vm.stack[n-1]
		} else {
			vm.setStackError("OP_SWAP: stack underflow (need 2, have %d)", len(vm.stack))
		}
	}
}

func (vm *VM) executeArith(ins Instruction) {
	switch ins.Op {
	case OP_ADD:
		b, a := vm.pop2()
		vm.push(vm.arith(a, b, "+"))
	case OP_SUB:
		b, a := vm.pop2()
		vm.push(vm.arith(a, b, "-"))
	case OP_MUL:
		b, a := vm.pop2()
		vm.push(vm.arith(a, b, "*"))
	case OP_DIV:
		b, a := vm.pop2()
		vm.push(vm.arith(a, b, "/"))
	case OP_MOD:
		b, a := vm.pop2()
		vm.push(vm.arith(a, b, "%"))
	case OP_FLOOR_DIV:
		b, a := vm.pop2()
		vm.push(vm.floorDiv(a, b))
	case OP_NEG:
		a := vm.pop()
		if a.Kind == interp.ValDecimal {
			vm.push(interp.DecimalVal(a.Decimal.Neg()))
		} else {
			vm.push(interp.IntVal(-a.ToInt()))
		}
	}
}

func (vm *VM) executeCompare(ins Instruction) {
	b, a := vm.pop2()
	switch ins.Op {
	case OP_EQ:
		vm.push(interp.BoolVal(a.Equal(b)))
	case OP_NE:
		vm.push(interp.BoolVal(!a.Equal(b)))
	case OP_LT:
		vm.push(interp.BoolVal(vm.compare(a, b) < 0))
	case OP_LE:
		vm.push(interp.BoolVal(vm.compare(a, b) <= 0))
	case OP_GT:
		vm.push(interp.BoolVal(vm.compare(a, b) > 0))
	case OP_GE:
		vm.push(interp.BoolVal(vm.compare(a, b) >= 0))
	}
}

func (vm *VM) executeLogical(ins Instruction) {
	switch ins.Op {
	case OP_AND:
		b, a := vm.pop2()
		vm.push(interp.BoolVal(a.IsTrue() && b.IsTrue()))
	case OP_OR:
		b, a := vm.pop2()
		vm.push(interp.BoolVal(a.IsTrue() || b.IsTrue()))
	case OP_NOT:
		a := vm.pop()
		vm.push(interp.BoolVal(!a.IsTrue()))
	}
}

func (vm *VM) executePushArray(ins Instruction, idx interp.Value) interp.Value {
	if int(ins.A) < len(vm.globals) {
		arrVal := vm.globals[ins.A]
		if arrVal.Kind == interp.ValArray {
			array := arrVal.ArrayData()
			i := int(idx.ToInt())
			if i >= 0 && i < len(array) {
				return array[i]
			}
		}
	}
	return interp.NoneVal()
}

func (vm *VM) executeStoreArray(ins Instruction, idx, val interp.Value) {
	if int(ins.A) >= len(vm.globals) {
		return
	}
	arrVal := vm.globals[ins.A]
	if arrVal.Kind != interp.ValArray {
		return
	}
	array := arrVal.ArrayData()
	i := int(idx.ToInt())
	if i >= 0 && i < len(array) {
		array[i] = val
	}
}

func (vm *VM) executeCallUser(ins Instruction) error {
	nArgs := int(ins.B)
	entryPC := ins.A
	if nArgs < 0 {
		return fmt.Errorf("VM invalid user-call argument count: %d", nArgs)
	}
	if entryPC < 0 || entryPC >= int32(len(vm.bc.Code)) {
		return fmt.Errorf("VM invalid user-call target: %d", entryPC)
	}
	fn, ok := vm.funcByEntryPC[entryPC]
	if !ok {
		return fmt.Errorf("VM user-call target %d is not a function entry", entryPC)
	}
	args := vm.popN(nArgs)
	if vm.stackError != "" {
		return fmt.Errorf("VM stack error: %s", vm.stackError)
	}

	if vm.callDepth >= MaxCallDepth {
		return fmt.Errorf("strategy exceeded max call depth (%d)", MaxCallDepth)
	}
	vm.callDepth++

	oldLocals := vm.locals
	newLocals := make([]interp.Value, fn.NumLocals)
	copy(newLocals, args)

	vm.locals = newLocals
	returnPC := vm.pc
	vm.pc = entryPC // Jump to function body start (EntryPC points at body, not marker)

	completed := false
	for {
		if vm.pc < 0 || vm.pc >= int32(len(vm.bc.Code)) {
			vm.locals = oldLocals
			vm.pc = returnPC
			vm.callDepth--
			return fmt.Errorf("VM function %q exited without RETURN at pc=%d", fn.Name, vm.pc)
		}
		if vm.ticks%10000 == 0 && vm.runCtx != nil {
			select {
			case <-vm.runCtx.Done():
				vm.locals = oldLocals
				vm.callDepth--
				return vm.runCtx.Err()
			default:
			}
		}
		ins2 := vm.bc.Code[vm.pc]
		vm.pc++
		if ins2.Op == OP_RETURN || ins2.Op == OP_HALT {
			completed = true
			break
		}
		vm.ticks++
		if vm.ticks > MaxTicks {
			vm.locals = oldLocals
			vm.callDepth--
			return vm.instructionLimitError()
		}
		if err := vm.execute(ins2); err != nil {
			vm.locals = oldLocals
			vm.callDepth--
			return err
		}
		if vm.stackError != "" {
			vm.locals = oldLocals
			vm.callDepth--
			return fmt.Errorf("VM stack error: %s", vm.stackError)
		}
		if vm.fatalError != "" {
			vm.locals = oldLocals
			vm.callDepth--
			return fmt.Errorf("VM fatal: %s", vm.fatalError)
		}
	}

	if !completed {
		vm.locals = oldLocals
		vm.callDepth--
		return fmt.Errorf("VM function %q did not complete", fn.Name)
	}
	vm.locals = oldLocals
	vm.pc = returnPC
	vm.callDepth--
	return nil
}
