// Package risksvc provides VaR / CVaR calculation and stress test scenarios (M10-BASE-D7).
//
// VaR: Historical simulation with 90-day rolling window.
// CVaR: Expected shortfall beyond VaR threshold.
// Stress tests: Predefined scenarios (2008 crash, 2015 SNB, 2020 Covid, FOMC flash crash).

package risksvc

import (
	"math"
	"sort"
)

// VaRConfig holds parameters for VaR calculation.
type VaRConfig struct {
	ConfidenceLevel float64 // e.g., 0.95 for 95% VaR
	WindowDays      int     // lookback window (default 90)
}

// DefaultVaRConfig returns standard VaR parameters.
func DefaultVaRConfig() VaRConfig {
	return VaRConfig{
		ConfidenceLevel: 0.95,
		WindowDays:      90,
	}
}

// VaRResult holds the computed risk metrics.
type VaRResult struct {
	VaR          float64 `json:"var"`
	CVaR         float64 `json:"cvar"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	DailyVol     float64 `json:"daily_vol"`
	AnnualVol    float64 `json:"annual_vol"`
	Confidence   float64 `json:"confidence"`
	WindowDays   int     `json:"window_days"`
	NumReturns   int     `json:"num_returns"`
}

// ComputeVaR runs historical simulation VaR on daily returns.
// returns is a slice of daily P&L returns (e.g., in account currency).
func ComputeVaR(dailyReturns []float64, cfg VaRConfig) VaRResult {
	if len(dailyReturns) == 0 {
		return VaRResult{Confidence: cfg.ConfidenceLevel, WindowDays: cfg.WindowDays}
	}

	sorted := make([]float64, len(dailyReturns))
	copy(sorted, dailyReturns)
	sort.Float64s(sorted)

	// Historical VaR: (1 - confidence) quantile of sorted returns.
	idx := int(math.Ceil(float64(len(sorted)) * (1 - cfg.ConfidenceLevel))) - 1
	if idx < 0 {
		idx = 0
	}
	vr := sorted[idx]

	// CVaR (Expected Shortfall): mean of returns below VaR threshold.
	var cvarSum float64
	var cvarCount int
	for _, r := range sorted {
		if r <= vr {
			cvarSum += r
			cvarCount++
		}
	}
	cvar := 0.0
	if cvarCount > 0 {
		cvar = cvarSum / float64(cvarCount)
	}

	// Daily vol (standard deviation).
	var sum, sumSq float64
	for _, r := range dailyReturns {
		sum += r
		sumSq += r * r
	}
	n := float64(len(dailyReturns))
	mean := sum / n
	variance := sumSq/n - mean*mean
	if variance < 0 {
		variance = 0
	}
	dailyVol := math.Sqrt(variance)

	// Max drawdown.
	maxDD := computeMaxDrawdown(dailyReturns)

	return VaRResult{
		VaR:         vr,
		CVaR:        cvar,
		MaxDrawdown: maxDD,
		DailyVol:    dailyVol,
		AnnualVol:   dailyVol * math.Sqrt(252),
		Confidence:  cfg.ConfidenceLevel,
		WindowDays:  cfg.WindowDays,
		NumReturns:  len(dailyReturns),
	}
}

func computeMaxDrawdown(returns []float64) float64 {
	peak := 0.0
	maxDD := 0.0
	cumulative := 0.0
	for _, r := range returns {
		cumulative += r
		if cumulative > peak {
			peak = cumulative
		}
		dd := peak - cumulative
		if dd > maxDD {
			maxDD = dd
		}
	}
	return maxDD
}

// StressScenario defines a historical stress event for portfolio testing.
type StressScenario struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Shock       float64 `json:"shock"` // P&L shock fraction (e.g., -0.20 = -20%)
}

// StressTestResult holds the outcome of applying a stress scenario to a portfolio.
type StressTestResult struct {
	Scenario       StressScenario `json:"scenario"`
	StartingEquity float64        `json:"starting_equity"`
	ShockedEquity  float64        `json:"shocked_equity"`
	LossAmount     float64        `json:"loss_amount"`
	Passed         bool           `json:"passed"` // true if equity stays above min_equity
}

// PredefinedScenarios returns the standard stress test catalog.
func PredefinedScenarios() []StressScenario {
	return []StressScenario{
		{Name: "2008_crash", Description: "2008 financial crisis — equities -20%", Shock: -0.20},
		{Name: "2015_snb", Description: "2015 SNB franc unpegging — CHF +30% shock", Shock: -0.30},
		{Name: "2020_covid", Description: "2020 Covid crash — broad -15% across assets", Shock: -0.15},
		{Name: "fomc_flash", Description: "FOMC surprise — instantaneous -5% with high vol", Shock: -0.05},
	}
}

// RunStressTests applies all predefined scenarios plus any custom ones to a portfolio.
func RunStressTests(equity float64, minEquity float64, extra ...StressScenario) []StressTestResult {
	scenarios := PredefinedScenarios()
	scenarios = append(scenarios, extra...)

	results := make([]StressTestResult, len(scenarios))
	for i, s := range scenarios {
		shocked := equity * (1 + s.Shock)
		results[i] = StressTestResult{
			Scenario:       s,
			StartingEquity: equity,
			ShockedEquity:  shocked,
			LossAmount:     equity - shocked,
			Passed:         shocked >= minEquity,
		}
	}
	return results
}
