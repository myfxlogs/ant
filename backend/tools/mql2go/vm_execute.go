package mql2go

import (
	"context"
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
			return vm.instructionLimitError()
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

// instructionLimitError returns a diagnostic error when the VM exceeds MaxTicks.
// MaxTicks is a per-event budget — runEvent resets ticks each bar/tick event —
// so exhausting it is almost certainly an infinite loop, not legitimate complexity.
// The message includes the enclosing user function (or event handler) and pc so
// the EA author can locate the offending loop.
func (vm *VM) instructionLimitError() error {
	return fmt.Errorf(
		"strategy exceeded instruction limit (%d instructions in a single bar/tick event) "+
			"— this is an infinite loop in %s at pc=%d. "+
			"Common causes: a while/for loop without a terminating condition, or unbounded recursion",
		MaxTicks, vm.currentSymbol(), vm.pc)
}

// currentSymbol identifies the user function or event handler at the current pc,
// for inclusion in instruction-limit diagnostics. It scans bc.Funcs for the
// function whose [EntryPC, next EntryPC) range contains vm.pc.
func (vm *VM) currentSymbol() string {
	var bestName string
	var bestEntry int32 = -1
	for _, fn := range vm.bc.Funcs {
		if fn.EntryPC <= vm.pc && fn.EntryPC > bestEntry {
			bestEntry = fn.EntryPC
			bestName = fn.Name
		}
	}
	if bestName != "" {
		return fmt.Sprintf("function %q", bestName)
	}
	// Not inside any user function → executing directly in an event handler.
	return "the event handler (OnTick/OnBar/OnInit)"
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
		// VM-AUDIT-2026-08-27-4: popN sets fatalError on stack underflow but
		// returns partial results. Without this early return, callBuiltin would
		// execute with too few args (e.g. OrderSend with empty symbol/volume),
		// producing side effects before runLoop's top-of-loop check fires.
		if vm.fatalError != "" {
			return fmt.Errorf("VM fatal: %s", vm.fatalError)
		}
		result := vm.callBuiltin(ins.A, args)
		vm.push(result)
		// VM-RUNTIME-FAILCLOSED-1: defense-in-depth — check fatalError after
		// builtin call (callBuiltin may have set it via handler or setStackError).
		if vm.fatalError != "" {
			return fmt.Errorf("VM fatal: %s", vm.fatalError)
		}

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
		v := vm.executePushArray(ins, idx)
		// VM-ARRAY-OOB-FAILCLOSED-1: defense-in-depth, same shape as
		// OP_CALL_BUILTIN — don't push the NoneVal fallback after a
		// stack error; runLoop's top-of-loop check fires next iteration.
		if vm.fatalError == "" {
			vm.push(v)
		}

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
			vm.setStackError(fmt.Sprintf("OP_PUSH_VAR slot %d out of range (locals=%d)", ins.A, len(vm.locals)))
		}
	case OP_PUSH_GLOBAL:
		if int(ins.A) < len(vm.globals) {
			vm.push(vm.globals[ins.A])
		} else {
			vm.setStackError(fmt.Sprintf("OP_PUSH_GLOBAL slot %d out of range (globals=%d)", ins.A, len(vm.globals)))
		}
	case OP_STORE_VAR:
		if int(ins.A) < len(vm.locals) {
			vm.locals[ins.A] = vm.pop()
		} else {
			vm.setStackError(fmt.Sprintf("OP_STORE_VAR slot %d out of range (locals=%d)", ins.A, len(vm.locals)))
			vm.pop()
		}
	case OP_STORE_GLOBAL:
		if int(ins.A) < len(vm.globals) {
			vm.globals[ins.A] = vm.pop()
		} else {
			vm.setStackError(fmt.Sprintf("OP_STORE_GLOBAL slot %d out of range (globals=%d)", ins.A, len(vm.globals)))
			vm.pop()
		}
	case OP_POP:
		vm.pop()
	case OP_DUP:
		if len(vm.stack) > 0 {
			vm.push(vm.stack[len(vm.stack)-1])
		} else {
			vm.setStackError("OP_DUP underflow")
		}
	case OP_SWAP:
		if len(vm.stack) >= 2 {
			n := len(vm.stack)
			vm.stack[n-1], vm.stack[n-2] = vm.stack[n-2], vm.stack[n-1]
		} else {
			vm.setStackError("OP_SWAP underflow")
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
	if ins.A < 0 {
		// VM-ARRAY-OOB-FAILCLOSED-1: negative-encoded slot = local array
		// access (compileSubscript). Never treat a local index as a global
		// slot — that silently read/wrote an unrelated global array.
		vm.setStackError(fmt.Sprintf("OP_PUSH_ARRAY local array slot %d not supported", -ins.A-1))
		return interp.NoneVal()
	}
	if int(ins.A) >= len(vm.globals) {
		vm.setStackError(fmt.Sprintf("OP_PUSH_ARRAY slot %d out of range (globals=%d)", ins.A, len(vm.globals)))
		return interp.NoneVal()
	}
	arrVal := vm.globals[ins.A]
	if arrVal.Kind != interp.ValArray {
		vm.setStackError(fmt.Sprintf("OP_PUSH_ARRAY slot %d is not an array", ins.A))
		return interp.NoneVal()
	}
	i := int(idx.ToInt())
	if i < 0 || i >= len(arrVal.Array) {
		vm.setStackError(fmt.Sprintf("OP_PUSH_ARRAY index %d out of range (len=%d)", i, len(arrVal.Array)))
		return interp.NoneVal()
	}
	return arrVal.Array[i]
}

func (vm *VM) executeStoreArray(ins Instruction, idx, val interp.Value) {
	if ins.A < 0 {
		// VM-ARRAY-OOB-FAILCLOSED-1: negative-encoded slot = local array
		// access (compileSubscript). See executePushArray.
		vm.setStackError(fmt.Sprintf("OP_STORE_ARRAY local array slot %d not supported", -ins.A-1))
		return
	}
	if int(ins.A) >= len(vm.globals) {
		vm.setStackError(fmt.Sprintf("OP_STORE_ARRAY slot %d out of range (globals=%d)", ins.A, len(vm.globals)))
		return
	}
	arrVal := vm.globals[ins.A]
	if arrVal.Kind != interp.ValArray {
		vm.setStackError(fmt.Sprintf("OP_STORE_ARRAY slot %d is not an array", ins.A))
		return
	}
	i := int(idx.ToInt())
	if i < 0 || i >= len(arrVal.Array) {
		vm.setStackError(fmt.Sprintf("OP_STORE_ARRAY index %d out of range (len=%d)", i, len(arrVal.Array)))
		return
	}
	arrVal.Array[i] = val
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
	vm.pc = entryPC // Jump to function body start (EntryPC points at body, not marker)

	for vm.pc < int32(len(vm.bc.Code)) {
		// VM-FUNC-FATAL-DELAY-1: mirror runLoop's top-of-loop fatal check
		// (ADR §5.4): a stack/slot/arith fault inside a user function must
		// stop the function now, not leak writes until OP_RETURN. Also covers
		// the entry state — popN above can set a fatal on stack underflow.
		if vm.fatalError != "" {
			vm.locals = oldLocals
			vm.callDepth--
			return fmt.Errorf("VM fatal: %s", vm.fatalError)
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
			break
		}
		vm.ticks++
		if vm.ticks > MaxTicks {
			vm.locals = oldLocals
			vm.callDepth--
			return vm.instructionLimitError()
		}
		// VM-AUDIT-2026-08-27-3: defense-in-depth — the outer runLoop checks
		// MaxStackDepth, but a long loop inside a user function never returns
		// to runLoop, so the stack could grow to MaxTicks (~80-160MB) before
		// the instruction limit fires. Check here too, restoring locals/callDepth
		// like the other error exit paths above.
		if len(vm.stack) > MaxStackDepth {
			vm.locals = oldLocals
			vm.callDepth--
			return fmt.Errorf("strategy exceeded max stack depth (%d)", len(vm.stack))
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
