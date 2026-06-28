package sdk

import "github.com/shopspring/decimal"

// Strategy is the interface every generated strategy implements.
//
// The runtime calls OnInit once at startup, then dispatches events based on
// which optional interfaces the strategy implements:
//
//	BarStrategy   — OnBar is called when a new bar closes
//	TickStrategy  — OnTick is called on every price update (Bid/Ask)
//	TimerStrategy — OnTimer is called when EventSetTimer fires
//	TradeStrategy — OnTrade is called after order fill/close/modify
//
// OnDeinit is called on shutdown.
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

// Signal is returned by OnBar/OnTick/OnTimer/OnTrade to request trade actions.
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

// ── Optional execution model interfaces ─────────────────────────────
// Strategies implement the interfaces whose events they need.
// The runner detects implemented interfaces and subscribes to the
// corresponding data streams.

// TickStrategy is implemented by strategies that need per-tick execution.
// OnTick is called on every price update (Bid/Ask) for the primary symbol.
// Used by scalping and market-making EAs.
type TickStrategy interface {
	OnTick(ctx Context, bid, ask decimal.Decimal) (*Signal, error)
}

// TimerStrategy is implemented by strategies that use periodic timers.
// OnTimer is called every n seconds, as configured by ctx.SetTimer(n) in OnInit.
type TimerStrategy interface {
	OnTimer(ctx Context) (*Signal, error)
}

// TradeStrategy is implemented by strategies that need trade event callbacks.
// OnTrade is called after an order is filled, closed, or modified.
// Used by post-fill pyramiding, hedge management, and trailing-stop EAs.
type TradeStrategy interface {
	OnTrade(ctx Context, event TradeEvent) (*Signal, error)
}

// TradeEvent carries information about a completed trade action.
type TradeEvent struct {
	Ticket     int64
	Symbol     string
	EventType  TradeEventType // fill, close, modify, cancel
	Side       PositionSide
	Volume     decimal.Decimal
	Price      decimal.Decimal
	StopLoss   decimal.Decimal
	TakeProfit decimal.Decimal
	Profit     decimal.Decimal
	Commission decimal.Decimal
	Swap       decimal.Decimal
}

// TradeEventType classifies the trade event.
type TradeEventType int8

const (
	TradeFilled  TradeEventType = iota // order executed
	TradeClosed                        // position closed
	TradeModified                      // SL/TP modified
	TradeCancelled                     // pending order cancelled
)
