package connect

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/repository"
)

func (s *AIService) CreateWorkflowRun(ctx context.Context, req *connect.Request[v1.CreateWorkflowRunRequest]) (*connect.Response[v1.CreateWorkflowRunResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.wfRepo == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("workflow repository not initialized"))
	}

	title := strings.TrimSpace(req.Msg.Title)
	ctxJSON := req.Msg.ContextJson
	run, err := s.wfRepo.CreateRun(ctx, userID, title, ctxJSON)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.CreateWorkflowRunResponse{Run: toWorkflowRunSummary(run)}), nil
}

func (s *AIService) AppendWorkflowStep(ctx context.Context, req *connect.Request[v1.AppendWorkflowStepRequest]) (*connect.Response[v1.AppendWorkflowStepResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.wfRepo == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("workflow repository not initialized"))
	}

	runID, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	step, err := s.wfRepo.AppendStep(
		ctx,
		userID,
		runID,
		strings.TrimSpace(req.Msg.Key),
		strings.TrimSpace(req.Msg.Title),
		strings.TrimSpace(req.Msg.Status),
		req.Msg.Input,
		req.Msg.Output,
		req.Msg.Error,
		req.Msg.DurationMs,
	)
	if err != nil {
		code := connect.CodeInternal
		if errors.Is(err, repository.ErrAIWorkflowRunNotFound) {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&v1.AppendWorkflowStepResponse{Step: toWorkflowStep(step)}), nil
}

func (s *AIService) ListWorkflowRuns(ctx context.Context, req *connect.Request[v1.ListWorkflowRunsRequest]) (*connect.Response[v1.ListWorkflowRunsResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.wfRepo == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("workflow repository not initialized"))
	}

	runs, err := s.wfRepo.ListRuns(ctx, userID, int(req.Msg.Limit), int(req.Msg.Offset))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.WorkflowRunSummary, 0, len(runs))
	for i := range runs {
		r := runs[i]
		run := r
		out = append(out, toWorkflowRunSummary(&run))
	}
	return connect.NewResponse(&v1.ListWorkflowRunsResponse{Runs: out}), nil
}

func (s *AIService) GetWorkflowRun(ctx context.Context, req *connect.Request[v1.GetWorkflowRunRequest]) (*connect.Response[v1.GetWorkflowRunResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.wfRepo == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("workflow repository not initialized"))
	}

	runID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	run, steps, err := s.wfRepo.GetRun(ctx, userID, runID)
	if err != nil {
		code := connect.CodeInternal
		if errors.Is(err, repository.ErrAIWorkflowRunNotFound) {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}

	pbSteps := make([]*v1.WorkflowStep, 0, len(steps))
	for i := range steps {
		s := steps[i]
		st := s
		pbSteps = append(pbSteps, toWorkflowStep(&st))
	}

	return connect.NewResponse(&v1.GetWorkflowRunResponse{
		Run:         toWorkflowRunSummary(run),
		Steps:       pbSteps,
		ContextJson: run.Context,
	}), nil
}

func toWorkflowRunSummary(run *repository.AIWorkflowRun) *v1.WorkflowRunSummary {
	if run == nil {
		return &v1.WorkflowRunSummary{}
	}
	return &v1.WorkflowRunSummary{
		Id:        run.ID.String(),
		Title:     run.Title,
		Status:    run.Status,
		CreatedAt: timestamppb.New(run.CreatedAt),
		UpdatedAt: timestamppb.New(run.UpdatedAt),
		StepCount: int32(run.StepCount),
	}
}

func toWorkflowStep(step *repository.AIWorkflowStep) *v1.WorkflowStep {
	if step == nil {
		return &v1.WorkflowStep{}
	}
	createdAt := step.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	return &v1.WorkflowStep{
		Id:         step.ID.String(),
		RunId:      step.RunID.String(),
		Key:        step.Key,
		Title:      step.Title,
		Status:     step.Status,
		Input:      step.Input,
		Output:     step.Output,
		Error:      step.Error,
		DurationMs: step.Duration,
		CreatedAt:  timestamppb.New(createdAt),
	}
}
