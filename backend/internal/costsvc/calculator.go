package costsvc

import (
	"time"

	"github.com/shopspring/decimal"
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
func (m *CostModel) SwapCostDate(side string, lots decimal.Decimal, start time.Time, holdingDays int) decimal.Decimal {
	effective := EffectiveSwapDays(start, holdingDays)
	return m.SwapCost(side, lots, decimal.Zero, decimal.Zero, decimal.NewFromInt(int64(effective)))
}

// SpreadCost computes the half-spread cost for a trade.
// For a buy, you cross the spread to the ask (pay half spread).
// For a sell, you cross the spread to the bid (receive half spread less).
// Returns the cost in account currency.
func (m *CostModel) SpreadCost(lots decimal.Decimal) decimal.Decimal {
	if m.SpreadPips.LessThanOrEqual(decimal.Zero) || m.PipValue.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return m.SpreadPips.Div(decimal.NewFromInt(2)).Mul(m.PipValue).Mul(lots)
}

// Commission computes the broker commission for a trade.
// Uses per-lot rate or per-notional (bps) rate, whichever produces the higher cost.
// The result is capped at MinCommission as a floor.
func (m *CostModel) Commission(lots, notional decimal.Decimal) decimal.Decimal {
	cost := decimal.Zero
	if m.CommissionPerLot.GreaterThan(decimal.Zero) {
		cost = m.CommissionPerLot.Mul(lots)
	}
	if m.CommissionBps.GreaterThan(decimal.Zero) {
		bpsCost := notional.Mul(m.CommissionBps).Div(decimal.NewFromInt(10000))
		if bpsCost.GreaterThan(cost) {
			cost = bpsCost
		}
	}
	if cost.LessThan(m.MinCommission) {
		cost = m.MinCommission
	}
	return cost
}

// SwapCost computes the overnight holding cost for holding days.
// Side: "buy" uses SwapLong rate, "sell" uses SwapShort rate.
func (m *CostModel) SwapCost(side string, lots, price, contractSize, holdingDays decimal.Decimal) decimal.Decimal {
	rate := m.SwapLong
	if side == "sell" {
		rate = m.SwapShort
	}
	if rate.Equal(decimal.Zero) || holdingDays.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return rate.Mul(lots).Mul(holdingDays)
}

// FundingCost computes the periodic funding payment for perpetual instruments.
// fundingRate is applied to the notional position value at each funding interval.
func (m *CostModel) FundingCost(lots, price, contractSize decimal.Decimal, holdingDuration time.Duration) decimal.Decimal {
	if m.FundingRate.LessThanOrEqual(decimal.Zero) || m.FundingInterval <= 0 || holdingDuration <= 0 {
		return decimal.Zero
	}
	intervals := decimal.NewFromFloat(float64(holdingDuration) / float64(m.FundingInterval))
	notional := lots.Mul(contractSize).Mul(price)
	return m.FundingRate.Mul(notional).Mul(intervals)
}

// SlippageCost estimates execution slippage for a trade.
func (m *CostModel) SlippageCost(lots, price, contractSize decimal.Decimal) decimal.Decimal {
	if m.SlippageBps.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	notional := lots.Mul(contractSize).Mul(price)
	return notional.Mul(m.SlippageBps).Div(decimal.NewFromInt(10000))
}
