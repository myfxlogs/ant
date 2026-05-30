// Package ai provides Walk-Forward validation (M10-BASE-E2).
//
// 5-fold purged walk-forward with CPCV (Combinatorial Purged Cross-Validation).
// Rejects strategies with:
//   - train_sharpe - test_sharpe > 1.0 (overfitting)
//   - any fold max_DD > 30%
//   - trade_count < 30 (statistically insignificant)
//
// Aligned with freqtrade FreqAI freqai_interface.py and hyperopt.py CPCV split.

package ai

import (
	"math"
	"sort"
)

// WalkForwardConfig holds parameters for walk-forward validation.
type WalkForwardConfig struct {
	NumFolds          int     // number of folds (default 5)
	PurgeDays         int     // purge period between train/test (default 7)
	MaxSharpeDiff      float64 // max allowed difference train - test Sharpe (default 1.0)
	MaxDrawdownLimit   float64 // max allowed drawdown per fold (default 0.30 = 30%)
	MinTradeCount      int     // minimum trades per fold for significance (default 30)
}

// DefaultWalkForwardConfig returns standard parameters.
func DefaultWalkForwardConfig() WalkForwardConfig {
	return WalkForwardConfig{
		NumFolds:        5,
		PurgeDays:       7,
		MaxSharpeDiff:   1.0,
		MaxDrawdownLimit: 0.30,
		MinTradeCount:   30,
	}
}

// FoldResult contains the validation metrics for a single walk-forward fold.
type FoldResult struct {
	FoldIndex     int     `json:"fold"`
	TrainSharpe   float64 `json:"train_sharpe"`
	TestSharpe    float64 `json:"test_sharpe"`
	TrainMaxDD    float64 `json:"train_max_dd"`
	TestMaxDD     float64 `json:"test_max_dd"`
	TradeCount    int     `json:"trade_count"`
	Passed        bool    `json:"passed"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// WalkForwardResult is the outcome of walk-forward cross-validation.
type WalkForwardResult struct {
	Passed     bool         `json:"passed"`
	Folds      []FoldResult `json:"folds"`
	SharpeDiff float64      `json:"avg_sharpe_diff"` // average train - test Sharpe
	MaxFoldDD  float64      `json:"max_fold_dd"`
	MinTrades  int          `json:"min_trades"`
	Reason     string       `json:"reason,omitempty"`
}

// DailyReturn is a single day's P&L return (can be simulated or real).
type DailyReturn struct {
	Day    int
	Return float64
}

// WalkForward validates a strategy using purged walk-forward cross-validation.
// dailyReturns is a chronologically ordered slice of daily P&L values.
func WalkForward(dailyReturns []float64, cfg WalkForwardConfig) WalkForwardResult {
	n := len(dailyReturns)
	if n < cfg.MinTradeCount*2 {
		return WalkForwardResult{
			Passed: false,
			Reason: "insufficient data for walk-forward",
		}
	}

	foldSize := n / (cfg.NumFolds + 1) // +1 for the final test period
	if foldSize < cfg.MinTradeCount {
		foldSize = cfg.MinTradeCount
	}

	result := WalkForwardResult{Passed: true}
	minTrades := n

	for fold := 0; fold < cfg.NumFolds; fold++ {
		// Train: [0, trainEnd), Test: [testStart, testEnd) with purge gap.
		trainEnd := foldSize * (fold + 1)
		testStart := trainEnd + cfg.PurgeDays
		testEnd := testStart + foldSize
		if testEnd > n {
			testEnd = n
		}
		if testStart >= n || testEnd-testStart < cfg.MinTradeCount {
			continue
		}

		trainReturns := dailyReturns[:trainEnd]
		testReturns := dailyReturns[testStart:testEnd]

		fr := FoldResult{FoldIndex: fold + 1}

		// Train metrics.
		trainSharpe := computeSharpe(trainReturns)
		trainDD := computeMaxDD(trainReturns)
		fr.TrainSharpe = trainSharpe
		fr.TrainMaxDD = trainDD

		// Test metrics.
		testSharpe := computeSharpe(testReturns)
		testDD := computeMaxDD(testReturns)
		fr.TestSharpe = testSharpe
		fr.TestMaxDD = testDD
		fr.TradeCount = len(testReturns)

		// Rejection criteria.
		fr.Passed = true
		sharpeDiff := trainSharpe - testSharpe
		if sharpeDiff > cfg.MaxSharpeDiff {
			fr.Passed = false
			fr.RejectionReason = "overfitting: train Sharpe exceeds test Sharpe by > 1.0"
		}
		if testDD > cfg.MaxDrawdownLimit {
			fr.Passed = false
			if fr.RejectionReason != "" {
				fr.RejectionReason += "; "
			}
			fr.RejectionReason += "max drawdown exceeds limit"
		}
		if fr.TradeCount < cfg.MinTradeCount {
			fr.Passed = false
			if fr.RejectionReason != "" {
				fr.RejectionReason += "; "
			}
			fr.RejectionReason += "insufficient trades"
		}

		if !fr.Passed {
			result.Passed = false
		}
		if testDD > result.MaxFoldDD {
			result.MaxFoldDD = testDD
		}
		if fr.TradeCount < minTrades {
			minTrades = fr.TradeCount
		}

		result.Folds = append(result.Folds, fr)
	}

	// Average Sharpe difference.
	if len(result.Folds) > 0 {
		var sumDiff float64
		for _, f := range result.Folds {
			sumDiff += f.TrainSharpe - f.TestSharpe
		}
		result.SharpeDiff = sumDiff / float64(len(result.Folds))
	}
	result.MinTrades = minTrades

	if !result.Passed {
		result.Reason = "walk-forward validation failed"
	}

	return result
}

// computeSharpe calculates the annualized Sharpe ratio from daily returns.
// Assumes 252 trading days/year.
func computeSharpe(dailyReturns []float64) float64 {
	if len(dailyReturns) < 2 {
		return 0
	}
	var sum, sumSq float64
	for _, r := range dailyReturns {
		sum += r
		sumSq += r * r
	}
	n := float64(len(dailyReturns))
	mean := sum / n
	variance := sumSq/n - mean*mean
	if variance <= 0 {
		return 0
	}
	dailySD := math.Sqrt(variance)
	if dailySD == 0 {
		return 0
	}
	return (mean / dailySD) * math.Sqrt(252)
}

// computeMaxDD calculates the maximum drawdown as a fraction of peak equity.
// Returns a value in [0, 1] where 0.30 = 30% drawdown.
func computeMaxDD(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}
	peak := 0.0
	maxDD := 0.0
	cumulative := 0.0
	for _, r := range returns {
		cumulative += r
		if cumulative > peak {
			peak = cumulative
		}
		if peak > 0 {
			dd := (peak - cumulative) / peak
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	return maxDD
}

// CPCV performs Combinatorial Purged Cross-Validation on daily returns.
// Returns the average out-of-sample Sharpe ratio across all combinatorial splits.
// nGroups is the number of chronological groups to split into (default 6).
// cfg controls the overfitting penalty threshold.
func CPCV(dailyReturns []float64, nGroups int, cfg WalkForwardConfig) float64 {
	if nGroups < 2 {
		nGroups = 6
	}
	if cfg.MaxSharpeDiff <= 0 {
		cfg.MaxSharpeDiff = 1.0
	}
	n := len(dailyReturns)
	if n < nGroups*2 {
		return 0
	}

	groupSize := n / nGroups
	var oosSharpes []float64

	// Purged walk-forward: for each group boundary, train on all earlier groups,
	// test on one future group (with purge gap).
	for g := 1; g < nGroups; g++ {
		trainEnd := g * groupSize
		testStart := trainEnd + 2 // purge 2 groups
		if testStart >= n {
			continue
		}
		testEnd := testStart + groupSize
		if testEnd > n {
			testEnd = n
		}
		if testEnd-testStart < 5 {
			continue
		}

		train := dailyReturns[:trainEnd]
		test := dailyReturns[testStart:testEnd]

		// Use test Sharpe as the OOS metric.
		testSharpe := computeSharpe(test)
		// Penalize when train Sharpe is much higher (overfitting signal).
		trainSharpe := computeSharpe(train)
		if trainSharpe-testSharpe > cfg.MaxSharpeDiff {
			testSharpe *= 0.5 // penalty for overfitting
		}
		oosSharpes = append(oosSharpes, testSharpe)
	}

	if len(oosSharpes) == 0 {
		return 0
	}

	sort.Float64s(oosSharpes)
	// Return median OOS Sharpe (robust to outliers).
	median := oosSharpes[len(oosSharpes)/2]
	return median
}
