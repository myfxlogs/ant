// Package oms provides the FillModel (M10-BASE-D4).
//
// FillModel decomposes a gross fill price into its cost components and produces
// a net fill price after all trading costs. Backtest paths require non-zero defaults
// for commission/slippage/spread to prevent unrealistically optimistic results.
//
// Aligned with NautilusTrader backtest/models.py (FillModel + LatencyModel).

package oms

import "anttrader/internal/costsvc"

// FillModel computes the net fill price from a gross price after all costs.
// Commission/spread/slippage are always applied; backtest paths enforce non-zero defaults.
type FillModel struct {
	costModel *costsvc.CostModel
}

// NewFillModel creates a FillModel from a cost model.
func NewFillModel(cm *costsvc.CostModel) *FillModel {
	return &FillModel{costModel: cm}
}

// FillResult contains the decomposed fill price components.
type FillResult struct {
	GrossPrice     float64 `json:"gross_price"`
	SpreadCost     float64 `json:"spread_cost"`
	Commission     float64 `json:"commission"`
	SlippageCost   float64 `json:"slippage_cost"`
	SwapCost       float64 `json:"swap_cost"`
	FundingCost    float64 `json:"funding_cost"`
	NetFillPrice   float64 `json:"net_fill_price"`
	TotalCost      float64 `json:"total_cost"`
	FilledVolume   float64 `json:"filled_volume"`
}

// Compute calculates the net fill price from the gross price.
// For backtest mode (isBacktest=true), commission/slippage/spread are forced to non-zero defaults.
// The receiver's costModel is never mutated — backtest defaults are applied to a local copy.
func (f *FillModel) Compute(grossPrice float64, p costsvc.EstimateParams, isBacktest bool) FillResult {
	if f == nil || f.costModel == nil {
		return FillResult{GrossPrice: grossPrice, NetFillPrice: grossPrice, FilledVolume: p.Lots}
	}
	cm := f.costModel // use directly for live; clone for backtest to avoid mutating shared state
	if isBacktest {
		cloned := *f.costModel
		if cloned.CommissionPerLot == 0 && cloned.CommissionBps == 0 {
			cloned.CommissionBps = 1.0
		}
		if cloned.SlippageBps == 0 {
			cloned.SlippageBps = 1.0
		}
		if cloned.SpreadPips == 0 {
			cloned.SpreadPips = 1.0
		}
		cm = &cloned
	}

	breakdown := cm.Estimate(p)
	lots := p.Lots
	contractSize := p.ContractSize

	costPerUnit := 0.0
	if lots > 0 && contractSize > 0 {
		costPerUnit = breakdown.TotalCost / (lots * contractSize)
	}

	var netPrice float64
	if p.Side == "buy" {
		netPrice = grossPrice + costPerUnit
	} else {
		netPrice = grossPrice - costPerUnit
	}

	return FillResult{
		GrossPrice:   grossPrice,
		SpreadCost:   breakdown.SpreadCost,
		Commission:   breakdown.Commission.InexactFloat64(),
		SlippageCost: breakdown.SlippageCost,
		SwapCost:     breakdown.SwapCost,
		FundingCost:  breakdown.FundingCost,
		NetFillPrice: netPrice,
		TotalCost:    breakdown.TotalCost,
		FilledVolume: p.Lots,
	}
}

// ComputeNet is a convenience method that returns only the net fill price.
func (f *FillModel) ComputeNet(grossPrice float64, p costsvc.EstimateParams, isBacktest bool) float64 {
	return f.Compute(grossPrice, p, isBacktest).NetFillPrice
}
