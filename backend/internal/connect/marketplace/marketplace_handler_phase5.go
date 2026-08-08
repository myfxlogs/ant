package marketplace

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
)

// ── Phase 5.1: AI Strategy Iteration Handlers ─────────────────────────────────

func (s *MarketplaceServer) DetectStrategyDecay(
	ctx context.Context,
	req *connect.Request[antv1.DetectStrategyDecayRequest],
) (*connect.Response[antv1.DetectStrategyDecayResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	result, err := s.svc.DetectDecay(ctx, req.Msg.GetStrategyId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &antv1.DetectStrategyDecayResponse{
		IsDecaying:        result.IsDecaying,
		DecayScore:        result.DecayScore,
		TriggerReason:     result.TriggerReason,
		BaselineSharpe:    decPtrToStr(result.BaselineSharpe),
		BaselineWinRate:   decPtrToStr(result.BaselineWinRate),
		RecentSharpe:      decPtrToStr(result.RecentSharpe),
		RecentWinRate:     decPtrToStr(result.RecentWinRate),
		SharpeDeclinePct:  result.SharpeDeclinePct.String(),
		WinrateDeclinePct: result.WinRateDeclinePct.String(),
	}

	// FEAT-5: No auto-create optimization task. Author must manually initiate
	// iteration via InitiateStrategyIteration RPC. Decay detection is free
	// diagnosis; AI iteration is credit-billed and author-initiated only.

	return connect.NewResponse(resp), nil
}

func (s *MarketplaceServer) ListOptimizationTasks(
	ctx context.Context,
	req *connect.Request[antv1.ListOptimizationTasksRequest],
) (*connect.Response[antv1.ListOptimizationTasksResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	tasks, total, err := s.svc.ListOptimizationTasks(ctx, userID, req.Msg.GetStatus(), int(req.Msg.GetLimit()), int(req.Msg.GetOffset()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.OptimizationTaskInfo, 0, len(tasks))
	for _, t := range tasks {
		item := &antv1.OptimizationTaskInfo{
			Id:            t.ID,
			StrategyId:    t.StrategyID,
			Status:        t.Status,
			TriggerReason: t.TriggerReason,
			ChangeSummary: t.ChangeSummary,
			CreatedAtMs:   t.CreatedAt.UnixMilli(),
		}
		if t.CompletedAt != nil {
			item.CompletedAtMs = t.CompletedAt.UnixMilli()
		}
		if t.PublishedVersionID != nil {
			item.PublishedVersionId = *t.PublishedVersionID
		}
		items = append(items, item)
	}

	return connect.NewResponse(&antv1.ListOptimizationTasksResponse{
		Tasks: items,
		Total: int32(total),
	}), nil
}

func (s *MarketplaceServer) GetOptimizationTask(
	ctx context.Context,
	req *connect.Request[antv1.GetOptimizationTaskRequest],
) (*connect.Response[antv1.GetOptimizationTaskResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	t, err := s.svc.GetOptimizationTask(ctx, req.Msg.GetTaskId(), userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	resp := &antv1.GetOptimizationTaskResponse{
		Task: &antv1.OptimizationTaskInfo{
			Id:            t.ID,
			StrategyId:    t.StrategyID,
			Status:        t.Status,
			TriggerReason: t.TriggerReason,
			ChangeSummary: t.ChangeSummary,
			CreatedAtMs:   t.CreatedAt.UnixMilli(),
		},
		SuggestedCode:    t.SuggestedCode,
		SuggestedParams:  t.SuggestedParams,
		BacktestSnapshot: t.BacktestSnapshot,
	}
	if len(t.DecayMetrics) > 0 {
		var dm antv1.DecayMetrics
		if err := proto.Unmarshal(t.DecayMetrics, &dm); err != nil {
			s.log.Warn("get opt task: unmarshal decay metrics", zap.Error(err))
		} else {
			resp.DecayMetrics = &dm
		}
	}
	if t.CompletedAt != nil {
		resp.Task.CompletedAtMs = t.CompletedAt.UnixMilli()
	}
	if t.PublishedVersionID != nil {
		resp.Task.PublishedVersionId = *t.PublishedVersionID
	}

	return connect.NewResponse(resp), nil
}

func (s *MarketplaceServer) RejectOptimizationTask(
	ctx context.Context,
	req *connect.Request[antv1.RejectOptimizationTaskRequest],
) (*connect.Response[antv1.RejectOptimizationTaskResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	if err := s.svc.RejectOptimizationTask(ctx, req.Msg.GetTaskId(), userID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.RejectOptimizationTaskResponse{Success: true}), nil
}

func (s *MarketplaceServer) PublishOptimization(
	ctx context.Context,
	req *connect.Request[antv1.PublishOptimizationRequest],
) (*connect.Response[antv1.PublishOptimizationResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	versionID, err := s.svc.PublishOptimization(ctx, req.Msg.GetTaskId(), userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.PublishOptimizationResponse{
		VersionId: versionID,
		Success:   true,
	}), nil
}

func (s *MarketplaceServer) PreviewOptimization(
	ctx context.Context,
	req *connect.Request[antv1.PreviewOptimizationRequest],
) (*connect.Response[antv1.PreviewOptimizationResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	result, err := s.svc.PreviewOptimization(ctx, req.Msg.GetTaskId(), userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Unmarshal backtest snapshots.
	var originalSnap, optimizedSnap antv1.BacktestSnapshot
	if len(result.OriginalBacktest) > 0 {
		if err := proto.Unmarshal(result.OriginalBacktest, &originalSnap); err != nil {
			s.log.Warn("preview opt: unmarshal original backtest", zap.Error(err))
		}
	}
	if len(result.OptimizedBacktest) > 0 {
		if err := proto.Unmarshal(result.OptimizedBacktest, &optimizedSnap); err != nil {
			s.log.Warn("preview opt: unmarshal optimized backtest", zap.Error(err))
		}
	}

	// Build task info.
	task := result.Task
	var completedAtMs int64
	if task.CompletedAt != nil {
		completedAtMs = task.CompletedAt.UnixMilli()
	}
	taskInfo := &antv1.OptimizationTaskInfo{
		Id:            task.ID,
		StrategyId:    task.StrategyID,
		Status:        task.Status,
		TriggerReason: task.TriggerReason,
		ChangeSummary: task.ChangeSummary,
		CreatedAtMs:   task.CreatedAt.UnixMilli(),
		CompletedAtMs: completedAtMs,
		PublishedVersionId: func() string {
			if task.PublishedVersionID != nil {
				return *task.PublishedVersionID
			}
			return ""
		}(),
	}

	// Truncate suggested code to first 2000 chars for preview.
	codePreview := result.SuggestedCode
	if len(codePreview) > 2000 {
		codePreview = codePreview[:2000] + "\n... (truncated)"
	}

	resp := &antv1.PreviewOptimizationResponse{
		Task:                 taskInfo,
		OriginalBacktest:     &originalSnap,
		OptimizedBacktest:    &optimizedSnap,
		ChangeSummary:        result.ChangeSummary,
		SuggestedCodePreview: codePreview,
	}
	if len(result.DecayMetrics) > 0 {
		var dm antv1.DecayMetrics
		if err := proto.Unmarshal(result.DecayMetrics, &dm); err != nil {
			s.log.Warn("preview opt: unmarshal decay metrics", zap.Error(err))
		} else {
			resp.DecayMetrics = &dm
		}
	}

	return connect.NewResponse(resp), nil
}

// decPtrToStr converts a *decimal.Decimal to string, returning "" if nil.
func decPtrToStr(d *decimal.Decimal) string {
	if d == nil {
		return ""
	}
	return d.String()
}

func (s *MarketplaceServer) InitiateStrategyIteration(
	ctx context.Context,
	req *connect.Request[antv1.InitiateStrategyIterationRequest],
) (*connect.Response[antv1.InitiateStrategyIterationResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	taskID, err := s.svc.InitiateStrategyIteration(ctx, req.Msg.GetStrategyId(), userID)
	if err != nil {
		if strings.Contains(err.Error(), "not the strategy owner") {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		if strings.Contains(err.Error(), "not found") {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.InitiateStrategyIterationResponse{
		TaskId:  taskID,
		Success: true,
	}), nil
}
