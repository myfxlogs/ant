package costsvc

import (
	"time"

	"github.com/shopspring/decimal"
)

// CostBreakdown is the full pre-trade cost estimate for an order.
type CostBreakdown struct {
	SpreadCost   decimal.Decimal `json:"spread_cost"`
	Commission   decimal.Decimal `json:"commission"`
	SlippageCost decimal.Decimal `json:"slippage_cost"`
	SwapCost     decimal.Decimal `json:"swap_cost"`
	FundingCost  decimal.Decimal `json:"funding_cost"`
	TotalCost    decimal.Decimal `json:"total_cost"`
	CostBps      decimal.Decimal `json:"cost_bps"` // total cost / notional in bps
}

// EstimateParams are the inputs for pre-trade cost estimation.
type EstimateParams struct {
	Symbol          string          // for multi-model lookup
	Side            string          // "buy" / "sell"
	Lots            decimal.Decimal
	Price           decimal.Decimal
	ContractSize    decimal.Decimal
	HoldingDays     decimal.Decimal  // expected holding period for swap
	HoldingDuration time.Duration    // alternative holding period for funding
}

// Estimate computes the total cost breakdown for a planned trade.
func (m *CostModel) Estimate(p EstimateParams) CostBreakdown {
	notional := p.Lots.Mul(p.ContractSize).Mul(p.Price)
	spread := m.SpreadCost(p.Lots)
	comm := m.Commission(p.Lots, notional)
	slip := m.SlippageCost(p.Lots, p.Price, p.ContractSize)
	swap := m.SwapCost(p.Side, p.Lots, p.Price, p.ContractSize, p.HoldingDays)
	funding := m.FundingCost(p.Lots, p.Price, p.ContractSize, p.HoldingDuration)

	total := spread.Add(comm).Add(slip).Add(swap).Add(funding)
	costBps := decimal.Zero
	if notional.GreaterThan(decimal.Zero) {
		costBps = total.Div(notional).Mul(decimal.NewFromInt(10000))
	}

	return CostBreakdown{
		SpreadCost:   spread,
		Commission:   comm,
		SlippageCost: slip,
		SwapCost:     swap,
		FundingCost:  funding,
		TotalCost:    total,
		CostBps:      costBps,
	}
}

// GrossToNetFillPrice converts a gross fill price to net after all costs.
// This is used by the FillModel to compute the realized fill price.
func (m *CostModel) GrossToNetFillPrice(grossPrice decimal.Decimal, p EstimateParams) decimal.Decimal {
	breakdown := m.Estimate(p)
	notional := p.Lots.Mul(p.ContractSize).Mul(grossPrice)
	if notional.LessThanOrEqual(decimal.Zero) {
		return grossPrice
	}
	costPerUnit := breakdown.TotalCost.Div(p.Lots.Mul(p.ContractSize))
	if p.Side == "buy" {
		return grossPrice.Add(costPerUnit)
	}
	return grossPrice.Sub(costPerUnit)
}
