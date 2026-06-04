// scoring.go: 7-component regime-aware strategy scoring.
// Scores backtest results with weights that adapt to market regime.
// Adapted from QuantDinger experiment/scoring.py.

package ai

import "math"

// ScoreWeights defines the weight distribution across 7 components.
type ScoreWeights struct {
	Return       float64
	AnnualReturn float64
	Sharpe       float64
	ProfitFactor float64
	WinRate      float64
	Drawdown     float64
	Stability    float64
}

// DefaultWeights are the balanced scoring weights.
var DefaultWeights = ScoreWeights{
	Return: 0.22, AnnualReturn: 0.12, Sharpe: 0.18,
	ProfitFactor: 0.14, WinRate: 0.09, Drawdown: 0.15, Stability: 0.10,
}

// RegimeWeights returns scoring weights adjusted for the given regime.
func RegimeWeights(r MarketRegime) ScoreWeights {
	switch r {
	case RegimeBullTrend:
		return ScoreWeights{Return: 0.30, AnnualReturn: 0.18, Sharpe: 0.20,
			ProfitFactor: 0.12, WinRate: 0.06, Drawdown: 0.08, Stability: 0.06}
	case RegimeBearTrend:
		return ScoreWeights{Return: 0.14, AnnualReturn: 0.10, Sharpe: 0.20,
			ProfitFactor: 0.14, WinRate: 0.07, Drawdown: 0.22, Stability: 0.13}
	case RegimeRangeCompression:
		return ScoreWeights{Return: 0.10, AnnualReturn: 0.08, Sharpe: 0.14,
			ProfitFactor: 0.14, WinRate: 0.20, Drawdown: 0.14, Stability: 0.20}
	case RegimeHighVolatility:
		return ScoreWeights{Return: 0.15, AnnualReturn: 0.10, Sharpe: 0.15,
			ProfitFactor: 0.10, WinRate: 0.05, Drawdown: 0.30, Stability: 0.15}
	default:
		return DefaultWeights
	}
}

// ScoredResult holds the scored output of a backtest.
type ScoredResult struct {
	Score        float64
	Grade        string
	Components   map[string]float64 // per-component 0-100 scores
	Trades       int
	SharpeRatio  float64
	TotalReturn  float64
	MaxDrawdown  float64
	WinRate      float64
	ProfitFactor float64
	AnnualReturn float64
}

// Score evaluates a backtest result using regime-aware weights.
func Score(metrics *BacktestMetrics, regime MarketRegime) *ScoredResult {
	if metrics == nil {
		return &ScoredResult{Score: 0, Grade: "E"}
	}
	weights := RegimeWeights(regime)

	// Per-component 0-100 scores
	cs := map[string]float64{
		"return":        boundedScore(metrics.TotalReturn, -20, 80),
		"annual_return": boundedScore(metrics.AnnualReturn, -15, 60),
		"sharpe":        boundedScore(metrics.SharpeRatio, -1, 3),
		"profit_factor": boundedScore(metrics.ProfitFactor, 0.5, 3),
		"win_rate":      boundedScore(metrics.WinRate, 20, 80),
		"drawdown":      inverseScore(metrics.MaxDrawdown, 5, 45),
		"stability":     boundedScore(metrics.Stability, 0, 1),
	}

	// Weighted score
	weighted := weights.Return*cs["return"] +
		weights.AnnualReturn*cs["annual_return"] +
		weights.Sharpe*cs["sharpe"] +
		weights.ProfitFactor*cs["profit_factor"] +
		weights.WinRate*cs["win_rate"] +
		weights.Drawdown*cs["drawdown"] +
		weights.Stability*cs["stability"]

	// Regime fit bonus (average of top 3 regime-aligned components)
	regimeFit := top3Avg(cs, regime)

	overall := clampScore(weighted*0.88 + regimeFit*0.12)

	// Penalties
	if metrics.TotalTrades < 5 {
		overall -= 12
	} else if metrics.TotalTrades < 12 {
		overall -= 5
	}
	overall = clampScore(overall)

	return &ScoredResult{
		Score:        math.Round(overall*10) / 10,
		Grade:        gradeForScore(overall),
		Components:   cs,
		Trades:       metrics.TotalTrades,
		SharpeRatio:  math.Round(metrics.SharpeRatio*100) / 100,
		TotalReturn:  math.Round(metrics.TotalReturn*100) / 100,
		MaxDrawdown:  math.Round(metrics.MaxDrawdown*100) / 100,
		WinRate:      math.Round(metrics.WinRate*100) / 100,
		ProfitFactor: math.Round(metrics.ProfitFactor*100) / 100,
		AnnualReturn: math.Round(metrics.AnnualReturn*100) / 100,
	}
}

// BacktestMetrics mirrors the key fields returned by any backtest engine.
type BacktestMetrics struct {
	TotalReturn  float64
	AnnualReturn float64
	SharpeRatio  float64
	MaxDrawdown  float64
	WinRate      float64
	ProfitFactor float64
	TotalTrades  int
	Stability    float64 // equity curve monotonicity (0-1)
}

// boundedScore maps value to 0-100 linearly between floor and ceiling.
func boundedScore(value, floor, ceiling float64) float64 {
	if ceiling <= floor {
		return 50
	}
	s := (value - floor) / (ceiling - floor) * 100
	return clampScore(s)
}

// inverseScore maps value inversely (lower = better) between floor and ceiling.
func inverseScore(value, floor, ceiling float64) float64 {
	if ceiling <= floor {
		return 50
	}
	s := (ceiling - value) / (ceiling - floor) * 100
	return clampScore(s)
}

func clampScore(v float64) float64 { return math.Max(0, math.Min(100, v)) }

func gradeForScore(s float64) string {
	switch {
	case s >= 85: return "A"
	case s >= 72: return "B"
	case s >= 60: return "C"
	case s >= 45: return "D"
	default: return "E"
	}
}

// top3Avg returns the average of the top 3 scores most relevant to the regime.
func top3Avg(cs map[string]float64, regime MarketRegime) float64 {
	keys := topKeysForRegime(regime)
	sum := 0.0
	n := 0
	for _, k := range keys {
		if v, ok := cs[k]; ok {
			sum += v
			n++
		}
	}
	if n == 0 {
		return 50
	}
	return sum / float64(n)
}

func topKeysForRegime(r MarketRegime) []string {
	switch r {
	case RegimeBullTrend:
		return []string{"return", "annual_return", "sharpe"}
	case RegimeBearTrend:
		return []string{"drawdown", "sharpe", "stability"}
	case RegimeRangeCompression:
		return []string{"win_rate", "stability", "profit_factor"}
	case RegimeHighVolatility:
		return []string{"drawdown", "stability", "sharpe"}
	default:
		return []string{"sharpe", "return", "drawdown"}
	}
}
