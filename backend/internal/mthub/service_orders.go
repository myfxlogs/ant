package mthub

import (
	"context"
	"fmt"
	"strconv"
	"google.golang.org/protobuf/proto"

	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
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
			int16(req.OrderType), req.Volume, req.Price,
			req.StopLoss, req.TakeProfit); err != nil {
			return nil, fmt.Errorf("oms insert: %w", err)
		}
	}

	// OMS: advance state before gate evaluation.
	s.omsTransition(ctx, orderID, req.AccountID, OMSStateNew, OMSStateValidated)
	s.omsTransition(ctx, orderID, req.AccountID, OMSStateValidated, OMSStateRiskApproved)

	// D6-A: single-chokepoint risk gate evaluated for every order path.
	if s.gate != nil && s.accountStateProvider != nil {
		intent := orderRequestToIntent(req)
		state, stateErr := s.accountStateProvider(ctx, req.AccountID)
		if stateErr != nil {
			s.logger.Warn("gate: account state fetch failed — fail-closed",
				zap.String("accountID", req.AccountID), zap.Error(stateErr))
		}
		decision := s.gate.Evaluate(ctx, intent, state)
		if !decision.GetAllow() {
			s.omsTransition(ctx, orderID, req.AccountID, OMSStateRiskApproved, OMSStateFailed)
			return nil, fmt.Errorf("gate rejected: %s", decision.GetReason())
		}
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

// orderRequestToIntent converts an mthub OrderRequest into an antv1.OrderIntent
// for Gate.Evaluate(). Prices are kept as decimal strings for precision.
func orderRequestToIntent(req *OrderRequest) *antv1.OrderIntent {
	return &antv1.OrderIntent{
		AccountId: req.AccountID,
		Symbol:    req.Canonical,
		Side:      sideToString(req.Side),
		Volume:    req.Volume.String(),
		Type:      orderTypeToString(req.OrderType),
		Price:     req.Price.String(),
		Sl:        req.StopLoss.String(),
		Tp:        req.TakeProfit.String(),
		Magic:     int64(req.Magic),
	}
}

func sideToString(s Side) string {
	switch s {
	case SideBuy:
		return "buy"
	case SideSell:
		return "sell"
	default:
		return "unknown"
	}
}

func orderTypeToString(ot OrderType) string {
	switch ot {
	case OrderMarket:
		return "market"
	case OrderLimit:
		return "limit"
	case OrderStop:
		return "stop"
	case OrderStopLimit:
		return "stop_limit"
	default:
		return "unknown"
	}
}

// estimateOrderCost runs pre-trade cost estimation and returns a JSON representation.
func (s *MtHubService) estimateOrderCost(ctx context.Context, req *OrderRequest) string {
	est := s.costEstimator.Estimate(ctx, costsvc.EstimateParams{
		Symbol:       req.Canonical,
		Side:         sideToString(req.Side),
		Lots:         req.Volume.InexactFloat64(),
		Price:        req.Price.InexactFloat64(),
		ContractSize: 100000,
	})
	b, err := proto.Marshal(costToProto(&est))
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
		Volume:            req.Volume,
		Price:             req.Price,
		StopLoss:          req.StopLoss,
		TakeProfit:        req.TakeProfit,
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

// lossyFloat64 converts a decimal to float64 for MT API proto boundaries.
// Precision loss is detected but not rejected — the MT proto requires float64.
func (s *MtHubService) omsTransition(ctx context.Context, orderID, accountID string, from, to OMSState) {
	if s.omsWriter == nil || orderID == "" {
		return
	}
	if err := s.omsWriter.Transition(ctx, orderID, accountID, from, to); err != nil {
		if s.logger != nil {
			s.logger.Error("oms transition failed",
				zap.Error(err),
				zap.String("orderID", orderID),
				zap.String("from", string(from)),
				zap.String("to", string(to)))
		}
	}
}

// lossyFloat64 converts a decimal to float64 for MT API proto boundaries.
// Precision loss is detected but not rejected — the MT proto requires float64.
func costToProto(est *costsvc.CostBreakdown) *antv1.CostEstimate {
	return &antv1.CostEstimate{
		SpreadCost: strconv.FormatFloat(est.SpreadCost, 'f', -1, 64), Commission: est.Commission.String(),
		SlippageCost: strconv.FormatFloat(est.SlippageCost, 'f', -1, 64), SwapCost: strconv.FormatFloat(est.SwapCost, 'f', -1, 64),
		TotalCost: strconv.FormatFloat(est.TotalCost, 'f', -1, 64),
	}
}
