package marketplace

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
)

func (s *MarketplaceServer) GetLivePerformance(ctx context.Context, req *connect.Request[antv1.GetLivePerformanceRequest]) (*connect.Response[antv1.GetLivePerformanceResponse], error) {
	m := req.Msg
	if m.StrategyId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("strategy_id is required"))
	}

	points, summary, err := s.svc.GetLivePerformance(ctx, m.StrategyId, int(m.Limit))
	if err != nil {
		s.log.Error("GetLivePerformance", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &antv1.GetLivePerformanceResponse{}
	for _, p := range points {
		resp.Points = append(resp.Points, &antv1.LivePerformancePoint{
			Date:          p.Date.Format("2006-01-02"),
			DailyPnl:      p.DailyPnL.StringFixed(2),
			DailyReturn:   p.DailyReturn.StringFixed(6),
			Equity:        p.Equity.StringFixed(2),
			Drawdown:      p.Drawdown.StringFixed(6),
			TotalTrades:   p.TotalTrades,
			WinningTrades: p.WinningTrades,
		})
	}
	if summary != nil {
		resp.Summary = &antv1.LivePerformanceSummary{
			TotalReturn:   summary.TotalReturn.StringFixed(6),
			MaxDrawdown:   summary.MaxDrawdown.StringFixed(6),
			TotalTrades:   summary.TotalTrades,
			TrackingSince: summary.TrackingSince.Format("2006-01-02"),
			LastUpdated:   summary.LastUpdated.Format("2006-01-02"),
		}
		if summary.AnnualReturn != nil {
			resp.Summary.AnnualReturn = summary.AnnualReturn.StringFixed(6)
		}
		if summary.SharpeRatio != nil {
			resp.Summary.SharpeRatio = summary.SharpeRatio.StringFixed(6)
		}
		if summary.WinRate != nil {
			resp.Summary.WinRate = summary.WinRate.StringFixed(6)
		}
		if summary.AvgMonthlyReturn != nil {
			resp.Summary.AvgMonthlyReturn = summary.AvgMonthlyReturn.StringFixed(6)
		}
	}

	return connect.NewResponse(resp), nil
}

func (s *MarketplaceServer) LinkLiveAccount(ctx context.Context, req *connect.Request[antv1.LinkLiveAccountRequest]) (*connect.Response[antv1.LinkLiveAccountResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	m := req.Msg
	if m.StrategyId == "" || m.AccountId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("strategy_id and account_id are required"))
	}

	if err := s.svc.LinkLiveAccount(ctx, m.StrategyId, m.AccountId); err != nil {
		s.log.Error("LinkLiveAccount", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.LinkLiveAccountResponse{Success: true}), nil
}
