package connect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

func (s *PythonStrategyService) StartBacktestRun(ctx context.Context, req *connect.Request[v1.StartBacktestRunRequest]) (*connect.Response[v1.StartBacktestRunResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}
	if s == nil || s.backtestRun == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("backtest run service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Hard limits to prevent users from overwhelming the server.
	maxActivePerUser := s.getIntConfig(ctx, "backtest.max_active_per_user", 2)
	maxActivePerAccount := s.getIntConfig(ctx, "backtest.max_active_per_account", 1)
	maxPendingPerUser := s.getIntConfig(ctx, "backtest.max_pending_per_user", 10)
	maxStartPerMinute := s.getIntConfig(ctx, "backtest.max_start_per_minute", 6)

	if n, cerr := s.backtestRun.CountActiveByUser(ctx, userID); cerr == nil && n >= maxActivePerUser {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("too many active backtests (limit %d)", maxActivePerUser))
	}
	if n, cerr := s.backtestRun.CountActiveByAccount(ctx, userID, accountID); cerr == nil && n >= maxActivePerAccount {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("too many active backtests for this account (limit %d)", maxActivePerAccount))
	}
	if n, cerr := s.backtestRun.CountPendingByUser(ctx, userID); cerr == nil && n >= maxPendingPerUser {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("too many pending backtests (limit %d)", maxPendingPerUser))
	}
	if n, cerr := s.backtestRun.CountRecentStartsByUser(ctx, userID, time.Now().Add(-1*time.Minute)); cerr == nil && n >= maxStartPerMinute {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("too many backtest starts per minute (limit %d)", maxStartPerMinute))
	}

	modeStr, fromT, toT, dsid, err := validateStartBacktestRunRequest(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var templateID *uuid.UUID
	if req.Msg.GetTemplateId() != "" {
		id, perr := uuid.Parse(req.Msg.GetTemplateId())
		if perr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid template_id"))
		}
		templateID = &id
	}
	var templateDraftID *uuid.UUID
	if req.Msg.GetTemplateDraftId() != "" {
		id, perr := uuid.Parse(req.Msg.GetTemplateDraftId())
		if perr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid template_draft_id"))
		}
		templateDraftID = &id
	}

	// Phase B2: normalise extra_symbols (dedupe + strip primary/blank entries)
	// before we persist them.
	extraSymbols := normalizeExtraSymbols(req.Msg.GetExtraSymbols(), req.Msg.Symbol)

	runID, err := s.backtestRun.CreatePending(ctx, &service.CreateBacktestRunRequest{
		UserID:          userID,
		AccountID:       accountID,
		Symbol:          req.Msg.Symbol,
		Timeframe:       req.Msg.Timeframe,
		DatasetID:       dsid,
		Mode:            modeStr,
		FromTs:          fromT,
		ToTs:            toT,
		StrategyCode:    req.Msg.Code,
		InitialCapital:  req.Msg.InitialCapital,
		TemplateID:      templateID,
		TemplateDraftID: templateDraftID,
		ExtraSymbols:    extraSymbols,
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&v1.StartBacktestRunResponse{RunId: runID.String()}), nil
}

func (s *PythonStrategyService) CancelBacktestRun(ctx context.Context, req *connect.Request[v1.CancelBacktestRunRequest]) (*connect.Response[v1.CancelBacktestRunResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}
	if s == nil || s.backtestRun == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("backtest run service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.backtestRun.RequestCancel(ctx, userID, id); err != nil {
		return nil, err
	}
	run, err := s.backtestRun.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.CancelBacktestRunResponse{Run: toProtoBacktestRun(run)}), nil
}

func (s *PythonStrategyService) DeleteBacktestRun(ctx context.Context, req *connect.Request[v1.DeleteBacktestRunRequest]) (*connect.Response[v1.DeleteBacktestRunResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}
	if s == nil || s.backtestRun == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("backtest run service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	deleted, err := s.backtestRun.Delete(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.DeleteBacktestRunResponse{Deleted: deleted}), nil
}

func (s *PythonStrategyService) WatchBacktestRun(ctx context.Context, req *connect.Request[v1.WatchBacktestRunRequest], stream *connect.ServerStream[v1.BacktestRunUpdate]) error {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return err
	}
	if s == nil || s.backtestRun == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("backtest run service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return err
	}
	id, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	var lastSig string
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		run, err := s.backtestRun.Get(ctx, userID, id)
		if err != nil {
			return err
		}
		protoRun := toProtoBacktestRun(run)
		metrics := &v1.BacktestMetrics{}
		var mp service.BacktestMetricsPython
		if run != nil && len(run.Metrics) > 0 {
			_ = json.Unmarshal(run.Metrics, &mp)
			metrics = &v1.BacktestMetrics{
				TotalReturn:   mp.TotalReturn,
				AnnualReturn:  mp.AnnualReturn,
				MaxDrawdown:   mp.MaxDrawdown,
				SharpeRatio:   mp.SharpeRatio,
				WinRate:       mp.WinRate,
				ProfitFactor:  mp.ProfitFactor,
				TotalTrades:   int32(mp.TotalTrades),
				WinningTrades: int32(mp.WinningTrades),
				LosingTrades:  int32(mp.LosingTrades),
				AverageProfit: mp.AverageProfit,
				AverageLoss:   mp.AverageLoss,
			}
		}
		var equity []float64
		if run != nil {
			_ = json.Unmarshal(run.EquityCurve, &equity)
		}

		sig := ""
		if protoRun != nil {
			sig = fmt.Sprintf("%s|%s|%s|%v|%v|%d|%d", protoRun.Id, protoRun.Status.String(), protoRun.Error, protoRun.StartedAt, protoRun.FinishedAt, len(run.Metrics), len(run.EquityCurve))
		}
		if sig != lastSig {
			if err := stream.Send(&v1.BacktestRunUpdate{Run: protoRun, Metrics: metrics, EquityCurve: equity}); err != nil {
				return err
			}
			lastSig = sig
		}

		if protoRun != nil {
			s := protoRun.Status
			if s == v1.BacktestRunStatus_BACKTEST_RUN_STATUS_SUCCEEDED || s == v1.BacktestRunStatus_BACKTEST_RUN_STATUS_FAILED || s == v1.BacktestRunStatus_BACKTEST_RUN_STATUS_CANCELED {
				return nil
			}
		}

		t := time.NewTimer(1 * time.Second)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil
		case <-t.C:
		}
	}
}

// normalizeExtraSymbols trims whitespace, drops empties, drops the primary
// symbol, and de-duplicates while preserving caller order. Returns nil when
// no secondary symbols remain (callers treat nil/empty identically).
func normalizeExtraSymbols(raw []string, primary string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" || s == primary {
			continue
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func validateStartBacktestRunRequest(req *v1.StartBacktestRunRequest) (mode string, fromT, toT *time.Time, dsid *uuid.UUID, err error) {
	if req == nil {
		return "", nil, nil, nil, errors.New("nil request")
	}
	if req.Code == "" {
		return "", nil, nil, nil, errors.New("code is required")
	}
	if req.AccountId == "" {
		return "", nil, nil, nil, errors.New("account_id is required")
	}
	if req.Symbol == "" {
		return "", nil, nil, nil, errors.New("symbol is required")
	}
	if req.Timeframe == "" {
		return "", nil, nil, nil, errors.New("timeframe is required")
	}
	if req.InitialCapital <= 0 {
		return "", nil, nil, nil, errors.New("initial_capital must be > 0")
	}

	switch req.Mode {
	case v1.BacktestRunMode_BACKTEST_RUN_MODE_KLINE_RANGE:
		if req.From == nil || req.To == nil {
			return "", nil, nil, nil, errors.New("from/to are required for KLINE_RANGE")
		}
		f := req.From.AsTime().UTC()
		t := req.To.AsTime().UTC()
		if !f.Before(t) {
			return "", nil, nil, nil, errors.New("from must be < to")
		}
		if req.GetDatasetId() != "" {
			return "", nil, nil, nil, errors.New("dataset_id must be empty for KLINE_RANGE")
		}
		return "KLINE_RANGE", &f, &t, nil, nil

	case v1.BacktestRunMode_BACKTEST_RUN_MODE_DATASET:
		if req.GetDatasetId() == "" {
			return "", nil, nil, nil, errors.New("dataset_id is required for DATASET")
		}
		if req.From != nil || req.To != nil {
			return "", nil, nil, nil, errors.New("from/to must be empty for DATASET")
		}
		id, perr := uuid.Parse(req.GetDatasetId())
		if perr != nil {
			return "", nil, nil, nil, errors.New("invalid dataset_id")
		}
		return "DATASET", nil, nil, &id, nil

	default:
		return "", nil, nil, nil, errors.New("mode is required")
	}
}

func (s *PythonStrategyService) runBacktestInBackground(userID, accountID, runID uuid.UUID, req *v1.StartBacktestRunRequest, modeStr string, fromT, toT *time.Time, dsid *uuid.UUID) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	_ = s.backtestRun.MarkRunning(bgCtx, userID, runID)

	klines, ticks, costPtr, err := s.prepareBacktestInputs(bgCtx, userID, accountID, req, modeStr, fromT, toT, dsid)
	if err != nil {
		_ = s.backtestRun.MarkFailed(bgCtx, userID, runID, err.Error())
		return
	}

	resp, err := s.pythonSvc.RunBacktest(bgCtx, uuid.New(), req.Code, klines, ticks, req.Symbol, req.Timeframe, req.InitialCapital, costPtr, nil)
	if err != nil {
		_ = s.backtestRun.MarkFailed(bgCtx, userID, runID, err.Error())
		return
	}
	if resp == nil || !resp.Success {
		errMsg := "backtest failed"
		if resp != nil && resp.Error != "" {
			errMsg = resp.Error
		}
		_ = s.backtestRun.MarkFailed(bgCtx, userID, runID, errMsg)
		return
	}

	metricsJSON := service.MarshalMetricsWithTrades(resp.Metrics, resp.Trades)
	equityJSON, _ := json.Marshal(resp.EquityCurve)
	_ = s.backtestRun.MarkSucceeded(bgCtx, userID, runID, metricsJSON, equityJSON)
}

func (s *PythonStrategyService) prepareBacktestInputs(ctx context.Context, userID, accountID uuid.UUID, req *v1.StartBacktestRunRequest, modeStr string, fromT, toT *time.Time, dsid *uuid.UUID) ([]*service.KlineResponse, []service.QuoteTickPython, *service.BacktestCostModel, error) {
	if s.klineSvc == nil {
		return nil, nil, nil, errors.New("kline service not available")
	}
	var (
		klines  []*service.KlineResponse
		costPtr *service.BacktestCostModel
	)

	switch modeStr {
	case "DATASET":
		if s.datasetSvc == nil {
			return nil, nil, nil, errors.New("dataset service not available")
		}
		if dsid == nil || *dsid == uuid.Nil {
			return nil, nil, nil, errors.New("dataset_id is required")
		}
		k, err := s.datasetSvc.GetFrozenDatasetKlines(ctx, userID, *dsid, 0)
		if err != nil {
			return nil, nil, nil, err
		}
		klines = k
		if c, ok, _ := s.datasetSvc.GetFrozenDatasetCostModel(ctx, userID, *dsid); ok {
			costPtr = c
		}

	case "KLINE_RANGE":
		if fromT == nil || toT == nil {
			return nil, nil, nil, errors.New("from/to are required")
		}
		fromUTC := fromT.UTC()
		toUTC := toT.UTC()
		nowUTC := time.Now().UTC()
		if toUTC.After(nowUTC) {
			toUTC = nowUTC
		}
		if !fromUTC.Before(toUTC) {
			return nil, nil, nil, errors.New("from must be < to (after normalization)")
		}
		// Estimate bars in [from,to) and cap.
		tfMin := timeframeToMinutes(req.Timeframe)
		if tfMin <= 0 {
			tfMin = 60
		}
		durMin := int(toUTC.Sub(fromUTC).Minutes())
		estBars := durMin/tfMin + 50
		if estBars < 200 {
			estBars = 200
		}
		if estBars > 20000 {
			estBars = 20000
		}

		k, err := s.klineSvc.GetKlines(ctx, userID, accountID, &service.KlineRequest{
			AccountID: req.AccountId,
			Symbol:    req.Symbol,
			Timeframe: req.Timeframe,
			From:      fromUTC.Format(time.RFC3339),
			To:        toUTC.Format(time.RFC3339),
			Count:     estBars,
		})
		if err != nil {
			return nil, nil, nil, err
		}
		if len(k) == 0 {
			return nil, nil, nil, errors.New("no kline data returned for the given range")
		}
		klines = k

	default:
		return nil, nil, nil, errors.New("unknown mode")
	}

	// Best-effort tick dataset replay.
	var ticks []service.QuoteTickPython
	if s.tickDataset != nil {
		fromTick, toTick := time.Now().Add(-24*time.Hour), time.Now()
		if fromT != nil {
			fromTick = fromT.UTC()
		}
		if toT != nil {
			toTick = toT.UTC()
		}
		dsid, err := s.tickDataset.CreateFrozenTickDatasetFromRedis(ctx, userID, accountID, req.Symbol, fromTick, toTick, 200000)
		if err == nil && dsid != uuid.Nil {
			rows, rerr := s.tickDataset.GetFrozenTickDatasetTicks(ctx, userID, dsid, 200000)
			if rerr == nil {
				ticks = make([]service.QuoteTickPython, 0, len(rows))
				for _, r := range rows {
					if r == nil {
						continue
					}
					ticks = append(ticks, service.QuoteTickPython{Time: r.Time, Bid: r.Bid, Ask: r.Ask, Symbol: req.Symbol})
				}
			}
		}
	}

	if costPtr == nil {
		cost := service.ResolveBacktestCostModel(ctx, s.dynamicCfg)
		costPtr = &cost
	}

	return klines, ticks, costPtr, nil
}

func (s *PythonStrategyService) GetBacktestRun(ctx context.Context, req *connect.Request[v1.GetBacktestRunRequest]) (*connect.Response[v1.GetBacktestRunResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}
	if s == nil || s.backtestRun == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("backtest run service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	run, err := s.backtestRun.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	protoRun := toProtoBacktestRun(run)
	metrics := &v1.BacktestMetrics{}
	var mp service.BacktestMetricsPython
	if len(run.Metrics) > 0 {
		_ = json.Unmarshal(run.Metrics, &mp)
		metrics = &v1.BacktestMetrics{
			TotalReturn:   mp.TotalReturn,
			AnnualReturn:  mp.AnnualReturn,
			MaxDrawdown:   mp.MaxDrawdown,
			SharpeRatio:   mp.SharpeRatio,
			WinRate:       mp.WinRate,
			ProfitFactor:  mp.ProfitFactor,
			TotalTrades:   int32(mp.TotalTrades),
			WinningTrades: int32(mp.WinningTrades),
			LosingTrades:  int32(mp.LosingTrades),
			AverageProfit: mp.AverageProfit,
			AverageLoss:   mp.AverageLoss,
		}
	}

	var equity []float64
	_ = json.Unmarshal(run.EquityCurve, &equity)

	resp := &v1.GetBacktestRunResponse{
		Run:         protoRun,
		Metrics:     metrics,
		EquityCurve: equity,
	}
	if run.DatasetID != nil {
		d := run.DatasetID.String()
		resp.DatasetId = &d
	}
	return connect.NewResponse(resp), nil
}

func (s *PythonStrategyService) ListBacktestRuns(ctx context.Context, req *connect.Request[v1.ListBacktestRunsRequest]) (*connect.Response[v1.ListBacktestRunsResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}
	if s == nil || s.backtestRun == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("backtest run service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var accountID *uuid.UUID
	if req.Msg.GetAccountId() != "" {
		id, perr := uuid.Parse(req.Msg.GetAccountId())
		if perr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, perr)
		}
		accountID = &id
	}

	items, err := s.backtestRun.List(ctx, userID, accountID, int(req.Msg.Limit), int(req.Msg.Offset))
	if err != nil {
		es := err.Error()
		if strings.Contains(es, "relation \"backtest_runs\" does not exist") || strings.Contains(es, "column \"") {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("errors.migration.backtest_runs_missing"))
		}
		return nil, err
	}

	runs := make([]*v1.BacktestRun, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		runs = append(runs, toProtoBacktestRun(it))
	}
	return connect.NewResponse(&v1.ListBacktestRunsResponse{Runs: runs}), nil
}

func toProtoBacktestRun(run *repository.BacktestRun) *v1.BacktestRun {
	if run == nil {
		return nil
	}
	out := &v1.BacktestRun{
		Id:        run.ID.String(),
		AccountId: run.AccountID.String(),
		Symbol:    run.Symbol,
		Timeframe: run.Timeframe,
		Error:     run.Error,
		CreatedAt: timestamppb.New(run.CreatedAt.UTC()),
	}
	if run.DatasetID != nil {
		d := run.DatasetID.String()
		out.DatasetId = &d
	}
	if run.PythonServiceVersion != nil {
		out.PythonServiceVersion = run.PythonServiceVersion
	}
	if run.StartedAt != nil {
		out.StartedAt = timestamppb.New(run.StartedAt.UTC())
	}
	if run.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(run.FinishedAt.UTC())
	}
	if run.FromTs != nil {
		out.From = timestamppb.New(run.FromTs.UTC())
	}
	if run.ToTs != nil {
		out.To = timestamppb.New(run.ToTs.UTC())
	}
	if run.TemplateID != nil {
		id := run.TemplateID.String()
		out.TemplateId = &id
	}
	if run.TemplateDraftID != nil {
		id := run.TemplateDraftID.String()
		out.TemplateDraftId = &id
	}
	if len(run.ExtraSymbols) > 0 {
		out.ExtraSymbols = []string(run.ExtraSymbols)
	}

	switch run.Mode {
	case "DATASET":
		out.Mode = v1.BacktestRunMode_BACKTEST_RUN_MODE_DATASET
	case "KLINE_RANGE":
		out.Mode = v1.BacktestRunMode_BACKTEST_RUN_MODE_KLINE_RANGE
	default:
		out.Mode = v1.BacktestRunMode_BACKTEST_RUN_MODE_UNSPECIFIED
	}

	switch run.Status {
	case "PENDING":
		out.Status = v1.BacktestRunStatus_BACKTEST_RUN_STATUS_PENDING
	case "RUNNING":
		out.Status = v1.BacktestRunStatus_BACKTEST_RUN_STATUS_RUNNING
	case "SUCCEEDED":
		out.Status = v1.BacktestRunStatus_BACKTEST_RUN_STATUS_SUCCEEDED
	case "FAILED":
		out.Status = v1.BacktestRunStatus_BACKTEST_RUN_STATUS_FAILED
	case "CANCEL_REQUESTED":
		out.Status = v1.BacktestRunStatus_BACKTEST_RUN_STATUS_CANCEL_REQUESTED
	case "CANCELED":
		out.Status = v1.BacktestRunStatus_BACKTEST_RUN_STATUS_CANCELED
	default:
		out.Status = v1.BacktestRunStatus_BACKTEST_RUN_STATUS_UNSPECIFIED
	}
	out.IsTerminal = run.Status == "SUCCEEDED" || run.Status == "FAILED" || run.Status == "CANCELED"
	out.IsSucceeded = run.Status == "SUCCEEDED"

	return out
}
