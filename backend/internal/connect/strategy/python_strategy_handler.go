package strategy

import (
	"context"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/strategysvc"
)

// PythonStrategyServer implements ant.v1.PythonStrategyServiceHandler.
type PythonStrategyServer struct {
	log    *zap.Logger
	client *strategysvc.PythonClient // nil if strategy-service not configured (S2.1)
}

var _ antv1c.PythonStrategyServiceHandler = (*PythonStrategyServer)(nil)

func NewPythonStrategyServer(log *zap.Logger) *PythonStrategyServer {
	return &PythonStrategyServer{log: log}
}

// SetClient injects the Python strategy-service client (S2.1).
func (s *PythonStrategyServer) SetClient(c *strategysvc.PythonClient) { s.client = c }

func (s *PythonStrategyServer) Execute(ctx context.Context, req *connect.Request[antv1.ExecuteStrategyRequest]) (*connect.Response[antv1.ExecuteStrategyResponse], error) {
	if s.client != nil {
		result, err := s.client.Execute(ctx, &strategysvc.ExecuteRequest{
			Code:      req.Msg.Code,
			AccountID: req.Msg.AccountId,
			Symbol:    req.Msg.Symbol,
			Timeframe: req.Msg.Timeframe,
			Mode:      "paper",
		})
		if err != nil {
			s.log.Warn("python execute failed, falling back to mock", zap.Error(err))
		} else if result.Success && result.Signal != nil {
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
		Success: true,
		Signal:  &antv1.StrategySignal{Reason: "mock execution"},
	}), nil
}

func (s *PythonStrategyServer) Validate(ctx context.Context, req *connect.Request[antv1.ValidateStrategyRequest]) (*connect.Response[antv1.ValidateStrategyResponse], error) {
	if s.client != nil {
		result, err := s.client.Validate(ctx, &strategysvc.ValidateRequest{Code: req.Msg.Code})
		if err != nil {
			s.log.Warn("python validate failed, falling back to mock", zap.Error(err))
		} else {
			return connect.NewResponse(&antv1.ValidateStrategyResponse{
				Valid:    result.Valid,
				Errors:   result.Errors,
				Warnings: result.Warnings,
			}), nil
		}
	}
	return connect.NewResponse(&antv1.ValidateStrategyResponse{
		Valid: true,
	}), nil
}

func (s *PythonStrategyServer) Backtest(ctx context.Context, req *connect.Request[antv1.BacktestStrategyRequest]) (*connect.Response[antv1.BacktestStrategyResponse], error) {
	if s.client != nil {
		result, err := s.client.Backtest(ctx, &strategysvc.BacktestRequest{
			Code:      req.Msg.Code,
			Symbol:    req.Msg.Symbol,
			Timeframe: req.Msg.Timeframe,
			Balance:   10000,
		})
		if err != nil {
			s.log.Warn("python backtest failed, falling back to mock", zap.Error(err))
		} else if result.Success {
			return connect.NewResponse(&antv1.BacktestStrategyResponse{
				Success:     true,
				EquityCurve: result.EquityCurve,
				Metrics: &antv1.BacktestMetrics{
					SharpeRatio: result.SharpeRatio,
					MaxDrawdown: result.MaxDrawdown,
				},
			}), nil
		}
	}
	return connect.NewResponse(&antv1.BacktestStrategyResponse{
		Success:     true,
		EquityCurve: []float64{10000, 10050, 10100, 10080, 10150, 10200},
		Metrics:     &antv1.BacktestMetrics{SharpeRatio: 1.5, MaxDrawdown: 0.05},
	}), nil
}

func (s *PythonStrategyServer) StartBacktestRun(ctx context.Context, req *connect.Request[antv1.StartBacktestRunRequest]) (*connect.Response[antv1.StartBacktestRunResponse], error) {
	return connect.NewResponse(&antv1.StartBacktestRunResponse{
		RunId: uuid.New().String(),
	}), nil
}

func (s *PythonStrategyServer) GetBacktestRun(ctx context.Context, req *connect.Request[antv1.GetBacktestRunRequest]) (*connect.Response[antv1.GetBacktestRunResponse], error) {
	return connect.NewResponse(&antv1.GetBacktestRunResponse{
		Run: &antv1.BacktestRun{Id: req.Msg.RunId, Status: antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_SUCCEEDED},
	}), nil
}

func (s *PythonStrategyServer) ListBacktestRuns(ctx context.Context, req *connect.Request[antv1.ListBacktestRunsRequest]) (*connect.Response[antv1.ListBacktestRunsResponse], error) {
	return connect.NewResponse(&antv1.ListBacktestRunsResponse{
		Runs: []*antv1.BacktestRun{},
	}), nil
}

func (s *PythonStrategyServer) WatchBacktestRun(ctx context.Context, req *connect.Request[antv1.WatchBacktestRunRequest], stream *connect.ServerStream[antv1.BacktestRunUpdate]) error {
	<-ctx.Done()
	return nil
}

func (s *PythonStrategyServer) CancelBacktestRun(ctx context.Context, req *connect.Request[antv1.CancelBacktestRunRequest]) (*connect.Response[antv1.CancelBacktestRunResponse], error) {
	return connect.NewResponse(&antv1.CancelBacktestRunResponse{
		Run: &antv1.BacktestRun{Id: req.Msg.RunId, Status: antv1.BacktestRunStatus_BACKTEST_RUN_STATUS_FAILED},
	}), nil
}

func (s *PythonStrategyServer) DeleteBacktestRun(ctx context.Context, req *connect.Request[antv1.DeleteBacktestRunRequest]) (*connect.Response[antv1.DeleteBacktestRunResponse], error) {
	return connect.NewResponse(&antv1.DeleteBacktestRunResponse{}), nil
}

func (s *PythonStrategyServer) GetTemplates(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetPythonTemplatesResponse], error) {
	return connect.NewResponse(&antv1.GetPythonTemplatesResponse{
		Templates: []*antv1.PythonTemplate{
			{Name: "均线交叉策略", Description: "MA交叉产生买卖信号", Code: "# SMA crossover\n..."},
			{Name: "RSI反转策略", Description: "RSI超买超卖区间交易", Code: "# RSI strategy\n..."},
			{Name: "布林带突破", Description: "布林带上轨下轨突破信号", Code: "# Bollinger Bands\n..."},
		},
	}), nil
}
