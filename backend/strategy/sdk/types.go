// Package sdk defines the Go strategy execution interface.
// Strategies implement the Strategy interface; the runtime provides
// Context, Broker, Indicators, and BarSeries.
package sdk

import (
	"time"

	"github.com/shopspring/decimal"
)

// ── Core types ──────────────────────────────────────────────────────

// OrderType matches the existing OrderType constants in mthub.
type OrderType int8

const (
	OrderMarket    OrderType = 0
	OrderLimit     OrderType = 1
	OrderStop      OrderType = 2
	OrderStopLimit OrderType = 3
)

// PositionSide indicates buy or sell.
type PositionSide int8

const (
	SideBuy  PositionSide = 1
	SideSell PositionSide = -1
)

// OrderState represents the lifecycle of an order.
type OrderState int8

const (
	StatePending   OrderState = 0
	StateOpen      OrderState = 1
	StateClosed    OrderState = 2
	StateCancelled OrderState = 3
	StateRejected  OrderState = 4
)

// FillPolicy controls how orders are filled.
type FillPolicy int8

const (
	FillIOC    FillPolicy = 0 // Immediate or Cancel
	FillReturn FillPolicy = 1 // Return unfilled (pending orders)
	FillFOK    FillPolicy = 2 // Fill or Kill
)

// AccountMode distinguishes hedging from netting accounts.
type AccountMode string

const (
	ModeHedging AccountMode = "hedging"
	ModeNetting AccountMode = "netting"
)

// ── Order request / result ──────────────────────────────────────────

// OrderRequest is sent to Broker.OrderSend.
type OrderRequest struct {
	Symbol     string
	Type       OrderType
	Side       PositionSide
	Volume     decimal.Decimal
	Price      decimal.Decimal // 0 for market orders
	StopLoss   decimal.Decimal
	TakeProfit decimal.Decimal
	Deviation  int32
	Magic      int32
	Comment    string
	FillPolicy FillPolicy
}

// RetCode mirrors the ConnectRPC Retcode enum.
type RetCode string

const (
	RetDone          RetCode = "done"
	RetDonePartial   RetCode = "done_partial"
	RetRejected      RetCode = "rejected"
	RetRiskBlocked   RetCode = "risk_blocked"
	RetInvalidVolume RetCode = "invalid_volume"
	RetNoMoney       RetCode = "no_money"
	RetInvalidPrice  RetCode = "invalid_price"
	RetOffQuotes     RetCode = "off_quotes"
	RetTooManyOrders RetCode = "too_many_orders"
)

// OrderResult is returned by Broker.OrderSend.
type OrderResult struct {
	RetCode RetCode
	Ticket  int64
	Volume  decimal.Decimal
	Price   decimal.Decimal
}

// ── Positions / Pending Orders ──────────────────────────────────────

// Position represents an open market position.
type Position struct {
	Ticket     int64
	Symbol     string
	Side       PositionSide
	Volume     decimal.Decimal
	OpenPrice  decimal.Decimal
	StopLoss   decimal.Decimal
	TakeProfit decimal.Decimal
	Profit     decimal.Decimal
	Swap       decimal.Decimal
	Commission decimal.Decimal
	Comment    string
	Magic      int32
	OpenTime   time.Time
}

// PendingOrder represents a pending (limit/stop) order.
type PendingOrder struct {
	Ticket     int64
	Symbol     string
	Type       OrderType
	Side       PositionSide
	Volume     decimal.Decimal
	Price      decimal.Decimal
	StopLoss   decimal.Decimal
	TakeProfit decimal.Decimal
	Comment    string
	Magic      int32
	OpenTime   time.Time
	Expiration time.Time
}

// Deal represents a historical trade.
type Deal struct {
	Ticket     int64
	OrderTicket int64
	Symbol     string
	Side       PositionSide
	Volume     decimal.Decimal
	Price      decimal.Decimal
	Profit     decimal.Decimal
	Commission decimal.Decimal
	Swap       decimal.Decimal
	Comment    string
	Magic      int32
	OpenTime   time.Time
	CloseTime  time.Time
}
