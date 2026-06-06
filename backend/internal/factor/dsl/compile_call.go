package dsl

import (
	"fmt"
	"strings"
)

// compileCall compiles a function call expression into an evaluable Op.
// Moved from compile.go for size compliance.
func (c *Compiler) compileCall(n *CallExpr) (Op, error) {
	switch strings.ToLower(n.Name) {
	case "sma":
		return c.newWindowOp(n.Args, func(n int) Op { return NewSMA(n) })
	case "ema":
		return c.newWindowOp(n.Args, func(n int) Op { return NewEMA(n) })
	case "wma":
		return c.newWindowOp(n.Args, func(n int) Op { return NewWMA(n) })
	case "std":
		return c.newWindowOp(n.Args, func(n int) Op { return NewSTD(n) })
	case "var":
		return c.newWindowOp(n.Args, func(n int) Op { return NewVAR(n) })
	case "min":
		return c.newWindowOp(n.Args, func(n int) Op { return NewMin(n) })
	case "max":
		return c.newWindowOp(n.Args, func(n int) Op { return NewMax(n) })
	case "sum":
		return c.newWindowOp(n.Args, func(n int) Op { return NewSum(n) })
	case "ref":
		return c.newWindowOp(n.Args, func(n int) Op { return NewRef(n) })
	case "delta":
		return c.newWindowOp(n.Args, func(n int) Op { return NewDelta(n) })
	case "pct_change":
		return c.newWindowOp(n.Args, func(n int) Op { return NewPctChange(n) })
	case "zscore":
		return c.newWindowOp(n.Args, func(n int) Op { return NewZScore(n) })
	case "rank":
		return c.newWindowOp(n.Args, func(n int) Op { return NewRank(n) })
	case "rsi":
		return c.newWindowOp(n.Args, func(n int) Op { return NewRSI(n) })
	case "atr":
		return c.newWindowOp(n.Args, func(n int) Op { return NewATR(n) })
	case "abs", "sign", "log", "exp", "sqrt":
		inner, err := c.compileNode(n.Args[0])
		if err != nil {
			return nil, err
		}
		return &scalarOp{name: n.Name, inner: inner}, nil
	case "pow":
		inner, err := c.compileNode(n.Args[0])
		if err != nil {
			return nil, err
		}
		expv, ok := n.Args[1].(*NumberLit)
		if !ok {
			return nil, fmt.Errorf("compile: pow second arg must be number literal")
		}
		return &powOp{inner: inner, exp: expv.Value}, nil
	case "if_":
		if len(n.Args) != 3 {
			return nil, fmt.Errorf("compile: if_ requires 3 args")
		}
		cond, err := c.compileNode(n.Args[0])
		if err != nil {
			return nil, err
		}
		a, err := c.compileNode(n.Args[1])
		if err != nil {
			return nil, err
		}
		b, err := c.compileNode(n.Args[2])
		if err != nil {
			return nil, err
		}
		return &ternaryOp{cond: cond, t: a, f: b}, nil
	case "macd":
		if len(n.Args) < 2 {
			return nil, fmt.Errorf("compile: macd requires at least 2 args (fast, slow)")
		}
		fast := argInt(n.Args, 0, 12)
		slow := argInt(n.Args, 1, 26)
		inner, err := c.compileNode(n.Args[0])
		if err != nil {
			return nil, err
		}
		macdOp := NewMACD(fast, slow)
		return &wrapOp{inner: inner, outer: macdOp}, nil
	case "bb_upper", "bb_lower":
		if len(n.Args) < 2 {
			return nil, fmt.Errorf("compile: %s requires at least 2 args", n.Name)
		}
		period := argInt(n.Args, 0, 20)
		k := argFloat(n.Args, 1, 2.0)
		inner, err := c.compileNode(n.Args[0])
		if err != nil {
			return nil, err
		}
		var outer Op
		if n.Name == "bb_upper" {
			outer = NewBBUpper(period, k)
		} else {
			outer = NewBBLower(period, k)
		}
		return &wrapOp{inner: inner, outer: outer}, nil
	default:
		return nil, fmt.Errorf("compile: unknown function %q", n.Name)
	}
}

func (c *Compiler) newWindowOp(args []Node, factory func(int) Op) (Op, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("compile: window op requires at least 1 arg")
	}
	n := argInt(args, 0, 14)
	inner, err := c.compileNode(args[0])
	if err != nil {
		return nil, err
	}
	outer := factory(n)
	return &wrapOp{inner: inner, outer: outer}, nil
}

func argInt(args []Node, idx, defaultVal int) int {
	if idx+1 >= len(args) {
		return defaultVal
	}
	if lit, ok := args[idx+1].(*NumberLit); ok {
		return int(lit.Value)
	}
	return defaultVal
}

func argFloat(args []Node, idx int, defaultVal float64) float64 {
	if idx+1 >= len(args) {
		return defaultVal
	}
	if lit, ok := args[idx+1].(*NumberLit); ok {
		return lit.Value
	}
	return defaultVal
}
