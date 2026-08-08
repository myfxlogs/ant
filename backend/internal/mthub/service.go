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
}

// NewMtHubService creates the service with a Hub, event broker, and optional idempotency guard.
func NewMtHubService(hub *Hub, broker *OrderEventBroker, accountBroker *AccountProfitBroker, snapshotBroker *PositionSnapshotBroker, idem *IdempotencyGuard, gate *ReconcileGate, store *TradeEventStore) *MtHubService {
	return &MtHubService{hub: hub, broker: broker, accountBroker: accountBroker, snapshotBroker: snapshotBroker, idem: idem, reconcileGate: gate, eventStore: store, reconnectLastAt: map[string]time.Time{}}
}

// SetUserLimiter injects the per-user rate limiter (nil-safe).
func (s *MtHubService) SetUserLimiter(l *usermgr.UserLimiter) { s.userLimiter = l }

// SetCostEstimator injects the pre-trade cost estimator (M10-BASE-D2).
func (s *MtHubService) SetCostEstimator(e costsvc.CostEstimator) { s.costEstimator = e }

// AccountStateProvider fetches account state for risk evaluation.
// Returns risk.AccountState (single source of truth, shared with Gate).
type AccountStateProvider func(ctx context.Context, accountID string) (*risk.AccountState, error)

// SetAccountStateProvider injects the account state fetcher for risk evaluation.
func (s *MtHubService) SetAccountStateProvider(p AccountStateProvider) { s.accountStateProvider = p }

// SetGate injects the risk gate for pre-trade evaluation (D6-A single chokepoint).
func (s *MtHubService) SetGate(g *risk.Gate) { s.gate = g }

// SetGuard injects the mandatory safety net (kill switch, duplicate, max lot size).
func (s *MtHubService) SetGuard(g *risk.Guard) { s.guard = g }

// SetOmsWriter injects the OMS state writer for order lifecycle tracking (S1.2).
func (s *MtHubService) SetOmsWriter(w *OmsWriter) { s.omsWriter = w }

// SetKillSwitch injects the global kill switch for emergency stop (V3-R-5).
func (s *MtHubService) SetKillSwitch(ks KillSwitchGate) { s.killSwitch = ks }

// SetBrokerRegistry injects the multi-broker registry (M12-C2).
// When configured, external components (e.g. AlgoExecutor) can resolve
// broker adapters by platform name. Existing PlaceOrder flow is unaffected.
func (s *MtHubService) SetBrokerRegistry(r BrokerRegistry) { s.brokerRegistry = r }

// BrokerRegistry returns the configured multi-broker registry, or nil.
func (s *MtHubService) BrokerRegistry() BrokerRegistry { return s.brokerRegistry }

// AccountOwnerVerifier checks that a user owns the given account.
type AccountOwnerVerifier func(ctx context.Context, userID, accountID string) (bool, error)

// SetAccountOwnerVerifier injects the account ownership checker.
func (s *MtHubService) SetAccountOwnerVerifier(v AccountOwnerVerifier) { s.accountOwnerVerifier = v }

// SetLogger injects a zap logger for error reporting.
func (s *MtHubService) SetLogger(l *zap.Logger) { s.logger = l }

// SetBarBroker injects the bar update broker for real-time K-line push.
func (s *MtHubService) SetBarBroker(b *BarBroker) { s.barBroker = b }

// SetTickBroker injects the tick broker for real-time quote (Bid/Ask) push.
func (s *MtHubService) SetTickBroker(b *TickBroker) { s.tickBroker = b }

// SetTradeBroker injects the trade event broker for fill/close/modify events.
func (s *MtHubService) SetTradeBroker(b *TradeBroker) { s.tradeBroker = b }

// SetStatusBroker injects the account status broker for real-time connection state push.
func (s *MtHubService) SetStatusBroker(b *AccountStatusBroker) { s.statusBroker = b }

// ScheduleResolver maps a live trade's magic number back to its schedule ID.
// Used by orderRecordToTradeRecord to attribute trades to strategies (ARCH-4 step⑥).
type ScheduleResolver interface {
	ResolveScheduleIDByMagic(ctx context.Context, accountID uuid.UUID, magic int32) (*uuid.UUID, error)
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
		return []*OrderRecord{}, nil
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
// When the session hasn't been established yet, returns an empty list
// instead of an error — the SSE stream will push data once connected.
func (s *MtHubService) OrderHistory(ctx context.Context, accountID string, from, to time.Time) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return []*OrderRecord{}, nil
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

// PublishAccountProfit publishes an account profit event to all subscribers.
func (s *MtHubService) PublishAccountProfit(ev *AccountProfitEvent) {
	s.accountBroker.Publish(ev)
}

// SubscribeAccountProfit returns a channel of account profit events for a single account.
func (s *MtHubService) SubscribeAccountProfit(ctx context.Context, accountID string) (<-chan *AccountProfitEvent, func()) {
	return s.accountBroker.Subscribe(accountID)
}

// PublishPositionSnapshot publishes a full position snapshot to all subscribers.
func (s *MtHubService) PublishPositionSnapshot(ev *PositionSnapshot) {
	s.snapshotBroker.Publish(ev)
}

// SubscribePositionSnapshots returns a channel of full position snapshots for a single account.
func (s *MtHubService) SubscribePositionSnapshots(ctx context.Context, accountID string) (<-chan *PositionSnapshot, func()) {
	return s.snapshotBroker.Subscribe(accountID)
}
