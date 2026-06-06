package strategy

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
)

// ObjectiveScoreServer handles CalculateObjectiveScore via ConnectRPC to Python strategy-service.
// Previously bridged through REST (/api/objective-score); now uses native ConnectRPC client.
type ObjectiveScoreServer struct {
	client antv1c.ObjectiveScoreServiceClient // ConnectRPC to Python service
	log    *zap.Logger
}

var _ antv1c.ObjectiveScoreServiceHandler = (*ObjectiveScoreServer)(nil)

// NewObjectiveScoreServer creates a handler that delegates scoring to the Python service via ConnectRPC.
func NewObjectiveScoreServer(client antv1c.ObjectiveScoreServiceClient, log *zap.Logger) *ObjectiveScoreServer {
	return &ObjectiveScoreServer{client: client, log: log}
}

// --- Handler ---

func (s *ObjectiveScoreServer) CalculateObjectiveScore(
	ctx context.Context,
	req *connect.Request[antv1.ObjectiveScoreRequest],
) (*connect.Response[antv1.ObjectiveScoreResponse], error) {
	// Authorization verified by authInterceptor.
	resp, err := s.client.CalculateObjectiveScore(ctx, connect.NewRequest(req.Msg))
	if err != nil {
		s.log.Warn("objective score failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeUnavailable,
			fmt.Errorf("objective score service: %w", err))
	}
	return connect.NewResponse(resp.Msg), nil
}
