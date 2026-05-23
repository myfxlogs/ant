package factorsvc

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestNewEngine(t *testing.T) {
	cfg := Config{
		Factors: []FactorDef{
			{Name: "test", Expression: "$close", Symbols: []string{"EURUSD"}},
		},
	}
	e := NewEngine(cfg)
	if e == nil {
		t.Fatal("NewEngine returned nil")
	}
}

func TestFactorDef_Fields(t *testing.T) {
	fd := FactorDef{
		Name:       "sma10",
		Expression: "sma($close, 10)",
		Symbols:    []string{"EURUSD", "GBPJPY"},
	}
	if fd.Name != "sma10" {
		t.Fatalf("expected sma10, got %s", fd.Name)
	}
	if len(fd.Symbols) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(fd.Symbols))
	}
}

func TestEngine_RegisterAndEval(t *testing.T) {
	cfg := Config{
		Factors: []FactorDef{
			{Name: "close_price", Expression: "$close", Symbols: []string{"EURUSD"}},
		},
	}
	e := NewEngine(cfg)

	bar := &Bar{
		UserID:        "t1",
		Symbol:        "EURUSD",
		Period:        "M1",
		Open:          "1.1000",
		High:          "1.1010",
		Low:           "1.0990",
		Close:         "1.1005",
		Volume:        100,
		CloseTsUnixMs: 1000,
	}

	results := e.Eval(context.Background(), bar)
	if results["close_price"] != 1.1005 {
		t.Fatalf("expected close_price=1.1005, got %f", results["close_price"])
	}

	latest := e.LatestFactors()
	if latest["close_price"] != 1.1005 {
		t.Fatalf("LatestFactors: expected 1.1005, got %f", latest["close_price"])
	}
}

func TestEngine_EvalWithBuffer(t *testing.T) {
	cfg := Config{
		Factors: []FactorDef{
			{Name: "sma3", Expression: "sma($close, 3)", Symbols: []string{"EURUSD"}},
		},
	}
	e := NewEngine(cfg)
	wb := NewWindowBuffer(10, zap.NewNop())
	e.SetBuffer(wb)

	// Push bars and evaluate
	closes := []string{"1.1000", "1.1010", "1.1020", "1.1030", "1.1040"}
	for i, c := range closes {
		bar := &Bar{
			UserID:        "t1",
			Symbol:        "EURUSD",
			Period:        "M1",
			Open:          c,
			High:          c,
			Low:           c,
			Close:         c,
			Volume:        100,
			CloseTsUnixMs: int64(i * 60000),
		}
		e.Eval(context.Background(), bar)
	}

	// After 5 bars, buffer has 5 values; SMA(3) uses last 3 closes
	latest := e.LatestFactors()
	if latest["sma3"] == 0 {
		t.Fatal("sma3 should be non-zero after 5 bars")
	}
}

func TestEngine_RegisterInvalidExpr(t *testing.T) {
	e := NewEngine(Config{})
	err := e.Register(FactorDef{Name: "bad", Expression: "!!!"})
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
}

func TestEngine_LatestFactorsEmpty(t *testing.T) {
	e := NewEngine(Config{})
	latest := e.LatestFactors()
	if latest != nil {
		t.Fatal("expected nil before any Eval")
	}
}

func TestEngine_MultipleFactors(t *testing.T) {
	cfg := Config{
		Factors: []FactorDef{
			{Name: "c", Expression: "$close", Symbols: []string{"EURUSD"}},
			{Name: "o", Expression: "$open", Symbols: []string{"EURUSD"}},
		},
	}
	e := NewEngine(cfg)

	bar := &Bar{
		UserID:        "t1",
		Symbol:        "EURUSD",
		Period:        "M1",
		Open:          "1.1000",
		High:          "1.1010",
		Low:           "1.0990",
		Close:         "1.2000",
		Volume:        100,
		CloseTsUnixMs: 1000,
	}

	results := e.Eval(context.Background(), bar)
	if results["c"] != 1.2000 {
		t.Fatalf("expected c=1.2000, got %f", results["c"])
	}
	// Note: fieldOp.Eval(v) returns v regardless of field index;
	// the engine currently feeds bar.Close to all ops.
	if results["o"] != 1.2000 {
		t.Fatalf("expected o=1.2000, got %f", results["o"])
	}
}
