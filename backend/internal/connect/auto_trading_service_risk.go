package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
)

func (s *AutoTradingService) GetRiskConfig(ctx context.Context, req *connect.Request[v1.GetRiskConfigRequest]) (*connect.Response[v1.RiskConfig], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeRead); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var accountID uuid.UUID
	if req.Msg.AccountId != "" {
		accountID, err = uuid.Parse(req.Msg.AccountId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	config, err := s.autoTradingSvc.GetRiskConfig(ctx, uid, accountID)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.RiskConfig{
		Id:                 config.ID.String(),
		UserId:             config.UserID.String(),
		AccountId:          config.AccountID.String(),
		MaxPositions:       int32(config.MaxPositions),
		MaxLotSize:         config.MaxLotSize,
		MaxDailyLoss:       config.MaxDailyLoss,
		DailyLossUsed:      config.DailyLossUsed,
		MaxDrawdownPercent: config.MaxDrawdownPercent,
		MaxRiskPercent:     config.MaxRiskPercent,
		CreatedAt:          timestamppb.New(config.CreatedAt),
		UpdatedAt:          timestamppb.New(config.UpdatedAt),
	}), nil
}

func (s *AutoTradingService) UpdateRiskConfig(ctx context.Context, req *connect.Request[v1.UpdateRiskConfigRequest]) (*connect.Response[v1.RiskConfig], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeWrite); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var accountID uuid.UUID
	if req.Msg.AccountId != "" {
		accountID, err = uuid.Parse(req.Msg.AccountId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	config := &model.RiskConfig{UserID: uid, AccountID: accountID}
	if req.Msg.MaxPositions != nil {
		config.MaxPositions = int(*req.Msg.MaxPositions)
	}
	if req.Msg.MaxLotSize != nil {
		config.MaxLotSize = *req.Msg.MaxLotSize
	}
	if req.Msg.MaxDailyLoss != nil {
		config.MaxDailyLoss = *req.Msg.MaxDailyLoss
	}
	if req.Msg.MaxDrawdownPercent != nil {
		config.MaxDrawdownPercent = *req.Msg.MaxDrawdownPercent
	}
	if req.Msg.MaxRiskPercent != nil {
		config.MaxRiskPercent = *req.Msg.MaxRiskPercent
	}

	err = s.autoTradingSvc.UpdateRiskConfig(ctx, uid, accountID, config)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.RiskConfig{
		MaxPositions:       int32(config.MaxPositions),
		MaxLotSize:         config.MaxLotSize,
		MaxDailyLoss:       config.MaxDailyLoss,
		MaxDrawdownPercent: config.MaxDrawdownPercent,
		MaxRiskPercent:     config.MaxRiskPercent,
	}), nil
}

func (s *AutoTradingService) CheckRiskLimits(ctx context.Context, req *connect.Request[v1.CheckRiskLimitsRequest]) (*connect.Response[v1.CheckRiskLimitsResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeRead); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	_, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	result, err := s.autoTradingSvc.CheckRiskLimits(ctx, &model.RiskCheckRequest{
		AccountID:      accountID,
		Symbol:         req.Msg.Symbol,
		Volume:         req.Msg.Volume,
		CurrentBalance: req.Msg.CurrentBalance,
		CurrentEquity:  req.Msg.CurrentEquity,
		OpenPositions:  int(req.Msg.OpenPositions),
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.CheckRiskLimitsResponse{
		Allowed:            result.Allowed,
		IsWithinLimits:     result.IsWithinLimits,
		Reason:             result.Reason,
		MaxPositions:       int32(result.MaxPositions),
		PositionCount:      int32(result.PositionCount),
		DailyLossLimit:     result.DailyLossLimit,
		DailyLossUsed:      result.DailyLossUsed,
		MaxDrawdownPercent: result.MaxDrawdownPercent,
		DrawdownPercent:    result.DrawdownPercent,
	}), nil
}

func (s *AutoTradingService) CalculatePositionSize(ctx context.Context, req *connect.Request[v1.CalculatePositionSizeRequest]) (*connect.Response[v1.CalculatePositionSizeResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeRead); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	_, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	result, err := s.autoTradingSvc.CalculatePositionSize(ctx, &model.PositionSizingRequest{
		AccountID:       accountID,
		Symbol:          req.Msg.Symbol,
		AccountBalance:  req.Msg.AccountBalance,
		StopLossPips:    req.Msg.StopLossPips,
		RiskPercent:     req.Msg.RiskPercent,
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.CalculatePositionSizeResponse{
		Volume:     result.Volume,
		RiskAmount: result.RiskAmount,
		PipValue:   result.PipValue,
		MinVolume:  result.MinVolume,
		MaxVolume:  result.MaxVolume,
	}), nil
}

func (s *AutoTradingService) GetAutoTradingStatus(ctx context.Context, req *connect.Request[v1.GetAutoTradingStatusRequest]) (*connect.Response[v1.AutoTradingStatus], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeRead); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	status, err := s.autoTradingSvc.GetAutoTradingStatus(ctx, uid)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.AutoTradingStatus{
		GlobalEnabled:    status.GlobalEnabled,
		ActiveStrategies: int32(status.ActiveStrategies),
		PendingSignals:   int32(status.PendingSignals),
		TodayExecutions:  int32(status.TodayExecutions),
		TodayProfit:      status.TodayProfit,
	}), nil
}
