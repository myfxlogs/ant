package strategy

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
)

// ListStrategyRuns returns recent live/paper strategy run records for the authenticated user.
func (s *StrategyExecutionServer) ListStrategyRuns(ctx context.Context, req *connect.Request[antv1.ListStrategyRunsRequest]) (*connect.Response[antv1.ListStrategyRunsResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	if s.runRepo == nil {
		return connect.NewResponse(&antv1.ListStrategyRunsResponse{}), nil
	}

	limit := int(req.Msg.GetLimit())
	offset := int(req.Msg.GetOffset())

	var runs []*repository.StrategyRun
	if req.Msg.GetAccountId() != "" {
		runs, err = s.runRepo.ListByAccount(ctx, req.Msg.GetAccountId(), limit)
	} else {
		runs, err = s.runRepo.ListByUser(ctx, uid, limit, offset)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list strategy runs: %w", err))
	}

	pbRuns := make([]*antv1.StrategyRun, len(runs))
	for i, r := range runs {
		pbRuns[i] = strategyRunToProto(r)
	}
	return connect.NewResponse(&antv1.ListStrategyRunsResponse{Runs: pbRuns}), nil
}

// GetStrategyRun returns a single strategy run by ID.
func (s *StrategyExecutionServer) GetStrategyRun(ctx context.Context, req *connect.Request[antv1.GetStrategyRunRequest]) (*connect.Response[antv1.GetStrategyRunResponse], error) {
	if _, err := userIDRequire(ctx); err != nil {
		return nil, err
	}
	if s.runRepo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("run repository not configured"))
	}

	runID, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid run_id: %w", err))
	}

	run, err := s.runRepo.GetByID(ctx, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("strategy run not found: %w", err))
	}

	return connect.NewResponse(&antv1.GetStrategyRunResponse{Run: strategyRunToProto(run)}), nil
}

// strategyRunToProto converts a repository StrategyRun to a proto StrategyRun.
func strategyRunToProto(r *repository.StrategyRun) *antv1.StrategyRun {
	pb := &antv1.StrategyRun{
		Id:           r.ID.String(),
		UserId:       r.UserID.String(),
		AccountId:    r.AccountID,
		Symbol:       r.Symbol,
		Timeframe:    r.Timeframe,
		Mode:         r.Mode,
		Status:       r.Status,
		Error:        r.Error,
		TotalSignals: int32(r.TotalSignals),
		StartedAt:    timestamppb.New(r.StartedAt),
	}
	if r.StoppedAt != nil {
		pb.StoppedAt = timestamppb.New(*r.StoppedAt)
	}
	return pb
}
