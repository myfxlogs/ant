// Package oms provides P&L attribution (M11-15).
//
// Net P&L = Gross P&L(signal) - SpreadCost - SlippageCost(execution) - Commission - SwapCost - FundingCost(holding)
//
// Three dimensions:
//  1. Signal — gross P&L from price movement (alpha)
//  2. Execution — spread + slippage (fill quality)
//  3. Holding — commission + swap + funding (carry costs)

package oms

import (
	"fmt"

	"github.com/shopspring/decimal"

	"alphaforge/internal/costsvc"
)

// PnLAttribution decomposes Net P&L into three independently measurable dimensions.
type PnLAttribution struct {
	// Dimension 1: Signal — gross P&L before any costs
	GrossPnL decimal.Decimal `json:"gross_pnl"`

	// Dimension 2: Execution — costs incurred at order fill
	SlippageCost decimal.Decimal `json:"slippage_cost"`
	SpreadCost   decimal.Decimal `json:"spread_cost"`

	// Dimension 3: Holding — costs from carrying the position
	Commission  decimal.Decimal `json:"commission"`
	SwapCost    decimal.Decimal `json:"swap_cost"`
	FundingCost decimal.Decimal `json:"funding_cost"`

	// Context
	Notional decimal.Decimal `json:"notional"`
	Side     string          `json:"side"`
}

// SignalPnL returns the signal dimension (gross P&L).
func (a PnLAttribution) SignalPnL() decimal.Decimal { return a.GrossPnL }

// ExecutionCost returns the execution dimension total.
func (a PnLAttribution) ExecutionCost() decimal.Decimal { return a.SlippageCost.Add(a.SpreadCost) }

// HoldingCost returns the holding dimension total.
func (a PnLAttribution) HoldingCost() decimal.Decimal { return a.Commission.Add(a.SwapCost).Add(a.FundingCost) }

// NetPnL computes the bottom-line P&L after all costs.
func (a PnLAttribution) NetPnL() decimal.Decimal {
	return a.GrossPnL.Sub(a.ExecutionCost()).Sub(a.HoldingCost())
}

// SignalBps returns the signal alpha in basis points of notional.
func (a PnLAttribution) SignalBps() decimal.Decimal {
	if a.Notional.Equal(decimal.Zero) {
		return decimal.Zero
	}
	return a.GrossPnL.Div(a.Notional).Mul(decimal.NewFromInt(10000))
}

// ExecutionBps returns the execution cost in basis points of notional.
func (a PnLAttribution) ExecutionBps() decimal.Decimal {
	if a.Notional.Equal(decimal.Zero) {
		return decimal.Zero
	}
	return a.ExecutionCost().Div(a.Notional).Mul(decimal.NewFromInt(10000))
}

// HoldingBps returns the holding cost in basis points of notional.
func (a PnLAttribution) HoldingBps() decimal.Decimal {
	if a.Notional.Equal(decimal.Zero) {
		return decimal.Zero
	}
	return a.HoldingCost().Div(a.Notional).Mul(decimal.NewFromInt(10000))
}

// NetBps returns net P&L in basis points of notional.
func (a PnLAttribution) NetBps() decimal.Decimal {
	if a.Notional.Equal(decimal.Zero) {
		return decimal.Zero
	}
	return a.NetPnL().Div(a.Notional).Mul(decimal.NewFromInt(10000))
}

// Validate checks the P&L identity: Net = Gross - Spread - Slippage - Commission - Swap - Funding.
func (a PnLAttribution) Validate() error {
	expected := a.GrossPnL.Sub(a.SpreadCost).Sub(a.SlippageCost).Sub(a.Commission).Sub(a.SwapCost).Sub(a.FundingCost)
	actual := a.NetPnL()
	diff := expected.Sub(actual).Abs()
	if diff.GreaterThan(decimal.NewFromFloat(0.005)) {
		return fmt.Errorf("PnL identity violated: %s != %s (diff=%s)", expected.String(), actual.String(), diff.String())
	}
	return nil
}

// Add aggregates two attributions (e.g., entry + exit legs).
func (a PnLAttribution) Add(b PnLAttribution) PnLAttribution {
	return PnLAttribution{
		GrossPnL:     a.GrossPnL.Add(b.GrossPnL),
		SlippageCost: a.SlippageCost.Add(b.SlippageCost),
		SpreadCost:   a.SpreadCost.Add(b.SpreadCost),
		Commission:   a.Commission.Add(b.Commission),
		SwapCost:     a.SwapCost.Add(b.SwapCost),
		FundingCost:  a.FundingCost.Add(b.FundingCost),
		Notional:     a.Notional.Add(b.Notional),
		Side:         a.Side,
	}
}

// PnLAttributor computes the 3D P&L decomposition for closed trades.
type PnLAttributor struct {
	fillModel *FillModel
}

// NewPnLAttributor creates a P&L attributor backed by a FillModel.
func NewPnLAttributor(fm *FillModel) *PnLAttributor {
	return &PnLAttributor{fillModel: fm}
}

// Attribute computes the 3D P&L decomposition for a round-trip trade.
//
// side: "buy" or "sell"
// openPrice/closePrice: gross fill prices at entry and exit
// lots/contractSize: position size
// holdingDays: number of overnight rolls (for swap cost)
func (at *PnLAttributor) Attribute(side string, openPrice, closePrice, lots, contractSize, holdingDays decimal.Decimal) PnLAttribution {
	cm := at.fillModel.costModel
	notional := lots.Mul(contractSize).Mul(openPrice)

	// Gross P&L (signal dimension)
	grossPnL := decimal.Zero
	if side == "buy" {
		grossPnL = closePrice.Sub(openPrice).Mul(lots).Mul(contractSize)
	} else {
		grossPnL = openPrice.Sub(closePrice).Mul(lots).Mul(contractSize)
	}

	// Entry leg costs
	entryNotional := lots.Mul(contractSize).Mul(openPrice)
	entrySpread := cm.SpreadCost(lots)
	entryComm := cm.Commission(lots, entryNotional)
	entrySlip := cm.SlippageCost(lots, openPrice, contractSize)

	// Exit leg costs
	exitNotional := lots.Mul(contractSize).Mul(closePrice)
	exitSpread := cm.SpreadCost(lots)
	exitComm := cm.Commission(lots, exitNotional)
	exitSlip := cm.SlippageCost(lots, closePrice, contractSize)

	// Swap only applies to holding duration
	swap := cm.SwapCost(side, lots, closePrice, contractSize, holdingDays)
	funding := cm.FundingCost(lots, closePrice, contractSize, 0) // not perpetual by default

	return PnLAttribution{
		GrossPnL:     grossPnL,
		SlippageCost: entrySlip.Add(exitSlip),
		SpreadCost:   entrySpread.Add(exitSpread),
		Commission:   entryComm.Add(exitComm),
		SwapCost:     swap,
		FundingCost:  funding,
		Notional:     notional,
		Side:         side,
	}
}

// CostModel returns the underlying cost model (for inspection / reporting).
func (at *PnLAttributor) CostModel() *costsvc.CostModel {
	return at.fillModel.costModel
}
