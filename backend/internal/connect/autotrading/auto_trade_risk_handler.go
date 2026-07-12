package autotrading

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
	"alphaforge/internal/risksvc"
)

// GetRiskConfig returns the risk configuration for the given account.
func (s *AutoTradingServer) GetRiskConfig(
	ctx context.Context,
	req *connect.Request[antv1.GetRiskConfigRequest],
) (*connect.Response[antv1.RiskConfig], error) {
	aid, err := uuid.Parse(req.Msg.AccountId)
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
	aid, err := uuid.Parse(req.Msg.AccountId)
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
	balance, _ := strconv.ParseFloat(m.CurrentBalance, 64)
	equity, _ := strconv.ParseFloat(m.CurrentEquity, 64)
	sig := &risksvc.SignalRequest{
		AccountID: m.AccountId,
		Symbol:    m.Symbol,
		Balance:   balance,
		Equity:    equity,
		Positions: int(m.OpenPositions),
	}
	result := s.riskPipe.Process(ctx, sig)

	// Resolve effective risk limits: account-level RiskConfig → user-level GlobalSettings → defaults.
	limits := s.resolveRiskLimit(ctx, m.AccountId)

	return connect.NewResponse(&antv1.CheckRiskLimitsResponse{
		Allowed:            result.Allowed,
		IsWithinLimits:     result.Allowed,
		Reason:             result.Reason,
		MaxPositions:       int32(limits.maxPositions),
		PositionCount:      int32(sig.Positions),
		DrawdownPercent:    "0",
		MaxDrawdownPercent: limits.maxDrawdownPercent.String(),
	}), nil
}

// CalculatePositionSize computes optimal position size based on risk parameters.
func (s *AutoTradingServer) CalculatePositionSize(
	ctx context.Context,
	req *connect.Request[antv1.CalculatePositionSizeRequest],
) (*connect.Response[antv1.CalculatePositionSizeResponse], error) {
	m := req.Msg
	balanceDec := decimal.RequireFromString(m.AccountBalance)
	if balanceDec.LessThanOrEqual(decimal.Zero) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("account_balance must be positive"))
	}
	riskPctDec := decimal.RequireFromString(m.RiskPercent)
	if riskPctDec.LessThanOrEqual(decimal.Zero) {
		riskPctDec = decimal.NewFromFloat(2.0)
	}
	slPipsDec := decimal.RequireFromString(m.StopLossPips)
	if slPipsDec.LessThanOrEqual(decimal.Zero) {
		slPipsDec = decimal.NewFromInt(20)
	}

	pipValueDec := decimal.NewFromFloat(10.0) // standard lots
	hundred := decimal.NewFromInt(100)

	riskAmountDec := balanceDec.Mul(riskPctDec).Div(hundred)
	volumeDec := riskAmountDec.Div(slPipsDec.Mul(pipValueDec))

	return connect.NewResponse(&antv1.CalculatePositionSizeResponse{
		Volume:     volumeDec.String(),
		RiskAmount: riskAmountDec.String(),
		PipValue:   pipValueDec.String(),
		MinVolume:  "0.01",
		MaxVolume:  "100",
	}), nil
}

// --- effective risk limits with fallback chain ---

// effectiveLimits holds resolved risk parameters after the fallback chain:
// account RiskConfig → user GlobalSettings → system defaults.
type effectiveLimits struct {
	maxRiskPercent     decimal.Decimal
	maxDrawdownPercent decimal.Decimal
	maxPositions       int
	maxLotSize         decimal.Decimal
}

// resolveRiskLimit loads the effective risk limits for an account.
// Returns defaults if neither RiskConfig nor GlobalSettings exist.
func (s *AutoTradingServer) resolveRiskLimit(ctx context.Context, accountID string) *effectiveLimits {
	defaults := &effectiveLimits{
		maxRiskPercent:     decimal.NewFromFloat(2.0),
		maxDrawdownPercent: decimal.NewFromFloat(10.0),
		maxPositions:       5,
		maxLotSize:         decimal.NewFromFloat(100.0),
	}

	aid, err := uuid.Parse(accountID)
	if err != nil {
		return defaults
	}

	rc, _ := s.autoRepo.GetRiskConfigByAccountID(ctx, aid)
	uid := s.userID(ctx)
	gs, _ := s.autoRepo.GetGlobalSettingsByUserID(ctx, uid)

	limits := *defaults // copy

	// Account-level RiskConfig takes precedence.
	if rc != nil {
		if rc.MaxRiskPercent.GreaterThan(decimal.Zero) {
			limits.maxRiskPercent = rc.MaxRiskPercent
		}
		if rc.MaxDrawdownPercent.GreaterThan(decimal.Zero) {
			limits.maxDrawdownPercent = rc.MaxDrawdownPercent
		}
		if rc.MaxPositions > 0 {
			limits.maxPositions = rc.MaxPositions
		}
		if rc.MaxLotSize.GreaterThan(decimal.Zero) {
			limits.maxLotSize = rc.MaxLotSize
		}
	}

	// Fallback to user-level GlobalSettings for any remaining zero/default values.
	if gs != nil {
		if limits.maxRiskPercent.Equal(defaults.maxRiskPercent) && gs.MaxRiskPercent.GreaterThan(decimal.Zero) {
			limits.maxRiskPercent = gs.MaxRiskPercent
		}
		if limits.maxDrawdownPercent.Equal(defaults.maxDrawdownPercent) && gs.MaxDrawdownPercent.GreaterThan(decimal.Zero) {
			limits.maxDrawdownPercent = gs.MaxDrawdownPercent
		}
		if limits.maxPositions == defaults.maxPositions && gs.MaxPositions > 0 {
			limits.maxPositions = gs.MaxPositions
		}
		if limits.maxLotSize.Equal(defaults.maxLotSize) && gs.MaxLotSize.GreaterThan(decimal.Zero) {
			limits.maxLotSize = gs.MaxLotSize
		}
	}

	return &limits
}
