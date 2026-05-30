// Package ai provides the 6-Gate Pipeline (M10-BASE-E6).
//
// The gate pipeline evaluates AI-generated strategies through six sequential gates:
//
//	Compliance → LookAhead → Walk-Forward+CPCV → DeflatedSharpe → Paper(14d) → Correlation
//
// Only strategies that pass all 6 gates are eligible for PromoteToLive.
// PromoteToLive conditions: Sharpe > 0, DSR >= 0.95, Paper ≥ 14d Net P&L > 0, Correlation < 0.7.

package ai

import (
	"sort"
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
	GatePaper          GateName = "paper"
	GateCorrelation    GateName = "correlation"
)

// GateOrder is the canonical 6-gate evaluation order.
var GateOrder = []GateName{
	GateCompliance,
	GateLookAhead,
	GateWalkForward,
	GateDeflatedSharpe,
	GatePaper,
	GateCorrelation,
}

// GateStatus represents a single gate's evaluation result.
type GateStatus struct {
	Gate     GateName `json:"gate"`
	Passed   bool     `json:"passed"`
	Reason   string   `json:"reason,omitempty"`
	Score    float64  `json:"score,omitempty"`    // numeric score (DSR, correlation, etc.)
	Duration int64    `json:"duration_ms"`        // evaluation time in ms
}

// PipelineResult is the aggregate result of running the full 6-gate pipeline.
type PipelineResult struct {
	Passed        bool          `json:"passed"`
	Gates         []GateStatus  `json:"gates"`
	FirstFail     GateName      `json:"first_fail,omitempty"`
	Summary       string        `json:"summary"`
	TotalDuration time.Duration `json:"total_duration_ms"`
}

// PipelineInput bundles all the data needed for gate evaluation.
type PipelineInput struct {
	Expression     string                     // DSL expression for lookahead scanning
	DailyReturns   []float64                  // daily P&L returns for walk-forward and DSR
	NumAttempts    int                        // number of user strategy attempts
	PaperMetrics   PaperGateMetrics           // paper trading metrics
	NewSignals     []SignalDirection           // new strategy's signal directions
	ExistingSignals map[string][]SignalDirection // existing live strategies' signals
}

// Pipeline evaluates a strategy through all 6 gates in order.
// Stops at the first failing gate.
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
		case GatePaper:
			status = evalPaper(input.PaperMetrics)
		case GateCorrelation:
			status = evalCorrelation(input.NewSignals, input.ExistingSignals)
		}

		status.Duration = time.Since(gateStart).Milliseconds()
		result.Gates = append(result.Gates, status)

		if !status.Passed {
			result.Passed = false
			result.FirstFail = gate
			result.Summary = status.Reason
			result.TotalDuration = time.Since(startedAt)
			return result
		}
	}

	result.Summary = "all 6 gates passed"
	result.TotalDuration = time.Since(startedAt)
	return result
}

// --- Individual gate evaluators ---

func evalCompliance(input PipelineInput) GateStatus {
	// Compliance: expression must be non-empty.
	if input.Expression == "" {
		return GateStatus{Gate: GateCompliance, Passed: false, Reason: "empty DSL expression"}
	}
	return GateStatus{Gate: GateCompliance, Passed: true}
}

func evalLookAhead(expression string) GateStatus {
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
	wfResult := WalkForward(dailyReturns, cfg)
	if !wfResult.Passed {
		return GateStatus{
			Gate: GateWalkForward, Passed: false,
			Reason: wfResult.Reason,
			Score:  wfResult.SharpeDiff,
		}
	}
	return GateStatus{
		Gate: GateWalkForward, Passed: true,
		Score: wfResult.SharpeDiff,
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

func evalPaper(metrics PaperGateMetrics) GateStatus {
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

// --- PromoteToLive ---

// PromoteToLiveConditions bundles the criteria for promoting a strategy to live.
type PromoteToLiveConditions struct {
	MinPaperDays      int     // minimum paper trading days (default 14)
	MinDSR            float64 // minimum deflated Sharpe ratio (default 0.95)
	MinPaperNetPnL    float64 // minimum paper Net P&L (must be > 0)
	MaxCorrelation    float64 // maximum allowed signal correlation (default 0.7)
}

// DefaultPromoteConditions returns standard promotion criteria.
func DefaultPromoteConditions() PromoteToLiveConditions {
	return PromoteToLiveConditions{
		MinPaperDays:   14,
		MinDSR:         0.95,
		MinPaperNetPnL: 0,
		MaxCorrelation: 0.7,
	}
}

// PromoteToLive evaluates whether a strategy meets all conditions for live deployment.
// It checks: DSR >= 0.95, Paper ≥ 14d Net P&L > 0, Correlation < 0.7.
func PromoteToLive(metrics PaperGateMetrics, dsr float64, maxCorrelation float64, cond PromoteToLiveConditions) (bool, string) {
	if metrics.PaperDays < cond.MinPaperDays {
		return false, "insufficient paper trading days"
	}
	if metrics.PaperNetPnL <= cond.MinPaperNetPnL {
		return false, "paper Net P&L not positive"
	}
	if dsr < cond.MinDSR {
		return false, "deflated Sharpe below threshold"
	}
	if maxCorrelation >= cond.MaxCorrelation {
		return false, "signal correlation too high with existing strategies"
	}
	return true, "ready for live deployment"
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
