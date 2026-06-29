package backtest

import (
	"time"

	"github.com/shopspring/decimal"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/strategy/sdk"
)

// Config holds backtest parameters.
type Config struct {
	Symbol         string
	Timeframe      string
	StartDate      time.Time
	EndDate        time.Time
	InitialCapital decimal.Decimal
	Leverage       int32
	Commission     decimal.Decimal // percentage, e.g. 0.0003 = 0.03%
	Slippage       decimal.Decimal // pips
	SwapRate       decimal.Decimal // overnight swap rate (e.g. 0.00001)
	StrictMode     bool            // if true, skip bars with missing data

	// Symbol properties for SymbolInfo
	SymbolDigits  int32           // e.g. 5 for EURUSD
	SymbolPoint   decimal.Decimal // e.g. 0.00001 for 5-digit
	VolumeMin     decimal.Decimal
	VolumeMax     decimal.Decimal
	VolumeStep    decimal.Decimal
	ContractSize  decimal.Decimal // typically 100000 for forex
}

// Result holds the complete backtest output.
type Result struct {
	Config     Config
	Metrics    *antv1.BacktestMetrics // reuses existing proto type
	Equity     []EquityPoint
	Trades     []Trade
	Signals    []sdk.Signal
	StartedAt  time.Time
	FinishedAt time.Time
}

// EquityPoint is a single point on the equity curve.
type EquityPoint struct {
	Time   time.Time
	Equity decimal.Decimal
	Bar    int
}

// Trade represents a completed round-trip (open → close).
type Trade struct {
	Symbol     string
	Side       sdk.PositionSide
	EntryTime  time.Time
	ExitTime   time.Time
	EntryPrice decimal.Decimal
	ExitPrice  decimal.Decimal
	Volume     decimal.Decimal
	Profit     decimal.Decimal
	ProfitPct  float64
	Commission decimal.Decimal
	Comment    string
}

// OrderRecord tracks a single order through its lifecycle.
type OrderRecord struct {
	Ticket      int64
	Symbol      string
	Side        sdk.PositionSide
	OrderType   sdk.OrderType
	Volume      decimal.Decimal
	Price       decimal.Decimal
	StopLoss    decimal.Decimal
	TakeProfit  decimal.Decimal
	OpenTime    time.Time
	CloseTime   time.Time
	ClosePrice  decimal.Decimal
	State       OrderState
	Profit      decimal.Decimal
	Commission  decimal.Decimal
	Swap        decimal.Decimal
	Comment     string
	Magic       int32
	OpenBar     int
}

// OrderState is the lifecycle state of an order.
type OrderState int8

const (
	OrderPending OrderState = iota
	OrderOpen
	OrderClosed
	OrderCancelled
)
