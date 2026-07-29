package analysis

import (
	"math"
	"sort"

	"alphaforge/internal/repository"
)

// detectSRLevels finds support and resistance levels using swing-point clustering.
// Uses a window-based peak/trough detection followed by proximity clustering.
type swingPoint struct {
	price  float64
	isHigh bool
}

type swingCluster struct {
	levels []float64
	isHigh bool
}

func detectSRLevels(bars []repository.KlineBar) []SRLevel {
	if len(bars) < 20 {
		return nil
	}
	n := len(bars)
	window := 5
	if n < window*2+1 {
		window = 2
	}

	swings := detectSwingPoints(bars, window)
	if len(swings) < 2 {
		return nil
	}

	closes, highs, lows := extractOHLC(bars, n)
	atrVal := atrPct(highs, lows, closes, 14)
	avgPrice := closes[n-1]
	clusterThreshold := avgPrice * atrVal / 100 * 0.5
	if clusterThreshold < avgPrice*0.001 {
		clusterThreshold = avgPrice * 0.001
	}

	clusters := clusterSwings(swings, clusterThreshold)
	levels := clustersToLevels(clusters)

	sort.Slice(levels, func(i, j int) bool {
		return levels[i].Price > levels[j].Price
	})
	if len(levels) > 8 {
		sort.Slice(levels, func(i, j int) bool {
			return levels[i].Touches > levels[j].Touches
		})
		levels = levels[:8]
		sort.Slice(levels, func(i, j int) bool {
			return levels[i].Price > levels[j].Price
		})
	}
	return levels
}

func detectSwingPoints(bars []repository.KlineBar, window int) []swingPoint {
	n := len(bars)
	var swings []swingPoint
	for i := window; i < n-window; i++ {
		isHigh := true
		isLow := true
		for j := i - window; j <= i+window; j++ {
			if j == i {
				continue
			}
			if bars[j].High.GreaterThanOrEqual(bars[i].High) {
				isHigh = false
			}
			if bars[j].Low.LessThanOrEqual(bars[i].Low) {
				isLow = false
			}
		}
		if isHigh {
			swings = append(swings, swingPoint{price: bars[i].High.InexactFloat64(), isHigh: true})
		}
		if isLow {
			swings = append(swings, swingPoint{price: bars[i].Low.InexactFloat64(), isHigh: false})
		}
	}
	return swings
}

func extractOHLC(bars []repository.KlineBar, n int) (closes, highs, lows []float64) {
	closes = make([]float64, n)
	highs = make([]float64, n)
	lows = make([]float64, n)
	for i, b := range bars {
		closes[i] = b.Close.InexactFloat64()
		highs[i] = b.High.InexactFloat64()
		lows[i] = b.Low.InexactFloat64()
	}
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		closes[i], closes[j] = closes[j], closes[i]
		highs[i], highs[j] = highs[j], highs[i]
		lows[i], lows[j] = lows[j], lows[i]
	}
	return
}

func clusterSwings(swings []swingPoint, threshold float64) []swingCluster {
	var clusters []swingCluster
	for _, sp := range swings {
		merged := false
		for ci := range clusters {
			if clusters[ci].isHigh != sp.isHigh {
				continue
			}
			for _, lvl := range clusters[ci].levels {
				if math.Abs(lvl-sp.price) <= threshold {
					clusters[ci].levels = append(clusters[ci].levels, sp.price)
					merged = true
					break
				}
			}
			if merged {
				break
			}
		}
		if !merged {
			clusters = append(clusters, swingCluster{
				levels: []float64{sp.price},
				isHigh: sp.isHigh,
			})
		}
	}
	return clusters
}

func clustersToLevels(clusters []swingCluster) []SRLevel {
	var levels []SRLevel
	for _, c := range clusters {
		if len(c.levels) < 2 {
			continue
		}
		sum := 0.0
		for _, p := range c.levels {
			sum += p
		}
		avg := sum / float64(len(c.levels))
		lvl := SRLevel{
			Price:   math.Round(avg*1e5) / 1e5,
			Touches: int32(len(c.levels)),
		}
		if c.isHigh {
			lvl.Type = "RESISTANCE"
		} else {
			lvl.Type = "SUPPORT"
		}
		if len(c.levels) >= 3 {
			lvl.Strength = "MAJOR"
		} else {
			lvl.Strength = "MINOR"
		}
		levels = append(levels, lvl)
	}
	return levels
}

// classifyVolatility returns ATR % of price and a state label.
func classifyVolatility(bars []repository.KlineBar) (float64, string) {
	n := len(bars)
	highs := make([]float64, n)
	lows := make([]float64, n)
	closes := make([]float64, n)
	for i, b := range bars {
		highs[i] = b.High.InexactFloat64()
		lows[i] = b.Low.InexactFloat64()
		closes[i] = b.Close.InexactFloat64()
	}
	// Reverse to chronological.
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		highs[i], highs[j] = highs[j], highs[i]
		lows[i], lows[j] = lows[j], lows[i]
		closes[i], closes[j] = closes[j], closes[i]
	}
	atrPctVal := atrPct(highs, lows, closes, 14)
	atrPctVal = math.Round(atrPctVal*100) / 100

	state := "NORMAL"
	switch {
	case atrPctVal < 1.0:
		state = "LOW"
	case atrPctVal >= 5.0:
		state = "EXTREME"
	case atrPctVal >= 3.0:
		state = "HIGH"
	}

	return atrPctVal, state
}
