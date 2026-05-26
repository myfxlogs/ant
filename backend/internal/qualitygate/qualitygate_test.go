package qualitygate

import (
	"context"
	"math"
	"testing"
)

// --- Syntax Gate Tests ---

func TestSyntaxGate_Pass(t *testing.T) {
	gate := &SyntaxGate{MinCodeLength: 10}
	info := &StrategyInfo{
		Code:     "def on_bar(bar):\n    if bar.close > bar.open:\n        return 'buy'\n    return None",
		Language: "python",
	}
	result := gate.Evaluate(context.Background(), info)
	if !result.Passed {
		t.Fatalf("valid code should pass, got: %s", result.Reason)
	}
	if result.Score < 0.8 {
		t.Fatalf("score should be >= 0.8, got %.2f", result.Score)
	}
}

func TestSyntaxGate_EmptyCode(t *testing.T) {
	gate := &SyntaxGate{MinCodeLength: 10}
	info := &StrategyInfo{Code: "", Language: "python"}
	result := gate.Evaluate(context.Background(), info)
	if result.Passed {
		t.Fatal("empty code should fail")
	}
	if result.Severity != SeverityCritical {
		t.Fatalf("empty code should be CRITICAL, got %s", result.Severity)
	}
}

func TestSyntaxGate_TooShort(t *testing.T) {
	gate := &SyntaxGate{MinCodeLength: 50}
	info := &StrategyInfo{Code: "x=1", Language: "python"}
	result := gate.Evaluate(context.Background(), info)
	if result.Passed {
		t.Fatal("too-short code should fail")
	}
}

func TestSyntaxGate_ForbiddenPattern(t *testing.T) {
	gate := &SyntaxGate{
		MinCodeLength:    20,
		ForbiddenPatterns: []string{"os.system", "subprocess"},
	}
	info := &StrategyInfo{
		Code:     "import os\ndef on_bar(bar):\n    os.system('rm -rf /')\n    return 'buy'",
		Language: "python",
	}
	result := gate.Evaluate(context.Background(), info)
	if result.Passed {
		t.Fatal("code with os.system should be rejected")
	}
	if result.Severity != SeverityError {
		t.Fatalf("forbidden pattern should be ERROR, got %s", result.Severity)
	}
}

func TestSyntaxGate_MissingImportWarning(t *testing.T) {
	gate := &SyntaxGate{
		MinCodeLength:  20,
		RequiredImports: []string{"numpy", "pandas"},
	}
	info := &StrategyInfo{
		Code:     "def on_bar(bar):\n    return 'buy'",
		Language: "python",
	}
	result := gate.Evaluate(context.Background(), info)
	if len(result.Warnings) == 0 {
		t.Fatal("should warn about missing imports")
	}
}

// --- Risk Gate Tests ---

func TestRiskGate_Pass(t *testing.T) {
	gate := &RiskGate{}
	info := &StrategyInfo{
		Schedule: &ScheduleInfo{
			RiskPerTradePct: 0.01,
			MaxDrawdownPct:  0.15,
			LeverageAllowed: 50,
			StopLossPct:     0.02,
			DailyTradeLimit: 10,
			MaxPositions:    3,
		},
	}
	result := gate.Evaluate(context.Background(), info)
	if !result.Passed {
		t.Fatalf("safe schedule should pass: %s", result.Reason)
	}
}

func TestRiskGate_HighRiskPerTrade(t *testing.T) {
	gate := &RiskGate{MaxRiskPerTrade: 0.05}
	info := &StrategyInfo{
		Schedule: &ScheduleInfo{RiskPerTradePct: 0.20}, // 20%!
	}
	result := gate.Evaluate(context.Background(), info)
	if result.Passed {
		t.Fatal("20% risk per trade should fail")
	}
	if result.Severity != SeverityCritical {
		t.Fatalf("high risk should be CRITICAL, got %s", result.Severity)
	}
}

func TestRiskGate_HighDrawdown(t *testing.T) {
	gate := &RiskGate{MaxDrawdownLimit: 0.30}
	info := &StrategyInfo{
		Schedule: &ScheduleInfo{
			RiskPerTradePct: 0.01,
			MaxDrawdownPct:  0.60, // 60%!
		},
	}
	result := gate.Evaluate(context.Background(), info)
	if result.Passed {
		t.Fatal("60% drawdown should fail")
	}
}

func TestRiskGate_ExcessiveLeverage(t *testing.T) {
	gate := &RiskGate{MaxLeverage: 100}
	info := &StrategyInfo{
		Schedule: &ScheduleInfo{
			RiskPerTradePct: 0.01,
			MaxDrawdownPct:  0.15,
			LeverageAllowed: 500,
		},
	}
	result := gate.Evaluate(context.Background(), info)
	if result.Score >= 1.0 {
		t.Fatalf("500x leverage should reduce score, got %.2f", result.Score)
	}
}

func TestRiskGate_NilSchedule(t *testing.T) {
	gate := &RiskGate{}
	info := &StrategyInfo{} // nil schedule → uses defaults
	result := gate.Evaluate(context.Background(), info)
	if !result.Passed {
		t.Fatalf("default schedule should pass, got: %s", result.Reason)
	}
}

// --- Backtest Gate Tests ---

func TestBacktestGate_Pass(t *testing.T) {
	gate := &BacktestGate{}
	info := &StrategyInfo{
		Backtest: &BacktestInfo{
			SharpeRatio:  1.5,
			WinRate:      0.55,
			ProfitFactor: 2.0,
			MaxDrawdown:  0.15,
			TotalTrades:  100,
			TotalReturn:  0.25,
		},
	}
	result := gate.Evaluate(context.Background(), info)
	if !result.Passed {
		t.Fatalf("good backtest should pass: %s", result.Reason)
	}
}

func TestBacktestGate_PoorSharpe(t *testing.T) {
	gate := &BacktestGate{MinSharpeRatio: 0.5}
	info := &StrategyInfo{
		Backtest: &BacktestInfo{
			SharpeRatio:  0.1,
			WinRate:      0.55,
			ProfitFactor: 1.5,
			MaxDrawdown:  0.15,
			TotalTrades:  50,
		},
	}
	result := gate.Evaluate(context.Background(), info)
	if result.Passed {
		t.Fatal("sharpe 0.1 should fail")
	}
}

func TestBacktestGate_NoBacktest(t *testing.T) {
	gate := &BacktestGate{}
	info := &StrategyInfo{} // no backtest
	result := gate.Evaluate(context.Background(), info)
	if result.Passed {
		t.Fatal("missing backtest should fail")
	}
	if result.Severity != SeverityWarning {
		t.Fatalf("missing backtest should be WARNING, got %s", result.Severity)
	}
}

func TestBacktestGate_LowTrades(t *testing.T) {
	gate := &BacktestGate{MinTotalTrades: 30}
	info := &StrategyInfo{
		Backtest: &BacktestInfo{
			SharpeRatio:  1.5,
			WinRate:      0.60,
			ProfitFactor: 2.0,
			MaxDrawdown:  0.10,
			TotalTrades:  5, // very few
		},
	}
	result := gate.Evaluate(context.Background(), info)
	if result.Score >= 1.0 {
		t.Fatalf("low trade count should reduce score, got %.2f", result.Score)
	}
}

// --- Reliability Gate Tests ---

func TestReliabilityGate_Pass(t *testing.T) {
	gate := &ReliabilityGate{}
	info := &StrategyInfo{
		Backtest: &BacktestInfo{
			TotalTrades:   200,
			WinningTrades: 120,
			LosingTrades:  80,
			AverageProfit: 100,
			AverageLoss:   50,
			TotalReturn:   0.30,
			MaxDrawdown:   0.15,
			ProfitFactor:  2.5,
		},
	}
	result := gate.Evaluate(context.Background(), info)
	if !result.Passed {
		t.Fatalf("good reliability should pass: %s", result.Reason)
	}
	if result.Score < 0.8 {
		t.Fatalf("score should be high, got %.2f", result.Score)
	}
}

func TestReliabilityGate_LowTrades(t *testing.T) {
	gate := &ReliabilityGate{MinTradesForReliability: 100}
	info := &StrategyInfo{
		Backtest: &BacktestInfo{
			TotalTrades:   20,
			WinningTrades: 12,
			LosingTrades:  8,
			AverageProfit: 100,
			AverageLoss:   50,
			TotalReturn:   0.10,
			MaxDrawdown:   0.10,
		},
	}
	result := gate.Evaluate(context.Background(), info)
	if result.Score >= 1.0 {
		t.Fatalf("low trade count should reduce reliability score")
	}
}

func TestReliabilityGate_Overfitting(t *testing.T) {
	gate := &ReliabilityGate{MaxReturnDrawdownRatio: 10.0}
	info := &StrategyInfo{
		Backtest: &BacktestInfo{
			TotalTrades:   200,
			WinningTrades: 180,
			LosingTrades:  20,
			AverageProfit: 500,
			AverageLoss:   10,
			TotalReturn:   5.0,  // 500% return
			MaxDrawdown:   0.02, // 2% drawdown → ratio=250!
		},
	}
	result := gate.Evaluate(context.Background(), info)
	// Should have warnings about overfitting
	hasOverfitWarning := false
	for _, w := range result.Warnings {
		if contains(w, "overfitting") || contains(w, "return/drawdown") {
			hasOverfitWarning = true
		}
	}
	if !hasOverfitWarning {
		t.Fatalf("suspicious return/drawdown should trigger warning, got: %v", result.Warnings)
	}
}

func TestReliabilityGate_NegativeReturn(t *testing.T) {
	gate := &ReliabilityGate{RequirePositiveReturn: true}
	info := &StrategyInfo{
		Backtest: &BacktestInfo{
			TotalTrades:   200,
			WinningTrades: 80,
			LosingTrades:  120,
			AverageProfit: 50,
			AverageLoss:   100,
			TotalReturn:   -0.15,
			MaxDrawdown:   0.30,
		},
	}
	result := gate.Evaluate(context.Background(), info)
	if result.Passed {
		t.Fatal("negative return should fail when RequirePositiveReturn=true")
	}
}

// --- Pipeline Tests ---

func TestPipeline_AllPass(t *testing.T) {
	p := DefaultPipeline()
	info := &StrategyInfo{
		Code:     "def on_bar(bar):\n    if bar.close > bar.open:\n        return 'buy'\n    return None",
		Language: "python",
		Schedule: &ScheduleInfo{
			RiskPerTradePct: 0.01,
			MaxDrawdownPct:  0.10,
			LeverageAllowed: 50,
		},
		Backtest: &BacktestInfo{
			SharpeRatio:   1.5,
			WinRate:       0.55,
			ProfitFactor:  2.0,
			MaxDrawdown:   0.12,
			TotalTrades:   150,
			WinningTrades: 90,
			LosingTrades:  60,
			AverageProfit: 100,
			AverageLoss:   50,
			TotalReturn:   0.25,
		},
	}
	result := p.Run(context.Background(), info)
	if !result.Passed {
		t.Fatalf("full pass expected, got errors: %v", result.Errors)
	}
	if result.Score < 0.7 {
		t.Fatalf("score should be high, got %.2f", result.Score)
	}
	if len(result.Results) != 4 {
		t.Fatalf("want 4 gate results, got %d", len(result.Results))
	}
	t.Logf("Pipeline pass: score=%.2f risk=%s reliable=%v", result.Score, result.RiskLevel, result.IsReliable)
}

func TestPipeline_StopsOnCritical(t *testing.T) {
	// Empty code → syntax gate CRITICAL → pipeline stops before risk/backtest
	gate1 := &SyntaxGate{MinCodeLength: 10}
	gate2 := &RiskGate{}
	p := NewPipeline(gate1, gate2)
	info := &StrategyInfo{Code: ""}
	result := p.Run(context.Background(), info)
	if len(result.Results) < 1 {
		t.Fatal("should have at least syntax result")
	}
	// After CRITICAL, subsequent gates should not run
	if result.Results[0].Severity != SeverityCritical {
		t.Fatalf("syntax should be CRITICAL, got %s", result.Results[0].Severity)
	}
}

func TestPipeline_RiskLevel(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
	}{
		{0.9, "low"},
		{0.8, "low"},
		{0.6, "medium"},
		{0.5, "medium"},
		{0.3, "high"},
	}
	for _, tt := range tests {
		pr := &PipelineResult{Score: tt.score}
		level := computeRiskLevel(pr)
		if level != tt.expected {
			t.Fatalf("score %.1f → want %s, got %s", tt.score, tt.expected, level)
		}
	}
}

// --- Scorer Tests ---

func TestComputeQualityScore_Approved(t *testing.T) {
	pr := &PipelineResult{
		Passed:    true,
		Score:     0.85,
		RiskLevel: "low",
		Results: []*GateResult{
			{Gate: "syntax", Passed: true, Score: 0.9, Severity: SeverityPass},
			{Gate: "risk", Passed: true, Score: 0.85, Severity: SeverityPass},
			{Gate: "backtest", Passed: true, Score: 0.8, Severity: SeverityPass},
			{Gate: "reliability", Passed: true, Score: 0.85, Severity: SeverityPass},
		},
	}
	qs := ComputeQualityScore(pr)
	if qs.Verdict != "APPROVED" {
		t.Fatalf("want APPROVED, got %s", qs.Verdict)
	}
	if !qs.IsReliable {
		t.Fatal("should be reliable")
	}
}

func TestComputeQualityScore_Rejected_Critical(t *testing.T) {
	pr := &PipelineResult{
		Passed: false,
		Score:  0.3,
		Results: []*GateResult{
			{Gate: "syntax", Passed: false, Score: 0, Severity: SeverityCritical, Reason: "empty code"},
			{Gate: "risk", Passed: true, Score: 0.8, Severity: SeverityPass},
		},
	}
	qs := ComputeQualityScore(pr)
	if qs.Verdict != "REJECTED" {
		t.Fatalf("want REJECTED, got %s", qs.Verdict)
	}
}

func TestComputeQualityScore_NeedsReview(t *testing.T) {
	pr := &PipelineResult{
		Passed: false,
		Score:  0.55,
		Results: []*GateResult{
			{Gate: "syntax", Passed: true, Score: 0.7, Severity: SeverityWarning},
			{Gate: "risk", Passed: false, Score: 0.4, Severity: SeverityError, Reason: "risk high"},
		},
	}
	qs := ComputeQualityScore(pr)
	if qs.Verdict != "NEEDS_REVIEW" {
		t.Fatalf("want NEEDS_REVIEW, got %s", qs.Verdict)
	}
}

// --- Approval Decision Tests ---

func TestDecideApproval_Approved(t *testing.T) {
	pr := &PipelineResult{
		Passed:    true,
		Score:     0.82,
		RiskLevel: "low",
		Results: []*GateResult{
			{Gate: "syntax", Passed: true, Score: 0.9, Severity: SeverityPass},
			{Gate: "risk", Passed: true, Score: 0.85, Severity: SeverityPass},
			{Gate: "backtest", Passed: true, Score: 0.8, Severity: SeverityPass},
			{Gate: "reliability", Passed: true, Score: 0.75, Severity: SeverityPass},
		},
	}
	ad := DecideApproval(pr)
	if !ad.Allowed {
		t.Fatalf("should be allowed: %s", ad.Reason)
	}
}

func TestDecideApproval_Rejected(t *testing.T) {
	pr := &PipelineResult{
		Passed: false,
		Score:  0.2,
		Errors: []string{"[syntax] empty code"},
		Results: []*GateResult{
			{Gate: "syntax", Passed: false, Score: 0, Severity: SeverityCritical, Reason: "empty code"},
		},
	}
	ad := DecideApproval(pr)
	if ad.Allowed {
		t.Fatal("should be rejected")
	}
	if len(ad.Requires) == 0 {
		t.Fatal("should list required fixes")
	}
}

func TestDecideApproval_NeedsReview(t *testing.T) {
	pr := &PipelineResult{
		Passed:   false,
		Score:    0.55,
		Warnings: []string{"[risk] high leverage"},
		Results: []*GateResult{
			{Gate: "risk", Passed: false, Score: 0.5, Severity: SeverityError, Reason: "high leverage"},
		},
	}
	ad := DecideApproval(pr)
	if ad.Allowed {
		t.Fatal("needs review should not be auto-approved")
	}
}

// --- Severity Tests ---

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		s Severity
		e string
	}{
		{SeverityPass, "PASS"},
		{SeverityWarning, "WARNING"},
		{SeverityError, "ERROR"},
		{SeverityCritical, "CRITICAL"},
		{Severity(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		if tt.s.String() != tt.e {
			t.Fatalf("Severity(%d) = %s, want %s", tt.s, tt.s.String(), tt.e)
		}
	}
}

// --- Default Schedule ---

func TestDefaultSchedule(t *testing.T) {
	s := DefaultSchedule()
	if s.MaxPositions != 5 {
		t.Fatalf("default positions: %d", s.MaxPositions)
	}
	if math.Abs(s.RiskPerTradePct-0.02) > 0.001 {
		t.Fatalf("default risk: %.4f", s.RiskPerTradePct)
	}
}

// --- Gate Names ---

func TestGateNames(t *testing.T) {
	if (&SyntaxGate{}).Name() != "syntax" {
		t.Fatal("syntax gate name")
	}
	if (&RiskGate{}).Name() != "risk" {
		t.Fatal("risk gate name")
	}
	if (&BacktestGate{}).Name() != "backtest" {
		t.Fatal("backtest gate name")
	}
	if (&ReliabilityGate{}).Name() != "reliability" {
		t.Fatal("reliability gate name")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
