// Package sdk defines the Go strategy execution interface.
// Strategies implement the Strategy interface; the runtime provides
// Context, Broker, Indicators, and BarSeries.
package sdk

import (
	"fmt"
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
	// Comment is sent to the broker verbatim. MT4 accepts ANSI codepage
	// bytes only — non-ASCII may be transcoded or truncated (venue limit,
	// VM-LIVE-PARITY-F3). Read-back via broker records is authoritative.
	Comment    string
	FillPolicy FillPolicy
}

// BrokerRejectError carries the broker's numeric rejection code verbatim
// across the signal-dispatch boundary (VM-ERR-CODE-COLLAPSE-1). MT4 codes
// are the MQL4 ERR_* values (130 invalid stops, 136 off quotes); MT5 codes
// are the platform retcodes (10016 invalid stops, 10019 no money). The VM
// maps Code onto GetLastError — the same number a real terminal surfaces.
// Code 0 means the broker gave no numeric code (unknown).
type BrokerRejectError struct {
	Op      string // dispatch op label, e.g. "mt4 OrderSend"
	Code    int32
	Message string
}

func (e *BrokerRejectError) Error() string {
	if e == nil {
		return "broker rejected order"
	}
	return fmt.Sprintf("%s: code=%d msg=%s", e.Op, e.Code, e.Message)
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
// VM-LIVE-PARITY-F1 contract: Volume/Price are BROKER FILL FACTS from the
// OrderSend receipt — zero = unknown. Never echo the request values.
type OrderResult struct {
	RetCode RetCode
	Ticket  int64
	Volume  decimal.Decimal
	Price   decimal.Decimal
}

// ── Positions / Pending Orders ──────────────────────────────────────

// Position represents an open or closed market position.
// For open positions, ClosePrice and CloseTime are zero values.
// For closed positions (returned by HistoryOrders), they carry the actual close data.
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
	ClosePrice decimal.Decimal
	CloseTime  time.Time
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
	Ticket      int64
	OrderTicket int64
	Symbol      string
	Side        PositionSide
	Volume      decimal.Decimal
	Price       decimal.Decimal
	Profit      decimal.Decimal
	Commission  decimal.Decimal
	Swap        decimal.Decimal
	Comment     string
	Magic       int32
	OpenTime    time.Time
	CloseTime   time.Time
}
