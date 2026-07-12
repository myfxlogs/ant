// Package ai provides strategy promotion criteria (M10-BASE-E6).
//
// PromoteToLive evaluates whether a strategy meets all conditions for live deployment
// after passing the 7-gate pipeline.

package ai

// PromoteToLiveConditions bundles the criteria for promoting a strategy to live.
type PromoteToLiveConditions struct {
	MinPaperDays   int     // minimum paper trading days (default 14)
	MinDSR         float64 // minimum deflated Sharpe ratio (default 0.95)
	MinPaperNetPnL float64 // minimum paper Net P&L (must be > 0)
	MaxCorrelation float64 // maximum allowed signal correlation (default 0.7)
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
