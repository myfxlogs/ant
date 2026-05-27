package mthub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"anttrader/internal/costsvc"
	"anttrader/internal/risksvc"
	"anttrader/internal/usermgr"
)

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

	// S1.1: pre-trade risk pipeline (capability → hardlimit → platform → engine → sizer)
	riskPipeline        RiskPipeline
	accountStateProvider AccountStateProvider

	// S1.2: OMS 15-state state machine writer (NEW → VALIDATED → RISK_APPROVED → SUBMITTED).
	omsWriter *OmsWriter
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

// ErrRateLimited is returned when the user exceeds their order rate limit.
var ErrRateLimited = errors.New("mthub: order rate limit exceeded")

// ErrDuplicateOrder is returned when the idempotency guard detects a duplicate client ID.
var ErrDuplicateOrder = errors.New("mthub: duplicate order")

// ErrReconciling is returned when PlaceOrder is called while the account is reconciling.
var ErrReconciling = errors.New("mthub: account reconciling, order rejected")

// PlaceOrder places an order on the account's broker via the registered executor.
// If an IdempotencyGuard is configured, duplicate client IDs are rejected before broker submission.
func (s *MtHubService) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error) {
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

	// S1.2: OMS — insert order with state=NEW before risk checks.
	var orderID string
	if s.omsWriter != nil {
		orderID = IdempotencyKey(req.AccountID, req.ClientID)
		if err := s.omsWriter.InsertOrder(ctx, orderID, req.AccountID, req.Canonical,
			int16(req.OrderType), req.Volume.InexactFloat64(), req.Price.InexactFloat64(),
			req.StopLoss.InexactFloat64(), req.TakeProfit.InexactFloat64()); err != nil {
			return nil, fmt.Errorf("oms insert: %w", err)
		}
	}

	// Pre-trade risk pipeline (S1.1): capability → hardlimit → platform → engine → sizer.
	if s.riskPipeline != nil {
		// Fetch account state for risk evaluation.
		var balance, equity, freeMargin, margin float64
		var positions int
		if s.accountStateProvider != nil {
			if state, err := s.accountStateProvider(ctx, req.AccountID); err == nil && state != nil {
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
			if s.omsWriter != nil && orderID != "" {
				_ = s.omsWriter.Transition(ctx, orderID, OMSStateNew, OMSStateRejected)
			}
			return nil, fmt.Errorf("risk rejected at %s: %s", result.Stage, result.Reason)
		}

		// OMS: pipeline passed → VALIDATED, then RISK_APPROVED.
		if s.omsWriter != nil && orderID != "" {
			if err := s.omsWriter.Transition(ctx, orderID, OMSStateNew, OMSStateValidated); err == nil {
				_ = s.omsWriter.Transition(ctx, orderID, OMSStateValidated, OMSStateRiskApproved)
			}
		}

		// Override requested volume with sizer output.
		if result.Lots > 0 {
			req.Volume = decimal.NewFromFloat(result.Lots)
		}
	} else {
		// No risk pipeline — auto-approve to RISK_APPROVED.
		if s.omsWriter != nil && orderID != "" {
			if err := s.omsWriter.Transition(ctx, orderID, OMSStateNew, OMSStateValidated); err == nil {
				_ = s.omsWriter.Transition(ctx, orderID, OMSStateValidated, OMSStateRiskApproved)
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
			_ = s.omsWriter.Transition(ctx, orderID, OMSStateRiskApproved, OMSStateFailed)
		}
		return nil, ErrSessionNotFound
	}
	ticket, err := exec.PlaceOrder(ctx, req)
	if err != nil {
		// OMS: record failure on broker rejection.
		if s.omsWriter != nil && orderID != "" {
			_ = s.omsWriter.Transition(ctx, orderID, OMSStateRiskApproved, OMSStateFailed)
		}
		return nil, err
	}

	// OMS: broker accepted → SUBMITTED.
	if s.omsWriter != nil && orderID != "" {
		_ = s.omsWriter.Transition(ctx, orderID, OMSStateRiskApproved, OMSStateSubmitted)
	}

	// Update the idempotency key with the real ticket after successful placement.
	if s.idem != nil && req.ClientID != "" {
		_ = s.idem.SetTicket(ctx, req.AccountID, req.ClientID, ticket)
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
		_ = s.eventStore.Publish(ctx, ev)
	}

	return &OrderRecord{Ticket: ticket, AccountID: req.AccountID, State: OrderStatePending}, nil
}

// CloseOrder closes an existing position.
func (s *MtHubService) CloseOrder(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error {
	exec := s.hub.Get(accountID)
	if exec == nil { return ErrSessionNotFound }
	return exec.CloseOrder(ctx, ticket, lots)
}

// OpenedOrders returns currently open positions.
func (s *MtHubService) OpenedOrders(ctx context.Context, accountID string) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil { return nil, ErrSessionNotFound }
	return exec.FetchOpenedOrders(ctx)
}

// OrderHistory returns historical orders for the account.
func (s *MtHubService) OrderHistory(ctx context.Context, accountID string, from, to time.Time) ([]*OrderRecord, error) {
	exec := s.hub.Get(accountID)
	if exec == nil { return nil, ErrSessionNotFound }
	return exec.FetchOrderHistory(ctx, from, to)
}

// SymbolParams returns trading parameters for the given symbols.
func (s *MtHubService) SymbolParams(ctx context.Context, accountID string, canonicals []string) ([]*SymbolParam, error) {
	exec := s.hub.Get(accountID)
	if exec == nil { return nil, ErrSessionNotFound }
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
