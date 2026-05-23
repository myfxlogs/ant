package connect

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
)

func (s *AdminService) ListConfigs(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.ListConfigsResponse], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	configs, err := s.adminSvc.ListConfigs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoConfigs := make([]*v1.SystemConfig, len(configs))
	for i, c := range configs {
		protoConfigs[i] = convertSystemConfigToProto(c)
	}

	return connect.NewResponse(&v1.ListConfigsResponse{
		Configs: protoConfigs,
	}), nil
}

func (s *AdminService) GetConfig(ctx context.Context, req *connect.Request[v1.GetConfigRequest]) (*connect.Response[v1.SystemConfig], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	config, err := s.adminSvc.GetConfig(ctx, req.Msg.Key)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(convertSystemConfigToProto(config)), nil
}

func (s *AdminService) SetConfig(ctx context.Context, req *connect.Request[v1.SetConfigRequest]) (*connect.Response[emptypb.Empty], error) {
	operatorID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.adminSvc.SetConfig(ctx, req.Msg.Key, req.Msg.Value, req.Msg.Description, operatorID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AdminService) ToggleConfigEnabled(ctx context.Context, req *connect.Request[v1.ToggleConfigEnabledRequest]) (*connect.Response[emptypb.Empty], error) {
	operatorID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.adminSvc.SetConfigEnabled(ctx, req.Msg.Key, req.Msg.Enabled, operatorID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func convertSystemConfigToProto(c *model.SystemConfig) *v1.SystemConfig {
	enabled := true
	if c.Enabled != nil {
		enabled = *c.Enabled
	}
	return &v1.SystemConfig{
		Key:         c.Key,
		Value:       c.Value,
		Description: c.Description,
		Enabled:     enabled,
	}
}
