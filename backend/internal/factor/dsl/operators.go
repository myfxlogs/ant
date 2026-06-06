package dsl

import (
	"math"
)

// Compiled operator wrappers — moved from compile.go for size compliance.

type constOp struct{ val float64 }

func (c *constOp) Eval(v float64) float64 { return c.val }
func (c *constOp) Reset()                 {}
func (c *constOp) Warmup() int            { return 0 }

type fieldOp struct{ idx int }

func (f *fieldOp) Eval(v float64) float64 { return v }
func (f *fieldOp) Reset()                 {}
func (f *fieldOp) Warmup() int            { return 0 }

type binaryOp struct {
	op          string
	left, right Op
}

func (b *binaryOp) Eval(v float64) float64 {
	l := b.left.Eval(v)
	r := b.right.Eval(v)
	switch b.op {
	case "+":
		return l + r
	case "-":
		return l - r
	case "*":
		return l * r
	case "/":
		if r == 0 {
			return math.NaN()
		}
		return l / r
	case "%":
		return math.Mod(l, r)
	case "==":
		if l == r {
			return 1
		}
		return 0
	case "!=":
		if l != r {
			return 1
		}
		return 0
	case "<":
		if l < r {
			return 1
		}
		return 0
	case "<=":
		if l <= r {
			return 1
		}
		return 0
	case ">":
		if l > r {
			return 1
		}
		return 0
	case ">=":
		if l >= r {
			return 1
		}
		return 0
	case "&&":
		if l != 0 && r != 0 {
			return 1
		}
		return 0
	case "||":
		if l != 0 || r != 0 {
			return 1
		}
		return 0
	}
	return math.NaN()
}
func (b *binaryOp) Reset() {
	b.left.Reset()
	b.right.Reset()
}
func (b *binaryOp) Warmup() int {
	return nMax(b.left.Warmup(), b.right.Warmup())
}

type unaryOp struct {
	op    string
	inner Op
}

func (u *unaryOp) Eval(v float64) float64 {
	x := u.inner.Eval(v)
	switch u.op {
	case "-":
		return -x
	case "!":
		if x == 0 {
			return 1
		}
		return 0
	}
	return math.NaN()
}
func (u *unaryOp) Reset()      { u.inner.Reset() }
func (u *unaryOp) Warmup() int { return u.inner.Warmup() }

type ternaryOp struct {
	cond, t, f Op
}

func (t *ternaryOp) Eval(v float64) float64 {
	if t.cond.Eval(v) != 0 {
		return t.t.Eval(v)
	}
	return t.f.Eval(v)
}
func (t *ternaryOp) Reset() { t.cond.Reset(); t.t.Reset(); t.f.Reset() }
func (t *ternaryOp) Warmup() int {
	return nMax(t.cond.Warmup(), nMax(t.t.Warmup(), t.f.Warmup()))
}

type scalarOp struct {
	name  string
	inner Op
}

func (s *scalarOp) Eval(v float64) float64 {
	x := s.inner.Eval(v)
	switch s.name {
	case "abs":
		return Abs(x)
	case "sign":
		return Sign(x)
	case "log":
		return Log(x)
	case "exp":
		return Exp(x)
	case "sqrt":
		return Sqrt(x)
	}
	return math.NaN()
}
func (s *scalarOp) Reset()      { s.inner.Reset() }
func (s *scalarOp) Warmup() int { return s.inner.Warmup() }

type powOp struct {
	inner Op
	exp   float64
}

func (p *powOp) Eval(v float64) float64 { return Pow(p.inner.Eval(v), p.exp) }
func (p *powOp) Reset()                 { p.inner.Reset() }
func (p *powOp) Warmup() int            { return p.inner.Warmup() }

type wrapOp struct {
	inner, outer Op
}

func (w *wrapOp) Eval(v float64) float64 {
	x := w.inner.Eval(v)
	if math.IsNaN(x) {
		return math.NaN()
	}
	return w.outer.Eval(x)
}
func (w *wrapOp) Reset()      { w.inner.Reset(); w.outer.Reset() }
func (w *wrapOp) Warmup() int { return nMax(w.inner.Warmup(), w.outer.Warmup()) }

func nMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
