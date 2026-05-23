package connect

import (
	"context"

	"connectrpc.com/connect"

	v1 "anttrader/gen/proto"
	"anttrader/internal/service"
)

type AIPrimaryService struct {
	aiCfg *service.AIConfigService
}

func NewAIPrimaryService(aiCfg *service.AIConfigService) *AIPrimaryService {
	return &AIPrimaryService{aiCfg: aiCfg}
}

func (s *AIPrimaryService) GetAIPrimary(ctx context.Context, _ *connect.Request[v1.GetAIPrimaryRequest]) (*connect.Response[v1.AIPrimaryResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	providerID, model, err := s.aiCfg.GetPrimary(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.AIPrimaryResponse{ProviderId: providerID, Model: model}), nil
}

func (s *AIPrimaryService) SetAIPrimary(ctx context.Context, req *connect.Request[v1.SetAIPrimaryRequest]) (*connect.Response[v1.AIPrimaryResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.aiCfg.SetPrimary(ctx, userID, req.Msg.ProviderId, req.Msg.Model); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	providerID, model, _ := s.aiCfg.GetPrimary(ctx, userID)
	return connect.NewResponse(&v1.AIPrimaryResponse{ProviderId: providerID, Model: model}), nil
}
