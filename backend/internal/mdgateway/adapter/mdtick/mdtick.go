// Package mdtick provides shared DTOs for mdgateway adapters.
// This package MUST NOT import mdgateway, mt4, or mt5.
package mdtick

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
)

const FinancialsSourceAccountSummary = "account_summary"

// Breaker is the common interface for circuit breakers used by gateway adapters.
type Breaker interface {
	Allow() bool
	OnSuccess()
	OnFailure()
}

// TickHandler is the callback that receives ticks from a gateway adapter.
type TickHandler func(t *Tick)

// ProfitHandler is the callback that receives profit/account updates from a gateway adapter.
type ProfitHandler func(p *ProfitUpdate)

// OrderUpdateHandler is the callback that receives real-time order updates from OnOrderUpdate stream.
// The handler receives the full account snapshot (metrics + all opened positions) on every order change.
type OrderUpdateHandler func(o *OrderUpdate)

// BrokerInfoHandler is the callback that receives broker-level settings after gateway connect.
// Called once per successful connection; values of 0 mean "use schema default".
type BrokerInfoHandler func(accountID, platform, broker string, info *BrokerInfo)

// BrokerInfo holds broker-level margin configuration and account financial
// snapshot fetched after mtapi Connect via AccountSummary.
// The financial fields (Balance/Equity/etc.) are authoritative broker values
// used to publish a profit_update on every connect/reconnect, ensuring the
// frontend never displays stale data when OnOrderProfit is silent.
type BrokerInfo struct {
	MarginCallPct float64 // broker margin_call_level (e.g. 60.0 == 60%)
	StopOutPct    float64 // broker stop_out_level (e.g. 30.0 == 30%)

	// HasAccountSummary is true when AccountSummary returned a valid result.
	// When false (e.g. investor accounts), financial fields are zero and should not be published.
	HasAccountSummary bool

	Balance     decimal.Decimal
	Credit      decimal.Decimal
	Equity      decimal.Decimal
	Margin      decimal.Decimal
	FreeMargin  decimal.Decimal
	MarginLevel decimal.Decimal
	Profit      decimal.Decimal
	Leverage    int32
	CapturedAt  time.Time
}

// BrokerInfoFetcher is implemented by mt4.Gateway and mt5.Gateway.
// After Connect succeeds, the runner calls FetchBrokerInfo to populate
// BrokerInfo and passes it to the OnBrokerInfo callback.
type BrokerInfoFetcher interface {
	FetchBrokerInfo(ctx context.Context) (*BrokerInfo, error)
}

// SymbolFetcher is implemented by mt4.Gateway and mt5.Gateway.
// Returns all available symbol names from the broker, used to filter
// subscription requests so non-existent symbols don't cause atomic
// SubscribeMany failures.
type SymbolFetcher interface {
	FetchAllSymbols(ctx context.Context) ([]string, error)
}

// AccountInfoProvider is implemented by gateway adapters that can return
// account-level metadata (investor flag, etc.) after connection.
type AccountInfoProvider interface {
	GetAccountInfo(ctx context.Context) (*MTAccountInfo, error)
}

// MTAccountInfo holds basic account details from AccountSummary.
type MTAccountInfo struct {
	Balance    decimal.Decimal
	Credit     decimal.Decimal
	Equity     decimal.Decimal
	Margin     decimal.Decimal
	FreeMargin decimal.Decimal
	Leverage   int32
	Currency   string
	IsInvestor bool   // true = read-only / investor password
	BrokerHost string // the access host that successfully connected
}

// ProfitUpdate represents an account profit/financial snapshot from mtapi OnOrderProfit.
type ProfitUpdate struct {
	AccountID              string
	Platform               string
	Balance                decimal.Decimal
	Credit                 decimal.Decimal
	Equity                 decimal.Decimal
	Margin                 decimal.Decimal
	FreeMargin             decimal.Decimal
	MarginLevel            decimal.Decimal
	Profit                 decimal.Decimal
	ProfitPercent          float64
	Leverage               int32
	FinancialSource        string
	CapturedAt             time.Time
	PositionsAuthoritative bool
	Positions              []ProfitPosition
}

// ProfitPosition is an open position snapshot within a ProfitUpdate.
type ProfitPosition struct {
	Ticket       int64
	Symbol       string
	Type         string // "buy", "sell", etc.
	Magic        int32  // strategy attribution magic number (ExpertID)
	Volume       decimal.Decimal
	OpenPrice    decimal.Decimal
	CurrentPrice decimal.Decimal
	StopLoss     decimal.Decimal
	TakeProfit   decimal.Decimal
	Profit       decimal.Decimal
	Swap         decimal.Decimal
	Commission   decimal.Decimal
	Comment      string
	OpenTime     int64 // unix seconds
}

// OrderUpdate represents a real-time order change event from OnOrderUpdate stream.
// Contains the triggering update + full account snapshot (metrics + opened positions).
type OrderUpdate struct {
	AccountID string
	Platform  string
	// The specific order change.
	UpdateTicket     int64
	UpdateType       string // "open", "close", "modify", "delete", "pending_open", "pending_close", etc.
	UpdateOrderType  string // "buy", "sell", "buy_limit", "sell_limit", etc. (original order type)
	UpdateSymbol     string
	UpdateVolume     decimal.Decimal
	UpdateOpenPrice  decimal.Decimal
	UpdateClosePrice decimal.Decimal
	UpdateProfit     decimal.Decimal
	UpdateSwap       decimal.Decimal
	UpdateCommission decimal.Decimal
	UpdateComment    string
	UpdateOpenTime   int64 // unix seconds
	UpdateCloseTime  int64 // unix seconds
	UpdateSL         decimal.Decimal
	UpdateTP         decimal.Decimal
	// UpdateMagic is the magic number of the triggering order (LIVE-ORDER-REENTRY-1).
	// Extracted from the Update's Order field — MT4 Order.MagicNumber / MT5 Order.ExpertId.
	// Used by the execution barrier to match confirmation events precisely.
	UpdateMagic int32
	// Account metrics from OrderUpdateSummary.
	Balance     decimal.Decimal
	Credit      decimal.Decimal
	Equity      decimal.Decimal
	Margin      decimal.Decimal
	FreeMargin  decimal.Decimal
	MarginLevel decimal.Decimal
	Profit      decimal.Decimal
	// ProfitPercent is profit as percentage of balance (Balance>0 only).
	ProfitPercent float64
	// Full opened positions list.
	Positions []OrderUpdatePosition
}

// OrderUpdatePosition is an opened position within an OrderUpdate.
type OrderUpdatePosition struct {
	Ticket       int64
	Symbol       string
	Type         string // "buy", "sell", etc.
	Magic        int32  // strategy attribution magic number (ExpertID)
	Volume       decimal.Decimal
	OpenPrice    decimal.Decimal
	CurrentPrice decimal.Decimal
	StopLoss     decimal.Decimal
	TakeProfit   decimal.Decimal
	Profit       decimal.Decimal
	Swap         decimal.Decimal
	Commission   decimal.Decimal
	Comment      string
	OpenTime     int64 // unix seconds
}

// Tick is the canonical tick representation flowing into mdgateway.
type Tick struct {
	UserID        string // ant user ID
	AccountID     string // ant account UUID
	Broker        string // broker unique identifier
	Platform      string // "mt4" or "mt5"
	SymbolRaw     string // broker-native symbol (e.g. "BTCUSDm")
	Canonical     string // normalized symbol; adapter leaves empty, mdgateway fills
	TsUnixMs      int64  // broker timestamp (ms, UTC)
	ArrivedUnixMs int64  // local arrival time (ms, UTC)
	Bid           decimal.Decimal
	Ask           decimal.Decimal
	BidVolume     float64
	AskVolume     float64
	IsReplay      bool // true when tick originates from spill_replay or backfiller (ADR-0009)
}

// Bar is produced by mdgateway.bar_aggregator from accumulated ticks.
type Bar struct {
	UserID        string
	AccountID     string
	Broker        string
	SymbolRaw     string
	Canonical     string
	Period        string // "1m","5m","15m","1h","4h","1d"
	OpenTsUnixMs  int64
	CloseTsUnixMs int64
	Open          decimal.Decimal
	High          decimal.Decimal
	Low           decimal.Decimal
	Close         decimal.Decimal
	Bid           decimal.Decimal // latest bid for real-time quote
	Ask           decimal.Decimal // latest ask for real-time quote
	Volume        float64
	TickCount     uint32
	IsClosed      bool // true when bar is finalized by AddTick; false for open bar snapshots
	IsReplay      bool // true when bar originates from spill_replay or backfiller (ADR-0009)
}

// PeriodMs returns the duration of a timeframe in milliseconds.
// This is the single source of truth for timeframe durations across the codebase.
func PeriodMs(period string) int64 {
	switch period {
	case "1m":
		return 60_000
	case "5m":
		return 300_000
	case "15m":
		return 900_000
	case "30m":
		return 1_800_000
	case "1h":
		return 3_600_000
	case "4h":
		return 14_400_000
	case "1d":
		return 86_400_000
	case "1w":
		return 604_800_000
	default:
		return 60_000
	}
}

// AccountConfig comes from PG mt_accounts_v2 view; runner decrypts and passes to adapter.
// Field names strictly align with SQL column names (see spec/13 §4.1).
type AccountConfig struct {
	AccountID  string   // mt_accounts_v2.id (UUID)
	UserID     string   // mt_accounts_v2.user_id
	Broker     string   // mt_accounts_v2.broker (from broker_company)
	Platform   string   // mt_accounts_v2.platform ("mt4" / "mt5")
	Login      string   // mt_accounts_v2.login
	Password   string   // password_encrypted decrypted plaintext (vault.Decrypt)
	Server     string   // mt_accounts_v2.server (from broker_server, display name)
	BrokerHost string   // mt_accounts_v2.broker_host (actual broker IP:port for mtapi Connect)
	MtapiHost  string   // mt_accounts_v2.mtapi_host (mtapi gateway endpoint, empty=mtapi.io)
	MtapiPort  string   // mt_accounts_v2.mtapi_port
	MtapiToken string   // mt_token plaintext from DB
	Symbols    []string // canonical_subscribed_symbols
}
