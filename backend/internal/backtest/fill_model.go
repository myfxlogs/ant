package backtest

import (
	"math/rand"
	"time"

	"anttrader/internal/costsvc"
)

// FillModel simulates order execution for backtesting.
type FillModel struct {
	SlippagePips    float64
	Latency         time.Duration
	PartialFillProb float64
	PartialFillRatio float64
	CostModel       *costsvc.CostModel
	// ContractSize is the instrument contract multiplier (e.g., 100000 for forex, 1 for crypto spot).
	ContractSize    float64
}

// FillResult is the outcome of a simulated fill.
type FillResult struct {
	GrossPrice   float64
	NetPrice     float64
	FilledVolume float64
	SpreadCost   float64
	Commission   float64
	SlippageCost float64
	SwapCost     float64
	TotalCost    float64
	IsPartial    bool
	Latency      time.Duration
}

// SimulateFill simulates order execution against the current bar.
// direction: 1 for buy, -1 for sell. volume is in lots.
func (m *FillModel) SimulateFill(direction int, volume, barClose float64, holdingDays float64, rng *rand.Rand) *FillResult {
	if m.SlippagePips < 0 {
		m.SlippagePips = 0.5
	}
	if m.Latency <= 0 {
		m.Latency = 50 * time.Millisecond
	}
	cs := m.ContractSize
	if cs <= 0 {
		cs = 100000 // default forex
	}

	result := &FillResult{
		GrossPrice: barClose,
		Latency:    m.Latency,
	}

	// Slippage: random within ±SlippagePips, biased against the trader
	slippagePips := m.SlippagePips
	if rng != nil {
		slippagePips = m.SlippagePips * (0.5 + rng.Float64())
	}
	slippagePrice := slippagePips * pipToPrice(m.CostModel)
	if direction > 0 {
		result.GrossPrice += slippagePrice
	} else {
		result.GrossPrice -= slippagePrice
	}

	// Partial fill
	if m.PartialFillProb > 0 && rng != nil && rng.Float64() < m.PartialFillProb {
		result.IsPartial = true
		result.FilledVolume = volume * m.PartialFillRatio
	} else {
		result.FilledVolume = volume
	}

	if m.CostModel != nil {
		cm := m.CostModel
		side := "buy"
		if direction < 0 {
			side = "sell"
		}
		notional := result.FilledVolume * cs * result.GrossPrice
		result.SpreadCost = cm.SpreadCost(result.FilledVolume)
		result.Commission = cm.Commission(result.FilledVolume, notional)
		result.SlippageCost = cm.SlippageCost(result.FilledVolume, result.GrossPrice, cs)
		result.SwapCost = cm.SwapCost(side, result.FilledVolume, result.GrossPrice, cs, holdingDays)
		result.TotalCost = result.SpreadCost + result.Commission + result.SlippageCost + result.SwapCost

		// Net fill price = gross ± cost per contract unit
		units := result.FilledVolume * cs
		if units > 0 {
			costPerUnit := result.TotalCost / units
			if direction > 0 {
				result.NetPrice = result.GrossPrice + costPerUnit
			} else {
				result.NetPrice = result.GrossPrice - costPerUnit
			}
		} else {
			result.NetPrice = result.GrossPrice
		}
	} else {
		result.NetPrice = result.GrossPrice
	}

	return result
}

func pipToPrice(cm *costsvc.CostModel) float64 {
	if cm != nil && cm.PipSize > 0 {
		return cm.PipSize
	}
	return 0.0001
}
