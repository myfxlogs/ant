package mthub

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/costsvc"
	"alphaforge/internal/risk"
	"alphaforge/internal/usermgr"
)

// PlaceOrder places an order on the account's broker via the registered executor.
// If an IdempotencyGuard is configured, duplicate client IDs are rejected before broker submission.
// Implements OMS state machine integration (S1.2) and pre-trade risk pipeline (S1.1).
func (s *MtHubService) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error) {
	broker := platform(req.AccountID, s.hub)
	start := time.Now()

	if err := s.preTradeChecks(ctx, req); err != nil {
		OrdersPlacedTotal.WithLabelValues(broker, orderStatusRejected).Inc()
		return nil, err
	}

	var orderID string
	if s.omsWriter != nil {
		orderID = IdempotencyKey(req.AccountID, req.ClientID)
		if err := s.omsWriter.InsertOrder(ctx, orderID, req.AccountID, platform(req.AccountID, s.hub), req.Canonical,
			int16(req.OrderType), req.Volume, req.Price,
			req.StopLoss, req.TakeProfit); err != nil {
			return nil, fmt.Errorf("oms insert: %w", err)
		}
	}

	s.omsTransition(ctx, orderID, req.AccountID, OMSStateNew, OMSStateValidated)
	s.omsTransition(ctx, orderID, req.AccountID, OMSStateValidated, OMSStateRiskApproved)

	if err := s.evaluatePlaceGate(ctx, req, orderID); err != nil {
		OrdersPlacedTotal.WithLabelValues(broker, orderStatusRejected).Inc()
		PlaceLatencySeconds.WithLabelValues(broker).Observe(time.Since(start).Seconds())
		return nil, err
	}

	var costEstimate *antv1.CostEstimate
	if s.costEstimator != nil {
		costEstimate = s.estimateOrderCost(ctx, req)
	}

	ticket, err := s.submitToBroker(ctx, req, orderID)
	if err != nil {
		OrdersPlacedTotal.WithLabelValues(broker, orderStatusErr).Inc()
		PlaceLatencySeconds.WithLabelValues(broker).Observe(time.Since(start).Seconds())
		return nil, err
	}

	if s.idem != nil && req.ClientID != "" {
		if err := s.idem.SetTicket(ctx, req.AccountID, req.ClientID, ticket); err != nil {
			if s.logger != nil {
				s.logger.Error("idempotency set ticket failed",
					zap.Error(err),
					zap.String("accountID", req.AccountID),
					zap.String("clientID", req.ClientID),
					zap.Int64("ticket", ticket))
			}
		}
	}

	s.publishOrderCreatedEvent(ctx, req, ticket, costEstimate)

	OrdersPlacedTotal.WithLabelValues(broker, orderStatusOK).Inc()
	PlaceLatencySeconds.WithLabelValues(broker).Observe(time.Since(start).Seconds())

	return &OrderRecord{Ticket: ticket, AccountID: req.AccountID, State: OrderStatePending}, nil
}

func (s *MtHubService) preTradeChecks(ctx context.Context, req *OrderRequest) error {
	if s.killSwitch != nil && s.killSwitch.IsEngaged() {
		return ErrKillSwitchEngaged
	}
	if s.guard != nil {
		side := "buy"
		if req.Side == SideSell {
			side = "sell"
		}
		if result := s.guard.Check(ctx, &risk.GuardRequest{
			Symbol: req.Canonical, Side: side,
			Volume: req.Volume, OrderType: "market", Price: req.Price,
		}); !result.Allowed {
			return fmt.Errorf("guard: %s", result.Reason)
		}
	}
	if s.accountOwnerVerifier != nil {
		uid := usermgr.GetUserID(ctx)
		if uid == "" {
			return fmt.Errorf("unauthenticated: user ID required for order placement")
		}
		owns, err := s.accountOwnerVerifier(ctx, uid, req.AccountID)
		if err != nil {
			return fmt.Errorf("account ownership check: %w", err)
		}
		if !owns {
			return fmt.Errorf("%w: %s", ErrAccountNotOwned, req.AccountID)
		}
	}
	if s.idem != nil && req.ClientID != "" {
		isDup, _, err := s.idem.CheckAndSet(ctx, req.AccountID, req.ClientID, 0)
		if err != nil {
			return err
		}
		if isDup {
			return ErrDuplicateOrder
		}
	}
	if s.reconcileGate != nil && !s.reconcileGate.CanAccept(req.AccountID) {
		return fmt.Errorf("%w: %s", ErrReconciling, req.AccountID)
	}
	if s.userLimiter != nil {
		uid := usermgr.GetUserID(ctx)
		if uid != "" && !s.userLimiter.AllowOrder(uid) {
			return ErrRateLimited
		}
	}
	return nil
}

func (s *MtHubService) evaluatePlaceGate(ctx context.Context, req *OrderRequest, orderID string) error {
	if s.gate == nil || s.accountStateProvider == nil {
		return nil
	}
	intent := orderRequestToIntent(req)
	intent.UserId = usermgr.GetUserID(ctx)

	// RISK-MARGIN1: For market orders, intent.Price is zero because the fill
	// price is unknown at submission time. Margin rules (MarginPreCheck,
	// MarginFloorRule) skip when price=0, so market orders bypass margin checks.
	// Resolve the current mid-price from TickBroker so margin rules can compute
	// required margin. If no tick is available, leave price=0 — the gate's
	// fail-closed logic (state==nil) still applies, and margin rules will skip
	// rather than block (graceful degradation when no tick feed is available).
	if req.OrderType == OrderMarket && intent.Price == "0" && s.tickBroker != nil {
		if tick := s.tickBroker.LatestTick(req.AccountID, req.Canonical); tick != nil {
			mid := tick.Bid.Add(tick.Ask).Div(decimal.NewFromInt(2))
			if mid.GreaterThan(decimal.Zero) {
				intent.Price = mid.String()
			}
		}
	}

	state, stateErr := s.accountStateProvider(ctx, req.AccountID)
	if stateErr != nil && s.logger != nil {
		s.logger.Warn("gate: account state fetch failed — fail-closed",
			zap.String("accountID", req.AccountID), zap.Error(stateErr))
	}
	decision := s.gate.Evaluate(ctx, intent, state)
	if !decision.GetAllow() {
		s.omsTransition(ctx, orderID, req.AccountID, OMSStateRiskApproved, OMSStateFailed)
		return fmt.Errorf("gate rejected: %s", decision.GetReason())
	}
	return nil
}

// submitToBroker resolves the account's executor and submits the order.
// All risk checks are handled by the Gate in evaluatePlaceGate (D6-A single chokepoint).
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
		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
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
	if err := s.eventStore.Publish(ctx, ev); err != nil && s.logger != nil {
		s.logger.Error("event store publish failed",
			zap.Error(err),
			zap.String("eventID", ev.EventID),
			zap.String("accountID", ev.AccountID))
	}
	EventPublishedTotal.WithLabelValues(string(TradeEventOrderCreated)).Inc()
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
