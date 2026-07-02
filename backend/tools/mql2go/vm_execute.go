package mql2go

import (
	"context"
	"errors"
	"fmt"

	"anttrader/tools/mql2go/interp"
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

	// ── Arithmetic ──
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

	case OP_NEG:
		a := vm.pop()
		if a.Kind == ValDecimal {
			vm.push(interp.DecimalVal(a.Decimal.Neg()))
		} else {
			vm.push(interp.IntVal(-a.ToInt()))
		}

	// ── Comparison ──
	case OP_EQ:
		b, a := vm.pop2()
		vm.push(interp.BoolVal(a.Equal(b)))

	case OP_NE:
		b, a := vm.pop2()
		vm.push(interp.BoolVal(!a.Equal(b)))

	case OP_LT:
		b, a := vm.pop2()
		vm.push(interp.BoolVal(vm.compare(a, b) < 0))

	case OP_LE:
		b, a := vm.pop2()
		vm.push(interp.BoolVal(vm.compare(a, b) <= 0))

	case OP_GT:
		b, a := vm.pop2()
		vm.push(interp.BoolVal(vm.compare(a, b) > 0))

	case OP_GE:
		b, a := vm.pop2()
		vm.push(interp.BoolVal(vm.compare(a, b) >= 0))

	// ── Logical ──
	case OP_AND:
		b, a := vm.pop2()
		vm.push(interp.BoolVal(a.IsTrue() && b.IsTrue()))

	case OP_OR:
		b, a := vm.pop2()
		vm.push(interp.BoolVal(a.IsTrue() || b.IsTrue()))

	case OP_NOT:
		a := vm.pop()
		vm.push(interp.BoolVal(!a.IsTrue()))

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
		nArgs := int(ins.B)
		entryPC := ins.A
		args := vm.popN(nArgs)

		// Check call depth to prevent Go stack overflow
		if vm.callDepth >= MaxCallDepth {
			return fmt.Errorf("strategy exceeded max call depth (%d)", MaxCallDepth)
		}
		vm.callDepth++

		// Save current frame state
		oldLocals := vm.locals
		// Create new frame: params + local slots
		var numLocals int
		if fn, ok := vm.funcByEntryPC[entryPC]; ok {
			numLocals = fn.NumLocals
		}
		newLocals := make([]interp.Value, numLocals)
		copy(newLocals, args)

		vm.locals = newLocals
		// Save return PC
		returnPC := vm.pc
		// Set PC to function entry (skip ENTER_FUNC)
		vm.pc = entryPC + 1

		// Run until RETURN
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
			if ins2.Op == OP_RETURN {
				break
			}
			if ins2.Op == OP_HALT {
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

		// Restore frame
		vm.locals = oldLocals
		vm.pc = returnPC
		vm.callDepth--

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
		if int(ins.A) < len(vm.globals) {
			arrVal := vm.globals[ins.A]
			if arrVal.Kind == interp.ValArray {
				i := int(idx.ToInt())
				if i >= 0 && i < len(arrVal.Array) {
					vm.push(arrVal.Array[i])
					break
				}
			}
		}
		vm.push(interp.NoneVal())

	case OP_STORE_ARRAY:
		idx := vm.pop()
		val := vm.pop()
		if int(ins.A) < len(vm.globals) {
			arrVal := vm.globals[ins.A]
			if arrVal.Kind == interp.ValArray {
				i := int(idx.ToInt())
				if i >= 0 && i < len(arrVal.Array) {
					arrVal.Array[i] = val
				}
			}
		}

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
