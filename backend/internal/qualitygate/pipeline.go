package qualitygate

import (
	"context"
)

// PipelineResult is the aggregated outcome of all gates in the pipeline.
type PipelineResult struct {
	Passed     bool          `json:"passed"`
	Score      float64       `json:"score"`       // 0.0–1.0 overall quality score
	RiskLevel  string        `json:"risk_level"`  // "low", "medium", "high"
	IsReliable bool          `json:"is_reliable"` // sufficient for live trading
	Results    []*GateResult `json:"results"`     // individual gate results
	Errors     []string      `json:"errors"`      // critical failure reasons
	Warnings   []string      `json:"warnings"`    // non-critical warnings
}

// Pipeline executes quality gates in order and aggregates results.
type Pipeline struct {
	gates []QualityGate
}

// NewPipeline creates a gate pipeline with the given gates (executed in order).
func NewPipeline(gates ...QualityGate) *Pipeline {
	return &Pipeline{gates: gates}
}

// Run executes all gates sequentially. A CRITICAL failure stops execution.
func (p *Pipeline) Run(ctx context.Context, info *StrategyInfo) *PipelineResult {
	pr := &PipelineResult{
		Passed:  true,
		Score:   1.0,
		Results: make([]*GateResult, 0, len(p.gates)),
	}

	totalWeight := 0.0
	weightedScore := 0.0
	// Gate weights: syntax=0.30, risk=0.25, backtest=0.30, reliability=0.15
	weights := map[string]float64{
		"syntax":      0.30,
		"risk":        0.25,
		"backtest":    0.30,
		"reliability": 0.15,
	}

	for _, gate := range p.gates {
		result := gate.Evaluate(ctx, info)
		pr.Results = append(pr.Results, result)

		w := weights[gate.Name()]
		if w <= 0 {
			w = 0.25
		}
		weightedScore += result.Score * w
		totalWeight += w

		// Collect warnings and errors
		for _, w := range result.Warnings {
			pr.Warnings = append(pr.Warnings, "["+gate.Name()+"] "+w)
		}
		if !result.Passed {
			pr.Errors = append(pr.Errors, "["+gate.Name()+"] "+result.Reason)
			pr.Passed = false
		}

		// Stop on critical failure
		if result.Severity == SeverityCritical {
			break
		}
	}

	if totalWeight > 0 {
		pr.Score = weightedScore / totalWeight
	}
	if pr.Score < 0 {
		pr.Score = 0
	}

	// Determine risk level
	pr.RiskLevel = computeRiskLevel(pr)
	pr.IsReliable = pr.Passed && pr.Score >= 0.6

	return pr
}

func computeRiskLevel(pr *PipelineResult) string {
	if pr.Score >= 0.8 {
		return "low"
	}
	if pr.Score >= 0.5 {
		return "medium"
	}
	return "high"
}

// DefaultPipeline returns the standard four-gate pipeline.
func DefaultPipeline() *Pipeline {
	return NewPipeline(
		&SyntaxGate{MinCodeLength: 20},
		&RiskGate{},
		&BacktestGate{},
		&ReliabilityGate{},
	)
}
