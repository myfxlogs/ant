package connect

import (
	"time"

	"github.com/google/uuid"
)

type StreamEvent struct {
	Type        string    `json:"type"`
	AccountID   uuid.UUID `json:"account_id"`
	Timestamp   time.Time `json:"timestamp"`
	Platform    string    `json:"platform"`
	QuoteEvent  *QuoteEvent  `json:"quote_event,omitempty"`
	OrderEvent  *OrderEvent  `json:"order_event,omitempty"`
	ProfitEvent *ProfitEvent `json:"profit_event,omitempty"`
	AccountStatus *AccountStatusEvent `json:"account_status,omitempty"`
}

type QuoteEvent struct {
	Symbol    string  `json:"symbol"`
	Bid       float64 `json:"bid"`
	Ask       float64 `json:"ask"`
	High      float64 `json:"high,omitempty"`
	Low       float64 `json:"low,omitempty"`
	Last      float64 `json:"last,omitempty"`
	Volume    float64 `json:"volume,omitempty"`
	Timestamp string  `json:"timestamp"`
}

type OrderEvent struct {
	Ticket       int64   `json:"ticket"`
	Symbol       string  `json:"symbol"`
	OrderType    int32   `json:"order_type"`
	Volume       float64 `json:"volume"`
	OpenPrice    float64 `json:"open_price"`
	CurrentPrice float64 `json:"current_price"`
	Profit       float64 `json:"profit"`
	Action       string  `json:"action"`
	StopLoss     float64 `json:"stop_loss,omitempty"`
	TakeProfit   float64 `json:"take_profit,omitempty"`
	ClosePrice   float64 `json:"close_price,omitempty"`
	OpenTime     int64   `json:"open_time"`
	CloseTime    int64   `json:"close_time,omitempty"`
	Swap         float64 `json:"swap,omitempty"`
	Commission   float64 `json:"commission,omitempty"`
	Comment      string  `json:"comment,omitempty"`
}

type ProfitEvent struct {
	AccountID   uuid.UUID `json:"account_id"`
	Balance     float64   `json:"balance"`
	Credit      float64   `json:"credit"`
	Profit      float64   `json:"profit"`
	Equity      float64   `json:"equity"`
	Margin      float64   `json:"margin"`
	FreeMargin  float64   `json:"free_margin"`
	MarginLevel float64   `json:"margin_level"`
	Orders      []*OrderProfitItem `json:"orders"`
	Platform    string    `json:"platform"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OrderProfitItem struct {
	Ticket       int64   `json:"ticket"`
	Symbol       string  `json:"symbol"`
	Profit       float64 `json:"profit"`
	Volume       float64 `json:"volume"`
	CurrentPrice float64 `json:"current_price"`
}

type AccountStatusEvent struct {
	AccountID string `json:"account_id"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}
