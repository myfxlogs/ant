package sdk

import (
	"context"

	"github.com/shopspring/decimal"
)

// Context provides the strategy with runtime services.
// It is passed to OnInit, OnBar, OnDeinit, and optional callbacks
// (OnTick, OnTimer, OnTrade).
type Context interface {
	// ── Parameters ───────────────────────────────────────────────

	// Param reads a user-configured parameter by name.
	// Returns defaultVal if the parameter is not set.
	Param(name string, defaultVal interface{}) interface{}

	// ParamDecimal reads a numeric parameter as Decimal.
	ParamDecimal(name string, defaultVal decimal.Decimal) decimal.Decimal

	// ParamInt reads a numeric parameter as int.
	ParamInt(name string, defaultVal int) int

	// ParamString reads a string parameter.
	ParamString(name, defaultVal string) string

	// ParamBool reads a boolean parameter.
	ParamBool(name string, defaultVal bool) bool

	// ── Market data ──────────────────────────────────────────────

	// Bars returns the primary timeframe bar series.
	// Series uses MQL-style inverse indexing: [0] = current, [1] = previous.
	Bars() BarSeries

	// BarsTF returns bar series for a specific timeframe.
	// E.g. BarsTF("H4") for higher-timeframe confirmation.
	BarsTF(timeframe string) BarSeries

	// BarsForSymbol returns the bar series for a specific symbol and timeframe.
	// Used in multi-symbol strategies to access data for symbols other than the primary.
	// timeframe="" means the primary timeframe.
	// Returns an empty series if the symbol is not available.
	BarsForSymbol(symbol, timeframe string) BarSeries

	// Symbol returns the primary trading symbol.
	Symbol() string

	// Timeframe returns the primary timeframe string ("M5", "H1", etc.).
	Timeframe() string

	// Point returns the point size (e.g. 0.00001 for 5-digit FX).
	Point() decimal.Decimal

	// Pip returns the pip size (usually 10 * point).
	Pip() decimal.Decimal

	// Digits returns the number of decimal places for prices.
	Digits() int32

	// Ask returns the current ask price.
	Ask() decimal.Decimal

	// Bid returns the current bid price.
	Bid() decimal.Decimal

	// Spread returns the current spread (Ask - Bid) in price units.
	Spread() decimal.Decimal

	// ── Account ──────────────────────────────────────────────────

	// Account returns the current account state.
	Account() AccountInfo

	// AccountMode returns the account mode (hedging or netting).
	Mode() AccountMode

	// ── Services ─────────────────────────────────────────────────

	// Broker returns the trading interface.
	Broker() Broker

	// Indicators returns the indicator computation interface.
	Indicators() IndicatorSet

	// ── Lifecycle ────────────────────────────────────────────────

	// SetTimer starts a periodic timer. OnTimer will be called every n seconds.
	SetTimer(seconds int)

	// KillTimer stops the periodic timer.
	KillTimer()

	// Log writes a message to the strategy log.
	Log(msg string)

	// ServerTime returns the current broker server time (UTC).
	ServerTime() int64 // unix_ms

	// GoContext returns the Go context.Context associated with the current
	// event dispatch. Used by the Bytecode VM for cancellation/timeout checks.
	// Returns context.Background() if no context is set (e.g. backtest).
	GoContext() context.Context
}
