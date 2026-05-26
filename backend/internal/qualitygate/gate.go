// Package qualitygate provides multi-level quality gates for AI-generated strategies (M10-BASE-E).
//
// Each gate evaluates a strategy from a specific dimension:
//
//	SyntaxGate     — code validity, import checks, lint compliance
//	RiskGate       — parameter risk scoring (leverage, drawdown, concentration)
//	BacktestGate   — backtest metric thresholds (Sharpe, drawdown, win rate)
//	ReliabilityGate— combined reliability assessment
//
// Gates are executed in order by the GatePipeline. A CRITICAL failure stops the pipeline.
// WARNING failures are accumulated and reported.

package qualitygate

import "context"

// Severity of a gate failure.
type Severity int

const (
	SeverityPass    Severity = 0
	SeverityWarning Severity = 1
	SeverityError   Severity = 2
	SeverityCritical Severity = 3
)

// String returns the severity label.
func (s Severity) String() string {
	switch s {
	case SeverityPass:
		return "PASS"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// GateResult is the outcome of a single quality gate evaluation.
type GateResult struct {
	Gate     string   `json:"gate"`
	Passed   bool     `json:"passed"`
	Severity Severity `json:"severity"`
	Score    float64  `json:"score"` // 0.0–1.0 normalized score
	Reason   string   `json:"reason"`
	Warnings []string `json:"warnings"`
	Details  map[string]interface{} `json:"details"`
}

// QualityGate evaluates a strategy and returns a gate result.
type QualityGate interface {
	Name() string
	Evaluate(ctx context.Context, info *StrategyInfo) *GateResult
}

// StrategyInfo is the input to all quality gates.
type StrategyInfo struct {
	Code       string            `json:"code"`
	Language   string            `json:"language"` // "python", "mql5", "mql4"
	Parameters map[string]float64 `json:"parameters"`
	Schedule   *ScheduleInfo     `json:"schedule"`
	Backtest   *BacktestInfo     `json:"backtest"`
}

// ScheduleInfo holds the strategy schedule configuration.
type ScheduleInfo struct {
	MaxPositions     int     `json:"max_positions"`
	RiskPerTradePct  float64 `json:"risk_per_trade_pct"`
	StopLossPct      float64 `json:"stop_loss_pct"`
	TakeProfitPct    float64 `json:"take_profit_pct"`
	MaxDrawdownPct   float64 `json:"max_drawdown_pct"`
	DailyTradeLimit  int     `json:"daily_trade_limit"`
	LeverageAllowed  float64 `json:"leverage_allowed"`
}

// BacktestInfo holds the results of the most recent backtest.
type BacktestInfo struct {
	TotalReturn   float64 `json:"total_return"`
	AnnualReturn  float64 `json:"annual_return"`
	MaxDrawdown   float64 `json:"max_drawdown"`
	SharpeRatio   float64 `json:"sharpe_ratio"`
	WinRate       float64 `json:"win_rate"`
	ProfitFactor  float64 `json:"profit_factor"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	AverageProfit float64 `json:"average_profit"`
	AverageLoss   float64 `json:"average_loss"`
}

// DefaultSchedule returns sensible default schedule parameters.
func DefaultSchedule() *ScheduleInfo {
	return &ScheduleInfo{
		MaxPositions:     5,
		RiskPerTradePct:  0.02,
		StopLossPct:      0,
		TakeProfitPct:    0,
		MaxDrawdownPct:   0.25,
		DailyTradeLimit:  20,
		LeverageAllowed:  100,
	}
}
