package mt5

import (
	"time"

	"github.com/google/uuid"
)

type ProfitUpdate struct {
	AccountID   uuid.UUID `json:"account_id"`
	Balance     float64   `json:"balance"`
	Equity      float64   `json:"equity"`
	Margin      float64   `json:"margin"`
	FreeMargin  float64   `json:"free_margin"`
	Profit      float64   `json:"profit"`
	MarginLevel float64   `json:"margin_level"`
	Orders      []*OrderProfitItem `json:"orders"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OrderProfitItem struct {
	Ticket       int64   `json:"ticket"`
	Symbol       string  `json:"symbol"`
	Profit       float64 `json:"profit"`
	Volume       float64 `json:"volume"`
	CurrentPrice float64 `json:"current_price"`
}
