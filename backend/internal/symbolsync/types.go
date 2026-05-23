// Package symbolsync pulls MT4/MT5 symbol metadata into broker_symbols.
// Adapted from AlfQ — removed RLS tenant_id, using ant's sqlx DB.
package symbolsync

import "time"

// BrokerSymbol is the unified row type for broker_symbols table.
type BrokerSymbol struct {
	BrokerID   string  `db:"broker_id"`
	SymbolRaw  string  `db:"symbol_raw"`
	Canonical  string  `db:"canonical"`
	Digits     int16   `db:"digits"`
	Point      float64 `db:"point"`
	TickSize   float64 `db:"tick_size"`
	MinLot     float64 `db:"min_lot"`
	MaxLot     float64 `db:"max_lot"`
	LotStep    float64 `db:"lot_step"`
	SwapLong   float64 `db:"swap_long"`
	SwapShort  float64 `db:"swap_short"`
	TradeMode  int16   `db:"trade_mode"` // 0=disabled 1=long_only 2=short_only 3=full
	Description string `db:"description"`
	// SessionsQuote  []byte    `db:"sessions_quote"`  // JSONB — deferred
	// SessionsTrade  []byte    `db:"sessions_trade"`  // JSONB — deferred
	RawPayload []byte `db:"raw_payload"` // JSONB
	Partial    bool   `db:"partial"`
	UpdatedAt  time.Time `db:"updated_at"`
}
