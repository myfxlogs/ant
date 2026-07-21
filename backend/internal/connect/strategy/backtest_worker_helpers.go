package strategy

import (
	"log"
	"strconv"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// paramsProtoToMap converts proto binary StrategyParams to a map for ExecuteBacktestRequest.
func paramsProtoToMap(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var sp antv1.StrategyParams
	if err := proto.Unmarshal(raw, &sp); err != nil {
		return nil
	}
	return sp.GetValues()
}

func parseDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		log.Printf("parseDecimal: invalid value %q: %v", s, err)
		return decimal.Zero
	}
	return d
}

func parseInt64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func parseInt32(s string) int32 {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(n)
}

// assessRisk produces a basic ExecuteRiskAssessment from backtest metrics.
// ADR-0023 §5.5 #14: transparency for the user.
func assessRisk(m *antv1.BacktestMetrics) *antv1.ExecuteRiskAssessment {
	if m == nil {
		return &antv1.ExecuteRiskAssessment{Score: 50, Level: "unknown", IsReliable: false}
	}

	score := 50
	var reasons, warnings []string

	maxDD := decimal.RequireFromString(m.MaxDrawdown)
	sharpe := decimal.RequireFromString(m.SharpeRatio)
	winRate := decimal.RequireFromString(m.WinRate)
	profitFactor := decimal.RequireFromString(m.ProfitFactor)

	switch {
	case maxDD.GreaterThan(decimal.NewFromFloat(0.5)):
		score -= 25
		warnings = append(warnings, "Max drawdown exceeds 50%")
	case maxDD.GreaterThan(decimal.NewFromFloat(0.3)):
		score -= 15
		reasons = append(reasons, "High drawdown (30-50%)")
	case maxDD.GreaterThan(decimal.NewFromFloat(0.15)):
		score -= 8
		reasons = append(reasons, "Moderate drawdown (15-30%)")
	default:
		reasons = append(reasons, "Low drawdown (<15%)")
	}

	switch {
	case sharpe.GreaterThanOrEqual(decimal.NewFromFloat(2.0)):
		score += 20
		reasons = append(reasons, "Excellent Sharpe ratio (≥2.0)")
	case sharpe.GreaterThanOrEqual(decimal.NewFromFloat(1.0)):
		score += 10
		reasons = append(reasons, "Good Sharpe ratio (≥1.0)")
	case sharpe.LessThan(decimal.Zero):
		score -= 15
		warnings = append(warnings, "Negative Sharpe ratio")
	default:
		score -= 5
		reasons = append(reasons, "Low Sharpe ratio (<1.0)")
	}

	if winRate.GreaterThanOrEqual(decimal.NewFromFloat(0.6)) {
		score += 10
	} else if winRate.LessThan(decimal.NewFromFloat(0.3)) {
		score -= 10
		warnings = append(warnings, "Low win rate (<30%)")
	}

	if m.TotalTrades < 10 {
		warnings = append(warnings, "Insufficient trades for reliable assessment")
	}

	if profitFactor.GreaterThan(decimal.Zero) && profitFactor.LessThan(decimal.NewFromFloat(1.0)) {
		score -= 10
		warnings = append(warnings, "Profit factor below 1.0 (unprofitable)")
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	level := "medium"
	switch {
	case score >= 70:
		level = "low"
	case score < 40:
		level = "high"
	}

	return &antv1.ExecuteRiskAssessment{
		Score:      int32(score),
		Level:      level,
		Reasons:    reasons,
		Warnings:   warnings,
		IsReliable: m.TotalTrades >= 10,
	}
}
