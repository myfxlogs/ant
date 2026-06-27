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
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"
	natsgo "github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
)

// sanitizeUTF8 replaces invalid UTF-8 bytes with the replacement character.
// Proto3 requires all string fields to be valid UTF-8; MT4 gateways
// occasionally emit non-UTF-8 bytes (e.g. Windows-1252 in comments).
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "�")
}

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
	Volume            decimal.Decimal        `json:"volume"`
	Price             decimal.Decimal        `json:"price"`
	StopLoss          decimal.Decimal        `json:"stop_loss"`
	TakeProfit        decimal.Decimal        `json:"take_profit"`
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

	payload, err := proto.Marshal(eventToPayload(ev))
	if err != nil {
		// Proto3 requires valid UTF-8 in all string fields. MT4 gateways
		// occasionally emit non-UTF-8 bytes in broker fields (comments,
		// raw symbol names, etc.). Sanitize and retry once.
		ev.Canonical = sanitizeUTF8(ev.Canonical)
		ev.Broker = sanitizeUTF8(ev.Broker)
		ev.ClientID = sanitizeUTF8(ev.ClientID)
		ev.CostBreakdownJSON = sanitizeUTF8(ev.CostBreakdownJSON)
		payload, err = proto.Marshal(eventToPayload(ev))
		if err != nil {
			return fmt.Errorf("trade_event_store: marshal: %w", err)
		}
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
func eventToPayload(ev *TradeEvent) *antv1.TradeEventPayload {
	return &antv1.TradeEventPayload{
		EventId: ev.EventID, EventType: string(ev.EventType),
		AccountId: ev.AccountID, UserId: ev.UserID, Broker: ev.Broker,
		Ticket: ev.Ticket, ClientId: ev.ClientID, Canonical: ev.Canonical,
		Side: ev.Side, OrderType: ev.OrderType,
		Volume: ev.Volume.String(), Price: ev.Price.String(),
		StopLoss: ev.StopLoss.String(), TakeProfit: ev.TakeProfit.String(),
		FromState: ev.FromState, ToState: ev.ToState,
		CostJson: ev.CostBreakdownJSON, TsUnixMs: ev.Timestamp.UnixMilli(),
	}
}
