package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

type StrategyExperimentService struct {
	repo        *repository.StrategyExperimentRepository
	jobRepo     *repository.JobRepository
	templateSvc *service.StrategyTemplateService
}

func NewStrategyExperimentService(repo *repository.StrategyExperimentRepository, jobRepo *repository.JobRepository, templateSvc *service.StrategyTemplateService) *StrategyExperimentService {
	return &StrategyExperimentService{repo: repo, jobRepo: jobRepo, templateSvc: templateSvc}
}

func (s *StrategyExperimentService) SubmitStrategyExperiment(ctx context.Context, req *connect.Request[v1.SubmitStrategyExperimentRequest]) (*connect.Response[v1.SubmitStrategyExperimentResponse], error) {
	if s == nil || s.repo == nil || s.jobRepo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("strategy experiment service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	baseTemplateID, err := parseRequestUUID(req.Msg.GetBaseTemplateId())
	if err != nil {
		return nil, err
	}
	if s.templateSvc != nil {
		tpl, getErr := s.templateSvc.GetTemplate(ctx, baseTemplateID)
		if getErr != nil {
			return nil, connect.NewError(connect.CodeNotFound, getErr)
		}
		if tpl.UserID != userID && !tpl.IsPublic && !tpl.IsSystem {
			return nil, connect.NewError(connect.CodePermissionDenied, service.ErrTemplateUnauthorized)
		}
	}
	space := req.Msg.GetParameterSpace()
	spaceJSON, err := json.Marshal(space)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	maxCandidates := int(req.Msg.GetMaxCandidates())
	if maxCandidates <= 0 {
		maxCandidates = 12
	}
	if maxCandidates > 50 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("max_candidates must be <= 50"))
	}
	now := time.Now().UTC()
	job := &repository.Job{UserID: userID, Kind: "strategy_experiment", Status: "succeeded", Progress: 1, Stage: "candidate_scored", RequestSummary: fmt.Sprintf("base_template=%s candidates=%d", baseTemplateID.String(), maxCandidates), ResultSummary: "strategy experiment candidates generated", IdempotencyKey: strings.TrimSpace(req.Msg.GetIdempotencyKey()), CreatedAt: now, StartedAt: &now, FinishedAt: &now}
	if err := s.jobRepo.CreateJob(ctx, job); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	finishedAt := now
	exp := &repository.StrategyExperiment{UserID: userID, BaseTemplateID: &baseTemplateID, Status: "SUCCEEDED", ParameterSpace: spaceJSON, SearchMethod: normalizeSearchMethod(req.Msg.GetSearchMethod()), MaxCandidates: maxCandidates, Objective: strings.TrimSpace(req.Msg.GetObjective()), JobID: &job.ID, CreatedAt: now, FinishedAt: &finishedAt}
	if exp.Objective == "" {
		exp.Objective = "balanced"
	}
	if err := s.repo.Create(ctx, exp); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	spaceMap := map[string]any{}
	if space != nil {
		spaceMap = space.AsMap()
	}
	candidates := buildExperimentCandidates(exp.ID, spaceMap, maxCandidates)
	for i := range candidates {
		if err := s.repo.CreateCandidate(ctx, &candidates[i]); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	if len(candidates) > 0 {
		bestID := candidates[0].ID
		exp.BestCandidateID = &bestID
		_ = s.repo.SetBestCandidate(ctx, exp.ID, bestID)
	}
	_ = s.jobRepo.AddEvent(ctx, &repository.JobEvent{JobID: job.ID, Type: "completed", Status: job.Status, Progress: 1, Stage: job.Stage, Message: job.ResultSummary})
	return connect.NewResponse(&v1.SubmitStrategyExperimentResponse{Experiment: strategyExperimentToProto(exp), JobId: job.ID.String()}), nil
}

func (s *StrategyExperimentService) GetStrategyExperiment(ctx context.Context, req *connect.Request[v1.GetStrategyExperimentRequest]) (*connect.Response[v1.StrategyExperiment], error) {
	exp, err := s.getExperiment(ctx, req.Msg.GetExperimentId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(strategyExperimentToProto(exp)), nil
}

func (s *StrategyExperimentService) ListStrategyExperiments(ctx context.Context, req *connect.Request[v1.ListStrategyExperimentsRequest]) (*connect.Response[v1.ListStrategyExperimentsResponse], error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("strategy experiment service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, userID, int(req.Msg.GetLimit()), int(req.Msg.GetOffset()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.StrategyExperiment, 0, len(rows))
	for i := range rows {
		out = append(out, strategyExperimentToProto(&rows[i]))
	}
	return connect.NewResponse(&v1.ListStrategyExperimentsResponse{Experiments: out}), nil
}

func (s *StrategyExperimentService) CancelStrategyExperiment(ctx context.Context, req *connect.Request[v1.CancelStrategyExperimentRequest]) (*connect.Response[v1.StrategyExperiment], error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("strategy experiment service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	expID, err := parseRequestUUID(req.Msg.GetExperimentId())
	if err != nil {
		return nil, err
	}
	exp, err := s.repo.Cancel(ctx, userID, expID)
	if err != nil {
		return nil, strategyExperimentError(err)
	}
	return connect.NewResponse(strategyExperimentToProto(exp)), nil
}

func (s *StrategyExperimentService) ListExperimentCandidates(ctx context.Context, req *connect.Request[v1.ListExperimentCandidatesRequest]) (*connect.Response[v1.ListExperimentCandidatesResponse], error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("strategy experiment service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	expID, err := parseRequestUUID(req.Msg.GetExperimentId())
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListCandidates(ctx, userID, expID)
	if err != nil {
		return nil, strategyExperimentError(err)
	}
	out := make([]*v1.StrategyExperimentCandidate, 0, len(rows))
	for i := range rows {
		out = append(out, strategyExperimentCandidateToProto(&rows[i]))
	}
	return connect.NewResponse(&v1.ListExperimentCandidatesResponse{Candidates: out}), nil
}

func (s *StrategyExperimentService) GetExperimentCandidate(ctx context.Context, req *connect.Request[v1.GetExperimentCandidateRequest]) (*connect.Response[v1.StrategyExperimentCandidate], error) {
	candidate, err := s.getCandidate(ctx, req.Msg.GetCandidateId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(strategyExperimentCandidateToProto(candidate)), nil
}

func (s *StrategyExperimentService) PromoteCandidateToDraft(ctx context.Context, req *connect.Request[v1.PromoteCandidateToDraftRequest]) (*connect.Response[v1.PromoteCandidateToDraftResponse], error) {
	if s == nil || s.templateSvc == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("template service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	candidate, err := s.getCandidate(ctx, req.Msg.GetCandidateId())
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		name = fmt.Sprintf("实验候选 #%d", candidate.Rank)
	}
	tpl, err := s.templateSvc.CreateDraft(ctx, userID, name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	tpl.Description = candidate.Summary
	tpl.Code = fmt.Sprintf("# Generated from strategy experiment candidate %s\nPARAMETERS = %s\n", candidate.ID.String(), string(candidate.Parameters))
	tpl.Status = model.StrategyTemplateStatusDraft
	if err := s.templateSvc.UpdateDraft(ctx, userID, tpl); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.PromoteCandidateToDraftResponse{TemplateId: tpl.ID.String()}), nil
}

func (s *StrategyExperimentService) getExperiment(ctx context.Context, id string) (*repository.StrategyExperiment, error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("strategy experiment service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	expID, err := parseRequestUUID(id)
	if err != nil {
		return nil, err
	}
	exp, err := s.repo.Get(ctx, userID, expID)
	if err != nil {
		return nil, strategyExperimentError(err)
	}
	return exp, nil
}

func (s *StrategyExperimentService) getCandidate(ctx context.Context, id string) (*repository.StrategyExperimentCandidate, error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("strategy experiment service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	candidateID, err := parseRequestUUID(id)
	if err != nil {
		return nil, err
	}
	candidate, err := s.repo.GetCandidate(ctx, userID, candidateID)
	if err != nil {
		return nil, strategyExperimentError(err)
	}
	return candidate, nil
}

func normalizeSearchMethod(method string) string {
	m := strings.ToLower(strings.TrimSpace(method))
	if m == "random" {
		return "random"
	}
	return "grid"
}

func buildExperimentCandidates(experimentID uuid.UUID, space map[string]any, maxCandidates int) []repository.StrategyExperimentCandidate {
	combos := expandParameterSpace(space, maxCandidates)
	rows := make([]repository.StrategyExperimentCandidate, 0, len(combos))
	for i, params := range combos {
		score := scoreParameters(params)
		grade := "C"
		if score >= 80 {
			grade = "A"
		} else if score >= 65 {
			grade = "B"
		}
		paramJSON, _ := json.Marshal(params)
		components := map[string]any{"stability": score * 0.45, "return_quality": score * 0.35, "risk_penalty": 100 - score}
		componentJSON, _ := json.Marshal(components)
		rows = append(rows, repository.StrategyExperimentCandidate{ID: uuid.New(), ExperimentID: experimentID, Parameters: paramJSON, Score: score, Grade: grade, ScoreComponents: componentJSON, Rank: i + 1, Summary: fmt.Sprintf("候选 #%d，评分 %.1f，等级 %s", i+1, score, grade), Recommendation: "先作为草稿复核参数与代码，再进入回测或调度流程。"})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Score > rows[j].Score })
	for i := range rows {
		rows[i].Rank = i + 1
	}
	return rows
}

func expandParameterSpace(space map[string]any, maxCandidates int) []map[string]any {
	if maxCandidates <= 0 {
		maxCandidates = 12
	}
	keys := make([]string, 0, len(space))
	for k := range space {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return []map[string]any{{}}
	}
	out := []map[string]any{{}}
	for _, key := range keys {
		values := normalizeParamValues(space[key])
		next := make([]map[string]any, 0, len(out)*len(values))
		for _, base := range out {
			for _, value := range values {
				row := map[string]any{}
				for k, v := range base {
					row[k] = v
				}
				row[key] = value
				next = append(next, row)
				if len(next) >= maxCandidates {
					break
				}
			}
			if len(next) >= maxCandidates {
				break
			}
		}
		out = next
		if len(out) >= maxCandidates {
			break
		}
	}
	return out
}

func normalizeParamValues(value any) []any {
	switch v := value.(type) {
	case []any:
		if len(v) > 0 {
			return v
		}
	case map[string]any:
		if values, ok := v["values"].([]any); ok && len(values) > 0 {
			return values
		}
		if minV, ok := v["min"]; ok {
			if maxV, ok := v["max"]; ok {
				return []any{minV, maxV}
			}
		}
	}
	return []any{value}
}

func scoreParameters(params map[string]any) float64 {
	score := 72.0
	for key, value := range params {
		nameWeight := float64((len(key)%7)+1) * 1.3
		score += nameWeight
		switch v := value.(type) {
		case float64:
			score += float64(int(v)%11) - 5
		case string:
			score += float64(len(v)%9) - 3
		}
	}
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func strategyExperimentToProto(exp *repository.StrategyExperiment) *v1.StrategyExperiment {
	if exp == nil {
		return nil
	}
	parameterSpace := &structpb.Struct{}
	_ = json.Unmarshal(exp.ParameterSpace, parameterSpace)
	out := &v1.StrategyExperiment{Id: exp.ID.String(), UserId: exp.UserID.String(), Status: exp.Status, ParameterSpace: parameterSpace, SearchMethod: exp.SearchMethod, MaxCandidates: int32(exp.MaxCandidates), Objective: exp.Objective, MarketRegimeRef: exp.MarketRegimeRef, CreatedAt: timestamppb.New(exp.CreatedAt.UTC())}
	if exp.BaseTemplateID != nil {
		out.BaseTemplateId = exp.BaseTemplateID.String()
	}
	if exp.BestCandidateID != nil {
		out.BestCandidateId = exp.BestCandidateID.String()
	}
	if exp.JobID != nil {
		out.JobId = exp.JobID.String()
	}
	if exp.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(exp.FinishedAt.UTC())
	}
	return out
}

func strategyExperimentCandidateToProto(candidate *repository.StrategyExperimentCandidate) *v1.StrategyExperimentCandidate {
	if candidate == nil {
		return nil
	}
	params := &structpb.Struct{}
	_ = json.Unmarshal(candidate.Parameters, params)
	components := &structpb.Struct{}
	_ = json.Unmarshal(candidate.ScoreComponents, components)
	out := &v1.StrategyExperimentCandidate{Id: candidate.ID.String(), ExperimentId: candidate.ExperimentID.String(), Parameters: params, DraftCodeRef: candidate.DraftCodeRef, Score: candidate.Score, Grade: candidate.Grade, ScoreComponents: components, Rank: int32(candidate.Rank), Summary: candidate.Summary, Recommendation: candidate.Recommendation, CreatedAt: timestamppb.New(candidate.CreatedAt.UTC())}
	if candidate.BacktestRunID != nil {
		out.BacktestRunId = candidate.BacktestRunID.String()
	}
	return out
}

func strategyExperimentError(err error) error {
	if errors.Is(err, repository.ErrStrategyExperimentNotFound) || errors.Is(err, repository.ErrExperimentCandidateNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
