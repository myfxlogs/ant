// Package mthub provides OMS state writer (M11-2, ADR-0012).
//
// OmsWriter records order state transitions in PG and publishes events to NATS.
// State constants mirror oms.OrderState to avoid a circular dependency
// (oms/adapter_mt.go imports mthub for OrderExecutor).
//
// Order lifecycle (16 states):
//
//	NEW → VALIDATED → RISK_APPROVED → SUBMITTED
//	                                    ├── WORKING → PARTIALLY_FILLED → FILLED
//	                                    ├── FILLED
//	                                    ├── CANCELLED
//	                                    ├── EXPIRED
//	                                    ├── FAILED
//	                                    ├── UNKNOWN (timeout: 30s no response)
//	                                    ├── REQUOTED
//	                                    ├── SLIPPAGE_REJECTED
//	                                    └── MARGIN_CALL
//	VALIDATED → REJECTED
//	RISK_APPROVED → REJECTED
//	RISK_APPROVED → FAILED
//	UNKNOWN → RECONCILING
//	RECONCILING → WORKING | FILLED | CANCELLED | FAILED
package mthub

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OMSState represents an order state in the 16-state machine.
type OMSState string

const (
	OMSStateNew             OMSState = "NEW"
	OMSStateValidated       OMSState = "VALIDATED"
	OMSStateRiskApproved    OMSState = "RISK_APPROVED"
	OMSStateSubmitted       OMSState = "SUBMITTED"
	OMSStateWorking         OMSState = "WORKING"
	OMSStatePartiallyFilled OMSState = "PARTIALLY_FILLED"
	OMSStateFilled          OMSState = "FILLED"
	OMSStateCancelled       OMSState = "CANCELLED"
	OMSStateRejected         OMSState = "REJECTED"
	OMSStateFailed           OMSState = "FAILED"
	OMSStateExpired          OMSState = "EXPIRED"
	OMSStateRequoted         OMSState = "REQUOTED"
	OMSStateSlippageRejected OMSState = "SLIPPAGE_REJECTED"
	OMSStateUnknown          OMSState = "UNKNOWN"
	OMSStateReconciling      OMSState = "RECONCILING"
	OMSStateMarginCall       OMSState = "MARGIN_CALL"
)

// isValidOMSTransition validates state transitions (mirrors oms.isValid).
func isValidOMSTransition(current, next OMSState) bool {
	transitions := map[OMSState][]OMSState{
		OMSStateNew:          {OMSStateValidated},
		OMSStateValidated:    {OMSStateRiskApproved, OMSStateRejected},
		OMSStateRiskApproved: {OMSStateSubmitted, OMSStateRejected, OMSStateFailed},
		OMSStateSubmitted:    {OMSStateWorking, OMSStatePartiallyFilled, OMSStateFilled, OMSStateCancelled, OMSStateExpired, OMSStateFailed, OMSStateUnknown, OMSStateRequoted, OMSStateSlippageRejected, OMSStateMarginCall},
		OMSStateWorking:      {OMSStatePartiallyFilled, OMSStateFilled, OMSStateCancelled, OMSStateExpired, OMSStateFailed, OMSStateRequoted},
		OMSStatePartiallyFilled: {OMSStatePartiallyFilled, OMSStateFilled, OMSStateCancelled, OMSStateExpired, OMSStateFailed},
		OMSStateRequoted:         {OMSStateRiskApproved, OMSStateCancelled, OMSStateExpired},
		OMSStateSlippageRejected: {OMSStateRiskApproved, OMSStateCancelled, OMSStateExpired},
		OMSStateUnknown:          {OMSStateReconciling, OMSStateWorking, OMSStateFilled, OMSStateCancelled, OMSStateFailed, OMSStateExpired},
		OMSStateReconciling:      {OMSStateWorking, OMSStatePartiallyFilled, OMSStateFilled, OMSStateCancelled, OMSStateFailed, OMSStateExpired},
		OMSStateMarginCall:       {OMSStateRiskApproved, OMSStateCancelled, OMSStateExpired, OMSStateFailed},
	}
	allowed, ok := transitions[current]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}

// OmsWriter records order lifecycle state transitions in PG.
type OmsWriter struct {
	pool             *pgxpool.Pool
	store            *TradeEventStore // may be nil
	orderEventBroker *OrderEventBroker
}

// NewOmsWriter creates a state writer backed by PG.
func NewOmsWriter(pool *pgxpool.Pool, store *TradeEventStore) *OmsWriter {
	return &OmsWriter{pool: pool, store: store}
}

// SetOrderEventBroker wires the OrderEventBroker so that state transitions
// publish events to active subscribers via SubscribeOrderEvents.
func (w *OmsWriter) SetOrderEventBroker(b *OrderEventBroker) {
	w.orderEventBroker = b
}

// InsertOrder inserts a new order with state=NEW.
// Uses ON CONFLICT DO NOTHING to handle idempotent re-insertion.
// platform must be "MT4" or "MT5".
func (w *OmsWriter) InsertOrder(ctx context.Context, orderID, accountID, platform, symbol string, orderType int16, volume, price, stopLoss, takeProfit float64) error {
	if platform == "" {
		platform = "MT5"
	}
	_, err := w.pool.Exec(ctx, `
		INSERT INTO orders (id, mt_account_id, platform, ticket, symbol, order_type, volume, price, stop_loss, take_profit, state)
		VALUES ($1, $2, $3, '0', $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO NOTHING
	`, orderID, accountID, platform, symbol, orderType, volume, price, stopLoss, takeProfit, string(OMSStateNew))
	if err != nil {
		return fmt.Errorf("oms insert order: %w", err)
	}
	return nil
}


// Transition validates and persists a state transition to PG.
func (w *OmsWriter) Transition(ctx context.Context, orderID, accountID string, current, next OMSState) error {
	if !isValidOMSTransition(current, next) {
		return fmt.Errorf("oms: invalid transition %s → %s", current, next)
	}
	tag, err := w.pool.Exec(ctx,
		`UPDATE orders SET state = $1, updated_at = now() WHERE id = $2 AND state = $3`,
		string(next), orderID, string(current))
	if err != nil {
		return fmt.Errorf("oms update state %s: %w", orderID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("oms: concurrent conflict on %s (expected state %s)", orderID, current)
	}

	// Publish state transition event to NATS.
	if w.store != nil {
		ev := &TradeEvent{
			EventID:   fmt.Sprintf("oms-%s-%s", orderID, next),
			EventType: TradeEventOrderCreated,
			AccountID: accountID,
			ToState:   string(next),
			FromState: string(current),
			Timestamp: Clk.Now(),
			Version:   1,
		}
		_ = w.store.Publish(ctx, ev)
	}

	// Publish event to OrderEventBroker subscribers (H13: wire SubscribeOrderEvents).
	if w.orderEventBroker != nil {
		oev := &OrderEvent{
			AccountID: accountID,
			EventType: fmt.Sprintf("%s→%s", string(current), string(next)),
			Timestamp: Clk.Now(),
		}
		w.orderEventBroker.PublishEvent(oev.AccountID, oev)
	}
	return nil
}

// IdempotencyKey generates a deterministic UUID for idempotent order insertion.
// Uses MD5 namespace hashing so same account+clientID always resolves to the same UUID.
// When clientID is empty (frontend didn't send one), a random UUID is used —
// no idempotency guarantee without a client-provided key.
func IdempotencyKey(accountID, clientID string) string {
	if clientID == "" {
		return uuid.New().String()
	}
	return uuid.NewMD5(uuid.NameSpaceOID, []byte(fmt.Sprintf("ord-%s-%s", accountID, clientID))).String()
}

