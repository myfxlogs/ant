package strategy

import (
	"context"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
)

func (s *StrategyServer) RunBacktest(
	ctx context.Context, req *connect.Request[antv1.RunBacktestRequest],
) (*connect.Response[antv1.RunBacktestResponse], error) {
	m := req.Msg
	if s.backtestClient == nil || m.TemplateId == "" {
		return connect.NewResponse(&antv1.RunBacktestResponse{
			Success: false, RiskLevel: "unknown",
		}), nil
	}
	tid, err := uuid.Parse(m.TemplateId)
	if err != nil {
		return connect.NewResponse(&antv1.RunBacktestResponse{
			Success: false, RiskLevel: "unknown",
		}), nil
	}
	userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
	tmpl, err := s.svc.GetTemplate(ctx, tid, userID)
	if err != nil || tmpl.Code == "" {
		s.log.Warn("RunBacktest: get template failed", zap.Error(err))
		return connect.NewResponse(&antv1.RunBacktestResponse{
			Success: false, RiskLevel: "unknown",
		}), nil
	}
	return s.runConnectBacktest(ctx, tmpl.Code, m)
}

// runConnectBacktest executes a ConnectRPC backtest and maps the result.
func (s *StrategyServer) runConnectBacktest(
	ctx context.Context, code string, m *antv1.RunBacktestRequest,
) (*connect.Response[antv1.RunBacktestResponse], error) {
	balance := m.InitialCapital
	if balance <= 0 {
		balance = 10000
	}
	klines, err := fetchKlines(ctx, s.marketDataRepo, m.Symbol, m.Timeframe)
	if err != nil {
		s.log.Warn("RunBacktest: fetch klines failed", zap.Error(err))
	}
	resp, err := s.backtestClient.RunBacktest(ctx, connect.NewRequest(
		&antv1.ExecuteBacktestRequest{
			StrategyCode:   code,
			Symbol:         m.Symbol,
			Timeframe:      m.Timeframe,
			InitialCapital: balance,
			Klines:         klines,
		}))
	if err != nil {
		s.log.Warn("RunBacktest: backtest failed", zap.Error(err))
		return connect.NewResponse(&antv1.RunBacktestResponse{
			Success: false, RiskLevel: "unknown",
		}), nil
	}
	if !resp.Msg.GetSuccess() {
		return connect.NewResponse(&antv1.RunBacktestResponse{
			Success: false, RiskLevel: "unknown",
		}), nil
	}
	return connect.NewResponse(mapBacktestResult(resp.Msg)), nil
}

// mapBacktestResult converts ExecuteBacktestResponse → RunBacktestResponse.
func mapBacktestResult(resp *antv1.ExecuteBacktestResponse) *antv1.RunBacktestResponse {
	met := resp.GetMetrics()
	rsk := resp.GetRisk()
	riskLevel := "medium"
	riskScore := int32(0)
	var riskReasons, riskWarnings []string
	isReliable := false
	if rsk != nil {
		riskLevel = rsk.GetLevel()
		riskScore = rsk.GetScore()
		riskReasons = rsk.GetReasons()
		riskWarnings = rsk.GetWarnings()
		isReliable = rsk.GetIsReliable()
	}
	if riskLevel == "" {
		riskLevel = "medium"
	}
	if !isReliable && met.GetTotalTrades() >= 10 {
		isReliable = true
	}
	return &antv1.RunBacktestResponse{
		Success: true,
		Metrics: &antv1.BacktestMetrics{
			TotalReturn: met.GetTotalReturn(), AnnualReturn: met.GetAnnualReturn(),
			MaxDrawdown: met.GetMaxDrawdown(), SharpeRatio: met.GetSharpeRatio(),
			WinRate: met.GetWinRate(), ProfitFactor: met.GetProfitFactor(),
			TotalTrades: met.GetTotalTrades(), WinningTrades: met.GetWinningTrades(),
			LosingTrades: met.GetLosingTrades(), AverageProfit: met.GetAverageProfit(),
			AverageLoss: met.GetAverageLoss(),
		},
		RiskScore: riskScore, RiskLevel: riskLevel,
		RiskReasons: riskReasons, RiskWarnings: riskWarnings,
		IsReliable: isReliable,
	}
}

// fetchKlines is the shared helper for fetching and transforming ClickHouse K-lines.
// Used by both StrategyServer and PythonStrategyServer sync backtest paths.
func fetchKlines(
	ctx context.Context, repo repository.MarketDataStore,
	symbol, timeframe string,
) ([]*antv1.ExecuteKlineBar, error) {
	if repo == nil {
		return nil, nil
	}
	fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	to := time.Now()
	from := to.AddDate(0, -1, 0)
	bars, err := repo.GetKlines(fetchCtx, symbol, "", timeframe, &from, &to, 2000)
	if err != nil {
		return nil, err
	}
	out := make([]*antv1.ExecuteKlineBar, len(bars))
	for i, b := range bars {
		out[i] = &antv1.ExecuteKlineBar{
			OpenTimeMs:  int64(b.OpenTsUnixMs),
			CloseTimeMs: int64(b.CloseTsUnixMs),
			Open:        b.Open, High: b.High, Low: b.Low,
			Close:       b.Close, Volume: b.Volume,
		}
	}
	return out, nil
}

// --- Signals ---

func (s *StrategyServer) ListSignals(
	ctx context.Context, req *connect.Request[antv1.ListSignalsRequest],
) (*connect.Response[antv1.ListSignalsResponse], error) {
	m := req.Msg
	accountID, _ := uuid.Parse(m.AccountId)
	uid := s.userID(ctx)
	rows, err := s.svc.ListSignals(ctx, uid, accountID, m.Status)
	if err != nil {
		return nil, err
	}
	signals := make([]*antv1.StrategySignal, len(rows))
	for i, r := range rows {
		signals[i] = signalRowToProto(&r)
	}
	return connect.NewResponse(&antv1.ListSignalsResponse{Signals: signals}), nil
}

func (s *StrategyServer) ExecuteSignal(
	ctx context.Context, req *connect.Request[antv1.ExecuteSignalRequest],
) (*connect.Response[antv1.ExecuteSignalResponse], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	row, err := s.svc.ExecuteSignal(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &antv1.ExecuteSignalResponse{
		Ticket: row.Ticket, Symbol: row.Symbol,
		Type: row.SignalType, Volume: row.Volume, Price: row.Price,
	}
	if row.ExecutedAt != nil {
		resp.ExecutedAt = timestamppb.New(*row.ExecutedAt)
	}
	return connect.NewResponse(resp), nil
}

func (s *StrategyServer) ConfirmSignal(
	ctx context.Context, req *connect.Request[antv1.ConfirmSignalRequest],
) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.ConfirmSignal(ctx, id, s.userID(ctx)); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyServer) CancelSignal(
	ctx context.Context, req *connect.Request[antv1.CancelSignalRequest],
) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.CancelSignal(ctx, id, s.userID(ctx)); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}
