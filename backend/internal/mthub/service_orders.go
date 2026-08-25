package mthub

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/risk"
	"alphaforge/internal/usermgr"
)

// PlaceOrder places an order on the account's broker via the registered executor.
// If an IdempotencyGuard is configured, duplicate client IDs are rejected before broker submission.
// Implements OMS state machine integration (S1.2) and pre-trade risk pipeline (S1.1).
func (s *MtHubService) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderRecord, error) {
	broker := platform(req.AccountID, s.hub)
	start := Clk.Now()

	if err := s.preTradeChecks(ctx, req); err != nil {
		OrdersPlacedTotal.WithLabelValues(broker, orderStatusRejected).Inc()
		return nil, preBrokerError(err)
	}

	var orderID string
	if s.omsWriter != nil {
		orderID = IdempotencyKey(req.AccountID, req.ClientID)
		if err := s.omsWriter.InsertOrder(ctx, orderID, req.AccountID, platform(req.AccountID, s.hub), req.Canonical,
			int16(req.OrderType), req.Volume, req.Price,
			req.StopLoss, req.TakeProfit, req.Magic); err != nil {
			return nil, fmt.Errorf("oms insert: %w", err)
		}
	}

	s.omsTransition(ctx, orderID, req.AccountID, OMSStateNew, OMSStateValidated)
	s.omsTransition(ctx, orderID, req.AccountID, OMSStateValidated, OMSStateRiskApproved)

	if err := s.evaluatePlaceGate(ctx, req, orderID); err != nil {
		OrdersPlacedTotal.WithLabelValues(broker, orderStatusRejected).Inc()
		PlaceLatencySeconds.WithLabelValues(broker).Observe(time.Since(start).Seconds())
		return nil, preBrokerError(err)
	}

	var costEstimate *antv1.CostEstimate
	if s.costEstimator != nil {
		costEstimate = s.estimateOrderCost(ctx, req)
	}

	ticket, err := s.submitToBroker(ctx, req, orderID)
	if err != nil {
		OrdersPlacedTotal.WithLabelValues(broker, orderStatusErr).Inc()
		PlaceLatencySeconds.WithLabelValues(broker).Observe(time.Since(start).Seconds())
		return nil, brokerError(err)
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
	if s.gate == nil {
		return wrapGateError("gate not configured: order rejected (fail-closed)")
	}
	if s.accountStateProvider == nil {
		return wrapGateError("account state provider not configured: order rejected (fail-closed)")
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
	// RISK-MARGIN2: Always overlay ContractSize from the broker for this order's
	// symbol before evaluating the gate. Cache + TTL avoids a broker round-trip
	// per order. Unknown contract size (zero or lookup failure) flows through to
	// the MarginPreCheck rule and is fail-closed with reason "contract size unknown".
	if state != nil && req.Canonical != "" {
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		param, err := s.CachedSymbolParam(sctx, req.AccountID, req.Canonical)
		cancel()
		if err != nil {
			return fmt.Errorf("contract size lookup failed for %s: %w", req.Canonical, err)
		}
		if param != nil && param.ContractSize.GreaterThan(decimal.Zero) {
			state.ContractSize = param.ContractSize
		}
		if exec := s.hub.Get(req.AccountID); exec != nil {
			if marginRequirer, ok := exec.(MarginRequirer); ok {
				price, _ := decimal.NewFromString(intent.GetPrice())
				marginCtx, marginCancel := context.WithTimeout(context.Background(), 5*time.Second)
				required, marginErr := marginRequirer.RequiredMargin(marginCtx, req.Canonical, req.Volume, req.Side, price)
				marginCancel()
				if marginErr != nil {
					return fmt.Errorf("broker required margin failed for %s: %w", req.Canonical, marginErr)
				}
				if required.IsNegative() {
					return fmt.Errorf("broker returned negative required margin for %s", req.Canonical)
				}
				state.BrokerMarginAvailable = true
				state.RequiredMarginKnown = true
				state.RequiredMargin = required
			}
		}
	}

	decision := s.gate.Evaluate(ctx, intent, state)
	if !decision.GetAllow() {
		s.omsTransition(ctx, orderID, req.AccountID, OMSStateRiskApproved, OMSStateFailed)
		return wrapGateError("gate rejected: " + decision.GetReason())
	}
	return nil
}

// submitToBroker resolves the account's executor and submits the order.
// All risk checks are handled by the Gate in evaluatePlaceGate (D6-A single chokepoint).
func (s *MtHubService) submitToBroker(ctx context.Context, req *OrderRequest, orderID string) (int64, error) {
	exec := s.hub.Get(req.AccountID)
	if exec == nil {
		s.omsTransition(ctx, orderID, req.AccountID, OMSStateRiskApproved, OMSStateFailed)
		return 0, preBrokerError(ErrSessionNotFound)
	}

	ticket, err := exec.PlaceOrder(ctx, req)
	if err != nil {
		s.omsTransition(ctx, orderID, req.AccountID, OMSStateRiskApproved, OMSStateFailed)
		return 0, err
	}
	s.omsTransition(ctx, orderID, req.AccountID, OMSStateRiskApproved, OMSStateSubmitted)
	// Store the real broker ticket so OnOrderUpdate can resolve orderID by ticket.
	if s.omsWriter != nil {
		if err := s.omsWriter.UpdateTicket(ctx, orderID, ticket); err != nil && s.logger != nil {
			s.logger.Error("oms: failed to update ticket after broker accept",
				zap.String("orderID", orderID), zap.Int64("ticket", ticket), zap.Error(err))
		}
	}
	return ticket, nil
}
