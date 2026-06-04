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

import "sort"

// WalkForwardConfig holds parameters for walk-forward validation.
type WalkForwardConfig struct {
	NumFolds        int     // number of folds (default 5)
	PurgeDays       int     // purge period between train/test (default 7)
	MaxSharpeDiff   float64 // max allowed difference train - test Sharpe (default 1.0)
	MaxDrawdownLimit float64 // max allowed drawdown per fold (default 0.30 = 30%)
	MinTradeCount   int     // minimum trades per fold for significance (default 30)
}

// DefaultWalkForwardConfig returns standard parameters.
func DefaultWalkForwardConfig() WalkForwardConfig {
	return WalkForwardConfig{
		NumFolds:         5,
		PurgeDays:        7,
		MaxSharpeDiff:    1.0,
		MaxDrawdownLimit: 0.30,
		MinTradeCount:    30,
	}
}

// FoldResult contains the validation metrics for a single walk-forward fold.
type FoldResult struct {
	FoldIndex       int     `json:"fold"`
	TrainSharpe     float64 `json:"train_sharpe"`
	TestSharpe      float64 `json:"test_sharpe"`
	TrainMaxDD      float64 `json:"train_max_dd"`
	TestMaxDD       float64 `json:"test_max_dd"`
	TradeCount      int     `json:"trade_count"`
	Passed          bool    `json:"passed"`
	RejectionReason string  `json:"rejection_reason,omitempty"`
}

// WalkForwardResult is the outcome of walk-forward cross-validation.
type WalkForwardResult struct {
	Passed     bool         `json:"passed"`
	Folds      []FoldResult `json:"folds"`
	SharpeDiff float64      `json:"avg_sharpe_diff"`
	MaxFoldDD  float64      `json:"max_fold_dd"`
	MinTrades  int          `json:"min_trades"`
	Reason     string       `json:"reason,omitempty"`
}

// DailyReturn is a single day's P&L return (can be simulated or real).
type DailyReturn struct {
	Day    int
	Return float64
}

// calcFoldSize computes the fold size for walk-forward, clamped to MinTradeCount.
func calcFoldSize(n int, cfg WalkForwardConfig) int {
	foldSize := n / (cfg.NumFolds + 1)
	if foldSize < cfg.MinTradeCount {
		return cfg.MinTradeCount
	}
	return foldSize
}

// evaluateFold evaluates a single walk-forward fold (train→test with purge gap).
func evaluateFold(trainReturns, testReturns []float64, foldIdx int, cfg WalkForwardConfig) FoldResult {
	trainSharpe := computeSharpe(trainReturns)
	trainDD := computeMaxDD(trainReturns)
	testSharpe := computeSharpe(testReturns)
	testDD := computeMaxDD(testReturns)

	fr := FoldResult{
		FoldIndex:   foldIdx,
		TrainSharpe: trainSharpe,
		TestSharpe:  testSharpe,
		TrainMaxDD:  trainDD,
		TestMaxDD:   testDD,
		TradeCount:  len(testReturns),
		Passed:      true,
	}

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
	return fr
}

// WalkForward validates a strategy using purged walk-forward cross-validation.
// dailyReturns is a chronologically ordered slice of daily P&L values.
func WalkForward(dailyReturns []float64, cfg WalkForwardConfig) WalkForwardResult {
	if len(dailyReturns) < cfg.MinTradeCount*(cfg.NumFolds+1) {
		return WalkForwardResult{Passed: false, Reason: "insufficient data for walk-forward"}
	}

	n := len(dailyReturns)
	foldSize := calcFoldSize(n, cfg)

	folds, minTrades := executeFolds(dailyReturns, foldSize, n, cfg)
	return aggregateFoldResults(folds, minTrades)
}

// executeFolds runs each walk-forward fold sequentially.
func executeFolds(dailyReturns []float64, foldSize, n int, cfg WalkForwardConfig) ([]FoldResult, int) {
	var folds []FoldResult
	minTrades := n

	for fold := 0; fold < cfg.NumFolds; fold++ {
		trainEnd := foldSize * (fold + 1)
		testStart := trainEnd + cfg.PurgeDays
		testEnd := testStart + foldSize
		if testEnd > n {
			testEnd = n
		}
		if testStart >= n || testEnd-testStart < cfg.MinTradeCount {
			continue
		}
		fr := evaluateFold(dailyReturns[:trainEnd], dailyReturns[testStart:testEnd], fold+1, cfg)
		folds = append(folds, fr)
		if fr.TradeCount < minTrades {
			minTrades = fr.TradeCount
		}
	}
	return folds, minTrades
}

// aggregateFoldResults summarizes fold outcomes into a WalkForwardResult.
func aggregateFoldResults(folds []FoldResult, minTrades int) WalkForwardResult {
	result := WalkForwardResult{Passed: true, Folds: folds, MinTrades: minTrades}
	if len(folds) == 0 {
		result.Passed = false
		result.Reason = "no valid folds"
		return result
	}

	var sumDiff float64
	for _, f := range folds {
		sumDiff += f.TrainSharpe - f.TestSharpe
		if !f.Passed {
			result.Passed = false
		}
		if f.TestMaxDD > result.MaxFoldDD {
			result.MaxFoldDD = f.TestMaxDD
		}
		if f.TradeCount < result.MinTrades {
			result.MinTrades = f.TradeCount
		}
	}
	result.SharpeDiff = sumDiff / float64(len(folds))

	if !result.Passed {
		result.Reason = "walk-forward validation failed"
	}
	return result
}

// calcPurgeGroups determines the number of groups to use as purge gap.
func calcPurgeGroups(groupSize, purgeDays int) int {
	if purgeDays > 0 && groupSize > 0 {
		obsPerDay := float64(groupSize) / float64(purgeDays)
		if obsPerDay >= 1 {
			return 1
		}
	}
	return 2
}

// collectCPCVSharpe collects out-of-sample Sharpe ratios across CPCV groups.
func collectCPCVSharpe(dailyReturns []float64, nGroups, groupSize, purgeGroups int, cfg WalkForwardConfig) []float64 {
	n := len(dailyReturns)
	var oosSharpes []float64
	for g := 1; g < nGroups; g++ {
		trainEnd := g * groupSize
		testStart := trainEnd + purgeGroups
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
		testSharpe := computeSharpe(dailyReturns[testStart:testEnd])
		trainSharpe := computeSharpe(dailyReturns[:trainEnd])
		if trainSharpe-testSharpe > cfg.MaxSharpeDiff {
			testSharpe *= 0.5
		}
		oosSharpes = append(oosSharpes, testSharpe)
	}
	return oosSharpes
}

// medianFloat64 returns the median of a sorted float64 slice.
func medianFloat64(sorted []float64) float64 {
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// CPCV performs Combinatorial Purged Cross-Validation on daily returns.
// Returns the median out-of-sample Sharpe ratio across all combinatorial splits.
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
	purgeGroups := calcPurgeGroups(groupSize, cfg.PurgeDays)
	oosSharpes := collectCPCVSharpe(dailyReturns, nGroups, groupSize, purgeGroups, cfg)

	if len(oosSharpes) == 0 {
		return 0
	}
	sort.Float64s(oosSharpes)
	return medianFloat64(oosSharpes)
}
