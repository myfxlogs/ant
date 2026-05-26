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
	"math"

	"anttrader/internal/costsvc"
)

// PnLAttribution decomposes Net P&L into three independently measurable dimensions.
type PnLAttribution struct {
	// Dimension 1: Signal — gross P&L before any costs
	GrossPnL float64 `json:"gross_pnl"`

	// Dimension 2: Execution — costs incurred at order fill
	SlippageCost float64 `json:"slippage_cost"`
	SpreadCost   float64 `json:"spread_cost"`

	// Dimension 3: Holding — costs from carrying the position
	Commission  float64 `json:"commission"`
	SwapCost    float64 `json:"swap_cost"`
	FundingCost float64 `json:"funding_cost"`

	// Context
	Notional float64 `json:"notional"`
	Side     string  `json:"side"`
}

// SignalPnL returns the signal dimension (gross P&L).
func (a PnLAttribution) SignalPnL() float64 { return a.GrossPnL }

// ExecutionCost returns the execution dimension total.
func (a PnLAttribution) ExecutionCost() float64 { return a.SlippageCost + a.SpreadCost }

// HoldingCost returns the holding dimension total.
func (a PnLAttribution) HoldingCost() float64 { return a.Commission + a.SwapCost + a.FundingCost }

// NetPnL computes the bottom-line P&L after all costs.
func (a PnLAttribution) NetPnL() float64 {
	return a.GrossPnL - a.ExecutionCost() - a.HoldingCost()
}

// SignalBps returns the signal alpha in basis points of notional.
func (a PnLAttribution) SignalBps() float64 {
	if a.Notional == 0 {
		return 0
	}
	return a.GrossPnL / a.Notional * 10000.0
}

// ExecutionBps returns the execution cost in basis points of notional.
func (a PnLAttribution) ExecutionBps() float64 {
	if a.Notional == 0 {
		return 0
	}
	return a.ExecutionCost() / a.Notional * 10000.0
}

// HoldingBps returns the holding cost in basis points of notional.
func (a PnLAttribution) HoldingBps() float64 {
	if a.Notional == 0 {
		return 0
	}
	return a.HoldingCost() / a.Notional * 10000.0
}

// NetBps returns net P&L in basis points of notional.
func (a PnLAttribution) NetBps() float64 {
	if a.Notional == 0 {
		return 0
	}
	return a.NetPnL() / a.Notional * 10000.0
}

// Validate checks the P&L identity: Net = Gross - Spread - Slippage - Commission - Swap - Funding.
func (a PnLAttribution) Validate() error {
	expected := a.GrossPnL - a.SpreadCost - a.SlippageCost - a.Commission - a.SwapCost - a.FundingCost
	actual := a.NetPnL()
	if math.Abs(expected-actual) > 0.005 {
		return fmt.Errorf("PnL identity violated: %.4f != %.4f (diff=%.6f)", expected, actual, expected-actual)
	}
	return nil
}

// Add aggregates two attributions (e.g., entry + exit legs).
func (a PnLAttribution) Add(b PnLAttribution) PnLAttribution {
	return PnLAttribution{
		GrossPnL:     a.GrossPnL + b.GrossPnL,
		SlippageCost: a.SlippageCost + b.SlippageCost,
		SpreadCost:   a.SpreadCost + b.SpreadCost,
		Commission:   a.Commission + b.Commission,
		SwapCost:     a.SwapCost + b.SwapCost,
		FundingCost:  a.FundingCost + b.FundingCost,
		Notional:     a.Notional + b.Notional,
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
func (at *PnLAttributor) Attribute(side string, openPrice, closePrice, lots, contractSize, holdingDays float64) PnLAttribution {
	cm := at.fillModel.costModel
	notional := lots * contractSize * openPrice

	// Gross P&L (signal dimension)
	grossPnL := 0.0
	if side == "buy" {
		grossPnL = (closePrice - openPrice) * lots * contractSize
	} else {
		grossPnL = (openPrice - closePrice) * lots * contractSize
	}

	// Entry leg costs
	entryNotional := lots * contractSize * openPrice
	entrySpread := cm.SpreadCost(lots)
	entryComm := cm.Commission(lots, entryNotional)
	entrySlip := cm.SlippageCost(lots, openPrice, contractSize)

	// Exit leg costs
	exitNotional := lots * contractSize * closePrice
	exitSpread := cm.SpreadCost(lots)
	exitComm := cm.Commission(lots, exitNotional)
	exitSlip := cm.SlippageCost(lots, closePrice, contractSize)

	// Swap only applies to holding duration
	swap := cm.SwapCost(side, lots, closePrice, contractSize, holdingDays)
	funding := cm.FundingCost(lots, closePrice, contractSize, 0) // not perpetual by default

	return PnLAttribution{
		GrossPnL:     grossPnL,
		SlippageCost: entrySlip + exitSlip,
		SpreadCost:   entrySpread + exitSpread,
		Commission:   entryComm + exitComm,
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
