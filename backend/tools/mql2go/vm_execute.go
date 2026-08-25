package mql2go

import (
	"context"
	"fmt"
)

// runLoop is the main VM execution loop.
func (vm *VM) runLoop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		if vm.pc < 0 || vm.pc > int32(len(vm.bc.Code)) {
			return fmt.Errorf("VM invalid program counter: %d", vm.pc)
		}

		// VM-RUNTIME-FAILCLOSED-3: Check for fatal/stack error BEFORE code-end
		// success return. If the last instruction triggered a fault, the fault
		// must be reported — not swallowed by pc==len(Code) returning nil.
		if vm.fatalError != "" {
			return fmt.Errorf("VM fatal: %s", vm.fatalError)
		}
		if vm.stackError != "" {
			return fmt.Errorf("VM stack error: %s", vm.stackError)
		}

		if vm.pc == int32(len(vm.bc.Code)) {
			return nil
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
		if vm.stackError != "" {
			return fmt.Errorf("VM stack error: %s", vm.stackError)
		}
		result, err := vm.callBuiltin(ins.A, args)
		if err != nil {
			return err
		}
		vm.push(result)
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
