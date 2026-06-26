package sdk

import "github.com/shopspring/decimal"

// Strategy is the interface every generated strategy implements.
//
// The runtime calls OnInit once at startup, then OnBar on every new bar
// of the primary timeframe. OnDeinit is called on shutdown.
//
// A strategy may also implement optional interfaces:
//
//	OnTrader  — called after any order fill or position close
//	OnTimer   — called periodically (if EventSetTimer was called in OnInit)
type Strategy interface {
	// OnInit is called once when the strategy starts.
	// Use ctx.Param() to read user-configured parameters.
	// Return an error to abort startup.
	OnInit(ctx Context) error

	// OnBar is called when a new bar closes on the primary timeframe.
	// timeframe is the period string ("M5", "H1", etc.).
	// Return a Signal to place orders, or nil for no action.
	OnBar(ctx Context, timeframe string) (*Signal, error)

	// OnDeinit is called when the strategy stops.
	// reason is "user_stop", "kill_switch", or "schedule_end".
	OnDeinit(ctx Context, reason string) error
}

// Signal is returned by OnBar to request trade actions.
// Only non-nil fields are acted upon.
type Signal struct {
	Action       SignalAction
	Symbol       string          // defaults to primary symbol if empty
	Volume       decimal.Decimal // 0 = use default volume
	Price        decimal.Decimal // 0 = market price
	StopLoss     decimal.Decimal
	TakeProfit   decimal.Decimal
	Deviation    int32
	Magic        int32
	Comment      string
	FillPolicy   FillPolicy
	OrderTicket  int64           // for modify/close/cancel: which order to act on
}

// SignalAction tells the runtime what to do.
type SignalAction int8

const (
	ActionNone      SignalAction = iota
	ActionBuy                    // market buy
	ActionSell                   // market sell
	ActionBuyLimit               // pending buy limit
	ActionSellLimit              // pending sell limit
	ActionBuyStop                // pending buy stop
	ActionSellStop               // pending sell stop
	ActionClose                  // position_close by ticket
	ActionModify                 // position_modify SL/TP
	ActionCancel                 // order_delete by ticket
	ActionCloseAll               // close all positions for this magic
	ActionCancelAll              // cancel all pending orders for this magic
)

// ── Optional interfaces ─────────────────────────────────────────────

// OnTrader is implemented by strategies that need trade-event callbacks.
// The runtime calls OnTrade after any order is filled or position closed.
type OnTrader interface {
	OnTrade(ctx Context) error
}

// OnTimer is implemented by strategies that use periodic timers.
// Call ctx.SetTimer(seconds) in OnInit to enable.
type OnTimer interface {
	OnTimer(ctx Context) error
}
