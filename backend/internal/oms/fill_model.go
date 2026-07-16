// Package oms provides the FillModel (M10-BASE-D4).
//
// FillModel decomposes a gross fill price into its cost components and produces
// a net fill price after all trading costs. Backtest paths require non-zero defaults
// for commission/slippage/spread to prevent unrealistically optimistic results.
//
// Aligned with NautilusTrader backtest/models.py (FillModel + LatencyModel).

package oms

import (
	"github.com/shopspring/decimal"

	"alphaforge/internal/costsvc"
)

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
	GrossPrice   decimal.Decimal `json:"gross_price"`
	SpreadCost   decimal.Decimal `json:"spread_cost"`
	Commission   decimal.Decimal `json:"commission"`
	SlippageCost decimal.Decimal `json:"slippage_cost"`
	SwapCost     decimal.Decimal `json:"swap_cost"`
	FundingCost  decimal.Decimal `json:"funding_cost"`
	NetFillPrice decimal.Decimal `json:"net_fill_price"`
	TotalCost    decimal.Decimal `json:"total_cost"`
	FilledVolume decimal.Decimal `json:"filled_volume"`
}

// Compute calculates the net fill price from the gross price.
// For backtest mode (isBacktest=true), commission/slippage/spread are forced to non-zero defaults.
// The receiver's costModel is never mutated — backtest defaults are applied to a local copy.
func (f *FillModel) Compute(grossPrice decimal.Decimal, p costsvc.EstimateParams, isBacktest bool) FillResult {
	if f == nil || f.costModel == nil {
		return FillResult{GrossPrice: grossPrice, NetFillPrice: grossPrice, FilledVolume: p.Lots}
	}
	cm := f.costModel // use directly for live; clone for backtest to avoid mutating shared state
	if isBacktest {
		cloned := *f.costModel
		if cloned.CommissionPerLot.Equal(decimal.Zero) && cloned.CommissionBps.Equal(decimal.Zero) {
			cloned.CommissionBps = decimal.NewFromInt(1)
		}
		if cloned.SlippageBps.Equal(decimal.Zero) {
			cloned.SlippageBps = decimal.NewFromInt(1)
		}
		if cloned.SpreadPips.Equal(decimal.Zero) {
			cloned.SpreadPips = decimal.NewFromInt(1)
		}
		cm = &cloned
	}

	breakdown := cm.Estimate(p)
	lots := p.Lots
	contractSize := p.ContractSize

	costPerUnit := decimal.Zero
	if lots.GreaterThan(decimal.Zero) && contractSize.GreaterThan(decimal.Zero) {
		costPerUnit = breakdown.TotalCost.Div(lots.Mul(contractSize))
	}

	var netPrice decimal.Decimal
	if p.Side == "buy" {
		netPrice = grossPrice.Add(costPerUnit)
	} else {
		netPrice = grossPrice.Sub(costPerUnit)
	}

	return FillResult{
		GrossPrice:   grossPrice,
		SpreadCost:   breakdown.SpreadCost,
		Commission:   breakdown.Commission,
		SlippageCost: breakdown.SlippageCost,
		SwapCost:     breakdown.SwapCost,
		FundingCost:  breakdown.FundingCost,
		NetFillPrice: netPrice,
		TotalCost:    breakdown.TotalCost,
		FilledVolume: p.Lots,
	}
}

// ComputeNet is a convenience method that returns only the net fill price.
func (f *FillModel) ComputeNet(grossPrice decimal.Decimal, p costsvc.EstimateParams, isBacktest bool) decimal.Decimal {
	return f.Compute(grossPrice, p, isBacktest).NetFillPrice
}
