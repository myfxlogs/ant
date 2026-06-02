package model

import (
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
	Volume       float64   `json:"volume" db:"volume"`
	OpenPrice    float64   `json:"open_price" db:"open_price"`
	CurrentPrice float64   `json:"current_price" db:"current_price"`
	StopLoss     float64   `json:"stop_loss" db:"stop_loss"`
	TakeProfit   float64   `json:"take_profit" db:"take_profit"`
	OpenTime     time.Time `json:"open_time" db:"open_time"`
	Profit       float64   `json:"profit" db:"profit"`
	Swap         float64   `json:"swap" db:"swap"`
	Commission   float64   `json:"commission" db:"commission"`
	Fee          float64   `json:"fee" db:"fee"`
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
	Volume         float64    `json:"volume" db:"volume"`
	Price          float64    `json:"price" db:"price"`
	StopLimitPrice float64    `json:"stop_limit_price" db:"stop_limit_price"`
	StopLoss       float64    `json:"stop_loss" db:"stop_loss"`
	TakeProfit     float64    `json:"take_profit" db:"take_profit"`
	Expiration     *time.Time `json:"expiration" db:"expiration"`
	ExpirationType string     `json:"expiration_type" db:"expiration_type"`
	PlacedType     string     `json:"placed_type" db:"placed_type"`
	OrderComment   string     `json:"order_comment" db:"order_comment"`
	MagicNumber    int        `json:"magic_number" db:"magic_number"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}
