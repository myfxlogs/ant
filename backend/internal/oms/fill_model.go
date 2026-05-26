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
}

// Compute calculates the net fill price from the gross price.
// For backtest mode (isBacktest=true), commission/slippage/spread are forced to non-zero defaults.
func (f *FillModel) Compute(grossPrice float64, p costsvc.EstimateParams, isBacktest bool) FillResult {
	if isBacktest && f.costModel.CommissionPerLot == 0 && f.costModel.CommissionBps == 0 {
		f.costModel.CommissionBps = 1.0 // force 1 bps commission in backtest
	}
	if isBacktest && f.costModel.SlippageBps == 0 {
		f.costModel.SlippageBps = 1.0 // force 1 bps slippage in backtest
	}
	if isBacktest && f.costModel.SpreadPips == 0 {
		f.costModel.SpreadPips = 1.0 // force 1 pip spread in backtest
	}

	breakdown := f.costModel.Estimate(p)
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
		Commission:   breakdown.Commission,
		SlippageCost: breakdown.SlippageCost,
		SwapCost:     breakdown.SwapCost,
		FundingCost:  breakdown.FundingCost,
		NetFillPrice: netPrice,
	}
}

// ComputeNet is a convenience method that returns only the net fill price.
func (f *FillModel) ComputeNet(grossPrice float64, p costsvc.EstimateParams, isBacktest bool) float64 {
	return f.Compute(grossPrice, p, isBacktest).NetFillPrice
}
