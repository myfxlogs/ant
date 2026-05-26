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

// Start runs a full reconciliation then polls every 30 seconds.
// On startup, all active accounts are placed into Reconciling state (M10-BASE-B2).
func (r *ReconciliationLoop) Start(ctx context.Context) {
	r.log.Info("reconciliation: starting loop")

	// Enter all active accounts into reconciling state on startup.
	if r.gate != nil {
		accountIDs := r.hub.ActiveAccountIDs()
		r.gate.EnterAll(accountIDs)
		r.log.Info("reconciliation: entered reconciling gate", zap.Int("accounts", len(accountIDs)))
	}

	// Full reconciliation on startup
	r.reconcileAll(ctx)

	ticker := Clk.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.log.Info("reconciliation: loop stopped")
			return
		case <-ticker.C():
			r.reconcileAll(ctx)
		}
	}
}

func (r *ReconciliationLoop) reconcileAll(ctx context.Context) {
	accountIDs := r.hub.ActiveAccountIDs()
	if len(accountIDs) == 0 {
		return
	}

	r.log.Debug("reconciliation: reconciling accounts", zap.Int("count", len(accountIDs)))
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

	// 1. Fetch current broker-side state
	brokerOpened, err := exec.FetchOpenedOrders(ctx)
	if err != nil {
		return fmt.Errorf("reconciliation: fetch opened orders: %w", err)
	}
	brokerHistory, err := exec.FetchOrderHistory(ctx, Clk.Now().Add(-5*time.Minute), Clk.Now())
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

	// 2. Fetch ant-side active orders from PG
	rows, err := r.pg.Query(ctx, `
		SELECT ticket, state, order_type, symbol, volume, created_at
		FROM orders
		WHERE mt_account_id = $1::uuid
		  AND state IN ('PENDING','SUBMITTED','PARTIAL')
		ORDER BY ticket
	`, accountID)
	if err != nil {
		return fmt.Errorf("reconciliation: query ant orders: %w", err)
	}
	defer rows.Close()

	antTickets := make(map[int64]string) // ticket → state
	for rows.Next() {
		var ticket int64
		var state, orderType, symbol string
		var volume float64
		var createdAt time.Time
		if err := rows.Scan(&ticket, &state, &orderType, &symbol, &volume, &createdAt); err != nil {
			return fmt.Errorf("reconciliation: scan order row: %w", err)
		}
		antTickets[ticket] = state
	}

	// 3. Three-way comparison
	for ticket, antState := range antTickets {
		if _, exists := brokerTickets[ticket]; !exists {
			// Orphan: ant has order, broker doesn't know about it
			r.log.Warn("reconciliation: orphan order (ant has, broker missing)",
				zap.Int64("ticket", ticket), zap.String("state", antState))
		}
	}

	for ticket, brokerOrder := range brokerTickets {
		if antState, exists := antTickets[ticket]; !exists {
			// Ghost: broker has order, ant doesn't know
			r.log.Warn("reconciliation: ghost order (broker has, ant missing)",
				zap.Int64("ticket", ticket), zap.String("broker_state", orderStateToString(brokerOrder.State)))
		} else if antState != orderStateToString(brokerOrder.State) {
			// State mismatch
			r.log.Warn("reconciliation: state mismatch",
				zap.Int64("ticket", ticket),
				zap.String("ant_state", antState),
				zap.String("broker_state", orderStateToString(brokerOrder.State)))
		}
	}

		// Mark account as reconciled — unblocks PlaceOrder (M10-BASE-B2).
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
