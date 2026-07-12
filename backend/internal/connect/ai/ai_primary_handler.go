package ai

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	systemai "alphaforge/internal/service/systemai"
)

// AIPrimaryServer implements ant.v1.AIPrimaryServiceHandler.
type AIPrimaryServer struct {
	systemSvc *systemai.Service
	log       *zap.Logger
}

var _ antv1c.AIPrimaryServiceHandler = (*AIPrimaryServer)(nil)

func NewAIPrimaryServer(systemSvc *systemai.Service, log *zap.Logger) *AIPrimaryServer {
	return &AIPrimaryServer{systemSvc: systemSvc, log: log}
}

func (s *AIPrimaryServer) GetAIPrimary(ctx context.Context, req *connect.Request[antv1.GetAIPrimaryRequest]) (*connect.Response[antv1.AIPrimaryResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	// Return user's saved primary first (persisted across page refreshes).
	if pid, model, perr := s.systemSvc.GetAIPrimary(ctx, uid); perr == nil && pid != "" {
		return connect.NewResponse(&antv1.AIPrimaryResponse{ProviderId: pid, Model: model}), nil
	}
	rows, err := s.systemSvc.List(ctx, uid)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("errors.ai.internal_error"))
	}
	for _, r := range rows {
		if r.Enabled {
			for _, p := range r.PrimaryFor {
				if p == "chat" {
					return connect.NewResponse(&antv1.AIPrimaryResponse{
						ProviderId: r.ProviderID,
						Model:      r.DefaultModel,
					}), nil
				}
			}
		}
	}
	// Fallback: first enabled config
	for _, r := range rows {
		if r.Enabled {
			return connect.NewResponse(&antv1.AIPrimaryResponse{
				ProviderId: r.ProviderID,
				Model:      r.DefaultModel,
			}), nil
		}
	}
	// No AI configured — normal state, return empty response.
	return connect.NewResponse(&antv1.AIPrimaryResponse{}), nil
}

func (s *AIPrimaryServer) SetAIPrimary(ctx context.Context, req *connect.Request[antv1.SetAIPrimaryRequest]) (*connect.Response[antv1.AIPrimaryResponse], error) {
	uid, err := userIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.ProviderId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("provider_id is required"))
	}
	if req.Msg.Model == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("model is required"))
	}
	if err := s.systemSvc.SetAIPrimary(ctx, uid, req.Msg.ProviderId, req.Msg.Model); err != nil {
		s.log.Error("SetAIPrimary transaction failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("errors.ai.internal_error"))
	}
	return connect.NewResponse(&antv1.AIPrimaryResponse{
		ProviderId: req.Msg.ProviderId,
		Model:      req.Msg.Model,
	}), nil
}
