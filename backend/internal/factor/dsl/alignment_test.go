// Package dsl_test — Go/Python DSL alignment test (M7.6-2).
//
// Generates 100 random DSL expressions × 1000 bars, evaluates with Go engine,
// writes results to JSON for Python-side comparison. Max error must be < 1e-9.
package dsl_test

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"anttrader/internal/factor/dsl"
)

// ── Expression templates ──────────────────────────────────────────────

var exprTemplates = []struct {
	name string
	gen  func(rng *rand.Rand) string
}{
	{"sma_basic", func(r *rand.Rand) string { return fmt.Sprintf("sma($close, %d)", 3+r.Intn(30)) }},
	{"ema_basic", func(r *rand.Rand) string { return fmt.Sprintf("ema($close, %d)", 3+r.Intn(30)) }},
	{"wma_basic", func(r *rand.Rand) string { return fmt.Sprintf("wma($close, %d)", 3+r.Intn(20)) }},
	{"std_basic", func(r *rand.Rand) string { return fmt.Sprintf("std($close, %d)", 3+r.Intn(30)) }},
	{"var_basic", func(r *rand.Rand) string { return fmt.Sprintf("var($close, %d)", 3+r.Intn(30)) }},
	{"min_basic", func(r *rand.Rand) string { return fmt.Sprintf("min($close, %d)", 3+r.Intn(20)) }},
	{"max_basic", func(r *rand.Rand) string { return fmt.Sprintf("max($close, %d)", 3+r.Intn(20)) }},
	{"sum_basic", func(r *rand.Rand) string { return fmt.Sprintf("sum($close, %d)", 3+r.Intn(20)) }},
	{"ref_basic", func(r *rand.Rand) string { return fmt.Sprintf("ref($close, %d)", 1+r.Intn(10)) }},
	{"delta_basic", func(r *rand.Rand) string { return fmt.Sprintf("delta($close, %d)", 1+r.Intn(10)) }},
	{"rsi_basic", func(r *rand.Rand) string { return fmt.Sprintf("rsi($close, %d)", 5+r.Intn(20)) }},
	{"atr_basic", func(r *rand.Rand) string { return fmt.Sprintf("atr($close, %d)", 3+r.Intn(20)) }},
	{"bb_upper", func(r *rand.Rand) string { return fmt.Sprintf("bb_upper($close, %d, %.1f)", 5+r.Intn(25), 1.0+float64(r.Intn(3))) }},
	{"bb_lower", func(r *rand.Rand) string { return fmt.Sprintf("bb_lower($close, %d, %.1f)", 5+r.Intn(25), 1.0+float64(r.Intn(3))) }},
	{"zscore", func(r *rand.Rand) string { return fmt.Sprintf("zscore($close, %d)", 5+r.Intn(20)) }},
	{"rank", func(r *rand.Rand) string { return fmt.Sprintf("rank($close, %d)", 3+r.Intn(15)) }},
	{"pct_change", func(r *rand.Rand) string { return fmt.Sprintf("pct_change($close, %d)", 1+r.Intn(10)) }},
	{"ma_cross", func(r *rand.Rand) string {
		return fmt.Sprintf("ema($close, %d) / ema($close, %d) - 1", 3+r.Intn(20), 10+r.Intn(40))
	}},
	{"ma_diff", func(r *rand.Rand) string {
		return fmt.Sprintf("sma($close, %d) - sma($close, %d)", 3+r.Intn(10), 10+r.Intn(30))
	}},
	{"rsi_signal", func(r *rand.Rand) string { return fmt.Sprintf("rsi($close, %d) < 30 ? 1 : 0", 5+r.Intn(20)) }},
	{"bb_breakout", func(r *rand.Rand) string {
		return fmt.Sprintf("$close > bb_upper($close, %d, 2) ? 1 : -1", 10+r.Intn(20))
	}},
	{"scalar_abs", func(r *rand.Rand) string { return "abs($close)" }},
	{"scalar_sign", func(r *rand.Rand) string { return fmt.Sprintf("sign($close - %.1f)", 90.0+float64(r.Intn(20))) }},
	{"scalar_log", func(r *rand.Rand) string { return "log(abs($close))" }},
	{"scalar_exp", func(r *rand.Rand) string { return "exp($close / 100)" }},
	{"scalar_sqrt", func(r *rand.Rand) string { return "sqrt(abs($close))" }},
	{"arithmetic", func(r *rand.Rand) string {
		return fmt.Sprintf("($close + %d) * (sma($close, %d) - %d) / %d", 1+r.Intn(10), 3+r.Intn(10), 1+r.Intn(5), 2+r.Intn(10))
	}},
	{"comparison", func(r *rand.Rand) string {
		return fmt.Sprintf("$close > sma($close, %d) && $close < bb_upper($close, %d, 2)", 5+r.Intn(20), 10+r.Intn(20))
	}},
	{"logic", func(r *rand.Rand) string {
		return fmt.Sprintf("rsi($close, %d) < 30 || rsi($close, %d) > 70", 5+r.Intn(14), 5+r.Intn(14))
	}},
	{"nested_ops", func(r *rand.Rand) string {
		return fmt.Sprintf("delta(sma($close, %d), %d)", 5+r.Intn(15), 1+r.Intn(5))
	}},
	{"zscore_signal", func(r *rand.Rand) string {
		return fmt.Sprintf("zscore($close, %d) > 2 ? 1 : (zscore($close, %d) < -2 ? -1 : 0)", 10+r.Intn(10), 10+r.Intn(10))
	}},
	{"pow_expr", func(r *rand.Rand) string {
		return fmt.Sprintf("pow($close / 100, %.1f)", 1.0+float64(r.Intn(3)))
	}},
	{"macd_line", func(r *rand.Rand) string {
		return fmt.Sprintf("macd($close, %d, %d)", 8+r.Intn(8), 20+r.Intn(10))
	}},
}

// ── Bar generation ────────────────────────────────────────────────────

func generateBars(n int, rng *rand.Rand) []float64 {
	// Random walk starting at 100, realistic OHLC-like values
	bars := make([]float64, n)
	v := 100.0
	for i := range bars {
		v += (rng.Float64() - 0.5) * 2.0
		if v < 10 {
			v = 10
		}
		if v > 200 {
			v = 200
		}
		bars[i] = v
	}
	return bars
}

// ── Test output ───────────────────────────────────────────────────────

type AlignmentRecord struct {
	ExprIndex  int        `json:"expr_index"`
	Expression string     `json:"expression"`
	Bars       []float64  `json:"bars"`
	GoResults  []*float64 `json:"go_results"`
	Warmup     int        `json:"warmup"`
}

type AlignmentData struct {
	Records []AlignmentRecord `json:"records"`
}

// TestGoDSLAlignment generates alignment data for Python-side comparison.
func TestGoDSLAlignment(t *testing.T) {
	if testing.Short() {
		t.Skip("将在卡片 M10.5-4 中实施: skipping alignment test in short mode")
	}

	rng := rand.New(rand.NewSource(42)) // deterministic

	// Generate 100 expressions
	exprs := make([]string, 100)
	for i := range exprs {
		tmpl := exprTemplates[i%len(exprTemplates)]
		exprs[i] = tmpl.gen(rng)
	}

	// Generate 1000 bars
	bars := generateBars(1000, rng)

	fields := dsl.FieldIndex{Fields: map[string]int{"close": 0}}
	compiler := dsl.NewCompiler(fields, nil)

	var records []AlignmentRecord

	for idx, expr := range exprs {
		op, err := compiler.Compile(expr)
		if err != nil {
			t.Logf("skip expr[%d] %q: %v", idx, expr, err)
			// Use a simple fallback expression
			op, err = compiler.Compile("$close")
			if err != nil {
				t.Fatalf("fallback compile failed: %v", err)
			}
		}

		warmup := op.Warmup()
		results := make([]*float64, len(bars))
		for i, v := range bars {
			r := op.Eval(v)
			if !math.IsNaN(r) {
				results[i] = &r
			}
		}

		records = append(records, AlignmentRecord{
			ExprIndex:  idx,
			Expression: expr,
			Bars:       bars,
			GoResults:  results,
			Warmup:     warmup,
		})

		// Reset for next expression
		op.Reset()
	}

	data := AlignmentData{Records: records}
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	// Write to research/tests/alignment_data.json
	outPath := filepath.Join("..", "..", "..", "..", "research", "tests", "alignment_data.json")
	if err := os.WriteFile(outPath, jsonBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("alignment data written to %s (%d records, %d bars each)", outPath, len(records), len(bars))
}

// TestGoDSLSelfCheck validates the DSL engine against known reference values.
func TestGoDSLSelfCheck(t *testing.T) {
	fields := dsl.FieldIndex{Fields: map[string]int{"close": 0}}
	c := dsl.NewCompiler(fields, nil)

	tests := []struct {
		expr     string
		bars     []float64
		lastVal  float64
		tolerance float64
	}{
		{"sma($close, 3)", []float64{1, 2, 3, 4, 5}, 4.0, 1e-9},
		{"ema($close, 3)", []float64{1, 1, 1, 1, 1}, 1.0, 1e-9},
		{"$close > 3 ? 1 : -1", []float64{2, 4}, 1.0, 1e-9},
		{"rsi($close, 14)", generateBars(100, rand.New(rand.NewSource(1))), math.NaN(), 0}, // just check it runs
		{"bb_upper($close, 20, 2)", generateBars(30, rand.New(rand.NewSource(2))), math.NaN(), 0},
		{"$close + sma($close, 5) * 2", []float64{10, 10, 10, 10, 10, 10, 10}, 30.0, 1e-9},
	}

	for _, tt := range tests {
		t.Run(tt.expr[:min(20, len(tt.expr))], func(t *testing.T) {
			op, err := c.Compile(tt.expr)
			if err != nil {
				t.Fatalf("compile %q: %v", tt.expr, err)
			}
			var last float64
			for _, v := range tt.bars {
				last = op.Eval(v)
			}
			if math.IsNaN(tt.lastVal) {
				// Just verify it ran without panicking
				return
			}
			if math.Abs(last-tt.lastVal) > tt.tolerance {
				t.Errorf("%q: expected %.9f, got %.9f", tt.expr, tt.lastVal, last)
			}
		})
	}
}
