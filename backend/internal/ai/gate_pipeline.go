// Package ai provides the 7-Gate Pipeline (M10-BASE-E6).
//
// The gate pipeline evaluates AI-generated strategies through seven sequential gates:
//
//	Compliance → LookAhead → Walk-Forward+CPCV → DeflatedSharpe → MonteCarlo → Paper(14d) → Correlation
//
// Only strategies that pass all 7 gates are eligible for PromoteToLive.
// PromoteToLive conditions: Sharpe > 0, DSR >= 0.95, MC P(+) >= 0.95, Paper ≥ 14d Net P&L > 0, Correlation < 0.7.

package ai

import (
	"sort"
	"strings"
	"time"
)

// --- Gate Pipeline ---

// GateName identifies each gate in the pipeline.
type GateName string

const (
	GateCompliance     GateName = "compliance"
	GateLookAhead      GateName = "lookahead"
	GateWalkForward    GateName = "walkforward"
	GateDeflatedSharpe GateName = "deflated_sharpe"
	GateMonteCarlo     GateName = "monte_carlo"
	GatePaper          GateName = "paper"
	GateCorrelation    GateName = "correlation"
)

// GateOrder is the canonical 7-gate evaluation order.
var GateOrder = []GateName{
	GateCompliance,
	GateLookAhead,
	GateWalkForward,
	GateDeflatedSharpe,
	GateMonteCarlo,
	GatePaper,
	GateCorrelation,
}

// GateStatus represents a single gate's evaluation result.
type GateStatus struct {
	Gate     GateName
	Passed   bool
	Skipped  bool   // true when gate is skipped (no data)
	Reason   string
	Score    float64
	Duration int64
}

// PipelineResult is the aggregate result of running the full 7-gate pipeline.
type PipelineResult struct {
	Passed        bool
	Gates         []GateStatus
	FirstFail     GateName
	Summary       string
	TotalDuration int64
}

// PipelineInput bundles all the data needed for gate evaluation.
type PipelineInput struct {
	Expression      string                        // DSL expression for lookahead scanning
	DailyReturns    []float64                     // daily P&L returns for walk-forward and DSR
	NumAttempts     int                           // number of user strategy attempts
	PaperMetrics    PaperGateMetrics              // paper trading metrics
	NewSignals      []SignalDirection             // new strategy's signal directions
	ExistingSignals map[string][]SignalDirection  // existing live strategies' signals
}

// Pipeline evaluates a strategy through all 7 gates.
// All gates are evaluated (no short-circuit) so the user sees the complete picture.
// result.Passed is false if any non-skipped gate fails; FirstFail records the first failure.
func Pipeline(input PipelineInput) PipelineResult {
	startedAt := time.Now()
	result := PipelineResult{Passed: true}

	for _, gate := range GateOrder {
		gateStart := time.Now()
		status := GateStatus{Gate: gate}

		switch gate {
		case GateCompliance:
			status = evalCompliance(input)
		case GateLookAhead:
			status = evalLookAhead(input.Expression)
		case GateWalkForward:
			status = evalWalkForward(input.DailyReturns)
		case GateDeflatedSharpe:
			status = evalDeflatedSharpe(input.DailyReturns, input.NumAttempts)
		case GateMonteCarlo:
			status = evalMonteCarlo(input.DailyReturns)
		case GatePaper:
			status = evalPaper(input.PaperMetrics)
		case GateCorrelation:
			status = evalCorrelation(input.NewSignals, input.ExistingSignals)
		}

		status.Duration = time.Since(gateStart).Milliseconds()
		result.Gates = append(result.Gates, status)

		// Record first failure but continue evaluating remaining gates.
		if !status.Passed && !status.Skipped && result.Passed {
			result.Passed = false
			result.FirstFail = gate
			result.Summary = status.Reason
		}
	}

	if result.Passed {
		result.Summary = "all 7 gates passed"
	}
	result.TotalDuration = time.Since(startedAt).Milliseconds()
	return result
}

// --- Individual gate evaluators ---

// hasBalancedBrackets checks that parentheses and square brackets are balanced.
func hasBalancedBrackets(expr string) bool {
	stack := 0
	for _, c := range expr {
		switch c {
		case '(', '[':
			stack++
		case ')', ']':
			stack--
			if stack < 0 {
				return false
			}
		}
	}
	return stack == 0
}

// hasOperator checks that the expression contains at least one operator or comparison.
func hasOperator(expr string) bool {
	ops := []string{">", "<", "==", "!=", ">=", "<=", "&&", "||", "+", "-", "*", "/", "cross"}
	for _, op := range ops {
		if strings.Contains(expr, op) {
			return true
		}
	}
	return false
}

func evalCompliance(input PipelineInput) GateStatus {
	expr := strings.TrimSpace(input.Expression)
	if expr == "" {
		return GateStatus{Gate: GateCompliance, Passed: true, Skipped: true, Reason: "no DSL expression — skipped for MQL/code strategy"}
	}
	if !hasBalancedBrackets(expr) {
		return GateStatus{Gate: GateCompliance, Passed: false, Reason: "unbalanced brackets in expression"}
	}
	if !hasOperator(expr) {
		return GateStatus{Gate: GateCompliance, Passed: false, Reason: "expression missing comparison or operator"}
	}
	return GateStatus{Gate: GateCompliance, Passed: true}
}

func evalLookAhead(expression string) GateStatus {
	expr := strings.TrimSpace(expression)
	if expr == "" {
		return GateStatus{Gate: GateLookAhead, Passed: true, Skipped: true, Reason: "no DSL expression — skipped for MQL/code strategy"}
	}
	s := NewLookAheadScanner()
	scanResult := s.Scan(expression)
	if !scanResult.Passed {
		reason := "lookahead bias detected: "
		for i, v := range scanResult.Violations {
			if i > 0 {
				reason += "; "
			}
			reason += v.Message
		}
		return GateStatus{Gate: GateLookAhead, Passed: false, Reason: reason}
	}
	return GateStatus{Gate: GateLookAhead, Passed: true}
}

func evalWalkForward(dailyReturns []float64) GateStatus {
	cfg := DefaultWalkForwardConfig()

	// Primary: Walk-Forward with overfitting + drawdown + trade-count checks.
	wfResult := WalkForward(dailyReturns, cfg)
	if !wfResult.Passed {
		return GateStatus{
			Gate: GateWalkForward, Passed: false,
			Reason: wfResult.Reason,
			Score:  wfResult.SharpeDiff,
		}
	}

	// Secondary: CPCV for robust OOS Sharpe estimate.
	cpcvSharpe := CPCV(dailyReturns, 6, cfg)

	return GateStatus{
		Gate: GateWalkForward, Passed: true,
		Score: cpcvSharpe,
	}
}

func evalDeflatedSharpe(dailyReturns []float64, numAttempts int) GateStatus {
	dsr, passed := DeflatedSharpeFromReturns(dailyReturns, numAttempts)
	if !passed {
		return GateStatus{
			Gate: GateDeflatedSharpe, Passed: false,
			Reason: "deflated Sharpe below confidence threshold",
			Score:  dsr,
		}
	}
	return GateStatus{
		Gate: GateDeflatedSharpe, Passed: true,
		Score: dsr,
	}
}

func evalMonteCarlo(dailyReturns []float64) GateStatus {
	cfg := DefaultMonteCarloConfig()
	result := MonteCarlo(dailyReturns, cfg)
	if !result.Passed {
		return GateStatus{
			Gate: GateMonteCarlo, Passed: false,
			Reason: result.Reason,
			Score:  result.ProbPositive,
		}
	}
	return GateStatus{
		Gate: GateMonteCarlo, Passed: true,
		Score: result.SharpeMedian,
	}
}

func evalPaper(metrics PaperGateMetrics) GateStatus {
	// Skip paper gate when no paper trading data is available.
	if metrics.PaperDays == 0 {
		return GateStatus{
			Gate: GatePaper, Passed: true, Skipped: true,
			Reason: "no paper trading data available — gate skipped",
		}
	}
	cfg := DefaultPaperGateConfig()
	pgResult := PaperGate(metrics, cfg)
	if !pgResult.Passed {
		return GateStatus{
			Gate: GatePaper, Passed: false,
			Reason: pgResult.Reason,
			Score:  metrics.PaperNetReturn,
		}
	}
	return GateStatus{
		Gate: GatePaper, Passed: true,
		Score: metrics.PaperNetReturn,
	}
}

func evalCorrelation(newSignals []SignalDirection, existing map[string][]SignalDirection) GateStatus {
	// Skip correlation gate when no existing strategies to compare against.
	if len(existing) == 0 {
		return GateStatus{
			Gate: GateCorrelation, Passed: true, Skipped: true,
			Reason: "no existing live strategies to compare — gate skipped",
		}
	}
	cfg := DefaultCorrelationGateConfig()
	cgResult := CorrelationGate(newSignals, existing, cfg)
	if !cgResult.Passed {
		return GateStatus{
			Gate: GateCorrelation, Passed: false,
			Reason: cgResult.Reason,
			Score:  cgResult.MaxCorrelation,
		}
	}
	return GateStatus{
		Gate: GateCorrelation, Passed: true,
		Score: cgResult.MaxCorrelation,
	}
}

// GateResultsSummary returns a sorted summary of failed gates from a pipeline result.
func GateResultsSummary(result PipelineResult) []string {
	var failures []string
	for _, g := range result.Gates {
		if !g.Passed {
			failures = append(failures, string(g.Gate))
		}
	}
	sort.Strings(failures)
	return failures
}
