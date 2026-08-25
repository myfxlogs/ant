package mthub

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/costsvc"
)

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
	contractSize := decimal.NewFromInt(100000)
	if p, err := s.CachedSymbolParam(ctx, req.AccountID, req.Canonical); err == nil && p != nil {
		if p.ContractSize.GreaterThan(decimal.Zero) {
			contractSize = p.ContractSize
		}
	}
	est := s.costEstimator.Estimate(ctx, costsvc.EstimateParams{
		Symbol:       req.Canonical,
		Side:         sideToString(req.Side),
		Lots:         req.Volume,
		Price:        req.Price,
		ContractSize: contractSize,
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
func costToProto(est *costsvc.CostBreakdown) *antv1.CostEstimate {
	return &antv1.CostEstimate{
		SpreadCost:   est.SpreadCost.String(),
		Commission:   est.Commission.String(),
		SlippageCost: est.SlippageCost.String(),
		SwapCost:     est.SwapCost.String(),
		TotalCost:    est.TotalCost.String(),
	}
}
