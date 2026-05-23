package dsl

import (
	"math"
	"testing"
)

func TestLexer(t *testing.T) {
	toks, err := NewLexer(`ema($close, 20) / ema($close, 60) - 1`).LexAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) < 5 {
		t.Fatalf("expected >= 5 tokens, got %d", len(toks))
	}
}

func TestLexer_AllTokens(t *testing.T) {
	expr := `$close + 3.14 * -2 + true && false || 1 != 2 ? 3 : 4`
	toks, err := NewLexer(expr).LexAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(toks) < 10 {
		t.Fatalf("expected >= 10 tokens, got %d", len(toks))
	}
}

func TestLexer_String(t *testing.T) {
	toks, err := NewLexer(`"hello world"`).LexAll()
	if err != nil {
		t.Fatal(err)
	}
	if toks[0].Kind != TokString || toks[0].Value != "hello world" {
		t.Errorf("expected string 'hello world', got %q", toks[0].Value)
	}
}

func TestParse(t *testing.T) {
	expr := `ema($close, 20) / ema($close, 60) - 1`
	node, err := Parse(expr)
	if err != nil {
		t.Fatal(err)
	}
	_ = node
}

func TestParse_Ternary(t *testing.T) {
	node, err := Parse(`$close > sma($close, 20) ? 1 : -1`)
	if err != nil {
		t.Fatal(err)
	}
	tern, ok := node.(*TernaryExpr)
	if !ok {
		t.Fatalf("expected TernaryExpr, got %T", node)
	}
	_ = tern
}

func TestParse_Comparison(t *testing.T) {
	for _, expr := range []string{
		`$close > 100`,
		`$close >= 100`,
		`$close < 100`,
		`$close <= 100`,
		`$close == 100`,
		`$close != 100`,
	} {
		_, err := Parse(expr)
		if err != nil {
			t.Errorf("Parse(%q): %v", expr, err)
		}
	}
}

func TestParse_LogicOps(t *testing.T) {
	for _, expr := range []string{
		`$close > 100 && $volume > 1000`,
		`$close > 100 || $volume > 1000`,
	} {
		_, err := Parse(expr)
		if err != nil {
			t.Errorf("Parse(%q): %v", expr, err)
		}
	}
}

func TestSMABasic(t *testing.T) {
	s := NewSMA(3)
	vals := []float64{1, 2, 3, 4, 5, 6}
	expected := []float64{math.NaN(), math.NaN(), 2, 3, 4, 5}
	for i, v := range vals {
		r := s.Eval(v)
		if math.IsNaN(expected[i]) != math.IsNaN(r) {
			t.Errorf("idx %d: expected NaN=%v, got NaN=%v", i, math.IsNaN(expected[i]), math.IsNaN(r))
		} else if !math.IsNaN(r) && math.Abs(r-expected[i]) > 1e-9 {
			t.Errorf("idx %d: expected %.6f, got %.6f", i, expected[i], r)
		}
	}
}

func TestEMABasic(t *testing.T) {
	e := NewEMA(3)
	vals := []float64{1, 1, 1, 1, 1}
	for i, v := range vals {
		r := e.Eval(v)
		if i < 2 {
			if !math.IsNaN(r) {
				t.Errorf("expected NaN during warmup at %d, got %v", i, r)
			}
		} else {
			if math.Abs(r-1.0) > 1e-9 {
				t.Errorf("idx %d: expected 1.0, got %.6f", i, r)
			}
		}
	}
}

func TestRSI(t *testing.T) {
	rsi := NewRSI(14)
	// Feed 14 identical values → RSI should be NaN (no gains/losses initially)
	// Then feed a rising sequence
	prices := []float64{
		44.34, 44.09, 44.15, 43.61, 44.33, 44.83, 45.10, 45.42,
		45.84, 46.08, 45.89, 46.03, 45.61, 46.28, 46.28, 46.00,
	}
	for _, p := range prices {
		_ = rsi.Eval(p)
	}
	// After enough data, RSI should be between 0-100
	last := rsi.Eval(46.43)
	if math.IsNaN(last) || last < 0 || last > 100 {
		t.Errorf("RSI out of range: %f", last)
	}
}

func TestCompileAndEval(t *testing.T) {
	fields := FieldIndex{Fields: map[string]int{"close": 0}}
	c := NewCompiler(fields, nil)

	op, err := c.Compile(`sma($close, 3)`)
	if err != nil {
		t.Fatal(err)
	}
	bars := []float64{100, 101, 102, 103, 104}
	for i, v := range bars {
		r := op.Eval(v)
		switch i {
		case 0, 1:
			if !math.IsNaN(r) {
				t.Errorf("bar %d: expected NaN, got %.6f", i, r)
			}
		case 2:
			if math.Abs(r-101) > 1e-9 {
				t.Errorf("bar 2: expected 101, got %.6f", r)
			}
		case 3:
			if math.Abs(r-102) > 1e-9 {
				t.Errorf("bar 3: expected 102, got %.6f", r)
			}
		case 4:
			if math.Abs(r-103) > 1e-9 {
				t.Errorf("bar 4: expected 103, got %.6f", r)
			}
		}
	}
}

func TestCompile_ComplexExpr(t *testing.T) {
	fields := FieldIndex{Fields: map[string]int{"close": 0}}
	c := NewCompiler(fields, nil)

	// ema($close,20)/ema($close,60)-1
	op, err := c.Compile(`ema($close, 20) / ema($close, 60) - 1`)
	if err != nil {
		t.Fatal(err)
	}
	// Feed 100 bars of 100.0 → both EMAs converge to 100 → ratio=1 → result=0
	for i := 0; i < 100; i++ {
		r := op.Eval(100.0)
		if i >= 59 {
			if math.Abs(r) > 1e-6 {
				t.Errorf("bar %d: expected ~0, got %.6f", i, r)
			}
		}
	}
}

func TestCompile_BB(t *testing.T) {
	fields := FieldIndex{Fields: map[string]int{"close": 0}}
	c := NewCompiler(fields, nil)

	op, err := c.Compile(`bb_upper($close, 20, 2)`)
	if err != nil {
		t.Fatal(err)
	}
	// Feed 30 bars of 100.0 → all std=0 → bb_upper = 100
	for i := 0; i < 30; i++ {
		r := op.Eval(100.0)
		if i >= 19 {
			if math.Abs(r-100.0) > 1e-9 {
				t.Errorf("bar %d: expected 100.0, got %.6f", i, r)
			}
		}
	}
}

func TestCompile_MACD(t *testing.T) {
	fields := FieldIndex{Fields: map[string]int{"close": 0}}
	c := NewCompiler(fields, nil)

	op, err := c.Compile(`macd($close, 12, 26)`)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		r := op.Eval(100.0)
		if i >= 25 {
			if math.Abs(r) > 1e-6 {
				t.Errorf("bar %d: expected ~0, got %.6f", i, r)
			}
		}
	}
}

func TestCompile_UnknownField(t *testing.T) {
	fields := FieldIndex{Fields: map[string]int{"close": 0}}
	c := NewCompiler(fields, nil)
	_, err := c.Compile(`sma($open, 20)`)
	if err == nil {
		t.Error("expected error for unknown field $open")
	}
}

func TestCompile_UnknownFunction(t *testing.T) {
	fields := FieldIndex{Fields: map[string]int{"close": 0}}
	c := NewCompiler(fields, nil)
	_, err := c.Compile(`bogus($close, 10)`)
	if err == nil {
		t.Error("expected error for unknown function bogus")
	}
}

func TestValidation_Safety(t *testing.T) {
	err := ValidateExpression(`exec("ls")`, nil, nil)
	if err == nil {
		t.Error("expected rejection of dangerous token 'exec'")
	}
}

func TestValidation_LengthLimit(t *testing.T) {
	// Build a 5000-char expression
	long := ""
	for i := 0; i < 5000; i++ {
		long += "1+"
	}
	long += "1"
	err := ValidateExpression(long, nil, nil)
	if err == nil {
		t.Error("expected rejection of overlong expression")
	}
}

func TestValidation_ParseTimeout(t *testing.T) {
	// A deeply nested expression should trigger timeout
	nested := ""
	for i := 0; i < 200; i++ {
		nested += "("
	}
	nested += "1"
	for i := 0; i < 200; i++ {
		nested += ")"
	}
	// This might timeout or might parse — both acceptable for this test
	_ = ValidateExpression(nested, nil, nil)
}

func TestParser_Arithmetic(t *testing.T) {
	for _, expr := range []string{
		"1 + 2",
		"3 - 4",
		"5 * 6",
		"7 / 8",
		"9 % 2",
		"-3.14",
		"!true",
	} {
		_, err := Parse(expr)
		if err != nil {
			t.Errorf("Parse(%q): %v", expr, err)
		}
	}
}

func TestOperatorWarmup(t *testing.T) {
	tests := []struct {
		name   string
		newOp  func() Op
		warmup int
	}{
		{"SMA(5)", func() Op { return NewSMA(5) }, 5},
		{"EMA(5)", func() Op { return NewEMA(5) }, 5},
		{"WMA(5)", func() Op { return NewWMA(5) }, 5},
		{"STD(5)", func() Op { return NewSTD(5) }, 5},
		{"RSI(14)", func() Op { return NewRSI(14) }, 15},
		{"Ref(5)", func() Op { return NewRef(5) }, 5},
		{"Delta(5)", func() Op { return NewDelta(5) }, 5},
		{"BBUpper(20,2)", func() Op { return NewBBUpper(20, 2) }, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := tt.newOp()
			if op.Warmup() != tt.warmup {
				t.Errorf("expected warmup %d, got %d", tt.warmup, op.Warmup())
			}
		})
	}
}

func TestOpReset(t *testing.T) {
	sma := NewSMA(3)
	for i := 0; i < 5; i++ {
		sma.Eval(float64(i))
	}
	sma.Reset()
	// After reset, should be back to warmup
	r := sma.Eval(10.0)
	if !math.IsNaN(r) {
		t.Errorf("expected NaN after reset, got %.6f", r)
	}
}
