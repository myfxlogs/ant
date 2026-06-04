package mthub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/costsvc"
	"anttrader/internal/usermgr"
)

// PlaceOrder places an order on the account's broker via the registered executor.
// If an IdempotencyGuard is configured, duplicate client IDs are rejected before broker submission.
// Implements OMS state machine integration (S1.2) and pre-trade risk pipeline (S1.1).
func (s *MtHubService) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error) {
	// Pre-trade gates.
	if s.killSwitch != nil && s.killSwitch.IsEngaged() {
		return nil, ErrKillSwitchEngaged
	}
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

	// OMS: insert order with state=NEW before risk checks.
	var orderID string
	if s.omsWriter != nil {
		orderID = IdempotencyKey(req.AccountID, req.ClientID)
		if err := s.omsWriter.InsertOrder(ctx, orderID, req.AccountID, platform(req.AccountID, s.hub), req.Canonical,
			int16(req.OrderType), lossyFloat64(req.Volume), lossyFloat64(req.Price),
			lossyFloat64(req.StopLoss), lossyFloat64(req.TakeProfit)); err != nil {
			return nil, fmt.Errorf("oms insert: %w", err)
		}
	}

	// Pre-trade risk pipeline (S1.1).
	if err := s.runPreTradeRisk(ctx, req, orderID); err != nil {
		return nil, err
	}

	// Pre-trade cost estimation (M10-BASE-D2).
	var costJSON string
	if s.costEstimator != nil {
		costJSON = s.estimateOrderCost(ctx, req)
	}

	// Submit to broker executor.
	ticket, err := s.submitToBroker(ctx, req, orderID)
	if err != nil {
		return nil, err
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

	// Write ORDER_CREATED event to NATS JetStream (Tier-0).
	s.publishOrderCreatedEvent(ctx, req, ticket, costJSON)

	return &OrderRecord{Ticket: ticket, AccountID: req.AccountID, State: OrderStatePending}, nil
}

// submitToBroker resolves the account's executor and submits the order.
func (s *MtHubService) submitToBroker(ctx context.Context, req *OrderRequest, orderID string) (int64, error) {
	exec := s.hub.Get(req.AccountID)
	if exec == nil {
		s.omsTransition(ctx, orderID, req.AccountID, OMSStateRiskApproved, OMSStateFailed)
		return 0, ErrSessionNotFound
	}
	ticket, err := exec.PlaceOrder(ctx, req)
	if err != nil {
		s.omsTransition(ctx, orderID, req.AccountID, OMSStateRiskApproved, OMSStateFailed)
		return 0, err
	}
	s.omsTransition(ctx, orderID, req.AccountID, OMSStateRiskApproved, OMSStateSubmitted)
	return ticket, nil
}

// estimateOrderCost runs pre-trade cost estimation and returns a JSON representation.
func (s *MtHubService) estimateOrderCost(ctx context.Context, req *OrderRequest) string {
	est := s.costEstimator.Estimate(ctx, costsvc.EstimateParams{
		Symbol:       req.Canonical,
		Side:         sideToString(req.Side),
		Lots:         lossyFloat64(req.Volume),
		Price:        lossyFloat64(req.Price),
		ContractSize: 100000,
	})
	b, err := json.Marshal(est)
	if err != nil {
		return ""
	}
	return string(b)
}

// publishOrderCreatedEvent emits an ORDER_CREATED event to the event store (Tier-0).
func (s *MtHubService) publishOrderCreatedEvent(ctx context.Context, req *OrderRequest, ticket int64, costJSON string) {
	if s.eventStore == nil {
		return
	}
	ev := &TradeEvent{
		EventID:           fmt.Sprintf("ord-%d-created", ticket),
		EventType:         TradeEventOrderCreated,
		AccountID:         req.AccountID,
		Ticket:            ticket,
		ClientID:          req.ClientID,
		Canonical:         req.Canonical,
		Side:              sideToString(req.Side),
		OrderType:         orderTypeToString(req.OrderType),
		Volume:            lossyFloat64(req.Volume),
		Price:             lossyFloat64(req.Price),
		StopLoss:          lossyFloat64(req.StopLoss),
		TakeProfit:        lossyFloat64(req.TakeProfit),
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
// CloseOrder closes an existing position.
// Gate order matches PlaceOrder: killSwitch → ownership → idempotency → reconcile → rateLimit.
func (s *MtHubService) CloseOrder(ctx context.Context, accountID string, ticket int64, lots decimal.Decimal) error {
	// Pre-trade gates — unified order matching PlaceOrder.
	if s.killSwitch != nil && s.killSwitch.IsEngaged() {
		return ErrKillSwitchEngaged
	}
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

	// Idempotency check: key is cleaned up after execution so retries work.
	closeOrderID := fmt.Sprintf("close-%s-%d", accountID, ticket)
	if s.idem != nil {
		isDup, _, err := s.idem.CheckAndSet(ctx, accountID, closeOrderID, ticket)
		if err != nil {
			return fmt.Errorf("idempotency check: %w", err)
		}
		if isDup {
			return nil // Concurrent close attempt — already being processed.
		}
		defer s.idem.DeleteKey(ctx, accountID, closeOrderID)
	}

	if s.reconcileGate != nil && !s.reconcileGate.CanAccept(accountID) {
		return fmt.Errorf("%w: %s", ErrReconciling, accountID)
	}
	if s.userLimiter != nil {
		uid := usermgr.GetUserID(ctx)
		if uid != "" && !s.userLimiter.AllowOrder(uid) {
			return ErrRateLimited
		}
	}

	exec := s.hub.Get(accountID)
	if exec == nil {
		if s.logger != nil {
			s.logger.Warn("CloseOrder: session not found", zap.String("accountID", accountID), zap.Int64("ticket", ticket))
		}
		return ErrSessionNotFound
	}

	if s.logger != nil {
		s.logger.Info("CloseOrder: calling executor", zap.String("accountID", accountID), zap.Int64("ticket", ticket), zap.String("lots", lots.String()))
	}

	// OMS: insert close order row first (mirrors PlaceOrder), then transition.
	if s.omsWriter != nil {
		pf := platform(accountID, s.hub)
		if err := s.omsWriter.InsertOrder(ctx, closeOrderID, accountID, pf, "",
			int16(OrderMarket), lossyFloat64(lots), 0, 0, 0); err != nil {
			if s.logger != nil {
				s.logger.Warn("CloseOrder: OMS insert skipped", zap.Error(err))
			}
		} else {
			s.omsTransition(ctx, closeOrderID, accountID, OMSStateNew, OMSStateRiskApproved)
			s.omsTransition(ctx, closeOrderID, accountID, OMSStateRiskApproved, OMSStateSubmitted)
		}
	}

	err := exec.CloseOrder(ctx, ticket, lots)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("CloseOrder: executor failed", zap.Error(err), zap.String("accountID", accountID), zap.Int64("ticket", ticket))
		}
		if s.omsWriter != nil {
			s.omsTransition(ctx, closeOrderID, accountID, OMSStateSubmitted, OMSStateFailed)
		}
		return err
	}

	if s.logger != nil {
		s.logger.Info("CloseOrder: executor success", zap.String("accountID", accountID), zap.Int64("ticket", ticket))
	}
	if s.omsWriter != nil {
		s.omsTransition(ctx, closeOrderID, accountID, OMSStateSubmitted, OMSStateFilled)
	}

	// Publish ORDER_FILLED event to NATS JetStream (mirrors PlaceOrder's event publishing).
	if s.eventStore != nil {
		ev := &TradeEvent{
			EventID:   fmt.Sprintf("close-%s-%d", accountID, ticket),
			EventType: TradeEventOrderFilled,
			AccountID: accountID, Ticket: ticket,
			ToState: string(OMSStateFilled), FromState: string(OMSStateSubmitted),
			Timestamp: Clk.Now(), Version: 1,
		}
		if err := s.eventStore.Publish(ctx, ev); err != nil && s.logger != nil {
			s.logger.Error("close order event publish failed", zap.Error(err),
				zap.String("accountID", accountID), zap.Int64("ticket", ticket))
		}
	}
	return nil
}

// --- Helpers ---

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

// lossyFloat64 converts a decimal to float64 for MT API proto boundaries.
// Precision loss is detected but not rejected — the MT proto requires float64.
func lossyFloat64(d decimal.Decimal) float64 {
	f, exact := d.Float64()
	_ = exact
	return f
}
