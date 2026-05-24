// Package mdtick provides shared DTOs for mdgateway adapters.
// This package MUST NOT import mdgateway, mt4, or mt5.
package mdtick

import "github.com/shopspring/decimal"

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
	Server       string // mt_accounts_v2.server (from broker_server)
	MtapiHost    string // mt_accounts_v2.mtapi_host (from broker_host)
	MtapiPort    string // mt_accounts_v2.mtapi_port
	MtapiToken   string // mtapi_token_encrypted decrypted plaintext
}
