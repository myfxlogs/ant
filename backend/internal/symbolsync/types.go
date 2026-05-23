// Package symbolsync pulls MT4/MT5 symbol metadata into broker_symbols.
// Adapted from AlfQ — removed RLS tenant_id, using ant's sqlx DB.
package symbolsync

import "time"

// BrokerSymbol is the unified row type for broker_symbols table.
type BrokerSymbol struct {
	BrokerID        string    `db:"broker_id"`
	SymbolRaw       string    `db:"symbol_raw"`
	Canonical       string    `db:"canonical"`
	Digits          int16     `db:"digits"`
	Point           float64   `db:"point"`
	TickSize        float64   `db:"tick_size"`
	TickValue       float64   `db:"tick_value"`
	ContractSize    float64   `db:"contract_size"`
	MinLot          float64   `db:"min_lot"`
	MaxLot          float64   `db:"max_lot"`
	LotStep         float64   `db:"lot_step"`
	MarginInitial   float64   `db:"margin_initial"`
	MarginCurrency  string    `db:"margin_currency"`
	ProfitCurrency  string    `db:"profit_currency"`
	SwapLong        float64   `db:"swap_long"`
	SwapShort       float64   `db:"swap_short"`
	SwapMode        int16     `db:"swap_mode"`
	SwapRolloverDay int16     `db:"swap_rollover_day"`
	TradeMode       int16     `db:"trade_mode"`
	Description     string    `db:"description"`
	SessionsQuote   []byte    `db:"sessions_quote"`
	SessionsTrade   []byte    `db:"sessions_trade"`
	ServerTimezone  string    `db:"server_timezone"`
	RawPayload      []byte    `db:"raw_payload"`
	Partial         bool      `db:"partial"`
	UpdatedAt       time.Time `db:"updated_at"`
}
