package connect

import (
	"context"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/repository"
)

// AdminTradingServer implements ant.v1.AdminTradingServiceHandler.
type AdminTradingServer struct {
	repo *repository.AdminRepository
	log  *zap.Logger
}

var _ antv1c.AdminTradingServiceHandler = (*AdminTradingServer)(nil)

func NewAdminTradingServer(repo *repository.AdminRepository, log *zap.Logger) *AdminTradingServer {
	return &AdminTradingServer{repo: repo, log: log}
}

func (s *AdminTradingServer) GetTradingSummary(ctx context.Context, req *connect.Request[antv1.GetTradingSummaryRequest]) (*connect.Response[antv1.TradingSummary], error) {
	summary, err := s.repo.GetTradingSummary(ctx, req.Msg.StartDate, req.Msg.EndDate)
	if err != nil {
		return nil, err
	}

	byPlatform := make(map[string]*antv1.PlatformSummary, len(summary.ByPlatform))
	for k, v := range summary.ByPlatform {
		byPlatform[k] = &antv1.PlatformSummary{
			Accounts: v.Accounts,
			Orders:   v.Orders,
			Volume:   v.Volume,
		}
	}

	return connect.NewResponse(&antv1.TradingSummary{
		Period: &antv1.TradingPeriod{
			StartDate: summary.Period.StartDate,
			EndDate:   summary.Period.EndDate,
		},
		Overview: &antv1.TradingOverview{
			TotalUsers:        summary.Overview.TotalUsers,
			ActiveUsers:       summary.Overview.ActiveUsers,
			TotalAccounts:     summary.Overview.TotalAccounts,
			ConnectedAccounts: summary.Overview.ConnectedAccounts,
		},
		Trading: &antv1.TradingStats{
			TotalOrders:   summary.Trading.TotalOrders,
			ClosedOrders:  summary.Trading.ClosedOrders,
			PendingOrders: summary.Trading.PendingOrders,
			TotalVolume:   summary.Trading.TotalVolume,
			TotalProfit:   summary.Trading.TotalProfit,
			TotalLoss:     summary.Trading.TotalLoss,
			NetProfit:     summary.Trading.NetProfit,
		},
		ByPlatform: byPlatform,
	}), nil
}
