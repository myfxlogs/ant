package model

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/google/uuid"
)

type TradeLog struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	UserID    uuid.UUID       `json:"user_id" db:"user_id"`
	AccountID uuid.UUID       `json:"account_id" db:"account_id"`
	Action    string          `json:"action" db:"action"`
	Symbol    string          `json:"symbol" db:"symbol"`
	OrderType string          `json:"order_type" db:"order_type"`
	Volume    decimal.Decimal `json:"volume" db:"volume"`
	Price     decimal.Decimal `json:"price" db:"price"`
	Ticket    int64           `json:"ticket" db:"ticket"`
	Profit    decimal.Decimal `json:"profit" db:"profit"`
	Message   string          `json:"message" db:"message"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

type TradeRecord struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	UserID       uuid.UUID       `json:"user_id" db:"user_id"`
	ScheduleID   *uuid.UUID      `json:"schedule_id" db:"schedule_id"`
	AccountID    uuid.UUID       `json:"account_id" db:"account_id"`
	Ticket       int64           `json:"ticket" db:"ticket"`
	Symbol       string          `json:"symbol" db:"symbol"`
	OrderType    string          `json:"order_type" db:"order_type"`
	Volume       decimal.Decimal `json:"volume" db:"volume"`
	OpenPrice    decimal.Decimal `json:"open_price" db:"open_price"`
	ClosePrice   decimal.Decimal `json:"close_price" db:"close_price"`
	Profit       decimal.Decimal `json:"profit" db:"profit"`
	Swap         decimal.Decimal `json:"swap" db:"swap"`
	Commission   decimal.Decimal `json:"commission" db:"commission"`
	OpenTime     time.Time       `json:"open_time" db:"open_time"`
	CloseTime    time.Time       `json:"close_time" db:"close_time"`
	StopLoss     decimal.Decimal `json:"stop_loss" db:"stop_loss"`
	TakeProfit   decimal.Decimal `json:"take_profit" db:"take_profit"`
	OrderComment string          `json:"order_comment" db:"order_comment"`
	MagicNumber  int             `json:"magic_number" db:"magic_number"`
	Platform     string          `json:"platform" db:"platform"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
	// FEAT-4: Hash chain fields for tamper-proof live trade records.
	Seq       int64  `json:"seq" db:"seq"`
	PrevHash  []byte `json:"prev_hash" db:"prev_hash"`
	EntryHash []byte `json:"entry_hash" db:"entry_hash"`
}

// ChainBreak describes a detected tampering in the trade record hash chain.
type ChainBreak struct {
	Seq    int64 `json:"seq"`
	Ticket int64 `json:"ticket"`
	// AccountID attributes the finding to its account (the chain is global —
	// VERIFY-CHAIN-SEMANTIC-1 — so per-account views are filters on it).
	AccountID uuid.UUID `json:"account_id"`
	// Type: "chain_break" (linkage gap), "hash_mismatch" (recompute miss) or
	// "unhashed" (informational: row outside the chain, disclosed as-is).
	// "deleted_link" (TRADE-RECORDS-DUP-1 exemption) is no longer produced —
	// the archived rows are chain members in the global union walk, which
	// supersedes the exemption.
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

type KlineData struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	Symbol     string          `json:"symbol" db:"symbol"`
	Timeframe  string          `json:"timeframe" db:"timeframe"`
	OpenTime   time.Time       `json:"open_time" db:"open_time"`
	CloseTime  time.Time       `json:"close_time" db:"close_time"`
	KlineDate  time.Time       `json:"kline_date" db:"kline_date"`
	OpenPrice  decimal.Decimal `json:"open_price" db:"open_price"`
	HighPrice  decimal.Decimal `json:"high_price" db:"high_price"`
	LowPrice   decimal.Decimal `json:"low_price" db:"low_price"`
	ClosePrice decimal.Decimal `json:"close_price" db:"close_price"`
	TickVolume int64           `json:"tick_volume" db:"tick_volume"`
	RealVolume decimal.Decimal `json:"real_volume" db:"real_volume"`
	Spread     int             `json:"spread" db:"spread"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}
