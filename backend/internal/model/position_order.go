package model

import (
	"github.com/shopspring/decimal"
	"time"

	"github.com/google/uuid"
)

type Position struct {
	ID           uuid.UUID `json:"id" db:"id"`
	MTAccountID  uuid.UUID `json:"mt_account_id" db:"mt_account_id"`
	Platform     string    `json:"platform" db:"platform"`
	Ticket       int64     `json:"ticket" db:"ticket"`
	Symbol       string    `json:"symbol" db:"symbol"`
	OrderType    int16     `json:"order_type" db:"order_type"`
	Volume       decimal.Decimal   `json:"volume" db:"volume"`
	OpenPrice    decimal.Decimal   `json:"open_price" db:"open_price"`
	CurrentPrice decimal.Decimal   `json:"current_price" db:"current_price"`
	StopLoss     decimal.Decimal   `json:"stop_loss" db:"stop_loss"`
	TakeProfit   decimal.Decimal   `json:"take_profit" db:"take_profit"`
	OpenTime     time.Time `json:"open_time" db:"open_time"`
	Profit       decimal.Decimal   `json:"profit" db:"profit"`
	Swap         decimal.Decimal   `json:"swap" db:"swap"`
	Commission   decimal.Decimal   `json:"commission" db:"commission"`
	Fee          decimal.Decimal   `json:"fee" db:"fee"`
	OrderComment string    `json:"order_comment" db:"order_comment"`
	MagicNumber  int       `json:"magic_number" db:"magic_number"`
	CloseReason  string    `json:"close_reason" db:"close_reason"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Order struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	MTAccountID    uuid.UUID  `json:"mt_account_id" db:"mt_account_id"`
	Platform       string     `json:"platform" db:"platform"`
	Ticket         int64      `json:"ticket" db:"ticket"`
	Symbol         string     `json:"symbol" db:"symbol"`
	OrderType      int16      `json:"order_type" db:"order_type"`
	Volume         decimal.Decimal    `json:"volume" db:"volume"`
	Price          decimal.Decimal    `json:"price" db:"price"`
	StopLimitPrice decimal.Decimal    `json:"stop_limit_price" db:"stop_limit_price"`
	StopLoss       decimal.Decimal    `json:"stop_loss" db:"stop_loss"`
	TakeProfit     decimal.Decimal    `json:"take_profit" db:"take_profit"`
	Expiration     *time.Time `json:"expiration" db:"expiration"`
	ExpirationType string     `json:"expiration_type" db:"expiration_type"`
	PlacedType     string     `json:"placed_type" db:"placed_type"`
	OrderComment   string     `json:"order_comment" db:"order_comment"`
	MagicNumber    int        `json:"magic_number" db:"magic_number"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}
