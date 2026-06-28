package mthub

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"anttrader/internal/costsvc"
	"anttrader/internal/risk"
	"anttrader/internal/usermgr"
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

	accountStateProvider  AccountStateProvider
	accountOwnerVerifier  AccountOwnerVerifier

	// D6-A: single-chokepoint risk gate evaluated before every order (Place/Close/Modify).
	gate *risk.Gate

	// S1.2: OMS 16-state state machine writer (NEW to VALIDATED to RISK_APPROVED to SUBMITTED).
	omsWriter      *OmsWriter
	brokerRegistry BrokerRegistry // M12-C2: multi-broker registry (optional)
	barBroker      *BarBroker
	tickBroker     *TickBroker
	tradeBroker    *TradeBroker
	statusBroker   *AccountStatusBroker
	logger         *zap.Logger
}

// NewMtHubService creates the service with a Hub, event broker, and optional idempotency guard.
func NewMtHubService(hub *Hub, broker *OrderEventBroker, accountBroker *AccountProfitBroker, snapshotBroker *PositionSnapshotBroker, idem *IdempotencyGuard, gate *ReconcileGate, store *TradeEventStore) *MtHubService {
	return &MtHubService{hub: hub, broker: broker, accountBroker: accountBroker, snapshotBroker: snapshotBroker, idem: idem, reconcileGate: gate, eventStore: store}
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
	return exec.FetchOpenedOrders(ctx)
}

// OrderHistory returns historical orders for the account.
// When the session hasn't been established yet, returns an empty list
// instead of an error — the SSE stream will push data once connected.
func (s *MtHubService) OrderHistory(ctx context.Context, accountID string, from, to time.Time) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return []*OrderRecord{}, nil
	}
	return exec.FetchOrderHistory(ctx, from, to)
}

// SymbolParams returns trading parameters for the given symbols.
func (s *MtHubService) SymbolParams(ctx context.Context, accountID string, canonicals []string) ([]*SymbolParam, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	return exec.FetchSymbolParams(ctx, canonicals)
}

// PriceHistory fetches K-line bars from the connected broker.
func (s *MtHubService) PriceHistory(ctx context.Context, accountID, symbol, period string, from, to int64, count int) ([]*Bar, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	return exec.FetchPriceHistory(ctx, symbol, period, from, to, count)
}

// SymbolList returns all available symbol names for a connected MT account.
func (s *MtHubService) SymbolList(ctx context.Context, accountID string) ([]string, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	return exec.FetchAllSymbols(ctx)
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

