package risksvc

import (
	"context"
	"fmt"
	"time"
)

// ── Rule 1: MaxPosition — reject if open positions exceed limit ─────

type MaxPosition struct{ Max int }

func (r *MaxPosition) Name() string { return "max_position" }
func (r *MaxPosition) Check(_ context.Context, req *CheckRequest) *CheckResult {
	if r.Max > 0 && req.Positions >= r.Max {
		return &CheckResult{Passed: false, Rule: r.Name(),
			Reason: fmt.Sprintf("max positions %d reached (current %d)", r.Max, req.Positions)}
	}
	return &CheckResult{Passed: true, Rule: r.Name()}
}

// ── Rule 2: DailyLoss — reject if daily P&L exceeds limit ───────────

type DailyLoss struct {
	Limit     float64
	DayStart  time.Time
	DailyPL   float64
}

func (r *DailyLoss) Name() string { return "daily_loss" }
func (r *DailyLoss) Check(_ context.Context, req *CheckRequest) *CheckResult {
	if time.Since(r.DayStart) > 24*time.Hour {
		r.DayStart = Clk.Now()
		r.DailyPL = 0
	}
	if r.Limit > 0 && r.DailyPL < -r.Limit {
		return &CheckResult{Passed: false, Rule: r.Name(),
			Reason: fmt.Sprintf("daily loss %.2f exceeds limit %.2f", r.DailyPL, r.Limit)}
	}
	return &CheckResult{Passed: true, Rule: r.Name()}
}

// ── Rule 3: Drawdown — reject if equity drawdown exceeds pct ─────────

type Drawdown struct {
	MaxPct      float64
	PeakEquity  float64
}

func (r *Drawdown) Name() string { return "drawdown" }

func (r *Drawdown) Check(_ context.Context, req *CheckRequest) *CheckResult {
	if req.Equity > r.PeakEquity {
		r.PeakEquity = req.Equity
	}
	if r.MaxPct > 0 && r.PeakEquity > 0 {
		dd := (r.PeakEquity - req.Equity) / r.PeakEquity * 100
		if dd > r.MaxPct {
			return &CheckResult{Passed: false, Rule: r.Name(),
				Reason: fmt.Sprintf("drawdown %.2f%% exceeds limit %.2f%%", dd, r.MaxPct)}
		}
	}
	return &CheckResult{Passed: true, Rule: r.Name()}
}

// ── Rule 4: Session — reject outside trading hours ───────────────────

type Session struct{}

func (r *Session) Name() string { return "session" }
func (r *Session) Check(_ context.Context, req *CheckRequest) *CheckResult {
	now := Clk.Now().UTC()
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return &CheckResult{Passed: false, Rule: r.Name(), Reason: "market closed (weekend)"}
	}
	return &CheckResult{Passed: true, Rule: r.Name()}
}

// ── Rule 5: Margin — reject if margin level too low ──────────────────

type Margin struct{ MinLevel float64 }

func (r *Margin) Name() string { return "margin" }
func (r *Margin) Check(_ context.Context, req *CheckRequest) *CheckResult {
	if req.Margin <= 0 || req.Equity <= 0 {
		return &CheckResult{Passed: true, Rule: r.Name()}
	}
	level := req.Equity / req.Margin
	if level < r.MinLevel {
		return &CheckResult{Passed: false, Rule: r.Name(),
			Reason: fmt.Sprintf("margin level %.2f below minimum %.2f", level, r.MinLevel)}
	}
	return &CheckResult{Passed: true, Rule: r.Name()}
}

// ── Rule 6: CanonicalAuth — reject if symbol not in whitelist ────────

type CanonicalAuth struct {
	Whitelist []string // if empty, all symbols allowed
}

func (r *CanonicalAuth) Name() string { return "canonical_auth" }
func (r *CanonicalAuth) Check(_ context.Context, req *CheckRequest) *CheckResult {
	if len(r.Whitelist) == 0 {
		return &CheckResult{Passed: true, Rule: r.Name()}
	}
	for _, s := range r.Whitelist {
		if s == req.Symbol {
			return &CheckResult{Passed: true, Rule: r.Name()}
		}
	}
	return &CheckResult{Passed: false, Rule: r.Name(),
		Reason: fmt.Sprintf("symbol %s not in whitelist", req.Symbol)}
}
