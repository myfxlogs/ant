package mthub

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"anttrader/internal/risksvc"
	"anttrader/internal/usermgr"
)

// runPreTradeRisk executes the pre-trade risk pipeline (S1.1).
// When no pipeline is configured, auto-approves the order.
func (s *MtHubService) runPreTradeRisk(ctx context.Context, req *OrderRequest, orderID string) error {
	if s.riskPipeline == nil {
		s.omsTransition(ctx, orderID, req.AccountID, OMSStateNew, OMSStateValidated)
		s.omsTransition(ctx, orderID, req.AccountID, OMSStateValidated, OMSStateRiskApproved)
		return nil
	}

	// Fetch account state for risk evaluation.
	var balance, equity, freeMargin, margin float64
	var positions int
	if s.accountStateProvider != nil {
		state, err := s.accountStateProvider(ctx, req.AccountID)
		if err != nil {
			return fmt.Errorf("account state fetch: %w", err)
		}
		if state != nil {
			balance, _ = state.Balance.Float64()
			equity, _ = state.Equity.Float64()
			freeMargin, _ = state.FreeMargin.Float64()
			margin, _ = state.UsedMargin.Float64()
			positions = state.OpenPositions
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
		s.omsTransition(ctx, orderID, req.AccountID, OMSStateNew, OMSStateValidated)
		s.omsTransition(ctx, orderID, req.AccountID, OMSStateValidated, OMSStateRejected)
		return fmt.Errorf("risk rejected at %s: %s", result.Stage, result.Reason)
	}

	// Approved: transition through VALIDATED → RISK_APPROVED.
	s.omsTransition(ctx, orderID, req.AccountID, OMSStateNew, OMSStateValidated)
	s.omsTransition(ctx, orderID, req.AccountID, OMSStateValidated, OMSStateRiskApproved)

	// Override volume with sizer output for automated signals.
	// For manual orders (user already set a volume), skip override
	// when the sizer cannot compute (missing Price/ATR for market orders).
	if result.Lots > 0 {
		req.Volume = decimal.NewFromFloat(result.Lots)
	} else if result.Stage == "sizer" && req.Volume.IsPositive() {
		// Manual order: user provided volume, sizer lacks data — let it through.
		return nil
	}
	return nil
}

// omsTransition is a fire-and-forget helper for OMS state transitions.
// Failures are logged but do not block order processing.
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
