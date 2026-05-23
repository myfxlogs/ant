package connect

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/model"
	"anttrader/internal/service"
)

type ScheduleHealthService struct {
	logSvc *service.LogService
	dynCfg *service.DynamicConfigService
}

type scheduleHealthConfig struct {
	GreenSuccessRate   float64 `json:"green_success_rate"`
	YellowSuccessRate  float64 `json:"yellow_success_rate"`
	GreenMaxFailedRuns int32   `json:"green_max_failed_runs"`
	MinSampleSize      int32   `json:"min_sample_size"`
}

func NewScheduleHealthService(logSvc *service.LogService, dynCfg *service.DynamicConfigService) *ScheduleHealthService {
	return &ScheduleHealthService{logSvc: logSvc, dynCfg: dynCfg}
}

func (s *ScheduleHealthService) GetScheduleHealth(ctx context.Context, req *connect.Request[v1.GetScheduleHealthRequest]) (*connect.Response[v1.GetScheduleHealthResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	scheduleID, err := uuid.Parse(req.Msg.ScheduleId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	userUUID := uuid.MustParse(userID)
	runLimit := boundedLimit(int(req.Msg.RunLimit), 30)
	orderLimit := boundedLimit(int(req.Msg.OrderLimit), 20)
	runRows, _, err := s.logSvc.GetScheduleRunLogs(ctx, userUUID, scheduleID, 1, runLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	orders, _, err := s.logSvc.GetOrderHistory(ctx, userUUID, &model.LogQueryParams{Page: 1, PageSize: orderLimit, ScheduleID: scheduleID.String()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	cfg := s.loadConfig(ctx)
	resp := &v1.GetScheduleHealthResponse{
		Summary: &v1.ScheduleHealthSummary{
			LatestOrderTicket:  "-",
			GreenSuccessRate:   cfg.GreenSuccessRate,
			YellowSuccessRate:  cfg.YellowSuccessRate,
			GreenMaxFailedRuns: cfg.GreenMaxFailedRuns,
			MinSampleSize:      cfg.MinSampleSize,
		},
		RunLogs: make([]*v1.ScheduleHealthRunLog, 0, len(runRows)),
		Orders:  make([]*v1.ScheduleHealthOrder, 0, len(orders)),
	}
	for _, row := range runRows {
		if strings.ToLower(row.Kind) != "execution" {
			continue
		}
		status := strings.ToLower(row.Status)
		resp.Summary.TotalRuns++
		if status == "completed" || status == "success" {
			resp.Summary.SuccessRuns++
		} else if status == "failed" {
			resp.Summary.FailedRuns++
		}
		if resp.Summary.LastRunAt == nil {
			resp.Summary.LastRunAt = timestamppb.New(row.CreatedAt)
		}
		if resp.Summary.LatestError == "" && strings.TrimSpace(row.ErrorMessage) != "" {
			resp.Summary.LatestError = strings.TrimSpace(row.ErrorMessage)
		}
		resp.RunLogs = append(resp.RunLogs, &v1.ScheduleHealthRunLog{Id: row.ID.String(), Status: row.Status, SignalType: row.SignalType, DurationMs: row.DurationMs, ErrorMessage: row.ErrorMessage, CreatedAt: timestamppb.New(row.CreatedAt)})
	}
	if resp.Summary.TotalRuns > 0 {
		resp.Summary.SuccessRate = float64(resp.Summary.SuccessRuns) / float64(resp.Summary.TotalRuns) * 100
	}
	for _, order := range orders {
		item := &v1.ScheduleHealthOrder{Id: order.ID.String(), Ticket: order.Ticket, OrderType: string(order.OrderType), Symbol: order.Symbol, Profit: order.Profit, OpenTime: timestamppb.New(order.OpenTime)}
		if order.CloseTime != nil {
			item.CloseTime = timestamppb.New(*order.CloseTime)
		}
		resp.Orders = append(resp.Orders, item)
	}
	if len(orders) > 0 {
		resp.Summary.LatestOrderTicket = strconv.FormatInt(orders[0].Ticket, 10)
		resp.Summary.LatestOrderProfit = orders[0].Profit
		resp.Summary.HasLatestOrderProfit = true
	}
	applyHealthGrade(resp.Summary)
	return connect.NewResponse(resp), nil
}

func (s *ScheduleHealthService) loadConfig(ctx context.Context) scheduleHealthConfig {
	cfg := scheduleHealthConfig{GreenSuccessRate: 90, YellowSuccessRate: 60, GreenMaxFailedRuns: 1, MinSampleSize: 1}
	if s.dynCfg == nil {
		return cfg
	}
	raw, enabled, _ := s.dynCfg.GetString(ctx, "strategy.schedule.health_grading_config", "")
	if !enabled || strings.TrimSpace(raw) == "" {
		return cfg
	}
	var parsed scheduleHealthConfig
	if json.Unmarshal([]byte(raw), &parsed) != nil {
		return cfg
	}
	if parsed.GreenSuccessRate >= 0 && parsed.GreenSuccessRate <= 100 {
		cfg.GreenSuccessRate = parsed.GreenSuccessRate
	}
	if parsed.YellowSuccessRate >= 0 && parsed.YellowSuccessRate <= 100 {
		cfg.YellowSuccessRate = parsed.YellowSuccessRate
	}
	if cfg.YellowSuccessRate > cfg.GreenSuccessRate {
		cfg.YellowSuccessRate = cfg.GreenSuccessRate
	}
	if parsed.GreenMaxFailedRuns >= 0 {
		cfg.GreenMaxFailedRuns = parsed.GreenMaxFailedRuns
	}
	if parsed.MinSampleSize >= 0 {
		cfg.MinSampleSize = parsed.MinSampleSize
	}
	return cfg
}

func applyHealthGrade(summary *v1.ScheduleHealthSummary) {
	if summary.TotalRuns < summary.MinSampleSize {
		summary.GradeLevel = "unknown"
		summary.GradeColor = "default"
		summary.GradeNoteCode = "no_sample"
		return
	}
	if summary.SuccessRate >= summary.GreenSuccessRate && summary.FailedRuns <= summary.GreenMaxFailedRuns {
		summary.GradeLevel = "green"
		summary.GradeColor = "success"
		summary.GradeNoteCode = "healthy"
		return
	}
	if summary.SuccessRate >= summary.YellowSuccessRate {
		summary.GradeLevel = "yellow"
		summary.GradeColor = "warning"
		summary.GradeNoteCode = "watch"
		return
	}
	summary.GradeLevel = "red"
	summary.GradeColor = "error"
	summary.GradeNoteCode = "alert"
}

func boundedLimit(v int, fallback int) int {
	if v <= 0 {
		v = fallback
	}
	if v > 100 {
		return 100
	}
	return v
}
