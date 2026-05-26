package ai

import (
	"context"

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

func (s *AIPrimaryServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

func (s *AIPrimaryServer) GetAIPrimary(ctx context.Context, req *connect.Request[antv1.GetAIPrimaryRequest]) (*connect.Response[antv1.AIPrimaryResponse], error) {
	rows, err := s.systemSvc.List(ctx, s.userID(ctx))
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
	return connect.NewResponse(&antv1.AIPrimaryResponse{}), nil
}

func (s *AIPrimaryServer) SetAIPrimary(ctx context.Context, req *connect.Request[antv1.SetAIPrimaryRequest]) (*connect.Response[antv1.AIPrimaryResponse], error) {
	uid := s.userID(ctx)
	rows, err := s.systemSvc.List(ctx, uid)
	if err != nil {
		return nil, err
	}
	// Clear "chat" from all providers, set it on the selected one.
	for _, r := range rows {
		newPrimaryFor := make([]string, 0, len(r.PrimaryFor))
		for _, p := range r.PrimaryFor {
			if p != "chat" {
				newPrimaryFor = append(newPrimaryFor, p)
			}
		}
		if r.ProviderID == req.Msg.ProviderId {
			newPrimaryFor = append(newPrimaryFor, "chat")
		}
		r.PrimaryFor = newPrimaryFor
		if err := s.systemSvc.UpdateConfig(ctx, r, uid.String()); err != nil {
			return nil, err
		}
	}
	return connect.NewResponse(&antv1.AIPrimaryResponse{
		ProviderId: req.Msg.ProviderId,
		Model:      req.Msg.Model,
	}), nil
}
