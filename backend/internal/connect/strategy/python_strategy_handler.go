package strategy

import (
	"context"
	"fmt"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	"anttrader/internal/strategysvc"
)

// PythonStrategyServer implements ant.v1.PythonStrategyServiceHandler.
type PythonStrategyServer struct {
	backtestRepo  *repository.BacktestRunRepository
	log           *zap.Logger
	client        *strategysvc.PythonClient
	backtestClient antv1c.BacktestServiceClient
	marketDataRepo *repository.MarketDataRepository
}

func (s *PythonStrategyServer) SetMarketDataRepo(r *repository.MarketDataRepository) {
	s.marketDataRepo = r
}

var _ antv1c.PythonStrategyServiceHandler = (*PythonStrategyServer)(nil)

func NewPythonStrategyServer(backtestRepo *repository.BacktestRunRepository, log *zap.Logger) *PythonStrategyServer {
	return &PythonStrategyServer{backtestRepo: backtestRepo, log: log}
}

func (s *PythonStrategyServer) SetClient(c *strategysvc.PythonClient) { s.client = c }
func (s *PythonStrategyServer) SetBacktestClient(c antv1c.BacktestServiceClient) { s.backtestClient = c }

// userIDRequire extracts and validates the authenticated user ID from context.
func userIDRequire(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	return id, nil
}

func (s *PythonStrategyServer) Execute(ctx context.Context, req *connect.Request[antv1.ExecuteStrategyRequest]) (*connect.Response[antv1.ExecuteStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid // authorization verified; downstream uses AccountId directly
	if s.client != nil {
		result, err := s.client.Execute(ctx, &strategysvc.ExecuteRequest{
			Code:      req.Msg.Code,
			AccountID: req.Msg.AccountId,
			Symbol:    req.Msg.Symbol,
			Timeframe: req.Msg.Timeframe,
			Mode:      "paper",
		})
		if err != nil {
			s.log.Warn("python execute failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if result.Success && result.Signal != nil {
			return connect.NewResponse(&antv1.ExecuteStrategyResponse{
				Success: true,
				Signal: &antv1.StrategySignal{
					SignalType: result.Signal.Side,
					Volume:     result.Signal.Lots,
					Price:      result.Signal.Price,
					StopLoss:   result.Signal.StopLoss,
					TakeProfit: result.Signal.TakeProfit,
					Reason:     result.Signal.Reason,
				},
			}), nil
		}
	}
	return connect.NewResponse(&antv1.ExecuteStrategyResponse{
		Success: false,
	}), nil
}

func (s *PythonStrategyServer) Validate(ctx context.Context, req *connect.Request[antv1.ValidateStrategyRequest]) (*connect.Response[antv1.ValidateStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid
	if s.client != nil {
		result, err := s.client.Validate(ctx, &strategysvc.ValidateRequest{Code: req.Msg.Code})
		if err != nil {
			s.log.Warn("python validate failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&antv1.ValidateStrategyResponse{
			Valid:    result.Valid,
			Errors:   result.Errors,
			Warnings: result.Warnings,
		}), nil
	}
	return connect.NewResponse(&antv1.ValidateStrategyResponse{
		Valid: true,
	}), nil
}

func (s *PythonStrategyServer) Backtest(ctx context.Context, req *connect.Request[antv1.BacktestStrategyRequest]) (*connect.Response[antv1.BacktestStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid
	if s.client != nil {
		result, err := s.client.Backtest(ctx, &strategysvc.BacktestRequest{
			Code:      req.Msg.Code,
			Symbol:    req.Msg.Symbol,
			Timeframe: req.Msg.Timeframe,
			Capital:   10000,
		})
		if err != nil {
			s.log.Warn("python backtest failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if result.Success {
			return connect.NewResponse(&antv1.BacktestStrategyResponse{
				Success:     true,
				EquityCurve: result.EquityCurve,
				Metrics: &antv1.BacktestMetrics{
					TotalReturn:   result.TotalReturn,
					AnnualReturn:  result.AnnualReturn,
					MaxDrawdown:   result.MaxDrawdown,
					SharpeRatio:   result.SharpeRatio,
					WinRate:       result.WinRate,
					ProfitFactor:  result.ProfitFactor,
					TotalTrades:   result.TotalTrades,
					WinningTrades: result.WinningTrades,
					LosingTrades:  result.LosingTrades,
					AverageProfit: result.AverageProfit,
					AverageLoss:   result.AverageLoss,
				},
			}), nil
		}
	}
	return connect.NewResponse(&antv1.BacktestStrategyResponse{
		Success: false,
	}), nil
}

func (s *PythonStrategyServer) StartBacktestRun(ctx context.Context, req *connect.Request[antv1.StartBacktestRunRequest]) (*connect.Response[antv1.StartBacktestRunResponse], error) {
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	accountID, _ := uuid.Parse(req.Msg.AccountId)
	mode := backtestModeToString(req.Msg.Mode)
	run := &repository.BacktestRun{
		ID:            uuid.New(),
		UserID:        userID,
		AccountID:     accountID,
		Symbol:        req.Msg.Symbol,
		Timeframe:     req.Msg.Timeframe,
		Mode:          mode,
		Status:        "PENDING",
		StrategyCode:  strPtr(req.Msg.Code),
		InitialCapital: f64Ptr(req.Msg.InitialCapital),
	}
	if run.Mode == "" {
		run.Mode = "KLINE_RANGE"
	}
	if req.Msg.InitialCapital <= 0 {
		run.InitialCapital = f64Ptr(10000)
	}
	if req.Msg.From != nil {
		t := req.Msg.From.AsTime()
		run.FromTs = &t
	}
	if req.Msg.To != nil {
		t := req.Msg.To.AsTime()
		run.ToTs = &t
	}
	run.ExtraSymbols = req.Msg.ExtraSymbols
	if req.Msg.DatasetId != nil {
		id, _ := uuid.Parse(*req.Msg.DatasetId)
		if id != uuid.Nil {
			run.DatasetID = &id
		}
	}
	if req.Msg.TemplateId != nil {
		id, _ := uuid.Parse(*req.Msg.TemplateId)
		if id != uuid.Nil {
			run.TemplateID = &id
		}
	}
	runID, err := s.backtestRepo.Create(ctx, run)
	if err != nil {
		s.log.Error("StartBacktestRun: create", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.StartBacktestRunResponse{
		RunId: runID.String(),
	}), nil
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
		Run:         toProtoBacktestRun(run),
		Metrics:     parseMetrics(run.Metrics),
		EquityCurve: parseEquityCurve(run.EquityCurve),
		Risk:        parseRisk(run.Metrics),
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
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	prevStatus := ""
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		run, err := s.backtestRepo.GetByID(ctx, userID, runID)
		if err != nil || run == nil {
			continue
		}
		if run.Status == prevStatus {
			continue
		}
		prevStatus = run.Status
		if err := stream.Send(&antv1.BacktestRunUpdate{
			Run:         toProtoBacktestRun(run),
			Metrics:     parseMetrics(run.Metrics),
			EquityCurve: parseEquityCurve(run.EquityCurve),
			Risk:        parseRisk(run.Metrics),
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

func (s *PythonStrategyServer) GetTemplates(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetPythonTemplatesResponse], error) {
	return connect.NewResponse(&antv1.GetPythonTemplatesResponse{
		Templates: []*antv1.PythonTemplate{
			{Name: "MA Crossover", Description: "双均线交叉策略", Code: `# @param fast_period 10 range=5:50:5
# @param slow_period 30 range=10:100:10
def run(context):
    p = context.get('params', {})
    fast_period = int(p.get('fast_period', 10))
    slow_period = int(p.get('slow_period', 30))
    prices = context['close']
    if len(prices) < slow_period + 1:
        return {'signal': 'hold', 'volume': 0}
    fast_ma = sum(prices[-fast_period:]) / fast_period
    slow_ma = sum(prices[-slow_period:]) / slow_period
    pos = context.get('position')
    if fast_ma > slow_ma:
        if pos and pos.get('side') == 'buy':
            return {'signal': 'hold', 'volume': 0}
        if pos:
            return {'signal': 'close', 'volume': 0}
        return {'signal': 'buy', 'volume': 1.0}
    elif fast_ma < slow_ma:
        if pos and pos.get('side') == 'sell':
            return {'signal': 'hold', 'volume': 0}
        if pos:
            return {'signal': 'close', 'volume': 0}
        return {'signal': 'sell', 'volume': 1.0}
    return {'signal': 'hold', 'volume': 0}`},
		{Name: "RSI Mean Reversion", Description: "RSI超买超卖反转策略", Code: `# @param rsi_period 14 range=7:28:7
# @param oversold 30 range=20:40:5
# @param overbought 70 range=60:80:5
def run(context):
    p = context.get('params', {})
    rsi_period = int(p.get('rsi_period', 14))
    oversold = int(p.get('oversold', 30))
    overbought = int(p.get('overbought', 70))
    prices = context['close']
    if len(prices) < rsi_period + 1:
        return {'signal': 'hold', 'volume': 0}
    deltas = [prices[i] - prices[i-1] for i in range(1, len(prices))]
    gains = [d if d > 0 else 0 for d in deltas[-rsi_period:]]
    losses = [-d if d < 0 else 0 for d in deltas[-rsi_period:]]
    avg_gain = sum(gains) / rsi_period if gains else 0
    avg_loss = sum(losses) / rsi_period if losses else 1e-10
    rs = avg_gain / avg_loss
    rsi = 100 - (100 / (1 + rs))
    pos = context.get('position')
    if rsi < oversold:
        if pos and pos.get('side') == 'sell':
            return {'signal': 'close', 'volume': 0}
        if not pos:
            return {'signal': 'buy', 'volume': 1.0}
    elif rsi > overbought:
        if pos and pos.get('side') == 'buy':
            return {'signal': 'close', 'volume': 0}
        if not pos:
            return {'signal': 'sell', 'volume': 1.0}
    return {'signal': 'hold', 'volume': 0}`},
		{Name: "Bollinger Breakout", Description: "布林带突破策略", Code: `# @param bb_period 20 range=10:50:10
# @param bb_std 2.0 range=1.0:4.0:0.5
def run(context):
    p = context.get('params', {})
    bb_period = int(p.get('bb_period', 20))
    bb_std = float(p.get('bb_std', 2.0))
    import math
    prices = context['close']
    if len(prices) < bb_period + 1:
        return {'signal': 'hold', 'volume': 0}
    window = prices[-bb_period:]
    ma = sum(window) / bb_period
    variance = sum((x - ma) ** 2 for x in window) / bb_period
    std = math.sqrt(variance)
    upper = ma + bb_std * std
    lower = ma - bb_std * std
    last_price = prices[-1]
    pos = context.get('position')
    if last_price > upper:
        if pos and pos.get('side') == 'sell':
            return {'signal': 'close', 'volume': 0}
        if not pos:
            return {'signal': 'buy', 'volume': 1.0}
    elif last_price < lower:
        if pos and pos.get('side') == 'buy':
            return {'signal': 'close', 'volume': 0}
        if not pos:
            return {'signal': 'sell', 'volume': 1.0}
    return {'signal': 'hold', 'volume': 0}`},
		},
	}), nil
}

func toProtoBacktestRun(r *repository.BacktestRun) *antv1.BacktestRun {
	if r == nil {
		return nil
	}
	out := &antv1.BacktestRun{
		Id:        r.ID.String(),
		Symbol:    r.Symbol,
		Timeframe: r.Timeframe,
		Mode:      stringToBacktestMode(r.Mode),
		Status:    backtestStatusToProto(r.Status),
		Error:     r.Error,
	}
	if r.AccountID != uuid.Nil {
		out.AccountId = r.AccountID.String()
	}
	if r.StartedAt != nil {
		out.StartedAt = timestamppb.New(*r.StartedAt)
	}
	if r.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(*r.FinishedAt)
	}
	out.CreatedAt = timestamppb.New(r.CreatedAt)
	return out
}

func backtestStatusToProto(s string) antv1.BacktestRunStatus {
	switch s {
	case "PENDING":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_PENDING
	case "RUNNING":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_RUNNING
	case "SUCCEEDED":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_SUCCEEDED
	case "FAILED":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_FAILED
	case "CANCEL_REQUESTED":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_CANCEL_REQUESTED
	case "CANCELED":
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_CANCELED
	default:
		return antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_UNSPECIFIED
	}
}

func backtestModeToString(m antv1.BacktestRunMode) string {
	switch m {
	case antv1.BacktestRunMode_BACKTEST_RUN_MODE_DATASET:
		return "DATASET"
	default:
		return "KLINE_RANGE"
	}
}

func stringToBacktestMode(s string) antv1.BacktestRunMode {
	switch s {
	case "DATASET":
		return antv1.BacktestRunMode_BACKTEST_RUN_MODE_DATASET
	default:
		return antv1.BacktestRunMode_BACKTEST_RUN_MODE_KLINE_RANGE
	}
}

func parseMetrics(raw []byte) *antv1.BacktestMetrics {
	if len(raw) == 0 {
		return nil
	}
	var m antv1.BacktestMetrics
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return &m
}

func parseRisk(raw []byte) *antv1.BacktestRisk {
	if len(raw) == 0 {
		return nil
	}
	var r antv1.BacktestRisk
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil
	}
	return &r
}

func parseEquityCurve(raw []byte) []float64 {
	if len(raw) == 0 {
		return nil
	}
	var ec []float64
	if err := json.Unmarshal(raw, &ec); err != nil {
		return nil
	}
	return ec
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func f64Ptr(v float64) *float64 { return &v }
