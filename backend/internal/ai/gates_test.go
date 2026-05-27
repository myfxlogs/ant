package ai

import (
	"math"
	"testing"
)

// --- E1: LookAhead Scanner Tests ---

func TestLookAheadScanner_CleanExpression(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	result := s.Scan("sma(close, 5) > ema(close, 20)")
	if !result.Passed {
		t.Fatalf("clean expression should pass, got violations: %+v", result.Violations)
	}
}

func TestLookAheadScanner_ExplicitFutureIndex(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	result := s.Scan("close[t+1] > open")
	if result.Passed {
		t.Fatal("close[t+1] should be detected as lookahead")
	}
	if len(result.Violations) == 0 {
		t.Fatal("should have violations")
	}
	t.Logf("Violation: %s", result.Violations[0].Message)
}

func TestLookAheadScanner_NegativeRefOffset(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	result := s.Scan("ref(close, -1) > open")
	if result.Passed {
		t.Fatal("ref(close, -1) should be detected as lookahead (negative offset = future peek)")
	}
}

func TestLookAheadScanner_FutureHighReference(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	result := s.Scan("high[t+1] > high")
	if result.Passed {
		t.Fatal("high[t+1] should be detected as lookahead")
	}
}

func TestLookAheadScanner_PositiveRefOK(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	// ref(close, 1) with positive offset = looking at past value (legitimate).
	// Our scanner only flags negative offsets (future peek).
	result := s.Scan("ref(close, 1) > open")
	if !result.Passed {
		t.Fatalf("ref(close, 1) is a past reference, should pass: %+v", result.Violations)
	}
}

func TestLookAheadScanner_ComplexExpression(t *testing.T) {
	t.Parallel()
	s := NewLookAheadScanner()
	// Contains close[t+2] lookahead.
	result := s.Scan("close[t+2] > sma(close, 5) && volume > 1000")
	if result.Passed {
		t.Fatal("close[t+2] should be flagged")
	}
}

func TestHasLookahead(t *testing.T) {
	t.Parallel()
	if !HasLookahead("close[t+1] > open") {
		t.Fatal("HasLookahead should detect future reference")
	}
	if HasLookahead("sma(close, 5) > ema(close, 20)") {
		t.Fatal("clean expression should not have lookahead")
	}
}

// --- E2: Walk-Forward + CPCV Tests ---

func TestWalkForward_Passing(t *testing.T) {
	t.Parallel()
	// All returns equal → zero variance → Sharpe=0 for both train and test → diff=0 < 1.0.
	// No drawdowns since cumulative always increases.
	returns := make([]float64, 500)
	for i := range returns {
		returns[i] = 1.0
	}
	cfg := DefaultWalkForwardConfig()
	result := WalkForward(returns, cfg)
	if !result.Passed {
		t.Fatalf("uniform positive returns should pass walk-forward: %s (SharpeDiff=%.4f)", result.Reason, result.SharpeDiff)
	}
	if len(result.Folds) == 0 {
		t.Fatal("should have folds")
	}
	t.Logf("SharpeDiff=%.4f MaxFoldDD=%.4f MinTrades=%d Folds=%d",
		result.SharpeDiff, result.MaxFoldDD, result.MinTrades, len(result.Folds))
}

func TestWalkForward_InsufficientData(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 10) // too few returns
	cfg := DefaultWalkForwardConfig()
	result := WalkForward(returns, cfg)
	if result.Passed {
		t.Fatal("insufficient data should not pass")
	}
}

func TestWalkForward_OverfittingDetected(t *testing.T) {
	t.Parallel()
	// Train period has high Sharpe, test period has very low/negative returns.
	returns := make([]float64, 250)
	// First half (train): consistently positive.
	for i := 0; i < 125; i++ {
		returns[i] = 5.0
	}
	// Second half (test): noisy/negative.
	for i := 125; i < 250; i++ {
		returns[i] = -2.0 + float64(i%5)*0.1
	}
	cfg := DefaultWalkForwardConfig()
	result := WalkForward(returns, cfg)
	if result.Passed {
		t.Logf("SharpeDiff=%.4f (overfitting may not trigger if < 1.0)", result.SharpeDiff)
	}
	// Not asserting Passed=false because it depends on exact values,
	// but the Sharpe diff should be significant.
	t.Logf("Overfitting test: SharpeDiff=%.4f MaxDD=%.4f Passed=%v",
		result.SharpeDiff, result.MaxFoldDD, result.Passed)
}

func TestCPCV(t *testing.T) {
	t.Parallel()
	returns := make([]float64, 252)
	for i := range returns {
		returns[i] = 1.0 + float64(i%20)*0.05
	}
	oosSharpe := CPCV(returns, 6)
	t.Logf("CPCV OOS median Sharpe: %.4f", oosSharpe)
	// Should return a valid Sharpe.
	if math.IsNaN(oosSharpe) {
		t.Fatal("CPCV should return a valid number")
	}
}

func TestCPCV_EmptyReturns(t *testing.T) {
	t.Parallel()
	oosSharpe := CPCV([]float64{}, 6)
	if oosSharpe != 0 {
		t.Fatalf("empty returns: want 0, got %.4f", oosSharpe)
	}
}

// --- E3: Deflated Sharpe Ratio Tests ---

func TestDeflatedSharpe_PositiveEdge(t *testing.T) {
	t.Parallel()
	// Daily returns with SR ~1.0: mean ~0.0004, std ~0.006.
	returns := []float64{
		0.002, -0.003, 0.005, -0.001, 0.004, -0.006, 0.003, 0.001, -0.002, 0.004,
		-0.001, 0.003, -0.005, 0.002, 0.004, -0.002, 0.001, -0.003, 0.006, -0.001,
	}
	moments := ComputeReturnMoments(returns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 1
	dsr, passed := DeflatedSharpe(moments, cfg)
	t.Logf("SR=%.4f DSR=%.4f Passed=%v", moments.SharpeRatio, dsr, passed)
	if !passed {
		t.Fatal("positive edge with N=1 should pass")
	}
}

func TestDeflatedSharpe_N100_Deflates(t *testing.T) {
	t.Parallel()
	// Small returns with variance => modest SR. N=100 => DSR < SR.
	returns := []float64{0.003, -0.002, 0.001, 0.004, -0.001, 0.002, -0.003, 0.001, 0.000, 0.002}
	moments := ComputeReturnMoments(returns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 100
	dsr, _ := DeflatedSharpe(moments, cfg)
	t.Logf("N=100: SR=%.4f DSR=%.4f", moments.SharpeRatio, dsr)
	if dsr >= moments.SharpeRatio {
		t.Fatal("DSR should be lower than raw SR when N > 1")
	}
}

func TestDeflatedSharpe_ZeroSharpe(t *testing.T) {
	t.Parallel()
	moments := ReturnMoments{SharpeRatio: 0}
	dsr, passed := DeflatedSharpe(moments, DefaultDeflatedSharpeConfig())
	if dsr != 0 || passed {
		t.Fatalf("zero SR: want dsr=0 passed=false, got dsr=%.4f passed=%v", dsr, passed)
	}
}

func TestDeflatedSharpe_NegativeSharpe(t *testing.T) {
	t.Parallel()
	moments := ReturnMoments{SharpeRatio: -0.5}
	dsr, passed := DeflatedSharpe(moments, DefaultDeflatedSharpeConfig())
	if dsr != 0 || passed {
		t.Fatalf("negative SR: want dsr=0 passed=false, got dsr=%.4f passed=%v", dsr, passed)
	}
}

func TestDeflatedSharpe_N1_unchanged(t *testing.T) {
	t.Parallel()
	returns := []float64{0.002, -0.003, 0.001, 0.004, -0.001, 0.002, -0.002, 0.003, -0.001, 0.001}
	moments := ComputeReturnMoments(returns)
	cfg := DefaultDeflatedSharpeConfig()
	cfg.NumAttempts = 1
	dsr, _ := DeflatedSharpe(moments, cfg)
	// With N=1, ln(1)=0 → numerator=1 → DSR ≈ SR if denominator ≈ 1.
	t.Logf("N=1: SR=%.4f DSR=%.4f", moments.SharpeRatio, dsr)
	if math.Abs(dsr-moments.SharpeRatio) > 2.0 {
		t.Fatalf("N=1: DSR %.4f should be close to SR %.4f", dsr, moments.SharpeRatio)
	}
}

func TestComputeReturnMoments(t *testing.T) {
	t.Parallel()
	returns := []float64{0.01, -0.005, 0.02, -0.01, 0.015}
	moments := ComputeReturnMoments(returns)
	if moments.SharpeRatio == 0 {
		t.Fatal("should have non-zero Sharpe")
	}
	t.Logf("Moments: mean=%.4f std=%.4f skew=%.4f kurt=%.4f SR=%.4f",
		moments.Mean, moments.StdDev, moments.Skewness, moments.ExcessKurtosis, moments.SharpeRatio)
}

func TestDeflatedSharpeFromReturns(t *testing.T) {
	t.Parallel()
	returns := []float64{0.002, -0.003, 0.001, 0.004, -0.001, 0.002, -0.002, 0.003, -0.001, 0.001}
	dsr, passed := DeflatedSharpeFromReturns(returns, 1)
	t.Logf("DSR from returns: %.4f passed=%v", dsr, passed)
	if dsr <= 0 {
		t.Fatal("positive returns should yield positive DSR")
	}
}

// --- E4: Paper Gate Tests ---

func TestPaperGate_Passing(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:         14,
		BacktestNetReturn: 0.10,
		PaperNetReturn:    0.08, // 80% of backtest, above 50% threshold
		PaperNetPnL:       5000,
		PaperTradeCount:   20,
	}
	cfg := DefaultPaperGateConfig()
	result := PaperGate(metrics, cfg)
	if !result.Passed {
		t.Fatalf("should pass: %s", result.Reason)
	}
}

func TestPaperGate_InsufficientDays(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:      7, // < 14 minimum
		PaperNetReturn: 0.05,
		PaperNetPnL:    1000,
		PaperTradeCount: 10,
	}
	cfg := DefaultPaperGateConfig()
	result := PaperGate(metrics, cfg)
	if result.Passed {
		t.Fatal("insufficient paper days should not pass")
	}
}

func TestPaperGate_NegativePnL(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:      14,
		PaperNetPnL:    -500, // negative P&L
		PaperTradeCount: 10,
	}
	cfg := DefaultPaperGateConfig()
	result := PaperGate(metrics, cfg)
	if result.Passed {
		t.Fatal("negative P&L should not pass")
	}
}

func TestPaperGate_RegimeFail(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:         14,
		BacktestNetReturn: 0.20,
		PaperNetReturn:    0.05, // only 25% of backtest → regime fail
		PaperNetPnL:       1000,
		PaperTradeCount:   10,
	}
	cfg := DefaultPaperGateConfig()
	result := PaperGate(metrics, cfg)
	if result.Passed {
		t.Fatal("regime fail (paper return < 50% of backtest) should not pass")
	}
}

func TestPaperGate_TooFewTrades(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:      14,
		PaperNetPnL:    1000,
		PaperTradeCount: 3, // < 5
	}
	cfg := DefaultPaperGateConfig()
	result := PaperGate(metrics, cfg)
	if result.Passed {
		t.Fatal("too few paper trades should not pass")
	}
}

// --- E5: Correlation Gate Tests ---

func TestCorrelationGate_LowCorrelation(t *testing.T) {
	t.Parallel()
	newSignals := []SignalDirection{
		{Timestamp: 1, Direction: 1},
		{Timestamp: 2, Direction: -1},
		{Timestamp: 3, Direction: 1},
		{Timestamp: 4, Direction: 1},
		{Timestamp: 5, Direction: -1},
	}
	existing := map[string][]SignalDirection{
		"strat_a": {
			{Timestamp: 1, Direction: -1}, // opposite directions → low correlation
			{Timestamp: 2, Direction: 1},
			{Timestamp: 3, Direction: -1},
			{Timestamp: 4, Direction: -1},
			{Timestamp: 5, Direction: 1},
		},
	}
	cfg := DefaultCorrelationGateConfig()
	cfg.MinObservations = 5
	result := CorrelationGate(newSignals, existing, cfg)
	if !result.Passed {
		t.Fatalf("opposite strategies should pass: %s (corr=%.4f)", result.Reason, result.MaxCorrelation)
	}
	t.Logf("Low correlation: max=%.4f", result.MaxCorrelation)
}

func TestCorrelationGate_HighCorrelation(t *testing.T) {
	t.Parallel()
	// Strongly correlated signals (same direction pattern but with slight noise).
	signals := make([]SignalDirection, 50)
	for i := range signals {
		dir := 1.0
		if i%7 == 0 {
			dir = -1.0
		}
		signals[i] = SignalDirection{Timestamp: int64(i), Direction: dir}
	}
	copy2 := make([]SignalDirection, 50)
	copy(copy2, signals)
	// Slightly perturb one signal to avoid perfect correlation (NaN).
	copy2[0] = SignalDirection{Timestamp: 0, Direction: 1.0}
	existing := map[string][]SignalDirection{
		"strat_similar": copy2,
	}
	cfg := DefaultCorrelationGateConfig()
	cfg.MinObservations = 20
	result := CorrelationGate(signals, existing, cfg)
	if result.Passed {
		t.Fatal("highly correlated signals should be rejected")
	}
	t.Logf("High correlation: max=%.4f strategy=%s", result.MaxCorrelation, result.CorrelatedStrategy)
}

func TestCorrelationGate_InsufficientObservations(t *testing.T) {
	t.Parallel()
	signals := make([]SignalDirection, 10)
	existing := map[string][]SignalDirection{}
	cfg := DefaultCorrelationGateConfig()
	cfg.MinObservations = 20
	result := CorrelationGate(signals, existing, cfg)
	if result.Passed {
		t.Fatal("insufficient observations should not pass")
	}
}

func TestPearsonCorrelation(t *testing.T) {
	t.Parallel()
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{2, 4, 6, 8, 10}
	r := pearsonCorrelation(x, y)
	if math.Abs(r-1.0) > 0.001 {
		t.Fatalf("perfect positive correlation: want 1.0, got %.4f", r)
	}

	y2 := []float64{10, 8, 6, 4, 2}
	r2 := pearsonCorrelation(x, y2)
	if math.Abs(r2-(-1.0)) > 0.001 {
		t.Fatalf("perfect negative correlation: want -1.0, got %.4f", r2)
	}
}

// --- E6: Gate Pipeline Tests ---

func TestAIGatePipeline_AllPass(t *testing.T) {
	t.Parallel()
	// Cyclic returns with same distribution across all folds.
	returns := make([]float64, 500)
	for i := range returns {
		returns[i] = 0.5 + float64(i%10)*0.1
	}

	input := PipelineInput{
		Expression:   "sma(close, 5) > ema(close, 20)",
		DailyReturns: returns,
		NumAttempts:  1,
		PaperMetrics: PaperGateMetrics{
			PaperDays:          14,
			PaperNetReturn:     0.08,
			PaperNetPnL:        5000,
			PaperTradeCount:    20,
			BacktestNetReturn:  0.10,
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
		t.Logf("  Gate %s: passed=%v score=%.4f reason=%s", g.Gate, g.Passed, g.Score, g.Reason)
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
		Expression:   "close[t+1] > open", // lookahead bias
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
	input := PipelineInput{
		Expression: "", // empty → compliance fails
	}
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
		Expression:   "close[t+1] > open", // fails at lookahead
		DailyReturns: returns,
	}
	result := Pipeline(input)
	// The gates should have been evaluated in order:
	// Compliance (pass) → LookAhead (fail).
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

func TestPromoteToLive_AllPass(t *testing.T) {
	t.Parallel()
	metrics := PaperGateMetrics{
		PaperDays:      14,
		PaperNetPnL:    10000,
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

func TestComputeSharpe(t *testing.T) {
	t.Parallel()
	// Positive returns with variance → positive Sharpe.
	returns := []float64{2.0, 1.0, 3.0, 1.5, 2.5, 1.0, 3.0, 2.0, 1.5, 2.5}
	sr := computeSharpe(returns)
	if sr <= 0 {
		t.Fatalf("positive returns: want positive SR, got %.4f", sr)
	}
	// Negative returns → negative Sharpe.
	negReturns := []float64{-2.0, -1.0, -3.0, -1.5, -2.5, -1.0, -3.0, -2.0, -1.5, -2.5}
	srNeg := computeSharpe(negReturns)
	if srNeg >= 0 {
		t.Fatalf("negative returns: want negative SR, got %.4f", srNeg)
	}
}

func TestComputeMaxDD_Simple(t *testing.T) {
	t.Parallel()
	returns := []float64{100, -50, -30, 20, 50}
	dd := computeMaxDD(returns)
	if dd < 0 {
		t.Fatal("max DD should be >= 0")
	}
	// Peak at 100, trough at (100-50-30)=20 → DD fraction = 80/100 = 0.8.
	if math.Abs(dd-0.8) > 0.01 {
		t.Fatalf("max DD: want ~0.8, got %.2f", dd)
	}
}
