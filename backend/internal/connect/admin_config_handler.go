package connect

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/model"
	"anttrader/internal/repository"
)

type AdminConfigServer struct {
	repo *repository.AdminRepository
	log  *zap.Logger
}

var _ antv1c.AdminConfigServiceHandler = (*AdminConfigServer)(nil)

func NewAdminConfigServer(repo *repository.AdminRepository, log *zap.Logger) *AdminConfigServer {
	return &AdminConfigServer{repo: repo, log: log}
}

func configToProto(c *model.SystemConfig) *antv1.SystemConfig {
	p := &antv1.SystemConfig{
		Key:         c.Key,
		Value:       c.Value,
		Description: c.Description,
		CreatedAt:   timestamppb.New(c.CreatedAt),
		UpdatedAt:   timestamppb.New(c.UpdatedAt),
	}
	if c.Enabled != nil {
		p.Enabled = *c.Enabled
	}
	return p
}

func (s *AdminConfigServer) ListConfigs(ctx context.Context, _ *connect.Request[antv1.ListConfigsRequest]) (*connect.Response[antv1.ListConfigsResponse], error) {
	configs, err := s.repo.ListConfigs(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*antv1.SystemConfig, len(configs))
	for i, c := range configs {
		items[i] = configToProto(c)
	}
	return connect.NewResponse(&antv1.ListConfigsResponse{Configs: items}), nil
}

func (s *AdminConfigServer) SetConfig(ctx context.Context, req *connect.Request[antv1.SetConfigRequest]) (*connect.Response[antv1.SetConfigResponse], error) {
	if err := s.repo.SetConfig(ctx, req.Msg.Key, req.Msg.Value, req.Msg.Description); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.SetConfigResponse{}), nil
}

func (s *AdminConfigServer) ToggleConfigEnabled(ctx context.Context, req *connect.Request[antv1.ToggleConfigEnabledRequest]) (*connect.Response[antv1.ToggleConfigEnabledResponse], error) {
	if err := s.repo.SetConfigEnabled(ctx, req.Msg.Key, req.Msg.Enabled); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.ToggleConfigEnabledResponse{}), nil
}
