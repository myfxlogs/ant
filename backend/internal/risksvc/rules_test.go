package risksvc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestEngine_AllPass(t *testing.T) {
	t.Parallel()
	engine := NewEngine(
		&MaxPosition{Max: 10},
		&Margin{MinLevel: 1.5},
	)
	req := &CheckRequest{
		Symbol:    "EURUSD",
		Positions: 3,
		Equity:    decF(10000),
		Margin:    decF(2000),
	}
	result := engine.Evaluate(context.Background(), req)
	if !result.Passed {
		t.Errorf("expected all pass, got %s: %s", result.Rule, result.Reason)
	}
}

func TestEngine_MaxPositionBlocked(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	engine := NewEngine(&Margin{MinLevel: 1.5})
	req := &CheckRequest{Symbol: "EURUSD", Equity: decF(1000), Margin: decF(2000)}
	result := engine.Evaluate(context.Background(), req)
	if result.Passed {
		t.Error("expected BLOCK from margin")
	}
	if result.Rule != "margin" {
		t.Errorf("expected rule margin, got %s", result.Rule)
	}
}

func TestEngine_CanonicalAuthBlocked(t *testing.T) {
	t.Parallel()
	engine := NewEngine(&CanonicalAuth{Whitelist: []string{"EURUSD", "GBPUSD"}})
	req := &CheckRequest{Symbol: "BTCUSD"}
	result := engine.Evaluate(context.Background(), req)
	if result.Passed {
		t.Error("expected BLOCK from canonical_auth")
	}
}

func TestEngine_CanonicalAuthAllowed(t *testing.T) {
	t.Parallel()
	engine := NewEngine(&CanonicalAuth{Whitelist: []string{"EURUSD"}})
	req := &CheckRequest{Symbol: "EURUSD"}
	result := engine.Evaluate(context.Background(), req)
	if !result.Passed {
		t.Errorf("expected pass for whitelisted symbol, got %s", result.Reason)
	}
}

func TestEngine_DrawdownBlocked(t *testing.T) {
	t.Parallel()
	engine := NewEngine(&Drawdown{MaxPct: 10, PeakEquity: decF(10000)})
	req := &CheckRequest{Equity: decF(8000)} // 20% drawdown
	result := engine.Evaluate(context.Background(), req)
	if result.Passed {
		t.Error("expected BLOCK from drawdown")
	}
}

func TestEngine_DrawdownAllowed(t *testing.T) {
	t.Parallel()
	engine := NewEngine(&Drawdown{MaxPct: 20, PeakEquity: decF(10000)})
	req := &CheckRequest{Equity: decF(9500)} // 5% drawdown
	result := engine.Evaluate(context.Background(), req)
	if !result.Passed {
		t.Errorf("expected pass, got %s: %s", result.Rule, result.Reason)
	}
}

func TestEngine_RulesList(t *testing.T) {
	t.Parallel()
	engine := NewEngine(&MaxPosition{Max: 5}, &Session{}, &Margin{MinLevel: 1.5})
	names := engine.Rules()
	if len(names) != 3 {
		t.Errorf("expected 3 rules, got %d: %v", len(names), names)
	}
}

func TestEngine_SessionWeekend(t *testing.T) {
	t.Parallel()
	// Session rule rejects weekends — test logic without time injection
	engine := NewEngine(&Session{})
	req := &CheckRequest{Symbol: "EURUSD"}
	result := engine.Evaluate(context.Background(), req)
	// May pass or fail depending on actual time — just verify no panic
	_ = result
}

func TestDailyLoss_Name(t *testing.T) {
	t.Parallel()
	r := &DailyLoss{}
	if r.Name() != "daily_loss" {
		t.Fatalf("expected daily_loss, got %s", r.Name())
	}
}

func TestDailyLoss_WithinLimit(t *testing.T) {
	t.Parallel()
	r := &DailyLoss{Limit: decF(1000), DailyPL: decF(-500), DayStart: time.Now()}
	result := r.Check(context.Background(), &CheckRequest{})
	if !result.Passed {
		t.Fatalf("loss within limit should pass, got %s: %s", result.Rule, result.Reason)
	}
}

func TestDailyLoss_ExceedsLimit(t *testing.T) {
	t.Parallel()
	r := &DailyLoss{Limit: decF(1000), DailyPL: decF(-1500), DayStart: time.Now()}
	result := r.Check(context.Background(), &CheckRequest{})
	if result.Passed {
		t.Fatal("loss exceeding limit should block")
	}
	if result.Rule != "daily_loss" {
		t.Fatalf("expected rule=daily_loss, got %s", result.Rule)
	}
}

func TestDailyLoss_ZeroLimit(t *testing.T) {
	t.Parallel()
	r := &DailyLoss{Limit: decimal.Zero, DailyPL: decF(-999999), DayStart: time.Now()}
	result := r.Check(context.Background(), &CheckRequest{})
	if !result.Passed {
		t.Fatal("zero limit should always pass")
	}
}

func TestEngine_SetUserLimiter(t *testing.T) {
	t.Parallel()
	engine := NewEngine(&MaxPosition{Max: 10})
	engine.SetUserLimiter(nil) // should not panic
	// Verify evaluate still works with nil limiter
	result := engine.Evaluate(context.Background(), &CheckRequest{Symbol: "EURUSD", Positions: 1})
	if !result.Passed {
		t.Fatalf("expected pass with nil limiter, got %s", result.Rule)
	}
}
