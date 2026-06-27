package admin

import (
	"context"

	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/repository"
)

type AdminSystemServer struct {
	repo *repository.AdminRepository
	log  *zap.Logger
}

var _ antv1c.AdminSystemServiceHandler = (*AdminSystemServer)(nil)

func NewAdminSystemServer(repo *repository.AdminRepository, log *zap.Logger) *AdminSystemServer {
	return &AdminSystemServer{repo: repo, log: log}
}

func (s *AdminSystemServer) HealthCheck(ctx context.Context, _ *connect.Request[antv1.HealthCheckRequest]) (*connect.Response[antv1.HealthCheckResponse], error) {
	dbStatus := "healthy"
	if err := s.repo.Ping(ctx); err != nil {
		dbStatus = "unhealthy"
	}
	return connect.NewResponse(&antv1.HealthCheckResponse{
		Status:   dbStatus,
		DbStatus: dbStatus,
	}), nil
}

func (s *AdminSystemServer) GetMetrics(ctx context.Context, _ *connect.Request[antv1.GetMetricsRequest]) (*connect.Response[antv1.GetMetricsResponse], error) {
	stats, err := s.repo.GetDashboardStats(ctx)
	if err != nil {
		return nil, err
	}
	metrics := []*antv1.MetricPoint{
		{Name: "total_users", Value: float64(stats.TotalUsers)},
		{Name: "active_users", Value: float64(stats.ActiveUsers)},
		{Name: "total_accounts", Value: float64(stats.TotalAccounts)},
		{Name: "online_accounts", Value: float64(stats.OnlineAccounts)},
		{Name: "today_trades", Value: float64(stats.TodayTrades)},
		{Name: "today_volume", Value: stats.TodayVolume.InexactFloat64()},
		{Name: "today_profit", Value: stats.TodayProfit.InexactFloat64()},
	}
	return connect.NewResponse(&antv1.GetMetricsResponse{Metrics: metrics}), nil
}
