// Package oms provides the dual-track P&L calculator (M10-BASE-D5).
//
// Net P&L = Gross P&L - SpreadCost - Commission - Swap - Slippage
// Backtest results must report both Gross and Net P&L.

package oms

import "anttrader/internal/costsvc"

// PnLCalculator computes both gross and net P&L for closed trades.
type PnLCalculator struct {
	fillModel *FillModel
}

// NewPnLCalculator creates a dual-track P&L calculator.
func NewPnLCalculator(fm *FillModel) *PnLCalculator {
	return &PnLCalculator{fillModel: fm}
}

// PnLResult contains both gross and net P&L for a closed position.
type PnLResult struct {
	GrossPnL    float64 `json:"gross_pnl"`
	SpreadCost  float64 `json:"spread_cost"`
	Commission  float64 `json:"commission"`
	SwapCost    float64 `json:"swap_cost"`
	SlippageCost float64 `json:"slippage_cost"`
	NetPnL      float64 `json:"net_pnl"`
}

// Calculate computes P&L for a round-trip trade (entry + exit).
// side: "buy" or "sell"
// openPrice/closePrice: gross fill prices
// lots/contractSize: position size
// holdingDays: how long the position was held (for swap)
func (c *PnLCalculator) Calculate(side string, openPrice, closePrice, lots, contractSize float64, holdingDays float64) PnLResult {
	notional := lots * contractSize
	grossPnL := 0.0
	if side == "buy" {
		grossPnL = (closePrice - openPrice) * notional / openPrice
	} else {
		grossPnL = (openPrice - closePrice) * notional / openPrice
	}

	// Entry costs
	entryBreakdown := c.fillModel.costModel.Estimate(costsvc.EstimateParams{
		Side: side, Lots: lots, Price: openPrice, ContractSize: contractSize, HoldingDays: 0,
	})
	// Exit costs
	exitSide := "sell"
	if side == "sell" {
		exitSide = "buy"
	}
	exitBreakdown := c.fillModel.costModel.Estimate(costsvc.EstimateParams{
		Side: exitSide, Lots: lots, Price: closePrice, ContractSize: contractSize, HoldingDays: 0,
	})

	swapCost := c.fillModel.costModel.SwapCost(side, lots, closePrice, contractSize, holdingDays)
	spreadCost := entryBreakdown.SpreadCost + exitBreakdown.SpreadCost
	commission := entryBreakdown.Commission + exitBreakdown.Commission
	slippage := entryBreakdown.SlippageCost + exitBreakdown.SlippageCost

	netPnL := grossPnL - spreadCost - commission - swapCost - slippage

	return PnLResult{
		GrossPnL:     grossPnL,
		SpreadCost:   spreadCost,
		Commission:   commission,
		SwapCost:     swapCost,
		SlippageCost: slippage,
		NetPnL:       netPnL,
	}
}
