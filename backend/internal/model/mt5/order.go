package mt5

import (
	"time"

	"github.com/google/uuid"
)

type PlacedType int32

const (
	PlacedTypeManually        PlacedType = 0
	PlacedTypeMobile          PlacedType = 16
	PlacedTypeWeb             PlacedType = 17
	PlacedTypeByExpert        PlacedType = 1
	PlacedTypeOnSL            PlacedType = 3
	PlacedTypeOnTP            PlacedType = 4
	PlacedTypeOnStopOut       PlacedType = 5
	PlacedTypeOnRollover      PlacedType = 6
	PlacedTypeOnVmargin       PlacedType = 8
	PlacedTypeOnSplit         PlacedType = 18
	PlacedTypeByDealer        PlacedType = 2
	PlacedTypeGateway         PlacedType = 9
	PlacedTypeSignal          PlacedType = 10
	PlacedTypeSettlement      PlacedType = 11
	PlacedTypeTransfer        PlacedType = 12
	PlacedTypeSync            PlacedType = 13
	PlacedTypeExternalService PlacedType = 14
	PlacedTypeMigration       PlacedType = 15
	PlacedTypeDefault         PlacedType = 20
)

type OrderType int32

const (
	OrderTypeBuy       OrderType = 0
	OrderTypeSell      OrderType = 1
	OrderTypeBuyLimit  OrderType = 2
	OrderTypeSellLimit OrderType = 3
	OrderTypeBuyStop   OrderType = 4
	OrderTypeSellStop  OrderType = 5
)

type DealType int32

const (
	DealTypeBuy    DealType = 0
	DealTypeSell   DealType = 1
	DealTypeBalance DealType = 2
	DealTypeCredit DealType = 3
)

type OrderState int32

const (
	OrderStateActive    OrderState = 0
	OrderStateFinished  OrderState = 1
	OrderStatePartially OrderState = 2
)

type Position struct {
	ID                uuid.UUID  `json:"id"`
	MTAccountID       uuid.UUID  `json:"mt_account_id"`
	Platform          string     `json:"platform"`
	Ticket            int64      `json:"ticket"`
	Symbol            string     `json:"symbol"`
	OrderType         OrderType  `json:"order_type"`
	DealType          DealType   `json:"deal_type"`
	Volume            float64    `json:"volume"`
	OpenPrice         float64    `json:"open_price"`
	ClosePrice        float64    `json:"close_price"`
	StopLoss          float64    `json:"stop_loss"`
	TakeProfit        float64    `json:"take_profit"`
	OpenTime          time.Time  `json:"open_time"`
	CloseTime         *time.Time `json:"close_time"`
	ExpirationTime    *time.Time `json:"expiration_time"`
	MagicNumber       int64      `json:"magic_number"`
	Swap              float64    `json:"swap"`
	Commission        float64    `json:"commission"`
	Fee               float64    `json:"fee"`
	OrderComment      string     `json:"order_comment"`
	Profit            float64    `json:"profit"`
	ProfitRate        float64    `json:"profit_rate"`
	PlacedType        PlacedType `json:"placed_type"`
	State             OrderState `json:"state"`
	ContractSize      float64    `json:"contract_size"`
	CloseVolume       float64    `json:"close_volume"`
	CloseLots         float64    `json:"close_lots"`
	CloseComment      string     `json:"close_comment"`
	StopLimitPrice    float64    `json:"stop_limit_price"`
	ExpertId          int64      `json:"expert_id"`
	ExchangeInternalIn  *DealInternal `json:"exchange_internal_in"`
	ExchangeInternalOut *DealInternal `json:"exchange_internal_out"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type DealInternal struct {
	Deal    int64   `json:"deal"`
	Profit  float64 `json:"profit"`
	Commission float64 `json:"commission"`
	Swap    float64 `json:"swap"`
}

type Order struct {
	ID                uuid.UUID  `json:"id"`
	MTAccountID       uuid.UUID  `json:"mt_account_id"`
	Platform          string     `json:"platform"`
	Ticket            int64      `json:"ticket"`
	Symbol            string     `json:"symbol"`
	OrderType         OrderType  `json:"order_type"`
	Volume            float64    `json:"volume"`
	OpenPrice         float64    `json:"open_price"`
	ClosePrice        float64    `json:"close_price"`
	StopLoss          float64    `json:"stop_loss"`
	TakeProfit        float64    `json:"take_profit"`
	OpenTime          time.Time  `json:"open_time"`
	CloseTime         *time.Time `json:"close_time"`
	ExpirationTime    *time.Time `json:"expiration_time"`
	ExpirationType    string     `json:"expiration_type"`
	FillPolicy        string     `json:"fill_policy"`
	PlacedType        PlacedType `json:"placed_type"`
	OrderComment      string     `json:"order_comment"`
	MagicNumber       int64      `json:"magic_number"`
	ContractSize      float64    `json:"contract_size"`
	StopLimitPrice    float64    `json:"stop_limit_price"`
	ExpertId          int64      `json:"expert_id"`
	ExchangeInternalIn  *DealInternal `json:"exchange_internal_in"`
	ExchangeInternalOut *DealInternal `json:"exchange_internal_out"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
