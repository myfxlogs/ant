package mql2go

import (
	"anttrader/tools/mql2go/interp"
)

// ── Expression compilation ───────────────────────────────────────────

func (c *astCompiler) compileExpr(e *interp.Expr) {
	if e == nil {
		c.emit(OP_PUSH_CONST, int32(c.addConst(interp.NoneVal())), 0, 0)
		return
	}

	switch e.Kind {
	case interp.ExprLiteral:
		cid := c.addConst(e.Val)
		c.emit(OP_PUSH_CONST, int32(cid), 0, 0)

	case interp.ExprVar:
		// Check if it's a builtin variable (Ask, Bid, Point, Digits, etc.)
		// These are MQL predefined variables that act as zero-arg builtins.
		if bid, ok := c.bc.Builtins[e.Name]; ok {
			c.emit(OP_CALL_BUILTIN, int32(bid), 0, 0)
			return
		}
		slot, isGlobal := c.resolveVar(e.Name)
		if isGlobal {
			c.emit(OP_PUSH_GLOBAL, int32(slot), 0, 0)
		} else {
			c.emit(OP_PUSH_VAR, int32(slot), 0, 0)
		}

	case interp.ExprConst:
		// Enum constant or predefined constant (OP_BUY, etc.)
		if val, ok := c.bc.Enums[e.Name]; ok {
			cid := c.addConst(interp.IntVal(val))
			c.emit(OP_PUSH_CONST, int32(cid), 0, 0)
		} else {
			// Unknown constant — push 0 and record blind spot
			c.bc.Coverage.AddBlindSpot("unknown constant: " + e.Name)
			cid := c.addConst(interp.IntVal(0))
			c.emit(OP_PUSH_CONST, int32(cid), 0, 0)
		}

	case interp.ExprBinary:
		c.compileBinary(e)

	case interp.ExprUnary:
		c.compileExpr(&e.Args[0])
		switch e.Op {
		case "-":
			c.emit(OP_NEG, 0, 0, 0)
		case "!":
			c.emit(OP_NOT, 0, 0, 0)
		case "+":
			// no-op
		default:
			c.bc.Coverage.AddBlindSpot("unary op: " + e.Op)
		}

	case interp.ExprCall:
		c.compileCall(e)

	case interp.ExprSubscript:
		// Series access: Close[i], Open[i], etc.
		c.compileExpr(e.Index)
		c.emit(OP_PUSH_SERIES, 0, int32(c.addSeriesName(e.Name)), 0)

	case interp.ExprField:
		c.compileField(e)

	case interp.ExprTernary:
		c.compileExpr(e.Cond)
		jmpFalse := c.emitJump(OP_JMP_IF_FALSE, 0)
		c.compileExpr(e.ThenExpr)
		jmpEnd := c.emitJump(OP_JMP, 0)
		c.patchJump(jmpFalse)
		c.compileExpr(e.ElseExpr)
		c.patchJump(jmpEnd)

	case interp.ExprUpdate:
		// i++ / i--
		slot, isGlobal := c.resolveVar(e.Name)
		if isGlobal {
			c.emit(OP_PUSH_GLOBAL, int32(slot), 0, 0)
		} else {
			c.emit(OP_PUSH_VAR, int32(slot), 0, 0)
		}
		c.emit(OP_PUSH_CONST, int32(c.addConst(interp.IntVal(1))), 0, 0)
		if e.Op == "++" {
			c.emit(OP_ADD, 0, 0, 0)
		} else {
			c.emit(OP_SUB, 0, 0, 0)
		}
		if isGlobal {
			c.emit(OP_STORE_GLOBAL, int32(slot), 0, 0)
		} else {
			c.emit(OP_STORE_VAR, int32(slot), 0, 0)
		}

	case interp.ExprAssignment:
		c.compileExpr(&e.Args[0])
		slot, isGlobal := c.resolveVar(e.Name)
		if isGlobal {
			c.emit(OP_STORE_GLOBAL, int32(slot), 0, 0)
		} else {
			c.emit(OP_STORE_VAR, int32(slot), 0, 0)
		}

	case interp.ExprCompoundAssign:
		// a += b → a = a + b
		slot, isGlobal := c.resolveVar(e.Name)
		if isGlobal {
			c.emit(OP_PUSH_GLOBAL, int32(slot), 0, 0)
		} else {
			c.emit(OP_PUSH_VAR, int32(slot), 0, 0)
		}
		c.compileExpr(&e.Args[0])
		c.emit(c.compoundAssignOp(e.Op), 0, 0, 0)
		if isGlobal {
			c.emit(OP_STORE_GLOBAL, int32(slot), 0, 0)
		} else {
			c.emit(OP_STORE_VAR, int32(slot), 0, 0)
		}

	case interp.ExprDecl:
		// Variable declaration with initializer: name := value
		c.compileExpr(&e.Args[0])
		// Allocate a local slot from the flat function-local space
		if len(c.localScopes) > 0 {
			scope := c.localScopes[len(c.localScopes)-1]
			scope[e.Name] = VarID(c.nextLocalSlot)
			c.nextLocalSlot++
		}
		slot, isGlobal := c.resolveVar(e.Name)
		if isGlobal {
			c.emit(OP_STORE_GLOBAL, int32(slot), 0, 0)
		} else {
			c.emit(OP_STORE_VAR, int32(slot), 0, 0)
		}
	}
}

func (c *astCompiler) compileBinary(e *interp.Expr) {
	// Constant folding: if both operands are literals, evaluate at compile time
	if folded, ok := c.foldConstBinary(e); ok {
		cid := c.addConst(folded)
		c.emit(OP_PUSH_CONST, int32(cid), 0, 0)
		return
	}

	// Short-circuit evaluation for && and ||
	if e.Op == "&&" {
		c.compileExpr(&e.Args[0])
		jmpFalse1 := c.emitJump(OP_JMP_IF_FALSE, 0)
		c.compileExpr(&e.Args[1])
		jmpFalse2 := c.emitJump(OP_JMP_IF_FALSE, 0)
		c.emit(OP_PUSH_CONST, int32(c.addConst(interp.BoolVal(true))), 0, 0)
		jmpEnd := c.emitJump(OP_JMP, 0)
		c.patchJump(jmpFalse1)
		c.patchJump(jmpFalse2)
		c.emit(OP_PUSH_CONST, int32(c.addConst(interp.BoolVal(false))), 0, 0)
		c.patchJump(jmpEnd)
		return
	}
	if e.Op == "||" {
		c.compileExpr(&e.Args[0])
		jmpTrue1 := c.emitJump(OP_JMP_IF_TRUE, 0)
		c.compileExpr(&e.Args[1])
		jmpTrue2 := c.emitJump(OP_JMP_IF_TRUE, 0)
		c.emit(OP_PUSH_CONST, int32(c.addConst(interp.BoolVal(false))), 0, 0)
		jmpEnd := c.emitJump(OP_JMP, 0)
		c.patchJump(jmpTrue1)
		c.patchJump(jmpTrue2)
		c.emit(OP_PUSH_CONST, int32(c.addConst(interp.BoolVal(true))), 0, 0)
		c.patchJump(jmpEnd)
		return
	}

	c.compileExpr(&e.Args[0])
	c.compileExpr(&e.Args[1])
	c.emit(c.binaryOp(e.Op), 0, 0, 0)
}

// foldConstBinary attempts to evaluate a binary expression at compile time
// when both operands are literals. Returns the folded value and true if successful.
func (c *astCompiler) foldConstBinary(e *interp.Expr) (interp.Value, bool) {
	if e.Kind != interp.ExprBinary {
		return interp.NoneVal(), false
	}
	if e.Args[0].Kind != interp.ExprLiteral || e.Args[1].Kind != interp.ExprLiteral {
		return interp.NoneVal(), false
	}
	a, b := e.Args[0].Val, e.Args[1].Val

	// String concatenation: "a" + "b" → "ab"
	if e.Op == "+" && a.Kind == interp.ValString && b.Kind == interp.ValString {
		return interp.StringVal(a.Str + b.Str), true
	}

	// Preserve int type when both operands are int
	bothInt := a.Kind == interp.ValInt && b.Kind == interp.ValInt
	if bothInt {
		ai, bi := a.ToInt(), b.ToInt()
		switch e.Op {
		case "+":
			return interp.IntVal(ai + bi), true
		case "-":
			return interp.IntVal(ai - bi), true
		case "*":
			return interp.IntVal(ai * bi), true
		case "/":
			if bi == 0 {
				return interp.NoneVal(), false
			}
			return interp.IntVal(ai / bi), true
		case "%":
			if bi == 0 {
				return interp.NoneVal(), false
			}
			return interp.IntVal(ai % bi), true
		case "==":
			return interp.BoolVal(ai == bi), true
		case "!=":
			return interp.BoolVal(ai != bi), true
		case "<":
			return interp.BoolVal(ai < bi), true
		case "<=":
			return interp.BoolVal(ai <= bi), true
		case ">":
			return interp.BoolVal(ai > bi), true
		case ">=":
			return interp.BoolVal(ai >= bi), true
		}
	}

	// Decimal arithmetic for mixed/decimal operands
	ad, bd := a.ToDecimal(), b.ToDecimal()
	switch e.Op {
	case "+":
		return interp.DecimalVal(ad.Add(bd)), true
	case "-":
		return interp.DecimalVal(ad.Sub(bd)), true
	case "*":
		return interp.DecimalVal(ad.Mul(bd)), true
	case "/":
		if bd.IsZero() {
			return interp.NoneVal(), false
		}
		return interp.DecimalVal(ad.Div(bd)), true
	case "%":
		if bd.IsZero() {
			return interp.NoneVal(), false
		}
		return interp.DecimalVal(ad.Mod(bd)), true
	case "==":
		return interp.BoolVal(a.Equal(b)), true
	case "!=":
		return interp.BoolVal(!a.Equal(b)), true
	case "<":
		return interp.BoolVal(ad.LessThan(bd)), true
	case "<=":
		return interp.BoolVal(ad.LessThanOrEqual(bd)), true
	case ">":
		return interp.BoolVal(ad.GreaterThan(bd)), true
	case ">=":
		return interp.BoolVal(ad.GreaterThanOrEqual(bd)), true
	}
	return interp.NoneVal(), false
}

func (c *astCompiler) compileCall(e *interp.Expr) {
	// Check if it's a user-defined function
	if fn, ok := c.bc.Funcs[e.Name]; ok {
		// Push arguments onto stack
		for i := range e.Args {
			c.compileExpr(&e.Args[i])
		}
		c.emit(OP_CALL_USER, fn.EntryPC, int32(fn.NumParams), 0)
		return
	}

	// Check if it's a builtin
	if bid, ok := c.bc.Builtins[e.Name]; ok {
		// Push arguments onto stack
		for i := range e.Args {
			c.compileExpr(&e.Args[i])
		}
		c.emit(OP_CALL_BUILTIN, int32(bid), int32(len(e.Args)), 0)
		return
	}

	// Unknown function — push args, pop them, push None
	for i := range e.Args {
		c.compileExpr(&e.Args[i])
	}
	for range e.Args {
		c.emit(OP_POP, 0, 0, 0)
	}
	c.bc.Coverage.AddBlindSpot("unknown function: " + e.Name)
	c.emit(OP_PUSH_CONST, int32(c.addConst(interp.NoneVal())), 0, 0)
}

func (c *astCompiler) compileField(e *interp.Expr) {
	// obj.field or obj.method(args)
	// Args[0] = object expression, Args[1:] = method args (if any)
	if len(e.Args) == 0 {
		return
	}

	// Compile object expression
	c.compileExpr(&e.Args[0])

	if e.IsAssign {
		// Field assignment: obj.field = value
		// Args[0] = obj, Args[1] = value
		if len(e.Args) >= 2 {
			c.compileExpr(&e.Args[1])
			fieldIdx := int32(c.addFieldOrMethodName(e.Name))
			c.emit(OP_SET_FIELD, fieldIdx, 0, 0)
		}
		return
	}

	// Check if this is a method call (has extra args beyond the object)
	if len(e.Args) > 1 {
		// Method call: obj.method(arg1, arg2)
		// Args[0] = obj, Args[1:] = method args
		// Pop the object — method builtins (CTrade.*) don't read it from the stack.
		c.emit(OP_POP, 0, 0, 0)
		for i := 1; i < len(e.Args); i++ {
			c.compileExpr(&e.Args[i])
		}
		// Use CALL_BUILTIN with a synthetic builtin for the method
		bid := c.registerMethodBuiltin(e.Name, len(e.Args)-1)
		c.emit(OP_CALL_BUILTIN, int32(bid), int32(len(e.Args)-1), 0)
	} else {
		// Field read: obj.field
		fieldIdx := int32(c.addFieldOrMethodName(e.Name))
		c.emit(OP_GET_FIELD, fieldIdx, 0, 0)
	}
}

func (c *astCompiler) binaryOp(op string) Opcode {
	switch op {
	case "+":
		return OP_ADD
	case "-":
		return OP_SUB
	case "*":
		return OP_MUL
	case "/":
		return OP_DIV
	case "%":
		return OP_MOD
	case "==":
		return OP_EQ
	case "!=":
		return OP_NE
	case "<":
		return OP_LT
	case "<=":
		return OP_LE
	case ">":
		return OP_GT
	case ">=":
		return OP_GE
	default:
		c.bc.Coverage.AddBlindSpot("binary op: " + op)
		return OP_ADD
	}
}

func (c *astCompiler) compoundAssignOp(op string) Opcode {
	switch op {
	case "+=":
		return OP_ADD
	case "-=":
		return OP_SUB
	case "*=":
		return OP_MUL
	case "/=":
		return OP_DIV
	case "%=":
		return OP_MOD
	default:
		return OP_ADD
	}
}

// addSeriesName registers a series name (Close, Open, High, Low, Volume, Time)
// and returns its index. The VM uses this to dispatch to the correct SDK method.
func (c *astCompiler) addSeriesName(name string) ConstID {
	// Series names are stored as string constants
	return c.addConst(interp.StringVal(name))
}

// addFieldOrMethodName registers a field/method name and returns its index.
func (c *astCompiler) addFieldOrMethodName(name string) ConstID {
	return c.addConst(interp.StringVal(name))
}
