package mthub

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

// --- Order / Symbol / Bar types ---

type OrderRequest struct {
	AccountID, Canonical                string
	Side                                Side
	OrderType                           OrderType
	Volume, Price, StopLoss, TakeProfit decimal.Decimal
	Comment, ClientID                   string
	Magic                               int32
}

type OrderRecord struct {
	Ticket                                                  int64
	AccountID, SymbolRaw, Canonical                         string
	Side                                                    Side
	OrderType                                               OrderType
	Volume, OpenPrice, ClosePrice, Profit, Commission, Swap decimal.Decimal
	StopLoss, TakeProfit                                    decimal.Decimal
	OpenTime, CloseTime                                     time.Time
	Comment                                                 string
	Magic                                                   int32
	State                                                   OrderState
}

type SymbolParam struct {
	Canonical, SymbolRaw                                       string
	Digits, TradeMode, StopLevel                               int32
	PointValue, ContractSize, LotSize, LotStep, LotMin, LotMax decimal.Decimal
	SpreadFloat                                                bool
}

type Bar struct {
	Time                           time.Time
	Open, High, Low, Close, Volume decimal.Decimal
}

type Side int8

const (
	SideBuy  Side = 1
	SideSell Side = -1
)

type OrderType int8

const (
	OrderMarket OrderType = iota
	OrderLimit
	OrderStop
	OrderStopLimit
	OrderBalance
	OrderCredit
)

type OrderState int8

const (
	OrderStatePending OrderState = iota
	OrderStateOpen
	OrderStateClosed
	OrderStateCancelled
	OrderStateRejected
)

type OrderEvent struct {
	AccountID string
	Ticket    int64
	EventType string
	Order     *OrderRecord
	Timestamp time.Time
}

type OrderEventHandler func(*OrderEvent)

// OrderTypeString returns a human-readable string for the order type,
// prefixed by side (e.g. "BUY_LIMIT", "SELL_STOP", "BALANCE").
// Shared by service/account_sync_service.go and connect/system/mthub_service_orders.go.
func (r *OrderRecord) OrderTypeString() string {
	prefix := "BUY"
	if r.Side == SideSell {
		prefix = "SELL"
	}
	switch r.OrderType {
	case OrderMarket:
		return prefix
	case OrderLimit:
		return prefix + "_LIMIT"
	case OrderStop:
		return prefix + "_STOP"
	case OrderStopLimit:
		return prefix + "_STOP_LIMIT"
	case OrderBalance:
		return "BALANCE"
	case OrderCredit:
		return "CREDIT"
	default:
		return prefix
	}
}

type OrderExecutor interface {
	Platform() string
	PlaceOrder(ctx context.Context, req *OrderRequest) (int64, error)
	CloseOrder(ctx context.Context, ticket int64, lots decimal.Decimal) error
	DeleteOrder(ctx context.Context, ticket int64) error
	ModifyOrder(ctx context.Context, ticket int64, sl, tp, price decimal.Decimal) error
	FetchOpenedOrders(ctx context.Context) ([]*OrderRecord, error)
	FetchOrderHistory(ctx context.Context, from, to time.Time) ([]*OrderRecord, error)
	FetchSymbolParams(ctx context.Context, canonicals []string) ([]*SymbolParam, error)
	FetchAllSymbols(ctx context.Context) ([]string, error)
	FetchPriceHistory(ctx context.Context, symbol, period string, from, to int64, count int) ([]*Bar, error)
	AddSymbols(ctx context.Context, symbols []string) error
	SubscribeOrderEvents(ctx context.Context, h OrderEventHandler) error
}

// MarginRequirer is implemented by executors that can query the broker for required margin.
type MarginRequirer interface {
	RequiredMargin(ctx context.Context, symbol string, lots decimal.Decimal, side Side, price decimal.Decimal) (decimal.Decimal, error)
}
