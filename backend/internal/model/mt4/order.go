package mt4

import (
	"time"

	"github.com/google/uuid"
)

type PlacedType int32

const (
	PlacedTypeClient    PlacedType = 0
	PlacedTypeExpert    PlacedType = 1
	PlacedTypeDealer    PlacedType = 2
	PlacedTypeSignal    PlacedType = 3
	PlacedTypeGateway   PlacedType = 4
	PlacedTypeMobile    PlacedType = 5
	PlacedTypeWeb       PlacedType = 6
	PlacedTypeApi       PlacedType = 7
	PlacedTypeDefault   PlacedType = 8
)

type OrderType int32

const (
	OpBuy         OrderType = 0
	OpSell        OrderType = 1
	OpBuyLimit    OrderType = 2
	OpSellLimit   OrderType = 3
	OpBuyStop     OrderType = 4
	OpSellStop    OrderType = 5
	OpBalance     OrderType = 6
	OpCredit      OrderType = 7
)

type Position struct {
	ID           uuid.UUID  `json:"id"`
	MTAccountID  uuid.UUID  `json:"mt_account_id"`
	Platform     string     `json:"platform"`
	Ticket       int32      `json:"ticket"`
	Symbol       string     `json:"symbol"`
	OrderType    OrderType  `json:"order_type"`
	Volume       float64    `json:"volume"`
	OpenPrice    float64    `json:"open_price"`
	ClosePrice   float64    `json:"close_price"`
	StopLoss     float64    `json:"stop_loss"`
	TakeProfit   float64    `json:"take_profit"`
	OpenTime     time.Time  `json:"open_time"`
	CloseTime    *time.Time `json:"close_time"`
	Expiration   *time.Time `json:"expiration"`
	MagicNumber  int32      `json:"magic_number"`
	Swap         float64    `json:"swap"`
	Commission   float64    `json:"commission"`
	OrderComment string     `json:"order_comment"`
	Profit       float64    `json:"profit"`
	RateOpen     float64    `json:"rate_open"`
	RateClose    float64    `json:"rate_close"`
	RateMargin   float64    `json:"rate_margin"`
	PlacedType   PlacedType `json:"placed_type"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Order struct {
	ID             uuid.UUID  `json:"id"`
	MTAccountID    uuid.UUID  `json:"mt_account_id"`
	Platform       string     `json:"platform"`
	Ticket         int32      `json:"ticket"`
	Symbol         string     `json:"symbol"`
	OrderType      OrderType  `json:"order_type"`
	Volume         float64    `json:"volume"`
	Price          float64    `json:"price"`
	StopLoss       float64    `json:"stop_loss"`
	TakeProfit     float64    `json:"take_profit"`
	Expiration     *time.Time `json:"expiration"`
	PlacedType     PlacedType `json:"placed_type"`
	OrderComment   string     `json:"order_comment"`
	MagicNumber    int32      `json:"magic_number"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
