package costsvc

import "time"

// CostBreakdown is the full pre-trade cost estimate for an order.
type CostBreakdown struct {
	SpreadCost    float64 `json:"spread_cost"`
	Commission    float64 `json:"commission"`
	SlippageCost  float64 `json:"slippage_cost"`
	SwapCost      float64 `json:"swap_cost"`
	FundingCost   float64 `json:"funding_cost"`
	TotalCost     float64 `json:"total_cost"`
	CostBps        float64 `json:"cost_bps"` // total cost / notional in bps
}

// EstimateParams are the inputs for pre-trade cost estimation.
type EstimateParams struct {
	Symbol         string  // for multi-model lookup
	Side           string  // "buy" / "sell"
	Lots           float64
	Price          float64
	ContractSize   float64
	HoldingDays    float64       // expected holding period for swap
	HoldingDuration time.Duration // alternative holding period for funding
}

// Estimate computes the total cost breakdown for a planned trade.
func (m *CostModel) Estimate(p EstimateParams) CostBreakdown {
	notional := p.Lots * p.ContractSize * p.Price
	spread := m.SpreadCost(p.Lots)
	comm := m.Commission(p.Lots, notional)
	slip := m.SlippageCost(p.Lots, p.Price, p.ContractSize)
	swap := m.SwapCost(p.Side, p.Lots, p.Price, p.ContractSize, p.HoldingDays)
	funding := m.FundingCost(p.Lots, p.Price, p.ContractSize, p.HoldingDuration)

	total := spread + comm + slip + swap + funding
	costBps := 0.0
	if notional > 0 {
		costBps = total / notional * 10000.0
	}

	return CostBreakdown{
		SpreadCost:   spread,
		Commission:   comm,
		SlippageCost: slip,
		SwapCost:     swap,
		FundingCost:  funding,
		TotalCost:    total,
		CostBps:       costBps,
	}
}

// GrossToNetFillPrice converts a gross fill price to net after all costs.
// This is used by the FillModel to compute the realized fill price.
func (m *CostModel) GrossToNetFillPrice(grossPrice float64, p EstimateParams) float64 {
	breakdown := m.Estimate(p)
	notional := p.Lots * p.ContractSize * grossPrice
	if notional <= 0 {
		return grossPrice
	}
	costPerUnit := breakdown.TotalCost / (p.Lots * p.ContractSize)
	if p.Side == "buy" {
		return grossPrice + costPerUnit
	}
	return grossPrice - costPerUnit
}
