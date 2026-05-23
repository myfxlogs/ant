package connect

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	v1 "anttrader/gen/proto"
)

func (s *AdminService) GetDashboard(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.DashboardStats], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	stats, err := s.adminSvc.GetDashboardStats(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&v1.DashboardStats{
		TotalUsers:        int32(stats.TotalUsers),
		ActiveUsers:       int32(stats.ActiveUsers),
		TotalAccounts:     int32(stats.TotalAccounts),
		OnlineAccounts:    int32(stats.OnlineAccounts),
		ConnectedAccounts: int32(stats.OnlineAccounts),
		TotalStrategies:   0,
		ActiveStrategies:  0,
		TotalTrades:       int32(stats.TodayTrades),
		TodayTrades:       int32(stats.TodayTrades),
		TodayProfit:       stats.TodayProfit,
	}), nil
}

func (s *AdminService) HealthCheck(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.HealthCheckResponse], error) {
	return connect.NewResponse(&v1.HealthCheckResponse{
		Status:        "ok",
		Version:       "1.0.0",
		UptimeSeconds: 0,
		Services: map[string]*v1.ServiceHealth{
			"database": {Healthy: true, Message: "ok"},
			"redis":    {Healthy: true, Message: "ok"},
		},
	}), nil
}

func (s *AdminService) GetMetrics(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[v1.MetricsResponse], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	metrics, err := s.adminSvc.GetSystemMetrics(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	riskWindows := make([]*v1.RiskMetricsWindow, 0, len(metrics.RiskWindows))
	for _, item := range metrics.RiskWindows {
		topCodes := make([]*v1.RiskCodeCount, 0, len(item.TopRejectRiskCodes))
		for _, code := range item.TopRejectRiskCodes {
			topCodes = append(topCodes, &v1.RiskCodeCount{
				RiskCode: code.RiskCode,
				Count:    code.Count,
			})
		}
		riskWindows = append(riskWindows, &v1.RiskMetricsWindow{
			Window:             item.Window,
			Hours:              int32(item.Hours),
			RiskValidateTotal:  item.RiskValidateTotal,
			RiskValidatePass:   item.RiskValidatePass,
			RiskValidateReject: item.RiskValidateReject,
			RiskValidateError:  item.RiskValidateError,
			OrderSendSuccess:   item.OrderSendSuccess,
			OrderSendFailed:    item.OrderSendFailed,
			OrderCloseSuccess:  item.OrderCloseSuccess,
			OrderCloseFailed:   item.OrderCloseFailed,
			TopRejectRiskCodes: topCodes,
		})
	}

	return connect.NewResponse(&v1.MetricsResponse{
		System: &v1.SystemMetrics{
			CpuUsagePercent:    metrics.CPUUsage,
			MemoryUsagePercent: metrics.MemoryUsage,
			MemoryUsedBytes:    metrics.MemoryUsed,
			MemoryTotalBytes:   metrics.MemoryTotal,
			DiskUsagePercent:   metrics.DiskUsage,
			DiskUsedBytes:      metrics.DiskUsed,
			DiskTotalBytes:     metrics.DiskTotal,
			GoroutinesCount:    int32(metrics.GoroutinesCount),
		},
		App: &v1.AppMetrics{
			TotalUsers:         int32(metrics.TotalUsers),
			ActiveUsers:        int32(metrics.ActiveUsers),
			TotalAccounts:      int32(metrics.TotalAccounts),
			ConnectedAccounts:  int32(metrics.ConnectedAccounts),
			OnlineAccounts:     int32(metrics.OnlineAccounts),
			TotalStrategies:    int32(metrics.TotalStrategies),
			ActiveStrategies:   int32(metrics.ActiveStrategies),
			TotalProfit:        metrics.TotalProfit,
			TotalTradesToday:   int32(metrics.TodayTrades),
			RiskValidateTotal:  metrics.RiskValidateTotal,
			RiskValidatePass:   metrics.RiskValidatePass,
			RiskValidateReject: metrics.RiskValidateReject,
			RiskValidateError:  metrics.RiskValidateError,
			OrderSendSuccess:   metrics.OrderSendSuccess,
			OrderSendFailed:    metrics.OrderSendFailed,
			OrderCloseSuccess:  metrics.OrderCloseSuccess,
			OrderCloseFailed:   metrics.OrderCloseFailed,
			RiskWindows:        riskWindows,
		},
	}), nil
}

func (s *AdminService) ClearCache(ctx context.Context, req *connect.Request[emptypb.Empty]) (*connect.Response[emptypb.Empty], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.adminSvc.ClearCache(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *AdminService) InvalidateCache(ctx context.Context, req *connect.Request[v1.InvalidateCacheRequest]) (*connect.Response[emptypb.Empty], error) {
	_, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	for _, tag := range req.Msg.Tags {
		if err := s.adminSvc.InvalidateCacheByTag(ctx, tag); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}
