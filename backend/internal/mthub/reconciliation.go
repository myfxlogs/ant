package mthub

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ReconciliationLoop periodically reconciles ant-side order state with the broker
// to detect ghost orders, orphans, and state mismatches (ADR-0013).
type ReconciliationLoop struct {
	hub   *Hub
	pg    *pgxpool.Pool
	redis *goredis.Client
	log   *zap.Logger
	gate  *ReconcileGate
}

// NewReconciliationLoop creates a reconciliation loop.
func NewReconciliationLoop(hub *Hub, pg *pgxpool.Pool, redis *goredis.Client, log *zap.Logger, gate *ReconcileGate) *ReconciliationLoop {
	return &ReconciliationLoop{hub: hub, pg: pg, redis: redis, log: log, gate: gate}
}

// Start runs a full reconciliation on startup then waits for event-driven triggers.
// No polling — reconciliation is triggered by gateway connect/reconnect events
// and OnOrderUpdate stream events (ADR-0013: event-driven architecture).
func (r *ReconciliationLoop) Start(ctx context.Context) {
	r.log.Info("reconciliation: starting loop")

	if r.gate != nil {
		accountIDs := r.hub.ActiveAccountIDs()
		r.gate.EnterAll(accountIDs)
		r.log.Info("reconciliation: entered reconciling gate", zap.Int("accounts", len(accountIDs)))
	}

	r.reconcileAll(ctx)

	<-ctx.Done()
	r.log.Info("reconciliation: loop stopped")
}

// ReconcileAccount is called by event-driven triggers (gateway connect/reconnect, OnOrderUpdate).
func (r *ReconciliationLoop) ReconcileAccount(ctx context.Context, accountID string) {
	if err := r.reconcileAccount(ctx, accountID); err != nil {
		r.log.Error("reconciliation: account failed", zap.String("accountID", accountID), zap.Error(err))
	}
}

// TriggerReconcile triggers a reconciliation for a specific account.
// Safe to call from OnBrokerInfo in main.go on broker reconnect events.
func (r *ReconciliationLoop) TriggerReconcile(accountID string) {
	r.log.Info("reconciliation: triggered for account", zap.String("accountID", accountID))
	if r.gate != nil {
		r.gate.EnterReconciling(accountID)
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		r.ReconcileAccount(ctx, accountID)
	}()
}

func (r *ReconciliationLoop) reconcileAll(ctx context.Context) {
	accountIDs := r.hub.ActiveAccountIDs()
	if len(accountIDs) == 0 {
		return
	}

	for _, accountID := range accountIDs {
		if err := r.reconcileAccount(ctx, accountID); err != nil {
			r.log.Error("reconciliation: account failed", zap.String("accountID", accountID), zap.Error(err))
		}
	}
}

func (r *ReconciliationLoop) reconcileAccount(ctx context.Context, accountID string) error {
	exec := r.hub.Get(accountID)
	if exec == nil {
		return nil
	}

	// 1. Fetch broker-side state (1h window)
	brokerOpened, err := exec.FetchOpenedOrders(ctx)
	if err != nil {
		return fmt.Errorf("reconciliation: fetch opened orders: %w", err)
	}
	brokerHistory, err := exec.FetchOrderHistory(ctx, Clk.Now().Add(-1*time.Hour), Clk.Now())
	if err != nil {
		return fmt.Errorf("reconciliation: fetch order history: %w", err)
	}

	brokerTickets := make(map[int64]*OrderRecord)
	for _, o := range brokerOpened {
		if o != nil {
			brokerTickets[o.Ticket] = o
		}
	}
	for _, o := range brokerHistory {
		if o != nil {
			brokerTickets[o.Ticket] = o
		}
	}

	// 2. Fetch ant-side orders from PG (all states)
	rows, err := r.pg.Query(ctx, `
		SELECT ticket, state FROM orders WHERE mt_account_id = $1::uuid
		UNION ALL
		SELECT ticket, 'CLOSED' FROM trade_records WHERE account_id = $1::uuid
	`, accountID)
	if err != nil {
		return fmt.Errorf("reconciliation: query ant orders: %w", err)
	}
	defer rows.Close()

	antTickets := make(map[int64]string)
	for rows.Next() {
		var ticket int64
		var state string
		if err := rows.Scan(&ticket, &state); err != nil {
			continue
		}
		antTickets[ticket] = state
	}

	// 3. Compare
	var ghosts, orphans, mismatches int
	for ticket, antState := range antTickets {
		if _, exists := brokerTickets[ticket]; !exists {
			r.log.Debug("reconciliation: orphan order (ant has, broker missing)",
				zap.Int64("ticket", ticket), zap.String("state", antState))
			orphans++
		}
	}

	for ticket, brokerOrder := range brokerTickets {
		brokerState := orderStateToString(brokerOrder.State)
		if antState, exists := antTickets[ticket]; !exists {
			r.log.Debug("reconciliation: ghost order (broker has, ant missing)",
				zap.Int64("ticket", ticket), zap.String("broker_state", brokerState))
			ghosts++
		} else if antState != brokerState {
			r.log.Debug("reconciliation: state mismatch",
				zap.Int64("ticket", ticket), zap.String("ant_state", antState),
				zap.String("broker_state", brokerState))
			mismatches++
		}
	}

	if ghosts+orphans+mismatches > 0 {
		r.log.Info("reconciliation: account summary",
			zap.String("accountID", accountID),
			zap.Int("ghosts", ghosts),
			zap.Int("orphans", orphans),
			zap.Int("mismatches", mismatches),
			zap.Int("broker_orders", len(brokerTickets)),
			zap.Int("ant_orders", len(antTickets)),
		)
	}

	if r.gate != nil {
		r.gate.MarkReconciled(accountID)
	}

	return nil
}

func orderStateToString(s OrderState) string {
	switch s {
	case OrderStatePending:
		return "PENDING"
	case OrderStateOpen:
		return "OPEN"
	case OrderStateClosed:
		return "CLOSED"
	case OrderStateCancelled:
		return "CANCELLED"
	case OrderStateRejected:
		return "REJECTED"
	default:
		return "UNKNOWN"
	}
}
