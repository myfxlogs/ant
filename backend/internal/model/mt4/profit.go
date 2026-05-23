package mt4

import (
	"time"

	"github.com/google/uuid"
)

type ProfitUpdate struct {
	AccountID     uuid.UUID `json:"account_id"`
	Balance       float64   `json:"balance"`
	Credit        float64   `json:"credit"`
	Profit        float64   `json:"profit"`
	Equity        float64   `json:"equity"`
	Margin        float64   `json:"margin"`
	FreeMargin    float64   `json:"free_margin"`
	MarginLevel   float64   `json:"margin_level"`
	Leverage      int32     `json:"leverage"`
	Currency      string    `json:"currency"`
	Type          AccountType `json:"type"`
	IsInvestor    bool      `json:"is_investor"`
	Orders        []*OrderProfitItem `json:"orders"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type OrderProfitItem struct {
	Ticket       int32   `json:"ticket"`
	Symbol       string  `json:"symbol"`
	Profit       float64 `json:"profit"`
	Volume       float64 `json:"volume"`
	CurrentPrice float64 `json:"current_price"`
}
