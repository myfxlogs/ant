package autotrading

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/model"
)

// GetGlobalSettings returns the current user's global auto-trading settings.
// Creates default settings if none exist yet.
func (s *AutoTradingServer) GetGlobalSettings(
	ctx context.Context,
	_ *connect.Request[antv1.GetGlobalSettingsRequest],
) (*connect.Response[antv1.GlobalSettings], error) {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			fmt.Errorf("authentication required"))
	}
	gs, err := s.autoRepo.GetGlobalSettingsByUserID(ctx, uid)
	if err != nil {
		// Create defaults on first access.
		gs = model.NewGlobalSettings(uid)
		if err := s.autoRepo.CreateGlobalSettings(ctx, gs); err != nil {
			s.log.Error("create default global settings failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(globalSettingsToProto(gs)), nil
}

// UpdateGlobalSettings updates the user's global auto-trading settings.
func (s *AutoTradingServer) UpdateGlobalSettings(
	ctx context.Context,
	req *connect.Request[antv1.UpdateGlobalSettingsRequest],
) (*connect.Response[antv1.GlobalSettings], error) {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			fmt.Errorf("authentication required"))
	}
	gs, err := s.autoRepo.GetGlobalSettingsByUserID(ctx, uid)
	if err != nil {
		gs = model.NewGlobalSettings(uid)
		if err := s.autoRepo.CreateGlobalSettings(ctx, gs); err != nil {
			s.log.Error("create global settings for update failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	applyGlobalSettings(gs, req.Msg)
	if err := s.autoRepo.UpdateGlobalSettings(ctx, gs); err != nil {
		s.log.Error("update global settings failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(globalSettingsToProto(gs)), nil
}

// ToggleAutoTrade enables or disables auto-trading for the current user.
func (s *AutoTradingServer) ToggleAutoTrade(
	ctx context.Context,
	req *connect.Request[antv1.ToggleAutoTradeRequest],
) (*connect.Response[antv1.ToggleAutoTradeResponse], error) {
	uid := s.userID(ctx)
	if uid == uuid.Nil {
		return nil, connect.NewError(connect.CodeUnauthenticated,
			fmt.Errorf("authentication required"))
	}
	if err := s.autoRepo.UpdateAutoTradeEnabled(ctx, uid, req.Msg.Enabled); err != nil {
		s.log.Error("toggle auto trade failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	msg := "Auto-trading disabled"
	if req.Msg.Enabled {
		msg = "Auto-trading enabled"
	}
	return connect.NewResponse(&antv1.ToggleAutoTradeResponse{
		Success: true, Message: msg,
	}), nil
}
