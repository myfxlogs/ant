package mql2go

import (
	"context"
	"errors"
	"fmt"

	"alphaforge/tools/mql2go/interp"
)

// runLoop is the main VM execution loop.
func (vm *VM) runLoop(ctx context.Context) error {
	for vm.pc < int32(len(vm.bc.Code)) {
		// Check for fatal error (ADR §5.4 — critical builtin missing)
		if vm.fatalError != "" {
			return fmt.Errorf("VM fatal: %s", vm.fatalError)
		}

		// Check for cancellation / timeout
		if vm.ticks%10000 == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
		}

		// Instruction limit check
		vm.ticks++
		if vm.ticks > MaxTicks {
			return errors.New("strategy exceeded instruction limit")
		}

		// Stack depth check
		if len(vm.stack) > MaxStackDepth {
			return fmt.Errorf("strategy exceeded max stack depth (%d)", len(vm.stack))
		}

		ins := vm.bc.Code[vm.pc]
		vm.pc++ // advance before execution (jumps will override)

		if err := vm.execute(ins); err != nil {
			return err
		}
	}
	return nil
}

// execute dispatches a single instruction.
func (vm *VM) execute(ins Instruction) error {
	switch ins.Op {
	// ── Stack operations ──
	case OP_PUSH_CONST, OP_PUSH_VAR, OP_PUSH_GLOBAL, OP_STORE_VAR, OP_STORE_GLOBAL, OP_POP, OP_DUP, OP_SWAP:
		vm.executeStack(ins)

	// ── Arithmetic ──
	case OP_ADD, OP_SUB, OP_MUL, OP_DIV, OP_MOD, OP_FLOOR_DIV, OP_NEG:
		vm.executeArith(ins)

	// ── Comparison ──
	case OP_EQ, OP_NE, OP_LT, OP_LE, OP_GT, OP_GE:
		vm.executeCompare(ins)

	// ── Logical ──
	case OP_AND, OP_OR, OP_NOT:
		vm.executeLogical(ins)

	// ── Control flow ──
	case OP_JMP:
		vm.pc = ins.A

	case OP_JMP_IF_FALSE:
		a := vm.pop()
		if !a.IsTrue() {
			vm.pc = ins.A
		}

	case OP_JMP_IF_TRUE:
		a := vm.pop()
		if a.IsTrue() {
			vm.pc = ins.A
		}

	// ── Function call ──
	case OP_CALL_BUILTIN:
		nArgs := int(ins.B)
		args := vm.popN(nArgs)
		result := vm.callBuiltin(ins.A, args)
		vm.push(result)

	case OP_CALL_USER:
		if err := vm.executeCallUser(ins); err != nil {
			return err
		}

	case OP_ENTER_FUNC:
		// No-op: local frame is set up by CALL_USER

	case OP_LEAVE_FUNC:
		// No-op: frame cleanup is handled by CALL_USER

	// ── Event entry markers ──
	case OP_ENTER_ONINIT, OP_ENTER_ONBAR, OP_ENTER_ONTICK, OP_ENTER_ONTRADE, OP_ENTER_ONTIMER, OP_ENTER_ONDEINIT, OP_ENTER_ONTRADETRANSACTION, OP_ENTER_ONBOOKEVENT:
		// No-op: just markers

	case OP_RETURN:
		// In event context, this ends the event
		vm.pc = int32(len(vm.bc.Code))

	// ── Series access ──
	case OP_PUSH_SERIES:
		seriesName := vm.bc.Consts[ins.B].Str
		idx := vm.pop()
		vm.push(vm.getSeries(seriesName, idx.ToInt()))

	// ── User array access ──
	case OP_PUSH_ARRAY:
		idx := vm.pop()
		vm.push(vm.executePushArray(ins, idx))

	case OP_STORE_ARRAY:
		idx := vm.pop()
		val := vm.pop()
		vm.executeStoreArray(ins, idx, val)

	// ── Field access ──
	case OP_GET_FIELD:
		fieldName := vm.bc.Consts[ins.A].Str
		obj := vm.pop()
		vm.push(vm.getField(obj, fieldName))

	case OP_SET_FIELD:
		fieldName := vm.bc.Consts[ins.A].Str
		val := vm.pop()
		obj := vm.pop()
		vm.setField(obj, fieldName, val)

	case OP_HALT:
		vm.pc = int32(len(vm.bc.Code))
	}

	return nil
}

func (vm *VM) executeStack(ins Instruction) {
	switch ins.Op {
	case OP_PUSH_CONST:
		vm.push(constToValue(vm.bc.Consts[ins.A]))
	case OP_PUSH_VAR:
		if int(ins.A) < len(vm.locals) {
			vm.push(vm.locals[ins.A])
		} else {
			vm.push(interp.NoneVal())
		}
	case OP_PUSH_GLOBAL:
		if int(ins.A) < len(vm.globals) {
			vm.push(vm.globals[ins.A])
		} else {
			vm.push(interp.NoneVal())
		}
	case OP_STORE_VAR:
		if int(ins.A) < len(vm.locals) {
			vm.locals[ins.A] = vm.pop()
		}
	case OP_STORE_GLOBAL:
		if int(ins.A) < len(vm.globals) {
			vm.globals[ins.A] = vm.pop()
		} else {
			vm.pop()
		}
	case OP_POP:
		vm.pop()
	case OP_DUP:
		if len(vm.stack) > 0 {
			vm.push(vm.stack[len(vm.stack)-1])
		}
	case OP_SWAP:
		if len(vm.stack) >= 2 {
			n := len(vm.stack)
			vm.stack[n-1], vm.stack[n-2] = vm.stack[n-2], vm.stack[n-1]
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
			i := int(idx.ToInt())
			if i >= 0 && i < len(arrVal.Array) {
				return arrVal.Array[i]
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
	i := int(idx.ToInt())
	if i >= 0 && i < len(arrVal.Array) {
		arrVal.Array[i] = val
	}
}

func (vm *VM) executeCallUser(ins Instruction) error {
	nArgs := int(ins.B)
	entryPC := ins.A
	args := vm.popN(nArgs)

	if vm.callDepth >= MaxCallDepth {
		return fmt.Errorf("strategy exceeded max call depth (%d)", MaxCallDepth)
	}
	vm.callDepth++

	oldLocals := vm.locals
	var numLocals int
	if fn, ok := vm.funcByEntryPC[entryPC]; ok {
		numLocals = fn.NumLocals
	}
	newLocals := make([]interp.Value, numLocals)
	copy(newLocals, args)

	vm.locals = newLocals
	returnPC := vm.pc
	vm.pc = entryPC + 1

	for vm.pc < int32(len(vm.bc.Code)) {
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
			break
		}
		vm.ticks++
		if vm.ticks > MaxTicks {
			vm.locals = oldLocals
			vm.callDepth--
			return errors.New("strategy exceeded instruction limit")
		}
		if err := vm.execute(ins2); err != nil {
			vm.locals = oldLocals
			vm.callDepth--
			return err
		}
	}

	vm.locals = oldLocals
	vm.pc = returnPC
	vm.callDepth--
	return nil
}
