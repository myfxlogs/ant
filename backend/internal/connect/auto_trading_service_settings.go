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

func (s *AutoTradingService) GetGlobalSettings(ctx context.Context, req *connect.Request[v1.GetGlobalSettingsRequest]) (*connect.Response[v1.GlobalSettings], error) {
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

	settings, err := s.autoTradingSvc.GetGlobalSettings(ctx, uid)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.GlobalSettings{
		Id:                 settings.ID.String(),
		UserId:             settings.UserID.String(),
		AutoTradeEnabled:   settings.AutoTradeEnabled,
		MaxRiskPercent:     settings.MaxRiskPercent,
		MaxPositions:       int32(settings.MaxPositions),
		MaxLotSize:         settings.MaxLotSize,
		MaxDailyLoss:       settings.MaxDailyLoss,
		MaxDrawdownPercent: settings.MaxDrawdownPercent,
		CreatedAt:          timestamppb.New(settings.CreatedAt),
		UpdatedAt:          timestamppb.New(settings.UpdatedAt),
	}), nil
}

func (s *AutoTradingService) UpdateGlobalSettings(ctx context.Context, req *connect.Request[v1.UpdateGlobalSettingsRequest]) (*connect.Response[v1.GlobalSettings], error) {
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

	settings := &model.GlobalSettings{}
	if req.Msg.AutoTradeEnabled != nil {
		settings.AutoTradeEnabled = *req.Msg.AutoTradeEnabled
	}
	if req.Msg.MaxRiskPercent != nil {
		settings.MaxRiskPercent = *req.Msg.MaxRiskPercent
	}
	if req.Msg.MaxPositions != nil {
		settings.MaxPositions = int(*req.Msg.MaxPositions)
	}
	if req.Msg.MaxLotSize != nil {
		settings.MaxLotSize = *req.Msg.MaxLotSize
	}
	if req.Msg.MaxDailyLoss != nil {
		settings.MaxDailyLoss = *req.Msg.MaxDailyLoss
	}
	if req.Msg.MaxDrawdownPercent != nil {
		settings.MaxDrawdownPercent = *req.Msg.MaxDrawdownPercent
	}

	err = s.autoTradingSvc.UpdateGlobalSettings(ctx, uid, settings)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.GlobalSettings{
		AutoTradeEnabled:   settings.AutoTradeEnabled,
		MaxRiskPercent:     settings.MaxRiskPercent,
		MaxPositions:       int32(settings.MaxPositions),
		MaxLotSize:         settings.MaxLotSize,
		MaxDailyLoss:       settings.MaxDailyLoss,
		MaxDrawdownPercent: settings.MaxDrawdownPercent,
	}), nil
}

func (s *AutoTradingService) ToggleAutoTrade(ctx context.Context, req *connect.Request[v1.ToggleAutoTradeRequest]) (*connect.Response[v1.ToggleAutoTradeResponse], error) {
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

	err = s.autoTradingSvc.ToggleAutoTrade(ctx, uid, req.Msg.Enabled)
	if err != nil {
		return nil, err
	}

	msgKey := "errors.auto_trading_disabled"
	if req.Msg.Enabled {
		msgKey = "errors.auto_trading_enabled"
	}
	return connect.NewResponse(&v1.ToggleAutoTradeResponse{
		Success: true,
		Message: msgKey,
	}), nil
}
