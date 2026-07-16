// Package costsvc provides trading cost models (M10-BASE-D1).
//
// Models all explicit and implicit trading costs:
//   - Spread cost (bid-ask half-spread)
//   - Commission (per-lot or per-notional)
//   - Swap/rollover (overnight holding)
//   - Funding rate (perpetual instruments)
//   - Slippage (execution deviation)
//
// A CostModel is per-symbol/per-broker and can be snapshotted (CostSnapshot)
// for deterministic backtest replay.

package costsvc

import (
	"time"

	"github.com/shopspring/decimal"
)

// CostModel defines the cost parameters for a symbol on a broker.
type CostModel struct {
	Symbol string
	Broker string

	// Spread in pips (e.g., 1.0 = 1 pip on EURUSD = 0.0001).
	SpreadPips decimal.Decimal
	// PipSize is the minimum price increment (e.g., 0.0001 for EURUSD, 1.0 for BTCUSD).
	PipSize decimal.Decimal
	// PipValue is the notional value of 1 pip per standard lot (e.g., $10 for EURUSD).
	PipValue decimal.Decimal

	// Commission per lot traded (e.g., $7 per lot round-turn).
	CommissionPerLot decimal.Decimal
	// CommissionBps is commission as basis points of notional (1 bps = 0.0001).
	CommissionBps decimal.Decimal

	// SwapLong is the daily swap rate for long positions (can be negative = you pay).
	SwapLong decimal.Decimal
	// SwapShort is the daily swap rate for short positions.
	SwapShort decimal.Decimal

	// FundingRate is the periodic funding rate for perpetuals (e.g., 0.0001 = 0.01%).
	FundingRate decimal.Decimal
	// FundingInterval is the time between funding payments.
	FundingInterval time.Duration

	// SlippageBps is the expected execution slippage in basis points.
	SlippageBps decimal.Decimal

	// MinCommission is the minimum commission per trade (floor).
	MinCommission decimal.Decimal
}

// CostSnapshot is a frozen copy of a CostModel for deterministic backtest replay.
type CostSnapshot struct {
	Symbol           string          `json:"symbol"`
	Broker           string          `json:"broker"`
	SpreadPips       decimal.Decimal `json:"spread_pips"`
	PipSize          decimal.Decimal `json:"pip_size"`
	PipValue         decimal.Decimal `json:"pip_value"`
	CommissionPerLot decimal.Decimal `json:"commission_per_lot"`
	CommissionBps    decimal.Decimal `json:"commission_bps"`
	SwapLong         decimal.Decimal `json:"swap_long"`
	SwapShort        decimal.Decimal `json:"swap_short"`
	FundingRate      decimal.Decimal `json:"funding_rate"`
	FundingInterval  int64           `json:"funding_interval_ns"`
	SlippageBps      decimal.Decimal `json:"slippage_bps"`
	MinCommission    decimal.Decimal `json:"min_commission"`
	FrozenAt         time.Time       `json:"frozen_at"`
}

// Snapshot creates a frozen copy of the cost model.
func (m *CostModel) Snapshot() CostSnapshot {
	return CostSnapshot{
		Symbol:           m.Symbol,
		Broker:           m.Broker,
		SpreadPips:       m.SpreadPips,
		PipSize:          m.PipSize,
		PipValue:         m.PipValue,
		CommissionPerLot: m.CommissionPerLot,
		CommissionBps:    m.CommissionBps,
		SwapLong:         m.SwapLong,
		SwapShort:        m.SwapShort,
		FundingRate:      m.FundingRate,
		FundingInterval:  int64(m.FundingInterval),
		SlippageBps:      m.SlippageBps,
		MinCommission:    m.MinCommission,
		FrozenAt:         Clk.Now(),
	}
}

// DefaultForexModel returns a typical retail forex cost model.
func DefaultForexModel(symbol string) *CostModel {
	return &CostModel{
		Symbol:           symbol,
		Broker:           "default",
		SpreadPips:       decimal.NewFromFloat(1.0),
		PipSize:          decimal.NewFromFloat(0.0001),
		PipValue:         decimal.NewFromFloat(10.0),
		CommissionPerLot: decimal.NewFromFloat(7.0),
		SwapLong:         decimal.NewFromFloat(-3.5),
		SwapShort:        decimal.NewFromFloat(0.5),
		SlippageBps:      decimal.NewFromFloat(0.5),
		MinCommission:    decimal.Zero,
	}
}

// DefaultCryptoModel returns a typical crypto cost model.
func DefaultCryptoModel(symbol string) *CostModel {
	return &CostModel{
		Symbol:           symbol,
		Broker:           "default",
		SpreadPips:       decimal.NewFromFloat(10.0),
		PipSize:          decimal.NewFromInt(1),
		PipValue:         decimal.NewFromInt(1),
		CommissionBps:    decimal.NewFromFloat(10.0), // 10 bps
		FundingRate:      decimal.NewFromFloat(0.0001),
		FundingInterval:  8 * time.Hour,
		SlippageBps:      decimal.NewFromFloat(2.0),
		MinCommission:    decimal.Zero,
	}
}
