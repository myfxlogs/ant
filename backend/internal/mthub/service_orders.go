package mthub

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/costsvc"
	"alphaforge/internal/risk"
	"alphaforge/internal/risksvc"
	"alphaforge/internal/usermgr"
)

// PlaceOrder places an order on the account's broker via the registered executor.
// If an IdempotencyGuard is configured, duplicate client IDs are rejected before broker submission.
// Implements OMS state machine integration (S1.2) and pre-trade risk pipeline (S1.1).
func (s *MtHubService) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error) {
	// Pre-trade gates.
	if s.killSwitch != nil && s.killSwitch.IsEngaged() {
		return nil, ErrKillSwitchEngaged
	}
	// Guard: mandatory 3-rule safety net (kill switch, duplicate, max lot size).
	if s.guard != nil {
		side := "buy"
		if req.Side == SideSell {
			side = "sell"
		}
		if result := s.guard.Check(ctx, &risk.GuardRequest{
			Symbol: req.Canonical, Side: side,
			Volume: req.Volume, OrderType: "market", Price: req.Price,
		}); !result.Allowed {
			return nil, fmt.Errorf("guard: %s", result.Reason)
		}
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
	var costEstimate *antv1.CostEstimate
	if s.costEstimator != nil {
		costEstimate = s.estimateOrderCost(ctx, req)
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
	s.publishOrderCreatedEvent(ctx, req, ticket, costEstimate)

	return &OrderRecord{Ticket: ticket, AccountID: req.AccountID, State: OrderStatePending}, nil
}

// submitToBroker resolves the account's executor and submits the order.
func (s *MtHubService) submitToBroker(ctx context.Context, req *OrderRequest, orderID string) (int64, error) {
	exec := s.hub.Get(req.AccountID)
	if exec == nil {
		s.omsTransition(ctx, orderID, req.AccountID, OMSStateRiskApproved, OMSStateFailed)
		return 0, ErrSessionNotFound
	}

	// P0-6: broker-backed margin precheck for MT5 accounts.
	if mr, ok := exec.(MarginRequirer); ok && s.accountStateProvider != nil {
		state, stateErr := s.accountStateProvider(ctx, req.AccountID)
		if stateErr == nil && state != nil {
			requiredMargin, rmErr := mr.RequiredMargin(ctx, req.Canonical, req.Volume, req.Side, req.Price)
			if rmErr == nil {
				check := &risksvc.CheckRequest{
					UserID:    usermgr.GetUserID(ctx),
					AccountID: req.AccountID,
					Symbol:    req.Canonical,
					Side:      sideToString(req.Side),
					Volume:    req.Volume,
					Price:     req.Price,
					Balance:   state.Balance,
					Equity:    state.Equity,
					Margin:    state.UsedMargin,
					Positions: state.OpenPositions + 1,
				}
				if result := risksvc.PreCheck(ctx, check, risksvc.DefaultRiskLimits(), 0, state.FreeMargin, requiredMargin); !result.Allowed {
					s.omsTransition(ctx, orderID, req.AccountID, OMSStateRiskApproved, OMSStateFailed)
					return 0, fmt.Errorf("precheck rejected: %s", result.Reason)
				}
			} else {
				s.logger.Warn("RequiredMargin RPC failed, skipping broker margin precheck",
					zap.String("account", req.AccountID), zap.Error(rmErr))
			}
		} else if stateErr != nil {
			s.logger.Warn("account state fetch failed for margin precheck",
				zap.String("account", req.AccountID), zap.Error(stateErr))
		}
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

// estimateOrderCost runs pre-trade cost estimation and returns a CostEstimate proto.
func (s *MtHubService) estimateOrderCost(ctx context.Context, req *OrderRequest) *antv1.CostEstimate {
	est := s.costEstimator.Estimate(ctx, costsvc.EstimateParams{
		Symbol:       req.Canonical,
		Side:         sideToString(req.Side),
		Lots:         req.Volume,
		Price:        req.Price,
		ContractSize: decimal.NewFromInt(100000),
	})
	return costToProto(&est)
}

// publishOrderCreatedEvent emits an ORDER_CREATED event to the event store (Tier-0).
func (s *MtHubService) publishOrderCreatedEvent(ctx context.Context, req *OrderRequest, ticket int64, costEstimate *antv1.CostEstimate) {
	if s.eventStore == nil {
		return
	}
	ev := &TradeEvent{
		EventID:      fmt.Sprintf("ord-%d-created", ticket),
		EventType:    TradeEventOrderCreated,
		AccountID:    req.AccountID,
		Ticket:       ticket,
		ClientID:     req.ClientID,
		Canonical:    req.Canonical,
		Side:         sideToString(req.Side),
		OrderType:    orderTypeToString(req.OrderType),
		Volume:       req.Volume,
		Price:        req.Price,
		StopLoss:     req.StopLoss,
		TakeProfit:   req.TakeProfit,
		ToState:      "SUBMITTED",
		FromState:    string(OMSStateRiskApproved),
		Timestamp:    Clk.Now(),
		Version:      1,
		CostEstimate: costEstimate,
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
		SpreadCost:   est.SpreadCost.String(),
		Commission:   est.Commission.String(),
		SlippageCost: est.SlippageCost.String(),
		SwapCost:     est.SwapCost.String(),
		TotalCost:    est.TotalCost.String(),
	}
}
