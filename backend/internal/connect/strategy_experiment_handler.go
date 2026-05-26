package connect

import (
	"context"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
)

// StrategyExperimentServer implements ant.v1.StrategyExperimentServiceHandler.
type StrategyExperimentServer struct{ log *zap.Logger }

var _ antv1c.StrategyExperimentServiceHandler = (*StrategyExperimentServer)(nil)

func NewStrategyExperimentServer(log *zap.Logger) *StrategyExperimentServer {
	return &StrategyExperimentServer{log: log}
}

func (s *StrategyExperimentServer) SubmitStrategyExperiment(ctx context.Context, req *connect.Request[antv1.SubmitStrategyExperimentRequest]) (*connect.Response[antv1.SubmitStrategyExperimentResponse], error) {
	id := uuid.New().String()
	now := timestamppb.New(time.Now())
	return connect.NewResponse(&antv1.SubmitStrategyExperimentResponse{
		Experiment: &antv1.StrategyExperiment{
			Id:             id,
			UserId:         "mock-user",
			BaseTemplateId: req.Msg.BaseTemplateId,
			Status:         "running",
			SearchMethod:   req.Msg.SearchMethod,
			MaxCandidates:  req.Msg.MaxCandidates,
			Objective:      req.Msg.Objective,
			CreatedAt:      now,
		},
		JobId: uuid.New().String(),
	}), nil
}

func (s *StrategyExperimentServer) GetStrategyExperiment(ctx context.Context, req *connect.Request[antv1.GetStrategyExperimentRequest]) (*connect.Response[antv1.StrategyExperiment], error) {
	return connect.NewResponse(&antv1.StrategyExperiment{
		Id:             req.Msg.ExperimentId,
		UserId:         "mock-user",
		BaseTemplateId: "tpl-001",
		Status:         "completed",
		SearchMethod:   "grid_search",
		MaxCandidates:  10,
		Objective:      "sharpe_ratio",
		CreatedAt:      timestamppb.New(time.Now().Add(-1 * time.Hour)),
		FinishedAt:     timestamppb.New(time.Now()),
	}), nil
}

func (s *StrategyExperimentServer) ListStrategyExperiments(ctx context.Context, req *connect.Request[antv1.ListStrategyExperimentsRequest]) (*connect.Response[antv1.ListStrategyExperimentsResponse], error) {
	return connect.NewResponse(&antv1.ListStrategyExperimentsResponse{
		Experiments: []*antv1.StrategyExperiment{},
	}), nil
}

func (s *StrategyExperimentServer) CancelStrategyExperiment(ctx context.Context, req *connect.Request[antv1.CancelStrategyExperimentRequest]) (*connect.Response[antv1.StrategyExperiment], error) {
	return connect.NewResponse(&antv1.StrategyExperiment{
		Id:     req.Msg.ExperimentId,
		Status: "cancelled",
	}), nil
}

func (s *StrategyExperimentServer) ListExperimentCandidates(ctx context.Context, req *connect.Request[antv1.ListExperimentCandidatesRequest]) (*connect.Response[antv1.ListExperimentCandidatesResponse], error) {
	return connect.NewResponse(&antv1.ListExperimentCandidatesResponse{
		Candidates: []*antv1.StrategyExperimentCandidate{},
	}), nil
}

func (s *StrategyExperimentServer) GetExperimentCandidate(ctx context.Context, req *connect.Request[antv1.GetExperimentCandidateRequest]) (*connect.Response[antv1.StrategyExperimentCandidate], error) {
	return connect.NewResponse(&antv1.StrategyExperimentCandidate{
		Id:             req.Msg.CandidateId,
		ExperimentId:   "exp-001",
		Score:          0.85,
		Grade:          "A",
		Rank:           1,
		Summary:        "参数优化候选方案，夏普比率 1.8",
		Recommendation: "建议采用该候选参数",
	}), nil
}

func (s *StrategyExperimentServer) PromoteCandidateToDraft(ctx context.Context, req *connect.Request[antv1.PromoteCandidateToDraftRequest]) (*connect.Response[antv1.PromoteCandidateToDraftResponse], error) {
	return connect.NewResponse(&antv1.PromoteCandidateToDraftResponse{
		TemplateId: uuid.New().String(),
	}), nil
}
