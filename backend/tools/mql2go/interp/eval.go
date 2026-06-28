package interp

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Evaluator handles expression evaluation within an Interpreter.
// It operates on pure Go Expr trees — no tree-sitter dependency.

// evalExpr evaluates an Expr and returns a Value.
func (it *Interpreter) evalExpr(e *Expr) Value {
	if e == nil {
		return NoneVal()
	}
	switch e.Kind {
	case ExprLiteral:
		return e.Val

	case ExprVar:
		return it.getVar(e.Name)

	case ExprConst:
		return it.lookupConstant(e.Name)

	case ExprBinary:
		left := it.evalExpr(&e.Args[0])
		right := it.evalExpr(&e.Args[1])
		return it.applyOp(left, right, e.Op)

	case ExprUnary:
		val := it.evalExpr(&e.Args[0])
		return it.applyUnary(val, e.Op)

	case ExprCall:
		return it.callBuiltin(e.Name, e.Args)

	case ExprSubscript:
		return it.evalSubscript(e)

	case ExprField:
		return it.evalField(e)

	case ExprTernary:
		if it.evalExpr(e.Cond).IsTrue() {
			return it.evalExpr(e.ThenExpr)
		}
		return it.evalExpr(e.ElseExpr)

	case ExprAssignment:
		val := it.evalExpr(&e.Args[0])
		it.setVar(e.Name, val)
		return val

	case ExprCompoundAssign:
		cur := it.getVar(e.Name)
		rhs := it.evalExpr(&e.Args[0])
		var result Value
		switch e.Op {
		case "+=":
			if cur.Kind == ValString || rhs.Kind == ValString {
				result = StringVal(cur.ToString() + rhs.ToString())
			} else {
				result = DecimalVal(cur.ToDecimal().Add(rhs.ToDecimal()))
			}
		case "-=":
			result = DecimalVal(cur.ToDecimal().Sub(rhs.ToDecimal()))
		case "*=":
			result = DecimalVal(cur.ToDecimal().Mul(rhs.ToDecimal()))
		case "/=":
			d := rhs.ToDecimal()
			if d.IsZero() {
				result = DecimalVal(decimal.Zero)
			} else {
				result = DecimalVal(cur.ToDecimal().Div(d))
			}
		case "%=":
			r := rhs.ToInt()
			if r == 0 {
				result = IntVal(0)
			} else {
				result = IntVal(cur.ToInt() % r)
			}
		default:
			result = rhs
		}
		it.setVar(e.Name, result)
		return result

	case ExprUpdate:
		val := it.getVar(e.Name)
		one := decimal.NewFromInt(1)
		if e.Op == "++" {
			val = Value{Kind: ValDecimal, Decimal: val.ToDecimal().Add(one)}
		} else {
			val = Value{Kind: ValDecimal, Decimal: val.ToDecimal().Sub(one)}
		}
		it.setVar(e.Name, val)
		return val
	}
	return NoneVal()
}

// applyOp applies a binary operator to two Values.
func (it *Interpreter) applyOp(left, right Value, op string) Value {
	switch op {
	case "+":
		if left.Kind == ValString || right.Kind == ValString {
			return StringVal(left.ToString() + right.ToString())
		}
		return DecimalVal(left.ToDecimal().Add(right.ToDecimal()))
	case "-":
		return DecimalVal(left.ToDecimal().Sub(right.ToDecimal()))
	case "*":
		return DecimalVal(left.ToDecimal().Mul(right.ToDecimal()))
	case "/":
		d := right.ToDecimal()
		if d.IsZero() {
			return DecimalVal(decimal.Zero)
		}
		return DecimalVal(left.ToDecimal().Div(d))
	case "%":
		l := left.ToInt()
		r := right.ToInt()
		if r == 0 {
			return IntVal(0)
		}
		return IntVal(l % r)
	case "==":
		return BoolVal(left.Equal(right))
	case "!=":
		return BoolVal(!left.Equal(right))
	case "<":
		return BoolVal(left.ToDecimal().LessThan(right.ToDecimal()))
	case ">":
		return BoolVal(left.ToDecimal().GreaterThan(right.ToDecimal()))
	case "<=":
		return BoolVal(left.ToDecimal().LessThanOrEqual(right.ToDecimal()))
	case ">=":
		return BoolVal(left.ToDecimal().GreaterThanOrEqual(right.ToDecimal()))
	case "&&":
		return BoolVal(left.IsTrue() && right.IsTrue())
	case "||":
		return BoolVal(left.IsTrue() || right.IsTrue())
	case "&":
		return IntVal(left.ToInt() & right.ToInt())
	case "|":
		return IntVal(left.ToInt() | right.ToInt())
	case "^":
		return IntVal(left.ToInt() ^ right.ToInt())
	case "<<":
		return IntVal(left.ToInt() << uint(right.ToInt()))
	case ">>":
		return IntVal(left.ToInt() >> uint(right.ToInt()))
	}
	return NoneVal()
}

// applyUnary applies a unary operator to a Value.
func (it *Interpreter) applyUnary(val Value, op string) Value {
	switch op {
	case "-":
		return DecimalVal(val.ToDecimal().Neg())
	case "!":
		return BoolVal(!val.IsTrue())
	case "~":
		return IntVal(^val.ToInt())
	case "+":
		return val
	}
	return NoneVal()
}

// evalSubscript handles array/series access: Close[1], High[shift], etc.
func (it *Interpreter) evalSubscript(e *Expr) Value {
	idx := it.evalExpr(e.Index)
	shift := int(idx.ToInt())

	// Time series access: Close, Open, High, Low, Volume, Time
	switch e.Name {
	case "Close", "close":
		return it.series.Close(shift)
	case "Open", "open":
		return it.series.Open(shift)
	case "High", "high":
		return it.series.High(shift)
	case "Low", "low":
		return it.series.Low(shift)
	case "Volume", "volume":
		return it.series.Volume(shift)
	case "Time", "time":
		return it.series.Time(shift)
	}

	// User array access
	if arr, ok := it.getArray(e.Name); ok {
		if shift >= 0 && shift < len(arr) {
			return arr[shift]
		}
	}
	return NoneVal()
}

// evalField handles object field/method access: obj.field, obj.method(args), obj.field = value
func (it *Interpreter) evalField(e *Expr) Value {
	if len(e.Args) == 0 {
		return NoneVal()
	}
	obj := it.evalExpr(&e.Args[0])
	if obj.Kind != ValClass || obj.Class == nil {
		return NoneVal()
	}

	// Field assignment: obj.field = value (IsAssign=true, last arg is the value)
	if e.IsAssign && len(e.Args) > 1 {
		val := it.evalExpr(&e.Args[len(e.Args)-1])
		obj.Class.Fields[e.Name] = val
		return val
	}

	// Method call (has additional args beyond the object)
	if len(e.Args) > 1 {
		return it.dispatchClassMethod(obj.Class, e.Name, e.Args[1:])
	}

	// Field read
	if val, ok := obj.Class.Fields[e.Name]; ok {
		return val
	}
	return NoneVal()
}

// lookupConstant resolves MQL predefined constants.
func (it *Interpreter) lookupConstant(name string) Value {
	if v, ok := mqlConstants[name]; ok {
		return v
	}
	// Check enum constants
	if it.ir != nil && it.ir.Enums != nil {
		if v, ok := it.ir.Enums[name]; ok {
			return IntVal(v)
		}
	}
	// Check if it's a global variable
	if v, ok := it.globals[name]; ok {
		return v
	}
	return NoneVal()
}

// MQL predefined constants
var mqlConstants = map[string]Value{
	// Order types
	"OP_BUY":         IntVal(0),
	"OP_SELL":        IntVal(1),
	"OP_BUYLIMIT":    IntVal(2),
	"OP_SELLLIMIT":   IntVal(3),
	"OP_BUYSTOP":     IntVal(4),
	"OP_SELLSTOP":    IntVal(5),
	// Selection modes
	"SELECT_BY_POS":    IntVal(0),
	"SELECT_BY_TICKET": IntVal(1),
	"MODE_TRADES":      IntVal(0),
	"MODE_HISTORY":     IntVal(1),
	// Applied price
	"PRICE_CLOSE":    IntVal(1),
	"PRICE_OPEN":     IntVal(2),
	"PRICE_HIGH":     IntVal(3),
	"PRICE_LOW":      IntVal(4),
	"PRICE_MEDIAN":   IntVal(5),
	"PRICE_TYPICAL":  IntVal(6),
	"PRICE_WEIGHTED": IntVal(7),
	// Timeframes
	"PERIOD_M1":  IntVal(1),
	"PERIOD_M5":  IntVal(5),
	"PERIOD_M15": IntVal(15),
	"PERIOD_M30": IntVal(30),
	"PERIOD_H1":  IntVal(60),
	"PERIOD_H4":  IntVal(240),
	"PERIOD_D1":  IntVal(1440),
	"PERIOD_W1":  IntVal(10080),
	"PERIOD_MN1": IntVal(43200),
	// MQL5 trade actions
	"TRADE_ACTION_DEAL":     IntVal(0),
	"TRADE_ACTION_PENDING":  IntVal(1),
	"TRADE_ACTION_SLTP":     IntVal(2),
	"TRADE_ACTION_PEND_CLOSE": IntVal(3),
	// MQL5 order types
	"ORDER_TYPE_BUY":         IntVal(0),
	"ORDER_TYPE_SELL":        IntVal(1),
	"ORDER_TYPE_BUY_LIMIT":   IntVal(2),
	"ORDER_TYPE_SELL_LIMIT":  IntVal(3),
	"ORDER_TYPE_BUY_STOP":    IntVal(4),
	"ORDER_TYPE_SELL_STOP":   IntVal(5),
	// Empty/invalid
	"EMPTY":      IntVal(-1),
	"INVALID_HANDLE": IntVal(-1),
	"CLR_NONE":   IntVal(-1),
	// Booleans
	"true":  BoolVal(true),
	"false": BoolVal(false),
}

// getArray retrieves a user-defined array variable.
func (it *Interpreter) getArray(name string) ([]Value, bool) {
	v, ok := it.globals[name]
	if !ok {
		for i := len(it.scopes) - 1; i >= 0; i-- {
			if v, ok = it.scopes[i][name]; ok {
				break
			}
		}
	}
	if !ok || v.Kind != ValArray {
		return nil, false
	}
	return v.Array, true
}

// formatExpr returns a debug string for an Expr tree.
func formatExpr(e *Expr, depth int) string {
	if e == nil {
		return "nil"
	}
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	switch e.Kind {
	case ExprLiteral:
		return fmt.Sprintf("%sLiteral(%v)", indent, e.Val)
	case ExprVar:
		return fmt.Sprintf("%sVar(%s)", indent, e.Name)
	case ExprConst:
		return fmt.Sprintf("%sConst(%s)", indent, e.Name)
	case ExprBinary:
		return fmt.Sprintf("%sBinary(%s)\n%s\n%s", indent, e.Op,
			formatExpr(&e.Args[0], depth+1), formatExpr(&e.Args[1], depth+1))
	case ExprCall:
		return fmt.Sprintf("%sCall(%s, %d args)", indent, e.Name, len(e.Args))
	case ExprSubscript:
		return fmt.Sprintf("%sSubscript(%s)", indent, e.Name)
	default:
		return fmt.Sprintf("%sExpr(%d)", indent, e.Kind)
	}
}
