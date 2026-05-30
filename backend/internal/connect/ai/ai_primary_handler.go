package ai

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	systemai "anttrader/internal/service/systemai"
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

func (s *AIPrimaryServer) userID(ctx context.Context) (uuid.UUID, error) {
	return uuid.Parse(interceptor.GetUserID(ctx))
}

func (s *AIPrimaryServer) GetAIPrimary(ctx context.Context, req *connect.Request[antv1.GetAIPrimaryRequest]) (*connect.Response[antv1.AIPrimaryResponse], error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	rows, err := s.systemSvc.List(ctx, uid)
	if err != nil {
		return nil, err
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
	return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("no enabled AI config"))
}

func (s *AIPrimaryServer) SetAIPrimary(ctx context.Context, req *connect.Request[antv1.SetAIPrimaryRequest]) (*connect.Response[antv1.AIPrimaryResponse], error) {
	uid, err := s.userID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if err := s.systemSvc.SetAIPrimary(ctx, uid, req.Msg.ProviderId, req.Msg.Model); err != nil {
		s.log.Error("SetAIPrimary transaction failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}
	return connect.NewResponse(&antv1.AIPrimaryResponse{
		ProviderId: req.Msg.ProviderId,
		Model:      req.Msg.Model,
	}), nil
}
