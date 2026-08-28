package mthub

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"go.uber.org/zap"

	"alphaforge/internal/costsvc"
	"alphaforge/internal/model"
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

	// tradeRecordRepo is used by ImportBrokerOrder to write closed ghost orders
	// (broker has, ant missing) into trade_records with hash chain preservation.
	// May be nil if not wired (ImportBrokerOrder skips trade_records write).
	tradeRecordRepo TradeRecordCreator

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

// PublishBar publishes a bar update to all subscribers for the given account.
func (s *MtHubService) PublishBar(ev *BarUpdate) {
	if s.barBroker != nil {
		s.barBroker.Publish(ev)
	}
}

// SubscribeBarUpdates returns a channel of bar updates for the given account.
func (s *MtHubService) SubscribeBarUpdates(accountID string) (<-chan *BarUpdate, func()) {
	if s.barBroker == nil {
		ch := make(chan *BarUpdate)
		close(ch)
		return ch, func() {}
	}
	return s.barBroker.Subscribe(accountID)
}

// PublishTick publishes a tick (Bid/Ask) update to all subscribers for the given account.
func (s *MtHubService) PublishTick(ev *TickUpdate) {
	if s.tickBroker != nil {
		s.tickBroker.Publish(ev)
	}
}

// SubscribeTickUpdates returns a channel of tick updates for the given account.
func (s *MtHubService) SubscribeTickUpdates(accountID string) (<-chan *TickUpdate, func()) {
	if s.tickBroker == nil {
		ch := make(chan *TickUpdate)
		close(ch)
		return ch, func() {}
	}
	return s.tickBroker.Subscribe(accountID)
}

// LatestTick returns the most recent tick for the given account+symbol.
// Returns nil if no tick has been received yet.
func (s *MtHubService) LatestTick(accountID, symbol string) *TickUpdate {
	if s.tickBroker == nil {
		return nil
	}
	return s.tickBroker.LatestTick(accountID, symbol)
}

// WatchAllTicks returns a channel that receives ALL tick updates across all accounts.
// Used by WatchActiveStrategies to push real-time prices to the Active Runs table.
func (s *MtHubService) WatchAllTicks() (<-chan *TickUpdate, func()) {
	if s.tickBroker == nil {
		ch := make(chan *TickUpdate)
		close(ch)
		return ch, func() {}
	}
	return s.tickBroker.WatchAll()
}

// PublishTradeEvent publishes a trade event to all subscribers for the given account.
func (s *MtHubService) PublishTradeEvent(ev *BrokerTradeEvent) {
	if s.tradeBroker != nil {
		s.tradeBroker.Publish(ev)
	}
}

// SubscribeTradeEvents returns a channel of trade events for the given account.
func (s *MtHubService) SubscribeTradeEvents(accountID string) (<-chan *BrokerTradeEvent, func()) {
	if s.tradeBroker == nil {
		ch := make(chan *BrokerTradeEvent)
		close(ch)
		return ch, func() {}
	}
	return s.tradeBroker.Subscribe(accountID)
}

// PublishAccountStatus publishes an account status event to all subscribers.
func (s *MtHubService) PublishAccountStatus(ev *AccountStatusEvent) {
	if s.statusBroker != nil {
		s.statusBroker.Publish(ev)
	}
}

// SubscribeAccountStatus returns a channel of account status events for the given account.
func (s *MtHubService) SubscribeAccountStatus(accountID string) (<-chan *AccountStatusEvent, func()) {
	if s.statusBroker == nil {
		ch := make(chan *AccountStatusEvent)
		close(ch)
		return ch, func() {}
	}
	return s.statusBroker.Subscribe(accountID)
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

func (s *MtHubService) OpenedOrders(ctx context.Context, accountID string) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	result, err := exec.FetchOpenedOrders(ctx)
	if err != nil && isSessionError(err) {
		if retryErr := s.reconnectAndRetry(accountID, func() error {
			exec = s.hub.Get(accountID)
			if exec == nil {
				return err
			}
			result, err = exec.FetchOpenedOrders(ctx)
			return err
		}); retryErr != nil {
			err = retryErr
		}
	}
	return result, err
}

// OrderHistory returns historical orders for the account.
func (s *MtHubService) OrderHistory(ctx context.Context, accountID string, from, to time.Time) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	result, err := exec.FetchOrderHistory(ctx, from, to)
	if err != nil && isSessionError(err) {
		if retryErr := s.reconnectAndRetry(accountID, func() error {
			exec = s.hub.Get(accountID)
			if exec == nil {
				return err
			}
			result, err = exec.FetchOrderHistory(ctx, from, to)
			return err
		}); retryErr != nil {
			err = retryErr
		}
	}
	return result, err
}

// SymbolParams returns trading parameters for the given symbols.
func (s *MtHubService) SymbolParams(ctx context.Context, accountID string, canonicals []string) ([]*SymbolParam, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	result, err := exec.FetchSymbolParams(ctx, canonicals)
	if err != nil && isSessionError(err) {
		if retryErr := s.reconnectAndRetry(accountID, func() error {
			exec = s.hub.Get(accountID)
			if exec == nil {
				return err
			}
			result, err = exec.FetchSymbolParams(ctx, canonicals)
			return err
		}); retryErr != nil {
			err = retryErr
		}
	}
	return result, err
}

// ActiveAccountIDs returns the IDs of all currently connected MT accounts.
// Delegates to Hub.ActiveAccountIDs.
func (s *MtHubService) ActiveAccountIDs() []string {
	return s.hub.ActiveAccountIDs()
}

// PriceHistory fetches K-line bars from the connected broker.
func (s *MtHubService) PriceHistory(ctx context.Context, accountID, symbol, period string, from, to int64, count int) ([]*Bar, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	result, err := exec.FetchPriceHistory(ctx, symbol, period, from, to, count)
	if err != nil && isSessionError(err) {
		if retryErr := s.reconnectAndRetry(accountID, func() error {
			exec = s.hub.Get(accountID)
			if exec == nil {
				return err
			}
			result, err = exec.FetchPriceHistory(ctx, symbol, period, from, to, count)
			return err
		}); retryErr != nil {
			err = retryErr
		}
	}
	return result, err
}

// SymbolList returns all available symbol names for a connected MT account.
func (s *MtHubService) SymbolList(ctx context.Context, accountID string) ([]string, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	result, err := exec.FetchAllSymbols(ctx)
	if err != nil && isSessionError(err) {
		if retryErr := s.reconnectAndRetry(accountID, func() error {
			exec = s.hub.Get(accountID)
			if exec == nil {
				return err
			}
			result, err = exec.FetchAllSymbols(ctx)
			return err
		}); retryErr != nil {
			err = retryErr
		}
	}
	return result, err
}

// SubscribeSymbols dynamically subscribes the gateway to additional symbols
// for the given account. Newly added symbols start receiving ticks through
// the existing quote stream without reconnection.
func (s *MtHubService) SubscribeSymbols(ctx context.Context, accountID string, symbols []string) error {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return ErrSessionNotFound
	}
	return exec.AddSymbols(ctx, symbols)
}

// SubscribeUserOrderEvents subscribes to all order events for a user.
func (s *MtHubService) SubscribeUserOrderEvents(ctx context.Context, userID string) (<-chan *OrderEvent, func()) {
	return s.broker.Subscribe(userID)
}

// PublishPositionSnapshot publishes a full position snapshot to all subscribers.
func (s *MtHubService) PublishPositionSnapshot(ev *PositionSnapshot) {
	s.snapshotBroker.Publish(ev)
}

// SubscribePositionSnapshots returns a channel of full position snapshots for a single account.
func (s *MtHubService) SubscribePositionSnapshots(ctx context.Context, accountID string) (<-chan *PositionSnapshot, func()) {
	return s.snapshotBroker.Subscribe(accountID)
}

// SnapshotBroker returns the underlying PositionSnapshotBroker.
// Used by the execution barrier to publish read-after-write confirmation
// snapshots into the existing position pipeline (LIVE-ORDER-REENTRY-1).
func (s *MtHubService) SnapshotBroker() *PositionSnapshotBroker { return s.snapshotBroker }

// TradeRecordCreator is the narrow interface used by ImportBrokerOrder to
// write closed ghost orders into trade_records (with hash chain preservation).
// Implemented by *repository.TradeRecordRepository.Create.
type TradeRecordCreator interface {
	Create(ctx context.Context, record *model.TradeRecord) error
}

// SetTradeRecordRepo wires the trade record repository for ImportBrokerOrder
// ghost-order convergence (FIX-2026-08-28-DATA-TRUTH-1). Nil-safe.
func (s *MtHubService) SetTradeRecordRepo(r TradeRecordCreator) { s.tradeRecordRepo = r }
