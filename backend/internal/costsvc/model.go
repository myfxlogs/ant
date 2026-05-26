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

import "time"

// CostModel defines the cost parameters for a symbol on a broker.
type CostModel struct {
	Symbol string
	Broker string

	// Spread in pips (e.g., 1.0 = 1 pip on EURUSD = 0.0001).
	SpreadPips float64
	// PipSize is the minimum price increment (e.g., 0.0001 for EURUSD, 1.0 for BTCUSD).
	PipSize float64
	// PipValue is the notional value of 1 pip per standard lot (e.g., $10 for EURUSD).
	PipValue float64

	// Commission per lot traded (e.g., $7 per lot round-turn).
	CommissionPerLot float64
	// CommissionBps is commission as basis points of notional (1 bps = 0.0001).
	CommissionBps float64

	// SwapLong is the daily swap rate for long positions (can be negative = you pay).
	SwapLong float64
	// SwapShort is the daily swap rate for short positions.
	SwapShort float64

	// FundingRate is the periodic funding rate for perpetuals (e.g., 0.0001 = 0.01%).
	FundingRate float64
	// FundingInterval is the time between funding payments.
	FundingInterval time.Duration

	// SlippageBps is the expected execution slippage in basis points.
	SlippageBps float64

	// MinCommission is the minimum commission per trade (floor).
	MinCommission float64
}

// CostSnapshot is a frozen copy of a CostModel for deterministic backtest replay.
type CostSnapshot struct {
	Symbol           string    `json:"symbol"`
	Broker           string    `json:"broker"`
	SpreadPips       float64   `json:"spread_pips"`
	PipSize          float64   `json:"pip_size"`
	PipValue         float64   `json:"pip_value"`
	CommissionPerLot float64   `json:"commission_per_lot"`
	CommissionBps    float64   `json:"commission_bps"`
	SwapLong         float64   `json:"swap_long"`
	SwapShort        float64   `json:"swap_short"`
	FundingRate      float64   `json:"funding_rate"`
	FundingInterval  int64     `json:"funding_interval_ns"`
	SlippageBps      float64   `json:"slippage_bps"`
	MinCommission    float64   `json:"min_commission"`
	FrozenAt         time.Time `json:"frozen_at"`
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
		SpreadPips:       1.0,
		PipSize:          0.0001,
		PipValue:         10.0,
		CommissionPerLot: 7.0,
		SwapLong:         -3.5,
		SwapShort:        0.5,
		SlippageBps:      0.5,
		MinCommission:    0,
	}
}

// DefaultCryptoModel returns a typical crypto cost model.
func DefaultCryptoModel(symbol string) *CostModel {
	return &CostModel{
		Symbol:           symbol,
		Broker:           "default",
		SpreadPips:       10.0,
		PipSize:          1.0,
		PipValue:         1.0,
		CommissionBps:    10.0, // 10 bps
		FundingRate:      0.0001,
		FundingInterval:  8 * time.Hour,
		SlippageBps:      2.0,
		MinCommission:    0,
	}
}
