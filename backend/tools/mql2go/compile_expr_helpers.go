package mql2go

import (
	"fmt"

	"alphaforge/tools/mql2go/interp"

	"github.com/shopspring/decimal"
)

func (c *astCompiler) foldConstBinary(e *interp.Expr) (interp.Value, bool) {
	if e.Kind != interp.ExprBinary || len(e.Args) != 2 {
		return interp.NoneVal(), false
	}
	if e.Args[0].Kind != interp.ExprLiteral || e.Args[1].Kind != interp.ExprLiteral {
		return interp.NoneVal(), false
	}
	a, b := e.Args[0].Val, e.Args[1].Val

	if e.Op == "+" && a.Kind == interp.ValString && b.Kind == interp.ValString {
		return interp.StringVal(a.Str + b.Str), true
	}

	if a.Kind == interp.ValInt && b.Kind == interp.ValInt {
		return foldIntBinary(e.Op, a.ToInt(), b.ToInt())
	}

	ad, bd := a.ToDecimal(), b.ToDecimal()
	return foldDecimalBinary(e.Op, ad, bd, a, b)
}

func foldIntBinary(op string, ai, bi int32) (interp.Value, bool) {
	switch op {
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
	case "//":
		if bi == 0 {
			return interp.NoneVal(), false
		}
		q := ai / bi
		if (ai < 0) != (bi < 0) && ai%bi != 0 {
			q--
		}
		return interp.IntVal(q), true
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
	return interp.NoneVal(), false
}

func foldDecimalBinary(op string, ad, bd decimal.Decimal, a, b interp.Value) (interp.Value, bool) {
	switch op {
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
	case "//":
		if bd.IsZero() {
			return interp.NoneVal(), false
		}
		return interp.DecimalVal(ad.Div(bd).Floor()), true
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
		if len(e.Args) != fn.NumParams {
			if c.err == nil {
				c.err = fmt.Errorf("function %s expects %d arguments, got %d", e.Name, fn.NumParams, len(e.Args))
			}
			return
		}
		// Push arguments onto stack
		for i := range e.Args {
			c.compileExpr(&e.Args[i])
		}
		callPC := c.emit(OP_CALL_USER, -1, int32(fn.NumParams), 0)
		c.userCallPatches = append(c.userCallPatches, userCallPatch{
			instruction: callPC,
			callee:      e.Name,
		})
		return
	}

	if sym, ok := interp.LookupAPI(e.Name); ok && sym.Status == interp.StatusUnsupported {
		if c.err == nil {
			c.err = fmt.Errorf("unsupported function %s: %s", e.Name, sym.Reason)
		}
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

	// Not a user function or registered builtin — check API registry.
	if sym, ok := interp.LookupAPI(e.Name); ok {
		switch sym.Status {
		case interp.StatusUnsupported:
			// Explicitly unsupported (iCustom, FileOpen, etc.) — compile error.
			if c.err == nil {
				c.err = fmt.Errorf("unsupported function %s: %s", e.Name, sym.Reason)
			}
			return
		case interp.StatusImplemented:
			// Registered as implemented but no VM builtin handler registered.
			// This is a registry/VM wiring mismatch — record as blind spot.
			c.bc.Coverage.AddBlindSpot(e.Name + " (registry:implemented, no VM handler)")
			for i := range e.Args {
				c.compileExpr(&e.Args[i])
			}
			for range e.Args {
				c.emit(OP_POP, 0, 0, 0)
			}
			c.emit(OP_PUSH_CONST, int32(c.addConst(interp.NoneVal())), 0, 0)
			return
		}
	}

	// Truly unknown function — record as blind spot (not in registry at all).
	// This catches typos and functions we haven't catalogued yet.
	c.bc.Coverage.AddBlindSpot("unknown function: " + e.Name)
	for i := range e.Args {
		c.compileExpr(&e.Args[i])
	}
	for range e.Args {
		c.emit(OP_POP, 0, 0, 0)
	}
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
		// Use CALL_BUILTIN with the class-qualified builtin name.
		bid := c.registerMethodBuiltin(c.methodBuiltinName(e), len(e.Args)-1)
		c.emit(OP_CALL_BUILTIN, int32(bid), int32(len(e.Args)-1), 0)
	} else {
		// Field read: obj.field
		fieldIdx := int32(c.addFieldOrMethodName(e.Name))
		c.emit(OP_GET_FIELD, fieldIdx, 0, 0)
	}
}

func (c *astCompiler) methodBuiltinName(e *interp.Expr) string {
	if len(e.Args) == 0 || e.Args[0].Kind != interp.ExprVar {
		return e.Name
	}
	for _, g := range c.ir.Globals {
		if g.Name == e.Args[0].Name && g.Type == "CTrade" {
			return "CTrade." + e.Name
		}
	}
	return e.Name
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
	case "//":
		return OP_FLOOR_DIV
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
		if c.err == nil {
			c.err = fmt.Errorf("unsupported binary operator: %s", op)
		}
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
	case "//=":
		return OP_FLOOR_DIV
	case "%=":
		return OP_MOD
	default:
		if c.err == nil {
			c.err = fmt.Errorf("unsupported compound assignment operator: %s", op)
		}
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
