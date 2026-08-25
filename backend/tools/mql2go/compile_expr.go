package mql2go

import (
	"fmt"
	"strings"

	"alphaforge/tools/mql2go/interp"
)

// isSeriesName returns true for MQL predefined time-series names.
func isSeriesName(name string) bool {
	switch strings.ToLower(name) {
	case "close", "open", "high", "low", nodeVolume, "time":
		return true
	}
	return false
}

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
		} else if val, ok := interp.LookupMQLConstant(e.Name); ok {
			cid := c.addConst(val)
			c.emit(OP_PUSH_CONST, int32(cid), 0, 0)
		} else {
			// Unknown constant — compile error, not silent push 0.
			if c.err == nil {
				c.err = fmt.Errorf("unknown constant: %s (not in MQL predefined constants or user enums)", e.Name)
			}
			cid := c.addConst(interp.IntVal(0))
			c.emit(OP_PUSH_CONST, int32(cid), 0, 0)
		}

	case interp.ExprBinary:
		c.compileBinary(e)

	case interp.ExprUnary:
		if len(e.Args) == 0 {
			if c.err == nil {
				c.err = fmt.Errorf("unary operator %q has no operand", e.Op)
			}
			return
		}
		c.compileExpr(&e.Args[0])
		switch e.Op {
		case "-":
			c.emit(OP_NEG, 0, 0, 0)
		case "!":
			c.emit(OP_NOT, 0, 0, 0)
		case "+":
			// no-op
		default:
			if c.err == nil {
				c.err = fmt.Errorf("unsupported unary operator: %s", e.Op)
			}
		}

	case interp.ExprCall:
		c.compileCall(e)

	case interp.ExprSubscript:
		c.compileSubscript(e)

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
		c.compileUpdate(e)

	case interp.ExprAssignment:
		c.compileExpr(&e.Args[0])
		slot, isGlobal := c.resolveVar(e.Name)
		if isGlobal {
			c.emit(OP_STORE_GLOBAL, int32(slot), 0, 0)
		} else {
			c.emit(OP_STORE_VAR, int32(slot), 0, 0)
		}

	case interp.ExprCompoundAssign:
		c.compileCompoundAssign(e)

	case interp.ExprDecl:
		c.compileDecl(e)

	case interp.ExprSeq:
		// Evaluate all children in order; only the last leaves a value on stack.
		for i := range e.Args {
			if i < len(e.Args)-1 {
				c.compileExpr(&e.Args[i])
				// Stack-neutral expressions (decl, assignment) don't leave a value.
				if !isStackNeutral(&e.Args[i]) {
					c.emit(OP_POP, 0, 0, 0)
				}
			} else {
				c.compileExpr(&e.Args[i])
			}
		}
	}
}

func (c *astCompiler) compileSubscript(e *interp.Expr) {
	if len(e.Args) > 0 {
		c.compileExpr(&e.Args[0])
		c.compileExpr(e.Index)
		slot, isGlobal := c.resolveVar(e.Name)
		if !isGlobal {
			c.bc.Coverage.AddBlindSpot("local array write: " + e.Name)
		}
		c.emit(OP_STORE_ARRAY, int32(slot), 0, 0)
		return
	}
	if isSeriesName(e.Name) {
		c.compileExpr(e.Index)
		c.emit(OP_PUSH_SERIES, 0, int32(c.addSeriesName(e.Name)), 0)
	} else {
		c.compileExpr(e.Index)
		slot, isGlobal := c.resolveVar(e.Name)
		if !isGlobal {
			c.bc.Coverage.AddBlindSpot("local array read: " + e.Name)
		}
		c.emit(OP_PUSH_ARRAY, int32(slot), 0, 0)
	}
}

func (c *astCompiler) compileUpdate(e *interp.Expr) {
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
}

func (c *astCompiler) compileCompoundAssign(e *interp.Expr) {
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
}

func (c *astCompiler) compileDecl(e *interp.Expr) {
	if len(e.Args) == 0 {
		if c.err == nil {
			c.err = fmt.Errorf("declaration %q has no initializer", e.Name)
		}
		c.emit(OP_PUSH_CONST, int32(c.addConst(interp.NoneVal())), 0, 0)
	} else {
		c.compileExpr(&e.Args[0])
	}
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

func (c *astCompiler) compileBinary(e *interp.Expr) {
	if len(e.Args) != 2 {
		if c.err == nil {
			c.err = fmt.Errorf("binary operator %q requires two operands", e.Op)
		}
		return
	}

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
