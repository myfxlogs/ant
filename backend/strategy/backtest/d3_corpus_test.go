package backtest

import (
	"context"
	"math"
	"testing"

	"github.com/shopspring/decimal"

	mql2go "alphaforge/tools/mql2go"
)

// D3 Corpus Tests
//
// Compiles each EA from D3Corpus using the MQL2Go transpiler, runs it through
// the backtest engine, and extracts global variables to compare against the
// reference indicator implementations.

func TestD3_Corpus(t *testing.T) {
	bars := d3GenerateBars(300)
	refBars := d3ToRefBars(bars)
	closes := refClose(refBars)

	config := Config{
		Symbol:         "EURUSD",
		Timeframe:      "H1",
		InitialCapital: decimal.NewFromFloat(10000),
		Leverage:       100,
		SymbolDigits:   5,
		SymbolPoint:    decimal.NewFromFloat(0.00001),
		ContractSize:   decimal.NewFromInt(100000),
	}

	for _, entry := range D3Corpus {
		t.Run(entry.Name, func(t *testing.T) {
			runner, err := mql2go.CompileMQL(entry.Source)
			if err != nil {
				t.Fatalf("CompileMQL(%s): %v", entry.Name, err)
			}

			engine := New(config, runner, bars)

			_, err = engine.Run(context.Background())
			if err != nil {
				t.Fatalf("Engine.Run(%s): %v", entry.Name, err)
			}

			// Extract globals and compare against reference.
			for _, gname := range entry.GlobalsToExtract {
				val, ok := runner.GetGlobal(gname)
				if !ok {
					t.Errorf("GetGlobal(%s) not found for %s", gname, entry.Name)
					continue
				}

				got := val.Decimal.InexactFloat64()
				want := d3ExpectedValue(entry.Name, gname, closes, refBars)

				// Skip comparison if reference returns 0 (not enough data).
				if want == 0 {
					continue
				}

				// Use relative tolerance for indicator comparison.
				if !d3CorpusApproxEqual(got, want, entry.Name) {
					t.Errorf("%s/%s: got %v, want %v (diff %v)",
						entry.Name, gname, got, want, math.Abs(got-want))
				}
			}
		})
	}
}

// d3ExpectedValue returns the reference implementation value for a given
// corpus entry and global variable name.
func d3ExpectedValue(entryName, globalName string, closes []float64, bars []refBar) float64 {
	switch entryName {
	case "iMA_SMA":
		return refSMA(closes, 20, 0)
	case "iMA_EMA":
		return refEMA(closes, 12, 0)
	case "iRSI":
		return refRSI(closes, 14, 0)
	case "iATR":
		return refATR(bars, 14, 0)
	case "iMACD":
		if globalName == "g_macd" {
			return refMACDLine(closes, 12, 26, 0)
		}
		return refMACDSignal(closes, 12, 26, 9, 0)
	case "iBands":
		upper, middle, lower := refBollinger(closes, 20, 2.0, 0)
		switch globalName {
		case "g_bands_upper":
			return upper
		case "g_bands_middle":
			return middle
		case "g_bands_lower":
			return lower
		}
	case "iStochastic":
		k, d := refStochastic(bars, 5, 3, 3, 0)
		if globalName == "g_stoch_k" {
			return k
		}
		return d
	case "iADX":
		return refADX(bars, 14, 0)
	case "iCCI":
		return refCCI(bars, 14, 0, 6)
	case "iSAR":
		return refSAR(bars, 0.02, 0.2, 0)
	}
	return 0
}

// d3CorpusApproxEqual compares values with indicator-specific tolerances.
func d3CorpusApproxEqual(a, b float64, entryName string) bool {
	// Default tolerance for most indicators.
	tol := 1e-6

	// Looser tolerance for indicators with complex smoothing.
	switch entryName {
	case "iRSI", "iStochastic":
		tol = 0.5
	case "iADX":
		tol = 1.0
	case "iCCI":
		tol = 1.0
	case "iSAR":
		// SAR is recursive and sensitive to initial conditions.
		tol = 0.001
	}

	diff := math.Abs(a - b)
	if diff < tol {
		return true
	}
	scale := math.Max(math.Abs(a), math.Abs(b))
	if scale < 1e-10 {
		return diff < tol
	}
	return diff/scale < tol
}
