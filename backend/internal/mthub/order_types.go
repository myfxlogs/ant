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
	// Deviation is the max slippage in points passed to the broker
	// (MT4/MT5 OrderSend slippage). 0 = use broker default.
	Deviation int32
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
	Canonical, SymbolRaw string
	Digits               int32
	// TradeMode canonical enum: 0=disabled,1=long_only,2=short_only,
	// 3=close_only,4=full. MT4/MT5 broker values are the same order and are
	// passed through as-is (distinct from the admin broker_symbols table's
	// same-named config column — resolver semantics untouched, R3).
	TradeMode              int32
	StopLevel, FreezeLevel int32
	// TradeExemode canonical enum: 0=instant,1=request,2=market,3=exchange;
	// -1=unknown. Sourced from mt4 SymbolInfoEx.Exemode; mt5 mtapi pb has no
	// execution-mode field, so mt5 always reports -1 (VM-LIVE-VENUE-R2).
	TradeExemode                     int32
	PointValue, ContractSize         decimal.Decimal
	LotSize, LotStep, LotMin, LotMax decimal.Decimal
	TickValue, TickSize              decimal.Decimal
	SwapLong, SwapShort              decimal.Decimal
	SpreadFloat                      bool
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

// SyncableClosedTrade reports whether r is a closed trade that belongs in
// the trade_records ledger. Broker history responses also carry open/pending
// rows (State != Closed, mapped verbatim with an epoch close time) and cash
// events (BALANCE/CREDIT deposits/withdrawals) — none of them are trades.
// Writing them bloats the ledger with phantom rows: TRADE-RECORDS-DUP-1
// traced 3,024 epoch rows and 160 BALANCE rows to the two unguarded sync
// sites. Single source for both call sites so the guard cannot drift.
func (r *OrderRecord) SyncableClosedTrade() bool {
	// CloseTime.Unix() <= 0 covers both the Go zero time (year 1) and the
	// Unix epoch: broker history maps "not closed" to seconds=0 → AsTime()
	// = 1970-01-01, which IsZero() does NOT catch (epoch is not the zero
	// time). No real close can be dated 1970 or earlier.
	if r.State != OrderStateClosed || r.CloseTime.Unix() <= 0 {
		return false
	}
	return r.OrderType != OrderBalance && r.OrderType != OrderCredit
}

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
	// PlaceOrder submits the order and returns the broker's synchronous
	// OrderSend receipt mapped into an OrderRecord. Fields absent from the
	// broker response stay zero (= unknown) — never echoed from the request.
	PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error)
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
