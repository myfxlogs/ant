package model

import (
	"time"

	"github.com/google/uuid"
)

type TradeLog struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	AccountID uuid.UUID `json:"account_id" db:"account_id"`
	Action    string    `json:"action" db:"action"`
	Symbol    string    `json:"symbol" db:"symbol"`
	OrderType string    `json:"order_type" db:"order_type"`
	Volume    float64   `json:"volume" db:"volume"`
	Price     float64   `json:"price" db:"price"`
	Ticket    int64     `json:"ticket" db:"ticket"`
	Profit    float64   `json:"profit" db:"profit"`
	Message   string    `json:"message" db:"message"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type TradeRecord struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	ScheduleID   *uuid.UUID `json:"schedule_id" db:"schedule_id"`
	AccountID    uuid.UUID  `json:"account_id" db:"account_id"`
	Ticket       int64      `json:"ticket" db:"ticket"`
	Symbol       string     `json:"symbol" db:"symbol"`
	OrderType    string     `json:"order_type" db:"order_type"`
	Volume       float64    `json:"volume" db:"volume"`
	OpenPrice    float64    `json:"open_price" db:"open_price"`
	ClosePrice   float64    `json:"close_price" db:"close_price"`
	Profit       float64    `json:"profit" db:"profit"`
	Swap         float64    `json:"swap" db:"swap"`
	Commission   float64    `json:"commission" db:"commission"`
	OpenTime     time.Time  `json:"open_time" db:"open_time"`
	CloseTime    time.Time  `json:"close_time" db:"close_time"`
	StopLoss     float64    `json:"stop_loss" db:"stop_loss"`
	TakeProfit   float64    `json:"take_profit" db:"take_profit"`
	OrderComment string     `json:"order_comment" db:"order_comment"`
	MagicNumber  int        `json:"magic_number" db:"magic_number"`
	Platform     string     `json:"platform" db:"platform"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

type KlineData struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Symbol     string    `json:"symbol" db:"symbol"`
	Timeframe  string    `json:"timeframe" db:"timeframe"`
	OpenTime   time.Time `json:"open_time" db:"open_time"`
	CloseTime  time.Time `json:"close_time" db:"close_time"`
	KlineDate  time.Time `json:"kline_date" db:"kline_date"`
	OpenPrice  float64   `json:"open_price" db:"open_price"`
	HighPrice  float64   `json:"high_price" db:"high_price"`
	LowPrice   float64   `json:"low_price" db:"low_price"`
	ClosePrice float64   `json:"close_price" db:"close_price"`
	TickVolume int64     `json:"tick_volume" db:"tick_volume"`
	RealVolume float64   `json:"real_volume" db:"real_volume"`
	Spread     int       `json:"spread" db:"spread"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
