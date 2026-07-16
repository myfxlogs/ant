// Package oms provides the dual-track P&L calculator (M10-BASE-D5).
//
// Net P&L = Gross P&L - SpreadCost - Commission - Swap - Slippage
// Backtest results must report both Gross and Net P&L.

package oms

import (
	"github.com/shopspring/decimal"

	"alphaforge/internal/costsvc"
)

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
	GrossPnL     decimal.Decimal `json:"gross_pnl"`
	SpreadCost   decimal.Decimal `json:"spread_cost"`
	Commission   decimal.Decimal `json:"commission"`
	SwapCost     decimal.Decimal `json:"swap_cost"`
	SlippageCost decimal.Decimal `json:"slippage_cost"`
	NetPnL       decimal.Decimal `json:"net_pnl"`
}

// Calculate computes P&L for a round-trip trade (entry + exit).
// side: "buy" or "sell"
// openPrice/closePrice: gross fill prices
// lots/contractSize: position size
// holdingDays: how long the position was held (for swap)
func (c *PnLCalculator) Calculate(side string, openPrice, closePrice, lots, contractSize, holdingDays decimal.Decimal) PnLResult {
	notional := lots.Mul(contractSize)
	grossPnL := decimal.Zero
	if side == "buy" {
		grossPnL = closePrice.Sub(openPrice).Mul(notional).Div(openPrice)
	} else {
		grossPnL = openPrice.Sub(closePrice).Mul(notional).Div(openPrice)
	}

	// Entry costs
	entryBreakdown := c.fillModel.costModel.Estimate(costsvc.EstimateParams{
		Side: side, Lots: lots, Price: openPrice, ContractSize: contractSize, HoldingDays: decimal.Zero,
	})
	// Exit costs
	exitSide := "sell"
	if side == "sell" {
		exitSide = "buy"
	}
	exitBreakdown := c.fillModel.costModel.Estimate(costsvc.EstimateParams{
		Side: exitSide, Lots: lots, Price: closePrice, ContractSize: contractSize, HoldingDays: decimal.Zero,
	})

	swapCost := c.fillModel.costModel.SwapCost(side, lots, closePrice, contractSize, holdingDays)
	spreadCost := entryBreakdown.SpreadCost.Add(exitBreakdown.SpreadCost)
	commission := entryBreakdown.Commission.Add(exitBreakdown.Commission)
	slippage := entryBreakdown.SlippageCost.Add(exitBreakdown.SlippageCost)

	netPnL := grossPnL.Sub(spreadCost).Sub(commission).Sub(swapCost).Sub(slippage)

	return PnLResult{
		GrossPnL:     grossPnL,
		SpreadCost:   spreadCost,
		Commission:   commission,
		SwapCost:     swapCost,
		SlippageCost: slippage,
		NetPnL:       netPnL,
	}
}
