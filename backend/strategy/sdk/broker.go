package sdk

import "github.com/shopspring/decimal"

// Broker provides trading operations to strategies.
// All methods are synchronous — they block until the broker responds.
type Broker interface {
	// OrderSend places a new order (market or pending).
	OrderSend(req OrderRequest) (OrderResult, error)

	// PositionClose closes a position by ticket.
	// volume=0 means close the full position.
	PositionClose(ticket int64, volume decimal.Decimal) (OrderResult, error)

	// PositionModify changes SL/TP on an open position.
	PositionModify(ticket int64, sl, tp decimal.Decimal) (OrderResult, error)

	// OrderDelete cancels a pending order by ticket.
	OrderDelete(ticket int64) (OrderResult, error)

	// Positions returns all open positions, optionally filtered by magic.
	// magic=0 means no filter.
	Positions(magic int32) []Position

	// Orders returns all pending orders, optionally filtered by magic.
	// magic=0 means no filter.
	Orders(magic int32) []PendingOrder

	// HistoryOrders returns closed/cancelled orders in the given time range.
	HistoryOrders(from, to int64) []Position

	// Deals returns executed deals in the given time range.
	Deals(from, to int64, magic int32) []Deal

	// SymbolInfo returns symbol parameters (digits, point, min/max volume, etc.).
	SymbolInfo(symbol string) (SymbolInfo, error)

	// Account returns the current account state.
	Account() AccountInfo
}

// AccountInfo holds the current account state.
type AccountInfo struct {
	Balance       decimal.Decimal
	Equity        decimal.Decimal
	Margin        decimal.Decimal
	FreeMargin    decimal.Decimal
	MarginLevel   decimal.Decimal
	Leverage      int32
	Currency      string
	Mode          AccountMode
}

// SymbolInfo holds static symbol parameters.
type SymbolInfo struct {
	Name        string
	Digits      int32
	Point       decimal.Decimal
	VolumeMin   decimal.Decimal
	VolumeMax   decimal.Decimal
	VolumeStep  decimal.Decimal
	StopsLevel  int32
	Spread      int32
	TickValue   decimal.Decimal
	TickSize    decimal.Decimal
	SwapLong    decimal.Decimal
	SwapShort   decimal.Decimal
	ContractSize decimal.Decimal
}
