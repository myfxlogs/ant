// Package ai — E6 Gate Pipeline + PromoteToLive tests.
package ai

import "testing"

func TestAIGatePipeline_AllPass(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 500)
	for i := range returns {
		returns[i] = 0.5 + float64(i%10)*0.1
	}

	input := PipelineInput{
		Expression:   "sma(close, 5) > ema(close, 20)",
		DailyReturns: returns,
		NumAttempts:  1,
		PaperMetrics: PaperGateMetrics{
			PaperDays:         14,
			PaperNetReturn:    0.08,
			PaperNetPnL:       5000,
			PaperTradeCount:   20,
			BacktestNetReturn: 0.10,
		},
		NewSignals: []SignalDirection{
			{1, 1}, {2, -1}, {3, 1}, {4, -1}, {5, 1},
			{6, -1}, {7, 1}, {8, -1}, {9, 1}, {10, -1},
			{11, 1}, {12, -1}, {13, 1}, {14, -1}, {15, 1},
			{16, -1}, {17, 1}, {18, -1}, {19, 1}, {20, -1},
		},
		ExistingSignals: map[string][]SignalDirection{},
	}

	result := Pipeline(input)
	t.Logf("Pipeline: passed=%v first_fail=%s summary=%s", result.Passed, result.FirstFail, result.Summary)
	for _, g := range result.Gates {
		t.Logf("  Gate %s: passed=%v skipped=%v score=%.4f reason=%s", g.Gate, g.Passed, g.Skipped, g.Score, g.Reason)
	}
	if !result.Passed {
		t.Fatalf("clean strategy should pass all gates, failed at: %s", result.FirstFail)
	}
	if len(result.Gates) != 6 {
		t.Fatalf("should evaluate all 6 gates, got %d", len(result.Gates))
	}
}

func TestAIGatePipeline_LookAheadFails(t *testing.T) {
	t.Parallel()
	input := PipelineInput{
		Expression:   "close[t+1] > open",
		DailyReturns: make([]float64, 200),
	}
	result := Pipeline(input)
	if result.Passed {
		t.Fatal("lookahead-biased expression should fail pipeline")
	}
	if result.FirstFail != GateLookAhead {
		t.Fatalf("should fail at lookahead gate, not %s", result.FirstFail)
	}
}

func TestAIGatePipeline_EmptyExpressionFails(t *testing.T) {
	t.Parallel()
	input := PipelineInput{Expression: ""}
	result := Pipeline(input)
	if result.Passed {
		t.Fatal("empty expression should fail at compliance gate")
	}
	if result.FirstFail != GateCompliance {
		t.Fatalf("should fail at compliance, not %s", result.FirstFail)
	}
}

func TestAIGatePipeline_OrderIsCorrect(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 200)
	input := PipelineInput{
		Expression:   "close[t+1] > open",
		DailyReturns: returns,
	}
	result := Pipeline(input)
	if len(result.Gates) != 2 {
		t.Fatalf("should stop after 2 gates, got %d", len(result.Gates))
	}
	if result.Gates[0].Gate != GateCompliance {
		t.Fatalf("first gate should be compliance, got %s", result.Gates[0].Gate)
	}
	if result.Gates[1].Gate != GateLookAhead {
		t.Fatalf("second gate should be lookahead, got %s", result.Gates[1].Gate)
	}
}

func TestAIGatePipeline_PaperSkipped(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 500)
	for i := range returns {
		returns[i] = 0.5 + float64(i%10)*0.1
	}
	input := PipelineInput{
		Expression:   "sma(close, 5) > ema(close, 20)",
		DailyReturns: returns,
		NumAttempts:  1,
		// PaperDays=0 → paper gate should be skipped.
		PaperMetrics:   PaperGateMetrics{PaperDays: 0},
		NewSignals:     make([]SignalDirection, 25),
		ExistingSignals: map[string][]SignalDirection{},
	}
	result := Pipeline(input)
	t.Logf("Pipeline with paper skipped: passed=%v summary=%s", result.Passed, result.Summary)
	// Paper gate should be skipped (not failed), and correlation should also be skipped (no existing).
	if !result.Passed {
		t.Fatalf("strategy should pass when paper and correlation are skipped: %s", result.FirstFail)
	}
}

func TestPromoteToLive_AllPass(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:       14,
		PaperNetPnL:     10000,
		PaperTradeCount: 30,
	}
	passed, msg := PromoteToLive(metrics, 0.98, 0.3, DefaultPromoteConditions())
	if !passed {
		t.Fatalf("should pass: %s", msg)
	}
	t.Logf("PromoteToLive: %s", msg)
}

func TestPromoteToLive_MissingDays(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{PaperDays: 7}
	passed, msg := PromoteToLive(metrics, 0.98, 0.3, DefaultPromoteConditions())
	if passed {
		t.Fatalf("should fail: %s", msg)
	}
}

func TestPromoteToLive_NegativePnL(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{PaperDays: 14, PaperNetPnL: -100}
	passed, msg := PromoteToLive(metrics, 0.98, 0.3, DefaultPromoteConditions())
	if passed {
		t.Fatalf("should fail: %s", msg)
	}
}

func TestPromoteToLive_LowDSR(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{PaperDays: 14, PaperNetPnL: 100}
	passed, msg := PromoteToLive(metrics, 0.80, 0.3, DefaultPromoteConditions())
	if passed {
		t.Fatalf("should fail: %s", msg)
	}
}

func TestPromoteToLive_HighCorrelation(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{PaperDays: 14, PaperNetPnL: 100}
	passed, msg := PromoteToLive(metrics, 0.98, 0.85, DefaultPromoteConditions())
	if passed {
		t.Fatalf("should fail: %s", msg)
	}
}

func TestGateResultsSummary(t *testing.T) {
	t.Parallel()
	result := PipelineResult{
		Passed: false,
		Gates: []GateStatus{
			{Gate: GateCompliance, Passed: true},
			{Gate: GateLookAhead, Passed: false, Reason: "future ref"},
			{Gate: GatePaper, Passed: false, Reason: "no paper days"},
		},
	}
	failures := GateResultsSummary(result)
	if len(failures) != 2 {
		t.Fatalf("want 2 failures, got %d: %v", len(failures), failures)
	}
}
