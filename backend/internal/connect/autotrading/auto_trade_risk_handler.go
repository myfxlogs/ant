package autotrading

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/model"
	"anttrader/internal/risksvc"
)

// GetRiskConfig returns the risk configuration for the given account.
func (s *AutoTradingServer) GetRiskConfig(
	ctx context.Context,
	req *connect.Request[antv1.GetRiskConfigRequest],
) (*connect.Response[antv1.RiskConfig], error) {
	aid, err := parseUUID(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rc, err := s.autoRepo.GetRiskConfigByAccountID(ctx, aid)
	if err != nil {
		s.log.Error("get risk config failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(riskConfigToProto(rc)), nil
}

// UpdateRiskConfig updates the risk configuration for an account.
func (s *AutoTradingServer) UpdateRiskConfig(
	ctx context.Context,
	req *connect.Request[antv1.UpdateRiskConfigRequest],
) (*connect.Response[antv1.RiskConfig], error) {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			fmt.Errorf("authentication required"))
	}
	aid, err := parseUUID(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rc, err := s.autoRepo.GetRiskConfigByAccountID(ctx, aid)
	if err != nil {
		rc = model.NewRiskConfig(uid, aid)
		if err := s.autoRepo.CreateRiskConfig(ctx, rc); err != nil {
			s.log.Error("create risk config failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	applyRiskConfig(rc, req.Msg)
	if err := s.autoRepo.UpdateRiskConfig(ctx, rc); err != nil {
		s.log.Error("update risk config failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(riskConfigToProto(rc)), nil
}

// CheckRiskLimits evaluates a proposed trade against the existing risk pipeline.
func (s *AutoTradingServer) CheckRiskLimits(
	ctx context.Context,
	req *connect.Request[antv1.CheckRiskLimitsRequest],
) (*connect.Response[antv1.CheckRiskLimitsResponse], error) {
	m := req.Msg
	if s.riskPipe == nil {
		return nil, connect.NewError(connect.CodeUnavailable,
			fmt.Errorf("risk pipeline not configured"))
	}
	sig := &risksvc.SignalRequest{
		AccountID: m.AccountId,
		Symbol:    m.Symbol,
		Balance:   m.CurrentBalance,
		Equity:    m.CurrentEquity,
		Positions: int(m.OpenPositions),
	}
	result := s.riskPipe.Process(ctx, sig)
	return connect.NewResponse(&antv1.CheckRiskLimitsResponse{
		Allowed:          result.Allowed,
		IsWithinLimits:   result.Allowed,
		Reason:           result.Reason,
		MaxPositions:     int32(sig.Positions),
		PositionCount:    int32(sig.Positions),
		DrawdownPercent:  0,
		MaxDrawdownPercent: 10.0,
	}), nil
}

// CalculatePositionSize computes optimal position size based on risk parameters.
func (s *AutoTradingServer) CalculatePositionSize(
	ctx context.Context,
	req *connect.Request[antv1.CalculatePositionSizeRequest],
) (*connect.Response[antv1.CalculatePositionSizeResponse], error) {
	m := req.Msg
	balance := m.AccountBalance
	if balance <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("account_balance must be positive"))
	}
	riskPct := m.RiskPercent
	if riskPct <= 0 {
		riskPct = 2.0
	}
	slPips := m.StopLossPips
	if slPips <= 0 {
		slPips = 20
	}
	pipValue := 10.0 // default for standard lots
	riskAmount := balance * riskPct / 100.0
	volume := riskAmount / (slPips * pipValue)
	return connect.NewResponse(&antv1.CalculatePositionSizeResponse{
		Volume:     volume,
		RiskAmount: riskAmount,
		PipValue:   pipValue,
		MinVolume:  0.01,
		MaxVolume:  100.0,
	}), nil
}
