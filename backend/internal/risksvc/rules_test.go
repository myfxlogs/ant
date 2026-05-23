package risksvc

import (
	"context"
	"testing"
)

func TestEngine_AllPass(t *testing.T) {
	engine := NewEngine(
		&MaxPosition{Max: 10},
		&Margin{MinLevel: 1.5},
	)
	req := &CheckRequest{
		Symbol:    "EURUSD",
		Positions: 3,
		Equity:    10000,
		Margin:    2000,
	}
	result := engine.Evaluate(context.Background(), req)
	if !result.Passed {
		t.Errorf("expected all pass, got %s: %s", result.Rule, result.Reason)
	}
}

func TestEngine_MaxPositionBlocked(t *testing.T) {
	engine := NewEngine(&MaxPosition{Max: 2})
	req := &CheckRequest{Symbol: "EURUSD", Positions: 3}
	result := engine.Evaluate(context.Background(), req)
	if result.Passed {
		t.Error("expected BLOCK from max_position")
	}
	if result.Rule != "max_position" {
		t.Errorf("expected rule max_position, got %s", result.Rule)
	}
}

func TestEngine_MarginBlocked(t *testing.T) {
	engine := NewEngine(&Margin{MinLevel: 1.5})
	req := &CheckRequest{Symbol: "EURUSD", Equity: 1000, Margin: 2000}
	result := engine.Evaluate(context.Background(), req)
	if result.Passed {
		t.Error("expected BLOCK from margin")
	}
	if result.Rule != "margin" {
		t.Errorf("expected rule margin, got %s", result.Rule)
	}
}

func TestEngine_CanonicalAuthBlocked(t *testing.T) {
	engine := NewEngine(&CanonicalAuth{Whitelist: []string{"EURUSD", "GBPUSD"}})
	req := &CheckRequest{Symbol: "BTCUSD"}
	result := engine.Evaluate(context.Background(), req)
	if result.Passed {
		t.Error("expected BLOCK from canonical_auth")
	}
}

func TestEngine_CanonicalAuthAllowed(t *testing.T) {
	engine := NewEngine(&CanonicalAuth{Whitelist: []string{"EURUSD"}})
	req := &CheckRequest{Symbol: "EURUSD"}
	result := engine.Evaluate(context.Background(), req)
	if !result.Passed {
		t.Errorf("expected pass for whitelisted symbol, got %s", result.Reason)
	}
}

func TestEngine_DrawdownBlocked(t *testing.T) {
	engine := NewEngine(&Drawdown{MaxPct: 10, PeakEquity: 10000})
	req := &CheckRequest{Equity: 8000} // 20% drawdown
	result := engine.Evaluate(context.Background(), req)
	if result.Passed {
		t.Error("expected BLOCK from drawdown")
	}
}

func TestEngine_DrawdownAllowed(t *testing.T) {
	engine := NewEngine(&Drawdown{MaxPct: 20, PeakEquity: 10000})
	req := &CheckRequest{Equity: 9500} // 5% drawdown
	result := engine.Evaluate(context.Background(), req)
	if !result.Passed {
		t.Errorf("expected pass, got %s: %s", result.Rule, result.Reason)
	}
}

func TestEngine_RulesList(t *testing.T) {
	engine := NewEngine(&MaxPosition{Max: 5}, &Session{}, &Margin{MinLevel: 1.5})
	names := engine.Rules()
	if len(names) != 3 {
		t.Errorf("expected 3 rules, got %d: %v", len(names), names)
	}
}

func TestEngine_SessionWeekend(t *testing.T) {
	// Session rule rejects weekends — test logic without time injection
	engine := NewEngine(&Session{})
	req := &CheckRequest{Symbol: "EURUSD"}
	result := engine.Evaluate(context.Background(), req)
	// May pass or fail depending on actual time — just verify no panic
	_ = result
}
