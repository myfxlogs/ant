// Package oms — BrokerAdapter interface (ADR-0012).
// All broker implementations (MT4, MT5, Binance, OKX, etc.) must implement this interface.
package oms

import (
	"context"

	"github.com/shopspring/decimal"
)

// OrderRequest is the canonical order request passed to the broker adapter.
type OrderRequest struct {
	AccountID       string  `json:"account_id"`
	Symbol          string  `json:"symbol"`           // canonical symbol
	BrokerSymbolRaw string  `json:"broker_symbol_raw"` // resolved broker-native symbol
	Side            string  `json:"side"`             // buy / sell
	Volume          decimal.Decimal `json:"volume"`
	Price           decimal.Decimal `json:"price"`            // 0 for market orders
	StopLoss        decimal.Decimal `json:"stop_loss"`
	TakeProfit      decimal.Decimal `json:"take_profit"`
	StrategyID      string  `json:"strategy_id"`
	Comment         string  `json:"comment"`
}

// BrokerResp is the broker's response to an order submission.
type BrokerResp struct {
	Ticket    string     `json:"ticket"`
	State     OrderState `json:"state"`
	FilledQty decimal.Decimal `json:"filled_qty"`
	FillPrice decimal.Decimal `json:"fill_price"`
	ErrorCode int32      `json:"error_code"`
	ErrorMsg  string     `json:"error_msg"`
}

// BrokerAdapter abstracts broker-specific order operations.
// All broker implementations must satisfy this interface.
type BrokerAdapter interface {
	// Submit sends an order to the broker. Returns the broker's response.
	Submit(ctx context.Context, req *OrderRequest) (*BrokerResp, error)

	// Cancel requests cancellation of an existing order.
	Cancel(ctx context.Context, ticket string) error

	// Modify adjusts price and/or stop-loss of an existing order.
	Modify(ctx context.Context, ticket string, price, stopPrice decimal.Decimal) error

	// Query retrieves the current state of an order from the broker.
	Query(ctx context.Context, ticket string) (*Order, error)
}

// Order is the canonical order record stored in the OMS.
type Order struct {
	ID              string     `db:"id" json:"id"`
	AccountID       string     `db:"mt_account_id" json:"account_id"`
	Platform        string     `db:"platform" json:"platform"`
	Ticket          string     `db:"ticket" json:"ticket"`
	Symbol          string     `db:"symbol" json:"symbol"`
	BrokerSymbolRaw string     `db:"broker_symbol_raw" json:"broker_symbol_raw"`
	OrderType       int16      `db:"order_type" json:"order_type"`
	Volume          decimal.Decimal `db:"volume" json:"volume"`
	Price           decimal.Decimal `db:"price" json:"price"`
	StopLoss        decimal.Decimal `db:"stop_loss" json:"stop_loss"`
	TakeProfit      decimal.Decimal `db:"take_profit" json:"take_profit"`
	State           OrderState `db:"state" json:"state"`
}
