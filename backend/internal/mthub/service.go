package mthub

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"go.uber.org/zap"

	"alphaforge/internal/costsvc"
	"alphaforge/internal/risk"
	"alphaforge/internal/usermgr"
)

// KillSwitchGate is checked before every order placement.
// Implementations must be concurrency-safe.
type KillSwitchGate interface {
	IsEngaged() bool
}

// BrokerRegistry resolves broker names to order execution capabilities.
// M12-C2: Enables multi-broker routing. Concrete implementation in
// mdgateway/adapter registers MT4/MT5 adapters and satisfies this interface.
type BrokerRegistry interface {
	// Resolve returns the BrokerExecutor registered under the given name
	// (e.g. "mt4", "mt5"). Returns an error if no adapter is found.
	Resolve(name string) (BrokerExecutor, error)

	// List returns all registered broker names.
	List() []string
}

// BrokerExecutor is a narrow order execution interface used by components
// that need broker-agnostic order submission (e.g. AlgoExecutor).
// It is a subset of the full OrderExecutor interface.
type BrokerExecutor interface {
	SubmitOrder(ctx context.Context, req *OrderRequest) (int64, error)
	Platform() string
}

// MtHubService is the business-layer facade for order operations.
// All MT account interactions go through this service.
type MtHubService struct {
	hub            *Hub
	broker         *OrderEventBroker
	accountBroker  *AccountProfitBroker
	snapshotBroker *PositionSnapshotBroker
	idem           *IdempotencyGuard // may be nil if Redis is not configured
	reconcileGate  *ReconcileGate
	eventStore     *TradeEventStore // may be nil if NATS is not configured
	userLimiter    *usermgr.UserLimiter
	costEstimator  costsvc.CostEstimator // M10-BASE-D2: pre-trade cost estimation
	killSwitch     KillSwitchGate        // V3-R-5: global kill switch

	accountStateProvider AccountStateProvider
	accountOwnerVerifier AccountOwnerVerifier

	// D6-A: single-chokepoint risk gate evaluated before every order (Place/Close/Modify).
	gate *risk.Gate

	// Guard is the mandatory 3-rule safety net (kill switch, duplicate, max lot size).
	// Evaluated BEFORE the optional Gate. Must pass for any order to reach a broker.
	guard *risk.Guard

	// S1.2: OMS 16-state state machine writer (NEW to VALIDATED to RISK_APPROVED to SUBMITTED).
	omsWriter      *OmsWriter
	brokerRegistry BrokerRegistry // M12-C2: multi-broker registry (optional)
	barBroker      *BarBroker
	tickBroker     *TickBroker
	tradeBroker    *TradeBroker
	statusBroker   *AccountStatusBroker
	logger         *zap.Logger

	// reconnectCooldown prevents repeated reconnect attempts for the same
	// account within a short window. Keyed by accountID, value = last reconnect time.
	reconnectMu     sync.Mutex
	reconnectLastAt map[string]time.Time

	// symbolParamCache stores per-symbol trading parameters with a TTL so the
	// risk gate can resolve ContractSize without a broker round-trip on every order.
	symbolParamMu    sync.RWMutex
	symbolParamCache map[string]symbolParamCacheEntry

	// reconcileTrigger (Task 2) allows TransitionOrderByTicket to trigger
	// a reconciliation when a broker fill event arrives before the ticket
	// is backfilled in OMS. Injected by cmd/server/pipeline.go.
	reconcileTrigger func(accountID string)
}

// NewMtHubService creates the service with a Hub, event broker, and optional idempotency guard.
func NewMtHubService(hub *Hub, broker *OrderEventBroker, accountBroker *AccountProfitBroker, snapshotBroker *PositionSnapshotBroker, idem *IdempotencyGuard, gate *ReconcileGate, store *TradeEventStore) *MtHubService {
	return &MtHubService{hub: hub, broker: broker, accountBroker: accountBroker, snapshotBroker: snapshotBroker, idem: idem, reconcileGate: gate, eventStore: store, reconnectLastAt: map[string]time.Time{}, symbolParamCache: map[string]symbolParamCacheEntry{}}
}

// ResolveScheduleID is the shared helper used by both orderRecordToTradeRecord paths.
// Returns nil for magic=0 (manual/non-strategy trades) or unknown magic.
// Errors are logged but do not block trade record writing — attribution is best-effort.
func ResolveScheduleID(ctx context.Context, resolver ScheduleResolver, log *zap.Logger, accountID uuid.UUID, magic int32) *uuid.UUID {
	if magic == 0 || resolver == nil {
		return nil
	}
	sid, err := resolver.ResolveScheduleIDByMagic(ctx, accountID, magic)
	if err != nil {
		if log != nil {
			log.Warn("resolveSchedule: failed to resolve magic to schedule_id",
				zap.Int32("magic", magic), zap.Stringer("accountID", accountID), zap.Error(err))
		}
		return nil
	}
	return sid
}

// ErrAccountNotOwned is returned when the authenticated user does not own the account.
var ErrAccountNotOwned = errors.New("mthub: account not owned by authenticated user")

// Platform returns the platform string ("mt4"/"mt5") for the account's executor.
func (s *MtHubService) Platform(accountID string) string {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return ""
	}
	return exec.Platform()
}

// SessionState returns "connected" if the Hub has a session for the account,
// or "not_found" otherwise. Expired sessions are auto-refreshed by EnsureSession.
func (s *MtHubService) SessionState(ctx context.Context, accountID string) string {
	_, err := s.hub.EnsureSession(ctx, accountID)
	if err != nil {
		return "not_found"
	}
	return "connected"
}

// ErrRateLimited is returned when the user exceeds their order rate limit.
var ErrRateLimited = errors.New("mthub: order rate limit exceeded")

// ErrDuplicateOrder is returned when the idempotency guard detects a duplicate client ID.
var ErrDuplicateOrder = errors.New("mthub: duplicate order")

// ErrReconciling is returned when PlaceOrder is called while the account is reconciling.
var ErrReconciling = errors.New("mthub: account reconciling, order rejected")

// PlaceOrder places an order on the account's broker via the registered executor.
// If an IdempotencyGuard is configured, duplicate client IDs are rejected before broker submission.
var ErrKillSwitchEngaged = errors.New("mthub: global kill switch engaged")

// ErrCircuitOpen is returned when the broker endpoint circuit breaker is open.
var ErrCircuitOpen = errors.New("mthub: circuit breaker open")

// platform resolves the account's broker platform name from the Hub.
func platform(accountID string, hub *Hub) string {
	exec := hub.Get(accountID)
	if exec == nil {
		return ""
	}
	return exec.Platform()
}
