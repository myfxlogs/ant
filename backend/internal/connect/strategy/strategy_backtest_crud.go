package strategy

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/backtest"
	"alphaforge/internal/pkg/ptr"
	"alphaforge/internal/repository"
	"alphaforge/tools/mql2go/interp"
)

// validateBacktestRun delegates to the shared backtest.ApplyDefaults.
func validateBacktestRun(run *repository.BacktestRun) {
	backtest.ApplyDefaults(run)
}

func (s *StrategyExecutionServer) StartBacktestRun(ctx context.Context, req *connect.Request[antv1.StartBacktestRunRequest]) (*connect.Response[antv1.StartBacktestRunResponse], error) {
	userID, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}

	// Verify account ownership — prevent cross-user backtest execution.
	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil || accountID == uuid.Nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id"))
	}
	var owns bool
	if err := s.backtestRepo.DB().QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM mt_accounts WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL)",
		accountID, userID,
	).Scan(&owns); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("account ownership check: %w", err))
	}
	if !owns {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("account not found or not owned by user"))
	}

	// I1: Enforce daily backtest quota before creating the run.
	if s.quotaChecker != nil {
		var todayCount int
		if err := s.backtestRepo.DB().QueryRow(ctx,
			"SELECT count(*) FROM backtest_runs WHERE user_id = $1 AND created_at >= CURRENT_DATE",
			userID,
		).Scan(&todayCount); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("backtest quota count: %w", err))
		}
		if !s.quotaChecker.CheckBacktestDailyLimit(userID, todayCount) {
			return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("daily backtest limit reached"))
		}
	}

	if err := s.validateBacktestRequest(ctx, req); err != nil {
		return nil, err
	}

	run := buildBacktestRunFromRequest(userID, req.Msg)

	// If strategy_id is provided, fetch source code + params from imported_strategies.
	if sid := req.Msg.GetStrategyId(); sid != "" {
		importedID, err := uuid.Parse(sid)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid strategy_id: %w", err))
		}
		if s.importedRepo == nil {
			return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("imported strategy repository not configured"))
		}
		imported, err := s.importedRepo.GetByIDAndUser(ctx, importedID, userID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("imported strategy not found: %w", err))
		}
		run.StrategyCode = ptr.Str(imported.SourceCode)
		run.StrategyID = &imported.ID
		// Convert interp params (SerializeParams format) to proto StrategyParams
		// so the backtest worker can use paramsProtoToMap uniformly.
		if len(imported.Params) > 0 && len(run.ParameterOverrides) == 0 {
			defaults := interp.ParamDefaultsToMap(imported.Params)
			if len(defaults) > 0 {
				sp := &antv1.StrategyParams{Values: defaults}
				if raw, err := proto.Marshal(sp); err == nil {
					run.ParameterOverrides = raw
				}
			}
		}
	}

	validateBacktestRun(run)
	if cfg := req.Msg.GetExecutionConfig(); cfg != nil {
		snap, err := proto.Marshal(cfg)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal config snapshot: %w", err))
		}
		run.ConfigSnapshot = snap
	}
	runID, err := s.backtestRepo.Create(ctx, run)
	if err != nil {
		s.log.Error("StartBacktestRun: create", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.StartBacktestRunResponse{RunId: runID.String()}), nil
}

func buildBacktestRunFromRequest(userID uuid.UUID, msg *antv1.StartBacktestRunRequest) *repository.BacktestRun {
	accountID, _ := uuid.Parse(msg.AccountId)
	run := &repository.BacktestRun{
		ID: uuid.New(), UserID: userID, AccountID: accountID,
		Symbol: msg.Symbol, Timeframe: msg.Timeframe,
		Mode: backtestModeToString(msg.Mode), Status: StatusPending,
		StrategyCode: ptr.Str(msg.Code), InitialCapital: ptr.Decimal(parseDecimal(msg.InitialCapital)),
	}
	if run.Mode == "" {
		run.Mode = "KLINE_RANGE"
	}
	if parseDecimal(msg.InitialCapital).LessThanOrEqual(decimal.Zero) {
		run.InitialCapital = ptr.Decimal(decimal.NewFromInt(10000))
	}
	if cfg := msg.GetExecutionConfig(); cfg != nil {
		run.Commission = ptr.Decimal(parseDecimal(cfg.GetCommission()))
		run.Slippage = ptr.Decimal(parseDecimal(cfg.GetSlippage()))
		run.Leverage = ptr.Decimal(parseDecimal(cfg.GetLeverage()))
		run.TradeDirection = ptr.Str(tradeDirectionToString(cfg.GetTradeDirection()))
		sMode := cfg.GetStrictMode()
		run.StrictMode = &sMode
	}
	if msg.From != nil {
		t := msg.From.AsTime()
		run.FromTs = &t
	}
	if msg.To != nil {
		t := msg.To.AsTime()
		run.ToTs = &t
	}
	if msg.DatasetId != nil {
		if id, _ := uuid.Parse(*msg.DatasetId); id != uuid.Nil {
			run.DatasetID = &id
		}
	}
	if msg.TemplateId != nil {
		if id, _ := uuid.Parse(*msg.TemplateId); id != uuid.Nil {
			run.TemplateID = &id
		}
	}
	run.AutoGate = msg.GetAutoGate()
	return run
}

func (s *StrategyExecutionServer) GetBacktestRun(ctx context.Context, req *connect.Request[antv1.GetBacktestRunRequest]) (*connect.Response[antv1.GetBacktestRunResponse], error) {
	userID, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	runID, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	run, err := s.backtestRepo.GetByID(ctx, userID, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	bp := parseBacktestResult(run.ProtoResponse)
	return connect.NewResponse(&antv1.GetBacktestRunResponse{
		Run:                  toProtoBacktestRun(run),
		Metrics:              bp.Metrics,
		EquityCurve:          bp.EquityCurve,
		Risk:                 bp.Risk,
		ExecutionAssumptions: bp.ExecutionAssumptions,
	}), nil
}

func (s *StrategyExecutionServer) ListBacktestRuns(ctx context.Context, req *connect.Request[antv1.ListBacktestRunsRequest]) (*connect.Response[antv1.ListBacktestRunsResponse], error) {
	userID, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	var accountID *uuid.UUID
	if req.Msg.AccountId != nil && *req.Msg.AccountId != "" {
		id, err := uuid.Parse(*req.Msg.AccountId)
		if err == nil {
			accountID = &id
		}
	}
	var templateID *uuid.UUID
	if req.Msg.TemplateId != nil && *req.Msg.TemplateId != "" {
		id, err := uuid.Parse(*req.Msg.TemplateId)
		if err == nil {
			templateID = &id
		}
	}
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 50
	}
	offset := int(req.Msg.Offset)
	runs, err := s.backtestRepo.ListByUser(ctx, userID, accountID, templateID, limit, offset)
	if err != nil {
		s.log.Error("ListBacktestRuns", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*antv1.BacktestRun, 0, len(runs))
	for _, r := range runs {
		out = append(out, toProtoBacktestRun(r))
	}
	return connect.NewResponse(&antv1.ListBacktestRunsResponse{Runs: out}), nil
}

func (s *StrategyExecutionServer) CancelBacktestRun(ctx context.Context, req *connect.Request[antv1.CancelBacktestRunRequest]) (*connect.Response[antv1.CancelBacktestRunResponse], error) {
	userID, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	runID, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.backtestRepo.RequestCancel(ctx, userID, runID); err != nil {
		s.log.Error("CancelBacktestRun", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	run, err := s.backtestRepo.GetByID(ctx, userID, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("query run after cancel: %w", err))
	}
	return connect.NewResponse(&antv1.CancelBacktestRunResponse{
		Run: toProtoBacktestRun(run),
	}), nil
}

func (s *StrategyExecutionServer) DeleteBacktestRun(ctx context.Context, req *connect.Request[antv1.DeleteBacktestRunRequest]) (*connect.Response[antv1.DeleteBacktestRunResponse], error) {
	userID, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	runID, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	deleted, err := s.backtestRepo.Delete(ctx, userID, runID)
	if err != nil {
		s.log.Error("DeleteBacktestRun", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("DeleteBacktestRun result", zap.Bool("deleted", deleted), zap.String("user_id", userID.String()), zap.String("run_id", runID.String()))
	return connect.NewResponse(&antv1.DeleteBacktestRunResponse{Deleted: deleted}), nil
}

// DeleteBacktestRuns implements the batch-delete RPC for backtest runs.
func (s *StrategyExecutionServer) DeleteBacktestRuns(ctx context.Context, req *connect.Request[antv1.DeleteBacktestRunsRequest]) (*connect.Response[antv1.DeleteBacktestRunsResponse], error) {
	userID, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	ids := req.Msg.RunIds
	if len(ids) == 0 {
		return connect.NewResponse(&antv1.DeleteBacktestRunsResponse{}), nil
	}
	uuids := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		uid, err := uuid.Parse(id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid run_id %q: %w", id, err))
		}
		uuids = append(uuids, uid)
	}
	deleted, err := s.backtestRepo.DeleteBatch(ctx, userID, uuids)
	if err != nil {
		s.log.Error("DeleteBacktestRuns", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.DeleteBacktestRunsResponse{
		DeletedCount: int32(deleted),
		FailedCount:  int32(len(uuids)) - int32(deleted),
	}), nil
}





