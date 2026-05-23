package connect

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/service"
)

func (s *StrategyService) RunBacktest(ctx context.Context, req *connect.Request[v1.RunBacktestRequest]) (*connect.Response[v1.RunBacktestResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.schedule == nil {
		return connect.NewResponse(&v1.RunBacktestResponse{Success: false, Error: "backtest service not available"}), nil
	}

	tplID, err := uuid.Parse(req.Msg.TemplateId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	accID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	params := make(map[string]interface{}, len(req.Msg.Parameters))
	for k, v := range req.Msg.Parameters {
		params[k] = v
	}

	btReq := &service.RunBacktestRequest{
		TemplateID:     tplID,
		AccountID:      accID,
		Symbol:         req.Msg.Symbol,
		Timeframe:      req.Msg.Timeframe,
		Parameters:     params,
		InitialCapital: req.Msg.InitialCapital,
		DatasetID:      req.Msg.GetDatasetId(),
	}
	btResp, err := s.schedule.RunBacktest(ctx, userID, btReq)
	if err != nil {
		return connect.NewResponse(&v1.RunBacktestResponse{Success: false, Error: err.Error()}), nil
	}

	resp := &v1.RunBacktestResponse{
		Success:      btResp.Success,
		RiskScore:     int32(btResp.RiskScore),
		RiskLevel:     btResp.RiskLevel,
		RiskReasons:   btResp.RiskReasons,
		RiskWarnings:  btResp.RiskWarnings,
		IsReliable:    btResp.IsReliable,
		Error:         btResp.Error,
	}
	if btResp.Metrics != nil {
		resp.Metrics = &v1.BacktestMetrics{
			TotalReturn:   btResp.Metrics.TotalReturn,
			AnnualReturn:  btResp.Metrics.AnnualReturn,
			MaxDrawdown:   btResp.Metrics.MaxDrawdown,
			SharpeRatio:   btResp.Metrics.SharpeRatio,
			WinRate:       btResp.Metrics.WinRate,
			ProfitFactor:  btResp.Metrics.ProfitFactor,
			TotalTrades:   int32(btResp.Metrics.TotalTrades),
			WinningTrades: int32(btResp.Metrics.WinningTrades),
			LosingTrades:  int32(btResp.Metrics.LosingTrades),
			AverageProfit: btResp.Metrics.AverageProfit,
			AverageLoss:   btResp.Metrics.AverageLoss,
		}
	}
	if btResp.DatasetID != "" {
		dsid := btResp.DatasetID
		resp.DatasetId = &dsid
	}
	return connect.NewResponse(resp), nil
}

func (s *StrategyService) ListSignals(ctx context.Context, req *connect.Request[v1.ListSignalsRequest]) (*connect.Response[v1.ListSignalsResponse], error) {
	return connect.NewResponse(&v1.ListSignalsResponse{
		Signals: []*v1.StrategySignal{},
	}), nil
}

func (s *StrategyService) ExecuteSignal(ctx context.Context, req *connect.Request[v1.ExecuteSignalRequest]) (*connect.Response[v1.ExecuteSignalResponse], error) {
	return connect.NewResponse(&v1.ExecuteSignalResponse{
		Ticket:     12345,
		Symbol:     "EURUSD",
		Type:       "buy",
		Volume:     0.1,
		Price:      1.0850,
		ExecutedAt: timestamppb.Now(),
	}), nil
}

func (s *StrategyService) ConfirmSignal(ctx context.Context, req *connect.Request[v1.ConfirmSignalRequest]) (*connect.Response[emptypb.Empty], error) {
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyService) CancelSignal(ctx context.Context, req *connect.Request[v1.CancelSignalRequest]) (*connect.Response[emptypb.Empty], error) {
	return connect.NewResponse(&emptypb.Empty{}), nil
}
