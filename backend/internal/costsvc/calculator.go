package costsvc

import (
	"math"
	"time"
)

// EffectiveSwapDays returns the number of swap charges accounting for
// Wednesday triple-swap convention (Wed→Thu rollover charges 3× for the weekend).
// start is the trade entry date, holdingDays is calendar days held.
func EffectiveSwapDays(start time.Time, holdingDays int) int {
	if holdingDays <= 0 {
		return 0
	}
	charges := 0
	for i := 0; i < holdingDays; i++ {
		d := start.AddDate(0, 0, i)
		if d.Weekday() == time.Wednesday {
			charges += 3
		} else {
			charges++
		}
	}
	return charges
}

// SwapCostDate computes swap cost with Wednesday triple-swap convention.
func (m *CostModel) SwapCostDate(side string, lots float64, start time.Time, holdingDays int) float64 {
	effective := EffectiveSwapDays(start, holdingDays)
	return m.SwapCost(side, lots, 0, 0, float64(effective))
}

// SpreadCost computes the half-spread cost for a trade.
// For a buy, you cross the spread to the ask (pay half spread).
// For a sell, you cross the spread to the bid (receive half spread less).
// Returns the cost in account currency.
func (m *CostModel) SpreadCost(lots float64) float64 {
	if m.SpreadPips <= 0 || m.PipValue <= 0 {
		return 0
	}
	return (m.SpreadPips / 2.0) * m.PipValue * lots
}

// Commission computes the broker commission for a trade.
// Uses per-lot rate or per-notional (bps) rate, whichever produces the higher cost.
// The result is capped at MinCommission as a floor.
func (m *CostModel) Commission(lots, notional float64) float64 {
	cost := 0.0
	if m.CommissionPerLot > 0 {
		cost = m.CommissionPerLot * lots
	}
	if m.CommissionBps > 0 {
		bpsCost := notional * m.CommissionBps / 10000.0
		if bpsCost > cost {
			cost = bpsCost
		}
	}
	return math.Max(cost, m.MinCommission)
}

// SwapCost computes the overnight holding cost for holding days.
// Side: "buy" uses SwapLong rate, "sell" uses SwapShort rate.
func (m *CostModel) SwapCost(side string, lots, price, contractSize float64, holdingDays float64) float64 {
	rate := m.SwapLong
	if side == "sell" {
		rate = m.SwapShort
	}
	if rate == 0 || holdingDays <= 0 {
		return 0
	}
	return rate * lots * holdingDays
}

// FundingCost computes the periodic funding payment for perpetual instruments.
// fundingRate is applied to the notional position value at each funding interval.
func (m *CostModel) FundingCost(lots, price, contractSize float64, holdingDuration time.Duration) float64 {
	if m.FundingRate <= 0 || m.FundingInterval <= 0 || holdingDuration <= 0 {
		return 0
	}
	intervals := float64(holdingDuration) / float64(m.FundingInterval)
	notional := lots * contractSize * price
	return m.FundingRate * notional * intervals
}

// SlippageCost estimates execution slippage for a trade.
func (m *CostModel) SlippageCost(lots, price, contractSize float64) float64 {
	if m.SlippageBps <= 0 {
		return 0
	}
	notional := lots * contractSize * price
	return notional * m.SlippageBps / 10000.0
}
