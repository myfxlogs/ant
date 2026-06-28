package backtest

import (
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/strategy/sdk"
)

// ParityConfig controls the tolerance for trade comparison.
type ParityConfig struct {
	// TimeTolerance is the maximum allowed difference in trade open/close times.
	// MT Strategy Tester and Go SimBroker may differ by a few seconds due to
	// bar timestamp interpretation.
	TimeTolerance time.Duration

	// PriceTolerance is the maximum allowed price difference as a decimal fraction.
	// e.g. 0.0001 = 1 pip for EURUSD. Accounts for slippage differences.
	PriceTolerance decimal.Decimal

	// VolumeTolerance is the maximum allowed volume difference (in lots).
	VolumeTolerance decimal.Decimal

	// RequireExactCount if true, fails if trade counts differ.
	// If false, allows missing/extra trades (reported as mismatches but not fatal).
	RequireExactCount bool
}

// DefaultParityConfig returns sensible defaults for forex strategies.
func DefaultParityConfig() ParityConfig {
	return ParityConfig{
		TimeTolerance:    2 * time.Minute,
		PriceTolerance:   decimal.NewFromFloat(0.0002), // 2 pips
		VolumeTolerance:  decimal.NewFromFloat(0.01),
		RequireExactCount: false,
	}
}

// ParityMismatchType classifies a mismatch.
type ParityMismatchType string

const (
	MismatchMissingInGo   ParityMismatchType = "missing_in_go"   // trade exists in MT but not Go
	MismatchMissingInMT   ParityMismatchType = "missing_in_mt"   // trade exists in Go but not MT
	MismatchSide          ParityMismatchType = "side_mismatch"   // buy vs sell
	MismatchVolume        ParityMismatchType = "volume_mismatch" // volume differs beyond tolerance
	MismatchOpenPrice     ParityMismatchType = "open_price_mismatch"
	MismatchClosePrice    ParityMismatchType = "close_price_mismatch"
	MismatchOpenTime      ParityMismatchType = "open_time_mismatch"
	MismatchCloseTime     ParityMismatchType = "close_time_mismatch"
	MismatchProfit        ParityMismatchType = "profit_mismatch"
	MismatchTradeCount    ParityMismatchType = "trade_count_mismatch"
)

// ParityMismatch describes a single difference between Go and MT results.
type ParityMismatch struct {
	Type       ParityMismatchType
	GoTrade    *Trade
	MTTrade    *MTReportTrade
	GoIndex    int
	MTIndex    int
	GoValue    string
	MTValue    string
	Diff       string
	Severity   string // "fatal" | "warning" | "info"
}

// ParityReport is the complete comparison result.
type ParityReport struct {
	Passed       bool
	GoTradeCount int
	MTTradeCount int
	MatchedCount int
	Mismatches   []ParityMismatch

	// Summary metrics comparison.
	GoTotalProfit  decimal.Decimal
	MTTotalProfit  decimal.Decimal
	ProfitDiff     decimal.Decimal

	// Config used for comparison.
	Config ParityConfig
}

// Passed returns true if the parity check passes (no fatal mismatches).
func (r *ParityReport) HasFatalMismatches() bool {
	for _, m := range r.Mismatches {
		if m.Severity == "fatal" {
			return true
		}
	}
	return false
}

// CompareParity compares Go backtest trades against MT reference trades.
// It aligns trades by open time, then checks side, volume, price, and profit.
func CompareParity(goTrades []Trade, mtTrades []MTReportTrade, cfg ParityConfig) *ParityReport {
	report := &ParityReport{
		Config:        cfg,
		GoTradeCount:  len(goTrades),
		MTTradeCount:  len(mtTrades),
		GoTotalProfit: sumGoProfit(goTrades),
		MTTotalProfit: sumMTProfit(mtTrades),
	}
	report.ProfitDiff = report.GoTotalProfit.Sub(report.MTTotalProfit)

	// Sort both by open time.
	sortedGo := make([]Trade, len(goTrades))
	copy(sortedGo, goTrades)
	sort.Slice(sortedGo, func(i, j int) bool {
		return sortedGo[i].EntryTime.Before(sortedGo[j].EntryTime)
	})

	sortedMT := make([]MTReportTrade, len(mtTrades))
	copy(sortedMT, mtTrades)
	sort.Slice(sortedMT, func(i, j int) bool {
		return sortedMT[i].OpenTime.Before(sortedMT[j].OpenTime)
	})

	// Greedy alignment by closest open time.
	usedMT := make([]bool, len(sortedMT))
	for i, gt := range sortedGo {
		bestMT := -1
		bestDiff := cfg.TimeTolerance + 1
		for j, mt := range sortedMT {
			if usedMT[j] {
				continue
			}
			diff := gt.EntryTime.Sub(mt.OpenTime)
			if diff < 0 {
				diff = -diff
			}
			if diff < bestDiff {
				bestDiff = diff
				bestMT = j
			}
		}

		if bestMT < 0 {
			report.Mismatches = append(report.Mismatches, ParityMismatch{
				Type:     MismatchMissingInMT,
				GoTrade:  &sortedGo[i],
				GoIndex:  i,
				Severity: "warning",
				GoValue:  fmt.Sprintf("open=%s side=%d vol=%s", gt.EntryTime.Format(time.RFC3339), gt.Side, gt.Volume.String()),
			})
			continue
		}

		usedMT[bestMT] = true
		mt := sortedMT[bestMT]
		report.MatchedCount++

		// Compare side. sdk.SideBuy=1, sdk.SideSell=-1.
		goSide := "buy"
		if gt.Side == sdk.SideSell {
			goSide = "sell"
		}
		if goSide != mt.Side {
			report.Mismatches = append(report.Mismatches, ParityMismatch{
				Type:     MismatchSide,
				GoTrade:  &sortedGo[i],
				MTTrade:  &mt,
				GoIndex:  i,
				MTIndex:  bestMT,
				GoValue:  goSide,
				MTValue:  mt.Side,
				Severity: "fatal",
			})
		}

		// Compare volume.
		volDiff := gt.Volume.Sub(mt.Volume).Abs()
		if volDiff.GreaterThan(cfg.VolumeTolerance) {
			report.Mismatches = append(report.Mismatches, ParityMismatch{
				Type:     MismatchVolume,
				GoTrade:  &sortedGo[i],
				MTTrade:  &mt,
				GoIndex:  i,
				MTIndex:  bestMT,
				GoValue:  gt.Volume.String(),
				MTValue:  mt.Volume.String(),
				Diff:     volDiff.String(),
				Severity: "warning",
			})
		}

		// Compare open price.
		priceDiff := gt.EntryPrice.Sub(mt.OpenPrice).Abs()
		if priceDiff.GreaterThan(cfg.PriceTolerance) {
			report.Mismatches = append(report.Mismatches, ParityMismatch{
				Type:     MismatchOpenPrice,
				GoTrade:  &sortedGo[i],
				MTTrade:  &mt,
				GoIndex:  i,
				MTIndex:  bestMT,
				GoValue:  gt.EntryPrice.String(),
				MTValue:  mt.OpenPrice.String(),
				Diff:     priceDiff.String(),
				Severity: "warning",
			})
		}

		// Compare close price (if MT report has it).
		if !mt.ClosePrice.IsZero() && !gt.ExitPrice.IsZero() {
			closeDiff := gt.ExitPrice.Sub(mt.ClosePrice).Abs()
			if closeDiff.GreaterThan(cfg.PriceTolerance) {
				report.Mismatches = append(report.Mismatches, ParityMismatch{
					Type:     MismatchClosePrice,
					GoTrade:  &sortedGo[i],
					MTTrade:  &mt,
					GoIndex:  i,
					MTIndex:  bestMT,
					GoValue:  gt.ExitPrice.String(),
					MTValue:  mt.ClosePrice.String(),
					Diff:     closeDiff.String(),
					Severity: "warning",
				})
			}
		}

		// Compare profit.
		profitDiff := gt.Profit.Sub(mt.Profit).Abs()
		profitTolerance := cfg.PriceTolerance.Mul(gt.Volume).Mul(decimal.NewFromInt(100))
		if profitDiff.GreaterThan(profitTolerance) {
			report.Mismatches = append(report.Mismatches, ParityMismatch{
				Type:     MismatchProfit,
				GoTrade:  &sortedGo[i],
				MTTrade:  &mt,
				GoIndex:  i,
				MTIndex:  bestMT,
				GoValue:  gt.Profit.String(),
				MTValue:  mt.Profit.String(),
				Diff:     profitDiff.String(),
				Severity: "warning",
			})
		}
	}

	// Check for MT trades not matched to Go trades.
	for j, mt := range sortedMT {
		if !usedMT[j] {
			report.Mismatches = append(report.Mismatches, ParityMismatch{
				Type:     MismatchMissingInGo,
				MTTrade:  &sortedMT[j],
				MTIndex:  j,
				Severity: "warning",
				MTValue:  fmt.Sprintf("open=%s side=%s vol=%s", mt.OpenTime.Format(time.RFC3339), mt.Side, mt.Volume.String()),
			})
		}
	}

	// Check trade count if required.
	if cfg.RequireExactCount && report.GoTradeCount != report.MTTradeCount {
		report.Mismatches = append(report.Mismatches, ParityMismatch{
			Type:     MismatchTradeCount,
			GoValue:  fmt.Sprintf("%d", report.GoTradeCount),
			MTValue:  fmt.Sprintf("%d", report.MTTradeCount),
			Severity: "fatal",
		})
	}

	report.Passed = !report.HasFatalMismatches()
	return report
}

// FormatReport produces a human-readable summary of the parity report.
func (r *ParityReport) FormatReport() string {
	status := "PASS"
	if !r.Passed {
		status = "FAIL"
	}

	out := fmt.Sprintf("=== Backtest Parity Report: %s ===\n", status)
	out += fmt.Sprintf("Go trades: %d | MT trades: %d | Matched: %d\n",
		r.GoTradeCount, r.MTTradeCount, r.MatchedCount)
	out += fmt.Sprintf("Go total profit: %s | MT total profit: %s | Diff: %s\n",
		r.GoTotalProfit.String(), r.MTTotalProfit.String(), r.ProfitDiff.String())

	fatal, warning := 0, 0
	for _, m := range r.Mismatches {
		if m.Severity == "fatal" {
			fatal++
		} else {
			warning++
		}
	}
	out += fmt.Sprintf("Mismatches: %d fatal, %d warning\n\n", fatal, warning)

	for i, m := range r.Mismatches {
		out += fmt.Sprintf("[%d] %s (%s)\n", i+1, m.Type, m.Severity)
		if m.GoValue != "" || m.MTValue != "" {
			out += fmt.Sprintf("  Go: %s | MT: %s", m.GoValue, m.MTValue)
			if m.Diff != "" {
				out += fmt.Sprintf(" | diff: %s", m.Diff)
			}
			out += "\n"
		}
	}

	return out
}

func sumGoProfit(trades []Trade) decimal.Decimal {
	total := decimal.Zero
	for _, t := range trades {
		total = total.Add(t.Profit)
	}
	return total
}

func sumMTProfit(trades []MTReportTrade) decimal.Decimal {
	total := decimal.Zero
	for _, t := range trades {
		total = total.Add(t.Profit)
	}
	return total
}
