package mthub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/costsvc"
	"anttrader/internal/risksvc"
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

	// S1.1: pre-trade risk pipeline (capability to hardlimit to platform to engine to sizer)
	riskPipeline          RiskPipeline
	accountStateProvider  AccountStateProvider
	accountOwnerVerifier  AccountOwnerVerifier

	// S1.2: OMS 16-state state machine writer (NEW to VALIDATED to RISK_APPROVED to SUBMITTED).
	omsWriter      *OmsWriter
	brokerRegistry BrokerRegistry // M12-C2: multi-broker registry (optional)
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

// RiskPipeline is the pre-trade risk and sizing pipeline (M10-BASE-C6, S1.1).
type RiskPipeline interface {
	Process(ctx context.Context, req *risksvc.SignalRequest) *risksvc.SignalResult
}

// SetRiskPipeline injects the risk pipeline for pre-trade checks.
func (s *MtHubService) SetRiskPipeline(p RiskPipeline) { s.riskPipeline = p }

// AccountState holds snapshot data needed for risk evaluation.
type AccountState struct {
	Balance, Equity, FreeMargin, Margin float64
	Positions                            int
}

// AccountStateProvider fetches account state for risk evaluation.
type AccountStateProvider func(ctx context.Context, accountID string) (*AccountState, error)

// SetAccountStateProvider injects the account state fetcher for risk evaluation.
func (s *MtHubService) SetAccountStateProvider(p AccountStateProvider) { s.accountStateProvider = p }

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

func (s *MtHubService) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error) {
	if s.killSwitch != nil && s.killSwitch.IsEngaged() {
		return nil, ErrKillSwitchEngaged
	}

	// H7: verify the authenticated user owns the accountID.
	if s.accountOwnerVerifier != nil {
		uid := usermgr.GetUserID(ctx)
		if uid != "" {
			owns, err := s.accountOwnerVerifier(ctx, uid, req.AccountID)
			if err != nil {
				return nil, fmt.Errorf("account ownership check: %w", err)
			}
			if !owns {
				return nil, fmt.Errorf("%w: %s", ErrAccountNotOwned, req.AccountID)
			}
		}
	}

	if s.idem != nil && req.ClientID != "" {
		isDup, existingTicket, err := s.idem.CheckAndSet(ctx, req.AccountID, req.ClientID, 0)
		if err != nil {
			return nil, err
		}
		if isDup {
			return &OrderRecord{Ticket: existingTicket, AccountID: req.AccountID, State: OrderStatePending}, ErrDuplicateOrder
		}
	}

	if s.reconcileGate != nil && !s.reconcileGate.CanAccept(req.AccountID) {
		return nil, fmt.Errorf("%w: %s", ErrReconciling, req.AccountID)
	}

	if s.userLimiter != nil {
		uid := usermgr.GetUserID(ctx)
		if uid != "" && !s.userLimiter.AllowOrder(uid) {
			return nil, ErrRateLimited
		}
	}

	// S1.2: OMS: insert order with state=NEW before risk checks.
	var orderID string
	if s.omsWriter != nil {
		orderID = IdempotencyKey(req.AccountID, req.ClientID)
		if err := s.omsWriter.InsertOrder(ctx, orderID, req.AccountID, req.Canonical,
			int16(req.OrderType), req.Volume.InexactFloat64(), req.Price.InexactFloat64(),
			req.StopLoss.InexactFloat64(), req.TakeProfit.InexactFloat64()); err != nil {
			return nil, fmt.Errorf("oms insert: %w", err)
		}
	}

	// Pre-trade risk pipeline (S1.1): capability to hardlimit to platform to engine to sizer.
	if s.riskPipeline != nil {
		// Fetch account state for risk evaluation.
		var balance, equity, freeMargin, margin float64
		var positions int
		if s.accountStateProvider != nil {
			state, err := s.accountStateProvider(ctx, req.AccountID)
			if err != nil {
				return nil, fmt.Errorf("account state fetch: %w", err)
			}
			if state != nil {
				balance, equity, freeMargin, margin = state.Balance, state.Equity, state.FreeMargin, state.Margin
				positions = state.Positions
			}
		}
		uid := usermgr.GetUserID(ctx)
		result := s.riskPipeline.Process(ctx, &risksvc.SignalRequest{
			UserID:     uid,
			AccountID:  req.AccountID,
			Symbol:     req.Canonical,
			Side:       sideToString(req.Side),
			Price:      req.Price.InexactFloat64(),
			Balance:    balance,
			Equity:     equity,
			FreeMargin: freeMargin,
			Margin:     margin,
			Positions:  positions,
		})
		if !result.Allowed {
			// OMS: record rejection if order was inserted.
			// Two-step: NEW to VALIDATED to REJECTED (no direct NEW to REJECTED edge).
			if s.omsWriter != nil && orderID != "" {
				if err := s.omsWriter.Transition(ctx, orderID, OMSStateNew, OMSStateValidated); err != nil {
					s.logger.Error("oms transition failed",
						zap.Error(err),
						zap.String("orderID", orderID),
						zap.String("from", string(OMSStateNew)),
						zap.String("to", string(OMSStateValidated)))
				}
				if err := s.omsWriter.Transition(ctx, orderID, OMSStateValidated, OMSStateRejected); err != nil {
					s.logger.Error("oms transition failed",
						zap.Error(err),
						zap.String("orderID", orderID),
						zap.String("from", string(OMSStateValidated)),
						zap.String("to", string(OMSStateRejected)))
				}
			}
			return nil, fmt.Errorf("risk rejected at %s: %s", result.Stage, result.Reason)
		}

		// OMS: pipeline passed: VALIDATED, then RISK_APPROVED.
		if s.omsWriter != nil && orderID != "" {
			if err := s.omsWriter.Transition(ctx, orderID, OMSStateNew, OMSStateValidated); err != nil {
				s.logger.Error("oms transition failed",
					zap.Error(err),
					zap.String("orderID", orderID),
					zap.String("from", string(OMSStateNew)),
					zap.String("to", string(OMSStateValidated)))
			} else if err := s.omsWriter.Transition(ctx, orderID, OMSStateValidated, OMSStateRiskApproved); err != nil {
				s.logger.Error("oms transition failed",
					zap.Error(err),
					zap.String("orderID", orderID),
					zap.String("from", string(OMSStateValidated)),
					zap.String("to", string(OMSStateRiskApproved)))
			}
		}

		// Override requested volume with sizer output.
		if result.Lots > 0 {
			req.Volume = decimal.NewFromFloat(result.Lots)
		}
	} else {
		// No risk pipeline: auto-approve to RISK_APPROVED.
		if s.omsWriter != nil && orderID != "" {
			if err := s.omsWriter.Transition(ctx, orderID, OMSStateNew, OMSStateValidated); err != nil {
				s.logger.Error("oms transition failed",
					zap.Error(err),
					zap.String("orderID", orderID),
					zap.String("from", string(OMSStateNew)),
					zap.String("to", string(OMSStateValidated)))
			} else if err := s.omsWriter.Transition(ctx, orderID, OMSStateValidated, OMSStateRiskApproved); err != nil {
				s.logger.Error("oms transition failed",
					zap.Error(err),
					zap.String("orderID", orderID),
					zap.String("from", string(OMSStateValidated)),
					zap.String("to", string(OMSStateRiskApproved)))
			}
		}
	}

	// Pre-trade cost estimation (M10-BASE-D2).
	var costJSON string
	if s.costEstimator != nil {
		est := s.costEstimator.Estimate(ctx, costsvc.EstimateParams{
			Symbol:       req.Canonical,
			Side:         sideToString(req.Side),
			Lots:         req.Volume.InexactFloat64(),
			Price:        req.Price.InexactFloat64(),
			ContractSize: 100000,
		})
		if b, err := json.Marshal(est); err == nil {
			costJSON = string(b)
		}
	}

	exec := s.hub.Get(req.AccountID)
	if exec == nil {
		// OMS: record failure if no session.
		if s.omsWriter != nil && orderID != "" {
			if err := s.omsWriter.Transition(ctx, orderID, OMSStateRiskApproved, OMSStateFailed); err != nil {
				s.logger.Error("oms transition failed",
					zap.Error(err),
					zap.String("orderID", orderID),
					zap.String("from", string(OMSStateRiskApproved)),
					zap.String("to", string(OMSStateFailed)))
			}
		}
		return nil, ErrSessionNotFound
	}
	ticket, err := exec.PlaceOrder(ctx, req)
	if err != nil {
		// OMS: record failure on broker rejection.
		if s.omsWriter != nil && orderID != "" {
			if err := s.omsWriter.Transition(ctx, orderID, OMSStateRiskApproved, OMSStateFailed); err != nil {
				s.logger.Error("oms transition failed",
					zap.Error(err),
					zap.String("orderID", orderID),
					zap.String("from", string(OMSStateRiskApproved)),
					zap.String("to", string(OMSStateFailed)))
			}
		}
		return nil, err
	}

	// OMS: broker accepted, transition to SUBMITTED.
	if s.omsWriter != nil && orderID != "" {
		if err := s.omsWriter.Transition(ctx, orderID, OMSStateRiskApproved, OMSStateSubmitted); err != nil {
			s.logger.Error("oms transition failed",
				zap.Error(err),
				zap.String("orderID", orderID),
				zap.String("from", string(OMSStateRiskApproved)),
				zap.String("to", string(OMSStateSubmitted)))
		}
	}

	// Update the idempotency key with the real ticket after successful placement.
	if s.idem != nil && req.ClientID != "" {
		if err := s.idem.SetTicket(ctx, req.AccountID, req.ClientID, ticket); err != nil {
			s.logger.Error("idempotency set ticket failed",
				zap.Error(err),
				zap.String("accountID", req.AccountID),
				zap.String("clientID", req.ClientID),
				zap.Int64("ticket", ticket))
		}
	}

	// Write ORDER_CREATED event to NATS JetStream (Tier-0) before PG update.
	if s.eventStore != nil {
		ev := &TradeEvent{
			EventID:           fmt.Sprintf("ord-%d-created", ticket),
			EventType:         TradeEventOrderCreated,
			AccountID:         req.AccountID,
			Ticket:            ticket,
			ClientID:          req.ClientID,
			Canonical:         req.Canonical,
			Side:              sideToString(req.Side),
			OrderType:         orderTypeToString(req.OrderType),
			Volume:            req.Volume.InexactFloat64(),
			Price:             req.Price.InexactFloat64(),
			StopLoss:          req.StopLoss.InexactFloat64(),
			TakeProfit:        req.TakeProfit.InexactFloat64(),
			ToState:           "SUBMITTED",
			FromState:         string(OMSStateRiskApproved),
			Timestamp:         Clk.Now(),
			Version:           1,
			CostBreakdownJSON: costJSON,
		}
		if err := s.eventStore.Publish(ctx, ev); err != nil {
			s.logger.Error("event store publish failed",
				zap.Error(err),
				zap.String("eventID", ev.EventID),
				zap.String("accountID", ev.AccountID))
		}
	}

	return &OrderRecord{Ticket: ticket, AccountID: req.AccountID, State: OrderStatePending}, nil
}

// CloseOrder closes an existing position.
// H6: Includes kill switch, reconcile gate, account ownership, and OMS state checks.
func (s *MtHubService) CloseOrder(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error {
	// Kill switch check.
	if s.killSwitch != nil && s.killSwitch.IsEngaged() {
		return ErrKillSwitchEngaged
	}

	// Reconcile gate check.
	if s.reconcileGate != nil && !s.reconcileGate.CanAccept(accountID) {
		return fmt.Errorf("%w: %s", ErrReconciling, accountID)
	}

	// Account ownership verification.
	if s.accountOwnerVerifier != nil {
		uid := usermgr.GetUserID(ctx)
		if uid != "" {
			owns, err := s.accountOwnerVerifier(ctx, uid, accountID)
			if err != nil {
				return fmt.Errorf("account ownership check: %w", err)
			}
			if !owns {
				return fmt.Errorf("%w: %s", ErrAccountNotOwned, accountID)
			}
		}
	}

	exec := s.hub.Get(accountID)
	if exec == nil {
		return ErrSessionNotFound
	}

	// OMS: record close attempt state transition.
	if s.omsWriter != nil {
		closeOrderID := fmt.Sprintf("close-%s-%d", accountID, ticket)
		if err := s.omsWriter.Transition(ctx, closeOrderID, OMSStateWorking, OMSStateFilled); err != nil {
			s.logger.Error("oms close transition failed",
				zap.Error(err),
				zap.String("orderID", closeOrderID),
				zap.String("accountID", accountID),
				zap.Int64("ticket", ticket))
		}
	}

	return exec.CloseOrder(ctx, ticket, lots)
}

// OpenedOrders returns currently open positions.
func (s *MtHubService) OpenedOrders(ctx context.Context, accountID string) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
	}
	return exec.FetchOpenedOrders(ctx)
}

// OrderHistory returns historical orders for the account.
func (s *MtHubService) OrderHistory(ctx context.Context, accountID string, from, to time.Time) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil {
		return nil, ErrSessionNotFound
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

func sideToString(s Side) string {
	switch s {
	case SideBuy:
		return "BUY"
	case SideSell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}

func orderTypeToString(ot OrderType) string {
	switch ot {
	case OrderMarket:
		return "MARKET"
	case OrderLimit:
		return "LIMIT"
	case OrderStop:
		return "STOP"
	case OrderStopLimit:
		return "STOP_LIMIT"
	default:
		return "UNKNOWN"
	}
}
