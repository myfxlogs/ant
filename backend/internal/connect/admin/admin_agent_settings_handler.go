package admin

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/agent"
)

// AdminAgentSettingsServer implements AdminAgentSettingsService (ADR-0025 §5.4).
type AdminAgentSettingsServer struct {
	settings *agent.SettingsStore
	log      *zap.Logger
}

var _ antv1c.AdminAgentSettingsServiceHandler = (*AdminAgentSettingsServer)(nil)

func NewAdminAgentSettingsServer(settings *agent.SettingsStore, log *zap.Logger) *AdminAgentSettingsServer {
	return &AdminAgentSettingsServer{settings: settings, log: log}
}

func (s *AdminAgentSettingsServer) GetManagedSettings(
	ctx context.Context,
	_ *connect.Request[antv1.GetManagedSettingsRequest],
) (*connect.Response[antv1.GetManagedSettingsResponse], error) {
	list, err := s.settings.ListManagedSettings(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	entries := make([]*antv1.ManagedSettingEntry, 0, len(list))
	for _, item := range list {
		entries = append(entries, &antv1.ManagedSettingEntry{
			Key:   item.Key,
			Value: item.Value,
		})
	}
	return connect.NewResponse(&antv1.GetManagedSettingsResponse{Settings: entries}), nil
}

func (s *AdminAgentSettingsServer) SetManagedSetting(
	ctx context.Context,
	req *connect.Request[antv1.SetManagedSettingRequest],
) (*connect.Response[antv1.SetManagedSettingResponse], error) {
	if req.Msg.Key == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("key is required"))
	}
	if err := s.settings.SetManagedSetting(ctx, req.Msg.Key, req.Msg.Value); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("admin: managed setting set",
		zap.String("actor", getActorID(ctx).String()),
		zap.String("key", req.Msg.Key))
	return connect.NewResponse(&antv1.SetManagedSettingResponse{Success: true}), nil
}

func (s *AdminAgentSettingsServer) DeleteManagedSetting(
	ctx context.Context,
	req *connect.Request[antv1.DeleteManagedSettingRequest],
) (*connect.Response[antv1.DeleteManagedSettingResponse], error) {
	if req.Msg.Key == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("key is required"))
	}
	if err := s.settings.DeleteManagedSetting(ctx, req.Msg.Key); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("admin: managed setting deleted",
		zap.String("actor", getActorID(ctx).String()),
		zap.String("key", req.Msg.Key))
	return connect.NewResponse(&antv1.DeleteManagedSettingResponse{Success: true}), nil
}
