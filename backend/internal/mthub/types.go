// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
// Package mthub — MT Session Hub: centralises MT4/MT5 session management.
package mthub

// ── Event types ──

// OrderEvent is a flat order event used for pub/sub within the hub.
type OrderEvent struct {
	AccountId string
	Type      string
	Order     *OrderRecord
}

// OrderRecord is a flat order struct used by callers.
type OrderRecord struct {
	Ticket       int64
	Symbol       string
	Side         string
	Lots         float64
	OpenPrice    float64
	ClosePrice   float64
	Profit       float64
	Swap         float64
	Commission   float64
	OpenTime     string
	CloseTime    string
	State        string
	OpenTimeMs   int64
	CurrentPrice float64
}

// ── Session types ──

// EnsureSessionResult holds the result of an EnsureSession call.
type EnsureSessionResult struct {
	SessionID     string
	AlreadyActive bool
}

// ── Symbol types ──

// SymbolParam describes a symbol's contract parameters.
type SymbolParam struct {
	Symbol       string
	Digits       int32
	Point        float64
	ContractSize float64
	MinLot       float64
	MaxLot       float64
	LotStep      float64
}

// ── Price types ──

// PriceBar is a single OHLCV bar.
type PriceBar struct {
	OpenTsMs int64
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Volume   float64
}

// ── Request types (service method inputs) ──

// OrderRequest is the input for placing an order.
type OrderRequest struct {
	Symbol   string
	Side     string
	Volume   float64
	Price    float64
	Sl       float64
	Tp       float64
	Comment  string
	Type     string
}

// CloseRequest is the input for closing an order.
type CloseRequest struct {
	Ticket int64
	Lots   float64
}

// HistoryRequest is the input for fetching order history.
type HistoryRequest struct {
	From string
	To   string
}

// ── Response types ──

// OrderSendResult holds the result of placing an order.
type OrderSendResult struct {
	Ticket int64
	Error  string
}

// CloseResult holds the result of closing an order.
type CloseResult struct {
	Error string
}

// ── History types (for executor interface) ──

// HistoryOrderInfo is a raw history record from the MT adapter.
type HistoryOrderInfo struct {
	Ticket       int64
	Symbol       string
	Type         string
	Lots         float64
	OpenPrice    float64
	ClosePrice   float64
	Profit       float64
	Swap         float64
	Commission   float64
	OpenTime     string
	CloseTime    string
	OpenTimeMs   int64
	CurrentPrice float64
}
