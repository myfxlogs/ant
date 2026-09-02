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
	svc   *MtHubService // optional: used to repair stuck OMS orders (SUBMITTED→FILLED/CANCELLED/FAILED)
}

// NewReconciliationLoop creates a reconciliation loop.
func NewReconciliationLoop(hub *Hub, pg *pgxpool.Pool, redis *goredis.Client, log *zap.Logger, gate *ReconcileGate) *ReconciliationLoop {
	return &ReconciliationLoop{hub: hub, pg: pg, redis: redis, log: log, gate: gate}
}

// SetMtHubService wires the OMS writer so reconciliation can repair stuck orders.
func (r *ReconciliationLoop) SetMtHubService(svc *MtHubService) {
	r.svc = svc
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
// It is idempotent: overlapping reconciliations for the same account are skipped.
func (r *ReconciliationLoop) TriggerReconcile(accountID string) {
	if r.gate != nil && !r.gate.CanAccept(accountID) {
		r.log.Debug("reconciliation: already in progress, skipping trigger", zap.String("accountID", accountID))
		return
	}
	r.log.Info("reconciliation: triggered for account", zap.String("accountID", accountID))
	if r.gate != nil {
		r.gate.EnterReconciling(accountID)
	}
	go func() {
		// Use a standalone context with timeout. The defer in reconcileAccount
		// will release the gate regardless of success/failure.
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

func brokerOrderStateToOMS(s OrderState) OMSState {
	switch s {
	case OrderStateOpen, OrderStateClosed:
		return OMSStateFilled
	case OrderStateCancelled:
		return OMSStateCancelled
	case OrderStateRejected:
		return OMSStateFailed
	case OrderStatePending:
		return OMSStateWorking
	}
	return ""
}

// isNonTerminalOMSState returns true if the OMS state is not a terminal state.
// Terminal states: FILLED, CANCELLED, FAILED, EXPIRED, REJECTED, SLIPPAGE_REJECTED.
// Used by reconciliation orphan repair (root cause B): broker confirms the
// order does not exist → any non-terminal ant-side state should be repaired
// to FAILED, not just SUBMITTED.
func isNonTerminalOMSState(state string) bool {
	switch OMSState(state) {
	case OMSStateFilled, OMSStateCancelled, OMSStateFailed, OMSStateExpired,
		OMSStateRejected, OMSStateSlippageRejected:
		return false
	}
	return true
}

func (r *ReconciliationLoop) repairOrder(ctx context.Context, accountID string, ticket int64, to OMSState) {
	if r.svc == nil {
		return
	}
	r.svc.TransitionOrderByTicket(ctx, accountID, ticket, to)
}

func (r *ReconciliationLoop) reconcileAccount(ctx context.Context, accountID string) error {
	exec := r.hub.Get(accountID)
	if exec == nil {
		return ErrSessionNotFound
	}

	// 1. Fetch broker-side state (24h window — extended from 1h to catch more gaps)
	brokerOpened, err := exec.FetchOpenedOrders(ctx)
	if err != nil {
		return fmt.Errorf("reconciliation: fetch opened orders: %w", err)
	}
	brokerHistory, err := exec.FetchOrderHistory(ctx, Clk.Now().Add(-24*time.Hour), Clk.Now())
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

	// 2. Fetch ant-side orders from PG (24h window — symmetric with broker
	//    FetchOrderHistory window above to avoid structural false orphans
	//    from comparing ant full-history vs broker 24h slice).
	cutoff := Clk.Now().Add(-24 * time.Hour)
	rows, err := r.pg.Query(ctx, `
		SELECT ticket, state FROM orders WHERE mt_account_id = $1::uuid AND created_at >= $2
		UNION ALL
		SELECT ticket, 'CLOSED' FROM trade_records WHERE account_id = $1::uuid AND close_time >= $2
	`, accountID, cutoff)
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

	// 3. Compare ticket existence and repair stuck SUBMITTED orders.
	//    OMS and broker use different state enums, so direct state comparison
	//    is meaningless; we drive OMS transitions from broker reality.
	var ghosts, orphans, repaired int
	for ticket, antState := range antTickets {
		br, exists := brokerTickets[ticket]
		if !exists {
			r.log.Warn("reconciliation: orphan order (ant has, broker missing)",
				zap.String("accountID", accountID),
				zap.Int64("ticket", ticket))
			orphans++
			if isNonTerminalOMSState(antState) && r.svc != nil {
				r.repairOrder(ctx, accountID, ticket, OMSStateFailed)
				repaired++
			}
			continue
		}
		if antState == string(OMSStateSubmitted) && r.svc != nil {
			to := brokerOrderStateToOMS(br.State)
			if to != "" && to != OMSStateSubmitted {
				r.repairOrder(ctx, accountID, ticket, to)
				repaired++
			}
		}
	}

	r.importGhostOrders(ctx, accountID, brokerTickets, antTickets, &ghosts, &repaired)

	if ghosts+orphans > 0 || repaired > 0 {
		r.log.Info("reconciliation: account summary",
			zap.String("accountID", accountID),
			zap.Int("ghosts", ghosts),
			zap.Int("orphans", orphans),
			zap.Int("repaired", repaired),
			zap.Int("broker_orders", len(brokerTickets)),
			zap.Int("ant_orders", len(antTickets)),
		)
	}

	if r.gate != nil {
		r.gate.MarkReconciled(accountID)
	}

	return nil
}

// importGhostOrders imports broker-side orders missing from ant (ghosts).
// Extracted from reconcileAccount to reduce cognitive complexity.
func (r *ReconciliationLoop) importGhostOrders(
	ctx context.Context,
	accountID string,
	brokerTickets map[int64]*OrderRecord,
	antTickets map[int64]string,
	ghosts, repaired *int,
) {
	for ticket, br := range brokerTickets {
		if _, exists := antTickets[ticket]; exists {
			continue
		}
		// Ghost order (broker has, ant missing): auto-import per ADR-0013 §2.3
		// ("broker has, PG missing → INSERT"). Idempotent via ON CONFLICT DO NOTHING.
		if r.svc != nil {
			if err := r.svc.ImportBrokerOrder(ctx, accountID, br); err != nil {
				r.log.Error("reconciliation: ghost import failed",
					zap.String("accountID", accountID),
					zap.Int64("ticket", ticket), zap.Error(err))
			} else {
				*repaired++
			}
		}
		r.log.Warn("reconciliation: ghost order (broker has, ant missing)",
			zap.String("accountID", accountID),
			zap.Int64("ticket", ticket))
		*ghosts++
	}
}
