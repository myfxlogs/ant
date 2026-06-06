package strategy

import (
	"context"
	"fmt"
	"time"
	"google.golang.org/protobuf/proto"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	"anttrader/internal/pglisten"
)

// StrategyExperimentServer implements ant.v1.StrategyExperimentServiceHandler.
type StrategyExperimentServer struct {
	repo *repository.StrategyExperimentRepository
	log  *zap.Logger
	pgListen *pglisten.Listener
}

var _ antv1c.StrategyExperimentServiceHandler = (*StrategyExperimentServer)(nil)

func NewStrategyExperimentServer(repo *repository.StrategyExperimentRepository, log *zap.Logger) *StrategyExperimentServer {
	return &StrategyExperimentServer{repo: repo, log: log}
}

func (s *StrategyExperimentServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

func expToProto(e *repository.StrategyExperiment) *antv1.StrategyExperiment {
	p := &antv1.StrategyExperiment{
		Id:             e.ID.String(),
		UserId:         e.UserID.String(),
		Status:         e.Status,
		SearchMethod:   e.SearchMethod,
		MaxCandidates:  int32(e.MaxCandidates),
		Objective:      e.Objective,
		MarketRegimeRef: e.MarketRegimeRef,
		CreatedAt:      timestamppb.New(e.CreatedAt),
	}
	if e.BaseTemplateID != nil {
		p.BaseTemplateId = e.BaseTemplateID.String()
	}
	if e.BestCandidateID != nil {
		p.BestCandidateId = e.BestCandidateID.String()
	}
	if e.JobID != nil {
		p.JobId = e.JobID.String()
	}
	if e.FinishedAt != nil {
		p.FinishedAt = timestamppb.New(*e.FinishedAt)
	}
	p.ParameterSpace = spaceProtoToStruct(e.ParameterSpace)
	return p
}

func candidateToProto(c *repository.StrategyExperimentCandidate) *antv1.StrategyExperimentCandidate {
	p := &antv1.StrategyExperimentCandidate{
		Id:            c.ID.String(),
		ExperimentId:  c.ExperimentID.String(),
		DraftCodeRef:  c.DraftCodeRef,
		Score:         c.Score,
		Grade:         c.Grade,
		Rank:          int32(c.Rank),
		Summary:       c.Summary,
		Recommendation: c.Recommendation,
		CreatedAt:     timestamppb.New(c.CreatedAt),
	}
	if c.BacktestRunID != nil {
		p.BacktestRunId = c.BacktestRunID.String()
	}
	p.Parameters = paramsProtoToStruct(c.Parameters)
	p.ScoreComponents = scoreProtoToStruct(c.ScoreComponents)
	return p
}

func (s *StrategyExperimentServer) SubmitStrategyExperiment(ctx context.Context, req *connect.Request[antv1.SubmitStrategyExperimentRequest]) (*connect.Response[antv1.SubmitStrategyExperimentResponse], error) {
	uid := s.userID(ctx)
	exp := &repository.StrategyExperiment{
		UserID:        uid,
		SearchMethod:  req.Msg.SearchMethod,
		MaxCandidates: int(req.Msg.MaxCandidates),
		Objective:     req.Msg.Objective,
		StrategyCode:  req.Msg.StrategyCode,
		Symbol:        req.Msg.Symbol,
		Timeframe:     req.Msg.Timeframe,
		FromTsUnixMs:  req.Msg.FromTsUnixMs,
		ToTsUnixMs:    req.Msg.ToTsUnixMs,
	}
	if req.Msg.BaseTemplateId != "" {
		tid, err := uuid.Parse(req.Msg.BaseTemplateId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		exp.BaseTemplateID = &tid
	}
	if req.Msg.ParameterSpace != nil {
		b, err := proto.Marshal(req.Msg.ParameterSpace)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal parameter space: %w", err))
		}
		exp.ParameterSpace = b
	}
	if err := s.repo.Create(ctx, exp); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.SubmitStrategyExperimentResponse{
		Experiment: expToProto(exp),
		JobId:      exp.ID.String(),
	}), nil
}

func (s *StrategyExperimentServer) GetStrategyExperiment(ctx context.Context, req *connect.Request[antv1.GetStrategyExperimentRequest]) (*connect.Response[antv1.StrategyExperiment], error) {
	uid := s.userID(ctx)
	id, err := uuid.Parse(req.Msg.ExperimentId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	exp, err := s.repo.Get(ctx, uid, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(expToProto(exp)), nil
}

func (s *StrategyExperimentServer) ListStrategyExperiments(ctx context.Context, req *connect.Request[antv1.ListStrategyExperimentsRequest]) (*connect.Response[antv1.ListStrategyExperimentsResponse], error) {
	limit := int(req.Msg.Limit)
	offset := int(req.Msg.Offset)
	rows, err := s.repo.List(ctx, s.userID(ctx), limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.StrategyExperiment, len(rows))
	for i := range rows {
		items[i] = expToProto(&rows[i])
	}
	return connect.NewResponse(&antv1.ListStrategyExperimentsResponse{Experiments: items}), nil
}

func (s *StrategyExperimentServer) CancelStrategyExperiment(ctx context.Context, req *connect.Request[antv1.CancelStrategyExperimentRequest]) (*connect.Response[antv1.StrategyExperiment], error) {
	uid := s.userID(ctx)
	id, err := uuid.Parse(req.Msg.ExperimentId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	exp, err := s.repo.Cancel(ctx, uid, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(expToProto(exp)), nil
}

func (s *StrategyExperimentServer) ListExperimentCandidates(ctx context.Context, req *connect.Request[antv1.ListExperimentCandidatesRequest]) (*connect.Response[antv1.ListExperimentCandidatesResponse], error) {
	uid := s.userID(ctx)
	expID, err := uuid.Parse(req.Msg.ExperimentId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rows, err := s.repo.ListCandidates(ctx, uid, expID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.StrategyExperimentCandidate, len(rows))
	for i := range rows {
		items[i] = candidateToProto(&rows[i])
	}
	return connect.NewResponse(&antv1.ListExperimentCandidatesResponse{Candidates: items}), nil
}

func (s *StrategyExperimentServer) GetExperimentCandidate(ctx context.Context, req *connect.Request[antv1.GetExperimentCandidateRequest]) (*connect.Response[antv1.StrategyExperimentCandidate], error) {
	uid := s.userID(ctx)
	id, err := uuid.Parse(req.Msg.CandidateId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	c, err := s.repo.GetCandidate(ctx, uid, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(candidateToProto(c)), nil
}

func (s *StrategyExperimentServer) PromoteCandidateToDraft(ctx context.Context, req *connect.Request[antv1.PromoteCandidateToDraftRequest]) (*connect.Response[antv1.PromoteCandidateToDraftResponse], error) {
	uid := s.userID(ctx)
	candID, err := uuid.Parse(req.Msg.CandidateId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	c, err := s.repo.GetCandidate(ctx, uid, candID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err := s.repo.SetBestCandidate(ctx, c.ExperimentID, candID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.PromoteCandidateToDraftResponse{TemplateId: c.DraftCodeRef}), nil
}

// WatchExperiment streams experiment status changes until completion.
// Push-first architecture: replaces client-side polling with SSE stream.
func (s *StrategyExperimentServer) WatchExperiment(ctx context.Context, req *connect.Request[antv1.WatchExperimentRequest], stream *connect.ServerStream[antv1.WatchExperimentEvent]) error {
	uid := s.userID(ctx)
	expID, err := uuid.Parse(req.Msg.ExperimentId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	prevStatus := ""
	notifCh, listenCancel, _ := s.pgListen.Listen(ctx, "experiment_status")
	if listenCancel != nil {
		defer listenCancel()
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-notifCh:
		case <-ticker.C:
		}

		exp, err := s.repo.Get(ctx, uid, expID)
		if err != nil {
			s.log.Warn("WatchExperiment: transient DB error", zap.Error(err), zap.String("expID", expID.String()))
			continue
		}
		if exp == nil {
			s.log.Warn("WatchExperiment: experiment not found", zap.String("expID", expID.String()))
			continue
		}
		if exp.Status == prevStatus {
			continue
		}
		prevStatus = exp.Status

		event := &antv1.WatchExperimentEvent{
			ExperimentId: expID.String(),
			Status:       exp.Status,
		}

		// On completion, include all candidates
		if exp.Status == "COMPLETED" {
			candidates, err := s.repo.ListCandidates(ctx, uid, expID)
			if err == nil {
				protoCands := make([]*antv1.StrategyExperimentCandidate, 0, len(candidates))
				for _, c := range candidates {
					protoCands = append(protoCands, candidateToProto(&c))
				}
				event.Candidates = protoCands
				event.CandidatesReady = int32(len(protoCands))
			}
		}

		if exp.Status == StatusFailed {
			event.Error = "experiment failed"
		}

		if err := stream.Send(event); err != nil {
			return err
		}

		// Terminal states — close the stream
		if exp.Status == StatusCompleted || exp.Status == StatusFailed {
			return nil
		}
	}
}

func (s *StrategyExperimentServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
