package service

import (
	"context"
	"strings"
)

type SystemMetrics struct {
	CPUUsage           float64
	MemoryUsage        float64
	MemoryUsed         int64
	MemoryTotal        int64
	DiskUsage          float64
	DiskUsed           int64
	DiskTotal          int64
	GoroutinesCount    int
	TotalUsers         int
	ActiveUsers        int
	TotalAccounts      int
	ConnectedAccounts  int
	OnlineAccounts     int
	TotalStrategies    int
	ActiveStrategies   int
	TotalProfit        float64
	TodayTrades        int
	RiskValidateTotal  int64
	RiskValidatePass   int64
	RiskValidateReject int64
	RiskValidateError  int64
	OrderSendSuccess   int64
	OrderSendFailed    int64
	OrderCloseSuccess  int64
	OrderCloseFailed   int64
	RiskWindows        []RiskMetricsWindow
}

type RiskMetricsWindow struct {
	Window             string
	Hours              int
	RiskValidateTotal  int64
	RiskValidatePass   int64
	RiskValidateReject int64
	RiskValidateError  int64
	OrderSendSuccess   int64
	OrderSendFailed    int64
	OrderCloseSuccess  int64
	OrderCloseFailed   int64
	TopRejectRiskCodes []RiskCodeCount
}

type RiskCodeCount struct {
	RiskCode string
	Count    int64
}

func (s *AdminService) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	stats, err := s.adminRepo.GetDashboardStats(ctx)
	if err != nil {
		return nil, err
	}

	riskCounters := readRiskMetricCounters()
	riskWindowsRaw, err := s.adminRepo.GetRiskMetricsWindows(ctx, []int{1, 24, 72}, 10)
	if err != nil {
		return nil, err
	}
	riskWindows := make([]RiskMetricsWindow, 0, len(riskWindowsRaw))
	for _, item := range riskWindowsRaw {
		codes := make([]RiskCodeCount, 0, len(item.TopRejectRiskCodes))
		for _, code := range item.TopRejectRiskCodes {
			codes = append(codes, RiskCodeCount{
				RiskCode: code.RiskCode,
				Count:    code.Count,
			})
		}
		riskWindows = append(riskWindows, RiskMetricsWindow{
			Window:             item.Window,
			Hours:              item.Hours,
			RiskValidateTotal:  item.RiskValidateTotal,
			RiskValidatePass:   item.RiskValidatePass,
			RiskValidateReject: item.RiskValidateReject,
			RiskValidateError:  item.RiskValidateError,
			OrderSendSuccess:   item.OrderSendSuccess,
			OrderSendFailed:    item.OrderSendFailed,
			OrderCloseSuccess:  item.OrderCloseSuccess,
			OrderCloseFailed:   item.OrderCloseFailed,
			TopRejectRiskCodes: codes,
		})
	}

	return &SystemMetrics{
		TotalUsers:         int(stats.TotalUsers),
		ActiveUsers:        int(stats.ActiveUsers),
		TotalAccounts:      int(stats.TotalAccounts),
		ConnectedAccounts:  int(stats.OnlineAccounts),
		OnlineAccounts:     int(stats.OnlineAccounts),
		TodayTrades:        int(stats.TodayTrades),
		TotalProfit:        stats.TodayProfit,
		RiskValidateTotal:  riskCounters["risk_validate_total"],
		RiskValidatePass:   riskCounters["risk_validate_pass"],
		RiskValidateReject: riskCounters["risk_validate_reject"],
		RiskValidateError:  riskCounters["risk_validate_error"],
		OrderSendSuccess:   riskCounters["order_send_success"],
		OrderSendFailed:    riskCounters["order_send_failed"],
		OrderCloseSuccess:  riskCounters["order_close_success"],
		OrderCloseFailed:   riskCounters["order_close_failed"],
		RiskWindows:        riskWindows,
	}, nil
}

func readRiskMetricCounters() map[string]int64 {
	out := map[string]int64{
		"risk_validate_total":  0,
		"risk_validate_pass":   0,
		"risk_validate_reject": 0,
		"risk_validate_error":  0,
		"order_send_success":   0,
		"order_send_failed":    0,
		"order_close_success":  0,
		"order_close_failed":   0,
	}
	snap := SnapshotTradeRiskMetrics()
	rawCounters, ok := snap["counters"].(map[string]int64)
	if !ok {
		return out
	}
	for key, val := range rawCounters {
		if strings.HasPrefix(key, "risk_validate_total|") {
			out["risk_validate_total"] += val
			result := extractMetricLabel(key, "result")
			switch result {
			case "pass":
				out["risk_validate_pass"] += val
			case "reject":
				out["risk_validate_reject"] += val
			case "error":
				out["risk_validate_error"] += val
			}
			continue
		}
		if strings.HasPrefix(key, "order_send_total|") {
			result := extractMetricLabel(key, "result")
			if result == "success" {
				out["order_send_success"] += val
			} else if result == "failed" {
				out["order_send_failed"] += val
			}
			continue
		}
		if strings.HasPrefix(key, "order_close_total|") {
			result := extractMetricLabel(key, "result")
			if result == "success" {
				out["order_close_success"] += val
			} else if result == "failed" {
				out["order_close_failed"] += val
			}
		}
	}
	return out
}

func extractMetricLabel(metricKey string, label string) string {
	parts := strings.Split(metricKey, "|")
	for _, part := range parts {
		if strings.HasPrefix(part, label+"=") {
			return strings.TrimPrefix(part, label+"=")
		}
	}
	return ""
}
