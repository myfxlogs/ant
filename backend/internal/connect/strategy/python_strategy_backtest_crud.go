package strategy

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	"anttrader/internal/pkg/ptr"
	"anttrader/internal/repository"
)


// validateBacktestRun clamps backtest run parameters to safe ranges.
// Backend is the final authority on value validation.
func validateBacktestRun(run *repository.BacktestRun) {
	if run.Mode == "" {
		run.Mode = "KLINE_RANGE"
	}
	if run.Commission == nil || *run.Commission == 0 {
		run.Commission = ptr.F64(0.001)
	}
	if run.Commission != nil && (*run.Commission < 0 || *run.Commission > 10) {
		run.Commission = ptr.F64(0.001)
	}
	if run.Slippage != nil && (*run.Slippage < 0 || *run.Slippage > 10) {
		run.Slippage = ptr.F64(0)
	}
	if run.Leverage == nil || *run.Leverage == 0 {
		run.Leverage = ptr.F64(1)
	}
	if run.Leverage != nil && (*run.Leverage < 1 || *run.Leverage > 125) {
		run.Leverage = ptr.F64(1)
	}
	if run.TradeDirection == nil || *run.TradeDirection == "" {
		run.TradeDirection = ptr.Str("both")
	}
	if run.StrictMode == nil {
		t := true
		run.StrictMode = &t
	}
	if run.ExtraSymbols == nil {
		run.ExtraSymbols = []string{}
	}
}

func (s *PythonStrategyServer) StartBacktestRun(ctx context.Context, req *connect.Request[antv1.StartBacktestRunRequest]) (*connect.Response[antv1.StartBacktestRunResponse], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	run := buildBacktestRunFromRequest(userID, req.Msg)
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
		StrategyCode: ptr.Str(msg.Code), InitialCapital: ptr.F64(msg.InitialCapital),
	}
	if run.Mode == "" { run.Mode = "KLINE_RANGE" }
	if msg.InitialCapital <= 0 { run.InitialCapital = ptr.F64(10000) }
	if cfg := msg.GetExecutionConfig(); cfg != nil {
		run.Commission = ptr.F64(cfg.GetCommission())
		run.Slippage = ptr.F64(cfg.GetSlippage())
		run.Leverage = ptr.F64(cfg.GetLeverage())
		run.TradeDirection = ptr.Str(tradeDirectionToString(cfg.GetTradeDirection()))
		sMode := cfg.GetStrictMode()
		run.StrictMode = &sMode
	}
	if msg.From != nil { t := msg.From.AsTime(); run.FromTs = &t }
	if msg.To != nil { t := msg.To.AsTime(); run.ToTs = &t }
	if msg.DatasetId != nil {
		if id, _ := uuid.Parse(*msg.DatasetId); id != uuid.Nil { run.DatasetID = &id }
	}
	if msg.TemplateId != nil {
		if id, _ := uuid.Parse(*msg.TemplateId); id != uuid.Nil { run.TemplateID = &id }
	}
	return run
}

func (s *PythonStrategyServer) GetBacktestRun(ctx context.Context, req *connect.Request[antv1.GetBacktestRunRequest]) (*connect.Response[antv1.GetBacktestRunResponse], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	runID, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	run, err := s.backtestRepo.GetByID(ctx, userID, runID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&antv1.GetBacktestRunResponse{
		Run:                   toProtoBacktestRun(run),
		Metrics:               parseMetrics(run.ProtoResponse),
		EquityCurve:           parseEquityCurve(run.ProtoResponse),
		Risk:                  parseRisk(run.ProtoResponse),
		ExecutionAssumptions:  parseExecutionAssumptions(run.ProtoResponse),
	}), nil
}

func (s *PythonStrategyServer) ListBacktestRuns(ctx context.Context, req *connect.Request[antv1.ListBacktestRunsRequest]) (*connect.Response[antv1.ListBacktestRunsResponse], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	var accountID *uuid.UUID
	if req.Msg.AccountId != nil && *req.Msg.AccountId != "" {
		id, err := uuid.Parse(*req.Msg.AccountId)
		if err == nil {
			accountID = &id
		}
	}
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 50
	}
	offset := int(req.Msg.Offset)
	runs, err := s.backtestRepo.ListByUser(ctx, userID, accountID, limit, offset)
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

func (s *PythonStrategyServer) WatchBacktestRun(ctx context.Context, req *connect.Request[antv1.WatchBacktestRunRequest], stream *connect.ServerStream[antv1.BacktestRunUpdate]) error {
	runID, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return err
	}
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	// LISTEN for status changes (push-first), fallback to 30s ticker
	notifCh, listenCancel, _ := s.pgListen.Listen(ctx, "backtest_status")
	if listenCancel != nil {
		defer listenCancel()
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	prevStatus := ""
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-notifCh:
		case <-ticker.C:
		}
		run, err := s.backtestRepo.GetByID(ctx, userID, runID)
		if err != nil {
			s.log.Warn("WatchBacktestRun: transient DB error", zap.Error(err), zap.String("runID", runID.String()))
			continue
		}
		if run == nil {
			s.log.Warn("WatchBacktestRun: run not found", zap.String("runID", runID.String()))
			continue
		}
		if run.Status == prevStatus {
			continue
		}
		prevStatus = run.Status
		if err := stream.Send(&antv1.BacktestRunUpdate{
			Run:                  toProtoBacktestRun(run),
			Metrics:              parseMetrics(run.ProtoResponse),
			EquityCurve:          parseEquityCurve(run.ProtoResponse),
			Risk:                 parseRisk(run.ProtoResponse),
			ExecutionAssumptions: parseExecutionAssumptions(run.ProtoResponse),
		}); err != nil {
			return err
		}
		if run.Status == "SUCCEEDED" || run.Status == "FAILED" || run.Status == "CANCELED" {
			return nil
		}
	}
}

func (s *PythonStrategyServer) CancelBacktestRun(ctx context.Context, req *connect.Request[antv1.CancelBacktestRunRequest]) (*connect.Response[antv1.CancelBacktestRunResponse], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	runID, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.backtestRepo.RequestCancel(ctx, userID, runID); err != nil {
		s.log.Error("CancelBacktestRun", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	run, _ := s.backtestRepo.GetByID(ctx, userID, runID)
	return connect.NewResponse(&antv1.CancelBacktestRunResponse{
		Run: toProtoBacktestRun(run),
	}), nil
}

func (s *PythonStrategyServer) DeleteBacktestRun(ctx context.Context, req *connect.Request[antv1.DeleteBacktestRunRequest]) (*connect.Response[antv1.DeleteBacktestRunResponse], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	runID, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	deleted, err := s.backtestRepo.Delete(ctx, userID, runID)
	if err != nil {
		s.log.Error("DeleteBacktestRun", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.DeleteBacktestRunResponse{Deleted: deleted}), nil
}

// DeleteBacktestRuns implements the batch-delete RPC for backtest runs.
func (s *PythonStrategyServer) DeleteBacktestRuns(ctx context.Context, req *connect.Request[antv1.DeleteBacktestRunsRequest]) (*connect.Response[antv1.DeleteBacktestRunsResponse], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
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
