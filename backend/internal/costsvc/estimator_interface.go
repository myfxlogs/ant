// Package costsvc provides the CostEstimator interface for pre-trade cost estimation (M10-BASE-D2).
package costsvc

import "context"

// CostEstimator estimates trading costs for a planned order.
// Implementations may use static CostModel tables or real-time spread data.
type CostEstimator interface {
	Estimate(ctx context.Context, params EstimateParams) CostBreakdown
}

// StaticEstimator uses a fixed CostModel for all estimates.
type StaticEstimator struct {
	Model *CostModel
}

func (e *StaticEstimator) Estimate(_ context.Context, params EstimateParams) CostBreakdown {
	return e.Model.Estimate(params)
}

// MultiModelEstimator selects a CostModel by symbol lookup.
type MultiModelEstimator struct {
	Models map[string]*CostModel // symbol → model
	Default *CostModel
}

func (e *MultiModelEstimator) Estimate(_ context.Context, params EstimateParams) CostBreakdown {
	m, ok := e.Models[params.Symbol]
	if !ok {
		if e.Default != nil {
			m = e.Default
		} else {
			return CostBreakdown{}
		}
	}
	return m.Estimate(params)
}
