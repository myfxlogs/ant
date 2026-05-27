// Package mthub provides the event-sourced trade ledger (M11-11, M10-BASE-B4).
//
// All order state changes are first written to NATS JetStream (Tier-0 append-only)
// and then projected to PG. This ensures a complete, replayable audit trail.
//
// Subject hierarchy: oms.order.<account_id>
// Event schema aligned with NautilusTrader OrderEvent hierarchy.

package mthub

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	natsgo "github.com/nats-io/nats.go"

	"anttrader/internal/interceptor"
)

// TradeEventType enumerates the kinds of order lifecycle events.
type TradeEventType string

const (
	TradeEventOrderCreated       TradeEventType = "ORDER_CREATED"
	TradeEventOrderSubmitted     TradeEventType = "ORDER_SUBMITTED"
	TradeEventOrderAccepted      TradeEventType = "ORDER_ACCEPTED"
	TradeEventOrderRejected      TradeEventType = "ORDER_REJECTED"
	TradeEventOrderWorking       TradeEventType = "ORDER_WORKING"
	TradeEventOrderPartiallyFilled TradeEventType = "ORDER_PARTIALLY_FILLED"
	TradeEventOrderFilled        TradeEventType = "ORDER_FILLED"
	TradeEventOrderCancelled     TradeEventType = "ORDER_CANCELLED"
	TradeEventOrderExpired       TradeEventType = "ORDER_EXPIRED"
	TradeEventOrderFailed        TradeEventType = "ORDER_FAILED"
	TradeEventOrderStateChanged  TradeEventType = "ORDER_STATE_CHANGED"
	TradeEventOrderRequoted      TradeEventType = "ORDER_REQUOTED"
	TradeEventOrderSlippageReject TradeEventType = "ORDER_SLIPPAGE_REJECTED"
	TradeEventOrderUnknown       TradeEventType = "ORDER_UNKNOWN"
	TradeEventOrderMarginCall    TradeEventType = "ORDER_MARGIN_CALL"
)

// TradeEvent is the canonical order event written to the append-only ledger.
// Schema aligned with NautilusTrader OrderEvent.
type TradeEvent struct {
	EventID           string         `json:"event_id"`
	EventType         TradeEventType `json:"event_type"`
	AccountID         string         `json:"account_id"`
	UserID            string         `json:"user_id"`
	Broker            string         `json:"broker"`
	Ticket            int64          `json:"ticket"`
	ClientID          string         `json:"client_id"`
	Canonical         string         `json:"canonical"`
	Side              string         `json:"side"`
	OrderType         string         `json:"order_type"`
	Volume            float64        `json:"volume"`
	Price             float64        `json:"price"`
	StopLoss          float64        `json:"stop_loss"`
	TakeProfit        float64        `json:"take_profit"`
	FromState         string         `json:"from_state"`
	ToState           string         `json:"to_state"`
	Timestamp         time.Time      `json:"timestamp"`
	ArrivedUnixMs     int64          `json:"arrived_unix_ms"`
	Version           int64          `json:"version"`
	CostBreakdownJSON string         `json:"cost_breakdown,omitempty"` // M10-BASE-D2: pre-trade cost estimate
}

// TradeEventStore publishes order events to NATS JetStream (Tier-0).
type TradeEventStore struct {
	js natsgo.JetStreamContext
}

// NewTradeEventStore creates a store backed by a JetStream context.
// js may be nil (events are silently dropped — for testing).
func NewTradeEventStore(js natsgo.JetStreamContext) *TradeEventStore {
	return &TradeEventStore{js: js}
}

// Publish writes an order event to the append-only NATS JetStream.
// This must be called BEFORE the corresponding PG state change.
func (s *TradeEventStore) Publish(ctx context.Context, ev *TradeEvent) error {
	if s.js == nil {
		return nil
	}

	ev.ArrivedUnixMs = Clk.Now().UnixMilli()

	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("trade_event_store: marshal: %w", err)
	}

	subject := fmt.Sprintf("oms.order.%s", ev.AccountID)

	// Idempotency key: event_id ensures at-least-once delivery dedup.
	msg := &natsgo.Msg{
		Subject: subject,
		Data:    payload,
		Header: natsgo.Header{
			"Nats-Msg-Id": []string{ev.EventID},
		},
	}

	interceptor.InjectNATSTraceHeaders(ctx, msg.Header)
	if _, err := s.js.PublishMsg(msg); err != nil {
		return fmt.Errorf("trade_event_store: publish: %w", err)
	}
	return nil
}
