package backtest

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"alphaforge/strategy/sdk"
)

// ParityTestInput holds all inputs for a parity test run.
type ParityTestInput struct {
	// MQLSource is the original MQL4/MQL5 EA source code.
	MQLSource string

	// GoCode is the transpiled Go strategy code (from mql2go).
	// If empty, the caller must provide a pre-transpiled strategy.
	GoCode string

	// Strategy is a pre-constructed strategy instance.
	// If non-nil, takes precedence over GoCode.
	Strategy sdk.Strategy

	// MTReportHTML is the raw HTML from MT Strategy Tester.
	MTReportHTML string

	// Bars is the historical bar data (must match MT Strategy Tester data range).
	Bars []sdk.Bar

	// Config is the backtest configuration.
	Config Config

	// ParityConfig controls comparison tolerance.
	ParityConfig ParityConfig
}

// RunParityTest executes a complete parity comparison:
//  1. Run Go strategy through SimBroker backtest
//  2. Parse MT Strategy Tester report
//  3. Compare trade lists
//  4. Return parity report
func RunParityTest(ctx context.Context, input ParityTestInput) (*ParityReport, error) {
	// Step 1: Run Go backtest.
	if input.Strategy == nil {
		return nil, fmt.Errorf("parity test: strategy is required")
	}
	if len(input.Bars) == 0 {
		return nil, fmt.Errorf("parity test: bars are required")
	}

	engine := New(input.Config, input.Strategy, input.Bars)
	result, err := engine.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("parity test: Go backtest failed: %w", err)
	}

	// Step 2: Parse MT report.
	if input.MTReportHTML == "" {
		return nil, fmt.Errorf("parity test: MT report HTML is required")
	}

	mtReport, err := ParseMTReport(input.MTReportHTML)
	if err != nil {
		return nil, fmt.Errorf("parity test: MT report parse failed: %w", err)
	}

	// Step 3: Compare.
	cfg := input.ParityConfig
	if cfg.TimeTolerance == 0 {
		cfg = DefaultParityConfig()
	}

	report := CompareParity(result.Trades, mtReport.Trades, cfg)
	return report, nil
}

// ParitySummary is a condensed result for API responses.
type ParitySummary struct {
	Passed       bool
	GoTrades     int
	MTTrades     int
	Matched      int
	FatalCount   int
	WarningCount int
	GoProfit     string
	MTProfit     string
	ProfitDiff   string
}

// Summarize produces a concise summary from a ParityReport.
func (r *ParityReport) Summarize() ParitySummary {
	fatal, warning := 0, 0
	for _, m := range r.Mismatches {
		if m.Severity == "fatal" {
			fatal++
		} else {
			warning++
		}
	}
	return ParitySummary{
		Passed:       r.Passed,
		GoTrades:     r.GoTradeCount,
		MTTrades:     r.MTTradeCount,
		Matched:      r.MatchedCount,
		FatalCount:   fatal,
		WarningCount: warning,
		GoProfit:     r.GoTotalProfit.String(),
		MTProfit:     r.MTTotalProfit.String(),
		ProfitDiff:   r.ProfitDiff.String(),
	}
}

// QuickParityCheck is a convenience function for testing a strategy against
// a known-good trade list (e.g. from a previous verified run).
// It runs the backtest and compares against the provided expected trades.
func QuickParityCheck(
	ctx context.Context,
	strategy sdk.Strategy,
	bars []sdk.Bar,
	cfg Config,
	expectedTrades []MTReportTrade,
) (*ParityReport, error) {
	engine := New(cfg, strategy, bars)
	result, err := engine.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("quick parity: backtest failed: %w", err)
	}

	report := CompareParity(result.Trades, expectedTrades, DefaultParityConfig())
	return report, nil
}

// MakeSyntheticMTReport creates an MTReport from a list of trades.
// Useful for generating reference data when an actual MT report is not available
// (e.g. for CI testing where MT Strategy Tester cannot run).
func MakeSyntheticMTReport(trades []MTReportTrade, symbol, timeframe string) *MTReport {
	totalProfit := decimal.Zero
	for _, t := range trades {
		totalProfit = totalProfit.Add(t.Profit)
	}
	profit, loss := 0, 0
	for _, t := range trades {
		if t.Profit.GreaterThanOrEqual(decimal.Zero) {
			profit++
		} else {
			loss++
		}
	}
	return &MTReport{
		Symbol:         symbol,
		Timeframe:      timeframe,
		Trades:         trades,
		TotalNetProfit: totalProfit,
		ProfitTrades:   profit,
		LossTrades:     loss,
		TotalTrades:    len(trades),
	}
}

// MakeMTReportTrade is a convenience constructor for test data.
func MakeMTReportTrade(
	side string,
	volume, openPrice, closePrice, profit decimal.Decimal,
	openTime, closeTime time.Time,
) MTReportTrade {
	return MTReportTrade{
		Side:       side,
		Volume:     volume,
		OpenPrice:  openPrice,
		ClosePrice: closePrice,
		Profit:     profit,
		OpenTime:   openTime,
		CloseTime:  closeTime,
	}
}
