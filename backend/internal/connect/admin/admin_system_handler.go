package admin

import (
	"context"
	"errors"

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
		{Name: "today_volume", Value: stats.TodayVolume},
		{Name: "today_profit", Value: stats.TodayProfit},
	}
	return connect.NewResponse(&antv1.GetMetricsResponse{Metrics: metrics}), nil
}

func (s *AdminSystemServer) ResolveAlert(context.Context, *connect.Request[antv1.ResolveAlertRequest]) (*connect.Response[antv1.ResolveAlertResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("ResolveAlert not implemented"))
}

func (s *AdminSystemServer) ClearCache(context.Context, *connect.Request[antv1.ClearCacheRequest]) (*connect.Response[antv1.ClearCacheResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("ClearCache not implemented"))
}

func (s *AdminSystemServer) InvalidateCache(context.Context, *connect.Request[antv1.InvalidateCacheRequest]) (*connect.Response[antv1.InvalidateCacheResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("InvalidateCache not implemented"))
}
