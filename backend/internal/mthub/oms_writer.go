// Package mthub provides OMS state writer (M11-2, ADR-0012).
//
// OmsWriter records order state transitions in PG and publishes events to NATS.
// State constants mirror oms.OrderState to avoid a circular dependency
// (oms/adapter_mt.go imports mthub for OrderExecutor).
//
// Order lifecycle (15 states):
//
//	NEW → VALIDATED → RISK_APPROVED → SUBMITTED
//	                                    ├── WORKING → PARTIALLY_FILLED → FILLED
//	                                    ├── FILLED
//	                                    ├── CANCELLED
//	                                    ├── EXPIRED
//	                                    ├── FAILED
//	                                    └── ...
package mthub

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OMSState represents an order state in the 15-state machine.
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
	OMSStateRejected        OMSState = "REJECTED"
	OMSStateFailed          OMSState = "FAILED"
	OMSStateExpired         OMSState = "EXPIRED"
)

// isValidOMSTransition validates state transitions (mirrors oms.isValid).
func isValidOMSTransition(current, next OMSState) bool {
	transitions := map[OMSState][]OMSState{
		OMSStateNew:          {OMSStateValidated},
		OMSStateValidated:    {OMSStateRiskApproved, OMSStateRejected},
		OMSStateRiskApproved: {OMSStateSubmitted, OMSStateRejected},
		OMSStateSubmitted:    {OMSStateWorking, OMSStatePartiallyFilled, OMSStateFilled, OMSStateCancelled, OMSStateExpired, OMSStateFailed},
		OMSStateWorking:      {OMSStatePartiallyFilled, OMSStateFilled, OMSStateCancelled, OMSStateExpired, OMSStateFailed},
		OMSStatePartiallyFilled: {OMSStatePartiallyFilled, OMSStateFilled, OMSStateCancelled, OMSStateExpired, OMSStateFailed},
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
	pool  *pgxpool.Pool
	store *TradeEventStore // may be nil
}

// NewOmsWriter creates a state writer backed by PG.
func NewOmsWriter(pool *pgxpool.Pool, store *TradeEventStore) *OmsWriter {
	return &OmsWriter{pool: pool, store: store}
}

// InsertOrder inserts a new order with state=NEW.
// Uses ON CONFLICT DO NOTHING to handle idempotent re-insertion.
func (w *OmsWriter) InsertOrder(ctx context.Context, orderID, accountID, symbol string, orderType int16, volume, price, stopLoss, takeProfit float64) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO orders (id, mt_account_id, platform, ticket, symbol, order_type, volume, price, stop_loss, take_profit, state)
		VALUES ($1, $2, 'MT5', '0', $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING
	`, orderID, accountID, symbol, orderType, volume, price, stopLoss, takeProfit, string(OMSStateNew))
	if err != nil {
		return fmt.Errorf("oms insert order: %w", err)
	}
	return nil
}

// Transition validates and persists a state transition to PG.
func (w *OmsWriter) Transition(ctx context.Context, orderID string, current, next OMSState) error {
	if !isValidOMSTransition(current, next) {
		return fmt.Errorf("oms: invalid transition %s → %s", current, next)
	}
	_, err := w.pool.Exec(ctx,
		`UPDATE orders SET state = $1, updated_at = now() WHERE id = $2`,
		string(next), orderID)
	if err != nil {
		return fmt.Errorf("oms update state %s: %w", orderID, err)
	}

	// Publish state transition event to NATS.
	if w.store != nil {
		ev := &TradeEvent{
			EventID:   fmt.Sprintf("oms-%s-%s", orderID, next),
			EventType: TradeEventOrderCreated,
			AccountID: accountIDFromOrderID(orderID),
			ToState:   string(next),
			FromState: string(current),
			Timestamp: Clk.Now(),
			Version:   1,
		}
		_ = w.store.Publish(ctx, ev)
	}
	return nil
}

// IdempotencyKey generates a deterministic order ID from account + client for idempotent insertion.
func IdempotencyKey(accountID, clientID string) string {
	return fmt.Sprintf("ord-%s-%s", accountID, clientID)
}

// accountIDFromOrderID extracts the account ID from the idempotent order key.
func accountIDFromOrderID(orderID string) string {
	// format: "ord-<accountID>-<clientID>"
	for i := 4; i < len(orderID); i++ {
		if orderID[i] == '-' {
			return orderID[4:i]
		}
	}
	return orderID
}
