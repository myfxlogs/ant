// Package mdtick provides shared DTOs for mdgateway adapters.
// This package MUST NOT import mdgateway, mt4, or mt5.
package mdtick

import (
	"context"

	"github.com/shopspring/decimal"
)

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

// BrokerInfo holds broker-level margin configuration fetched after mtapi Connect.
// Zero values signal that the broker did not expose these settings and the
// schema DEFAULTs (100.0 / 50.0) should be used.
type BrokerInfo struct {
	MarginCallPct float64 // broker margin_call_level (e.g. 60.0 == 60%)
	StopOutPct    float64 // broker stop_out_level (e.g. 30.0 == 30%)
}

// BrokerInfoFetcher is implemented by mt4.Gateway and mt5.Gateway.
// After Connect succeeds, the runner calls FetchBrokerInfo to populate
// BrokerInfo and passes it to the OnBrokerInfo callback.
type BrokerInfoFetcher interface {
	FetchBrokerInfo(ctx context.Context) (*BrokerInfo, error)
}

// ProfitUpdate represents an account profit/financial snapshot from mtapi OnOrderProfit.
type ProfitUpdate struct {
	AccountID    string
	Platform     string
	Balance      float64
	Credit       float64
	Equity       float64
	Margin       float64
	FreeMargin   float64
	MarginLevel  float64
	Profit       float64
	ProfitPercent float64
	Positions    []ProfitPosition
}

// ProfitPosition is an open position snapshot within a ProfitUpdate.
type ProfitPosition struct {
	Ticket       int64
	Symbol       string
	Profit       float64
	Volume       float64
	CurrentPrice float64
}

// OrderUpdate represents a real-time order change event from OnOrderUpdate stream.
// Contains the triggering update + full account snapshot (metrics + opened positions).
type OrderUpdate struct {
	AccountID   string
	Platform    string
	// The specific order change.
	UpdateTicket  int64
	UpdateType    string // "open", "close", "modify", "delete", "pending_open", "pending_close", etc.
	UpdateSymbol  string
	UpdateVolume  float64
	UpdateOpenPrice  float64
	UpdateClosePrice float64
	UpdateProfit     float64
	UpdateSwap       float64
	UpdateCommission float64
	UpdateComment    string
	UpdateOpenTime   int64 // unix seconds
	UpdateCloseTime  int64 // unix seconds
	UpdateSL         float64
	UpdateTP         float64
	// Account metrics from OrderUpdateSummary.
	Balance     float64
	Credit      float64
	Equity      float64
	Margin      float64
	FreeMargin  float64
	MarginLevel float64
	Profit      float64
	// Full opened positions list.
	Positions []OrderUpdatePosition
}

// OrderUpdatePosition is an opened position within an OrderUpdate.
type OrderUpdatePosition struct {
	Ticket       int64
	Symbol       string
	Type         string  // "buy", "sell", etc.
	Volume       float64
	OpenPrice    float64
	CurrentPrice float64
	StopLoss     float64
	TakeProfit   float64
	Profit       float64
	Swap         float64
	Commission   float64
	Comment      string
	OpenTime     int64   // unix seconds
}

// Tick is the canonical tick representation flowing into mdgateway.
type Tick struct {
	UserID        string          // ant user ID
	AccountID     string          // ant account UUID
	Broker        string          // broker unique identifier
	Platform      string          // "mt4" or "mt5"
	SymbolRaw     string          // broker-native symbol (e.g. "BTCUSDm")
	Canonical     string          // normalized symbol; adapter leaves empty, mdgateway fills
	TsUnixMs      int64           // broker timestamp (ms, UTC)
	ArrivedUnixMs int64           // local arrival time (ms, UTC)
	Bid           decimal.Decimal
	Ask           decimal.Decimal
	BidVolume     float64
	AskVolume     float64
	IsReplay      bool            // true when tick originates from spill_replay or backfiller (ADR-0009)
}

// Bar is produced by mdgateway.bar_aggregator from accumulated ticks.
type Bar struct {
	UserID        string
	AccountID     string
	Broker        string
	Canonical     string
	Period        string // "1m","5m","15m","1h","4h","1d"
	OpenTsUnixMs  int64
	CloseTsUnixMs int64
	Open          decimal.Decimal
	High          decimal.Decimal
	Low           decimal.Decimal
	Close         decimal.Decimal
	Volume        float64
	TickCount     uint32
	IsReplay      bool   // true when bar originates from spill_replay or backfiller (ADR-0009)
}

// AccountConfig comes from PG mt_accounts_v2 view; runner decrypts and passes to adapter.
// Field names strictly align with SQL column names (see spec/13 §4.1).
type AccountConfig struct {
	AccountID    string // mt_accounts_v2.id (UUID)
	UserID       string // mt_accounts_v2.user_id
	Broker       string // mt_accounts_v2.broker (from broker_company)
	Platform     string // mt_accounts_v2.platform ("mt4" / "mt5")
	Login        string // mt_accounts_v2.login
	Password     string // password_encrypted decrypted plaintext (vault.Decrypt)
	Server       string // mt_accounts_v2.server (from broker_server, display name)
	BrokerHost   string // mt_accounts_v2.broker_host (actual broker IP:port for mtapi Connect)
	MtapiHost    string // mt_accounts_v2.mtapi_host (mtapi gateway endpoint, empty=mtapi.io)
	MtapiPort    string // mt_accounts_v2.mtapi_port
	MtapiToken   string   // mt_token plaintext from DB
	Symbols      []string // canonical_subscribed_symbols
}
