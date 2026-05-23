package connect

import (
	"context"

	"connectrpc.com/connect"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/service"
)

type ObjectiveScoreService struct {
	pythonSvc *service.PythonStrategyService
}

func NewObjectiveScoreService(pythonSvc *service.PythonStrategyService) *ObjectiveScoreService {
	return &ObjectiveScoreService{pythonSvc: pythonSvc}
}

func (s *ObjectiveScoreService) CalculateObjectiveScore(ctx context.Context, req *connect.Request[v1.ObjectiveScoreRequest]) (*connect.Response[v1.ObjectiveScoreResponse], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeStrategyRun); err != nil {
		return nil, err
	}
	if _, err := getUserIDFromContext(ctx); err != nil {
		return nil, err
	}
	klines := make([]service.ObjectiveKlinePython, 0, len(req.Msg.Klines))
	for _, k := range req.Msg.Klines {
		if k == nil {
			continue
		}
		klines = append(klines, service.ObjectiveKlinePython{OpenTime: k.OpenTime, CloseTime: k.CloseTime, OpenPrice: k.OpenPrice, HighPrice: k.HighPrice, LowPrice: k.LowPrice, ClosePrice: k.ClosePrice, Volume: k.Volume})
	}
	resp, err := s.pythonSvc.CalculateObjectiveScore(ctx, service.ObjectiveScoreRequestPython{Symbol: req.Msg.Symbol, Timeframe: req.Msg.Timeframe, Klines: klines})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(toObjectiveScoreProto(resp)), nil
}

func toObjectiveScoreProto(resp *service.ObjectiveScoreResponsePython) *v1.ObjectiveScoreResponse {
	out := &v1.ObjectiveScoreResponse{Decision: resp.Decision, OverallScore: resp.OverallScore, TechnicalScore: resp.TechnicalScore}
	if resp.Signals == nil {
		return out
	}
	out.Signals = &v1.ObjectiveSignals{}
	if resp.Signals.RSI != nil {
		out.Signals.Rsi = &v1.RSISignal{Value: resp.Signals.RSI.Value, Signal: resp.Signals.RSI.Signal}
	}
	if resp.Signals.MACD != nil {
		m := resp.Signals.MACD
		out.Signals.Macd = &v1.MACDSignal{Value: m.Value, SignalLine: m.SignalLine, Histogram: m.Histogram, Signal: m.Signal, Trend: m.Trend}
	}
	if resp.Signals.MA != nil {
		m := resp.Signals.MA
		out.Signals.Ma = &v1.MASignal{Ma5: m.MA5, Ma10: m.MA10, Ma20: m.MA20, Trend: m.Trend}
	}
	return out
}
