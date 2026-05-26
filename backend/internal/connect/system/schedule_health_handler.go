package connect

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
)

type healthGradingConfig struct {
	GreenSuccessRate  float64 `json:"green_success_rate"`
	GreenMaxFailedRuns int    `json:"green_max_failed_runs"`
	YellowSuccessRate float64 `json:"yellow_success_rate"`
	MinSampleSize     int     `json:"min_sample_size"`
}

func defaultHealthGradingConfig() healthGradingConfig {
	return healthGradingConfig{
		GreenSuccessRate:  90,
		GreenMaxFailedRuns: 1,
		YellowSuccessRate: 60,
		MinSampleSize:     1,
	}
}

// ScheduleHealthServer implements ant.v1.ScheduleHealthServiceHandler.
type ScheduleHealthServer struct {
	pg  *pgxpool.Pool
	log *zap.Logger
}

var _ antv1c.ScheduleHealthServiceHandler = (*ScheduleHealthServer)(nil)

func NewScheduleHealthServer(pg *pgxpool.Pool, log *zap.Logger) *ScheduleHealthServer {
	return &ScheduleHealthServer{pg: pg, log: log}
}

func (s *ScheduleHealthServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

func (s *ScheduleHealthServer) getGradingConfig(ctx context.Context) healthGradingConfig {
	cfg := defaultHealthGradingConfig()
	var raw string
	err := s.pg.QueryRow(ctx,
		"SELECT value FROM system_configs WHERE key = 'strategy.schedule.health_grading_config' AND enabled = TRUE",
	).Scan(&raw)
	if err != nil || raw == "" {
		return cfg
	}
	var parsed healthGradingConfig
	if json.Unmarshal([]byte(raw), &parsed) == nil {
		if parsed.GreenSuccessRate > 0 {
			cfg.GreenSuccessRate = parsed.GreenSuccessRate
		}
		if parsed.GreenMaxFailedRuns > 0 {
			cfg.GreenMaxFailedRuns = parsed.GreenMaxFailedRuns
		}
		if parsed.YellowSuccessRate > 0 {
			cfg.YellowSuccessRate = parsed.YellowSuccessRate
		}
		if parsed.MinSampleSize > 0 {
			cfg.MinSampleSize = parsed.MinSampleSize
		}
	}
	return cfg
}

func computeGrade(successRate float64, failedRuns int, cfg healthGradingConfig) (level, color, noteCode string) {
	if failedRuns <= cfg.GreenMaxFailedRuns && successRate >= cfg.GreenSuccessRate {
		return "green", "#52c41a", "all_clear"
	}
	if successRate >= cfg.YellowSuccessRate {
		return "yellow", "#faad14", "degraded"
	}
	return "red", "#ff4d4f", "critical"
}

func ts(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func (s *ScheduleHealthServer) GetScheduleHealth(ctx context.Context, req *connect.Request[antv1.GetScheduleHealthRequest]) (*connect.Response[antv1.GetScheduleHealthResponse], error) {
	uid := s.userID(ctx)
	scheduleID, err := uuid.Parse(req.Msg.ScheduleId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	runLimit := int(req.Msg.RunLimit)
	if runLimit <= 0 || runLimit > 100 {
		runLimit = 20
	}
	orderLimit := int(req.Msg.OrderLimit)
	if orderLimit <= 0 || orderLimit > 100 {
		orderLimit = 20
	}

	// Summary stats
	var totalRuns, successRuns, failedRuns int32
	var successRate float64
	var lastRunTime *time.Time
	var latestError string
	err = s.pg.QueryRow(ctx,
		"SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'success'), COUNT(*) FILTER (WHERE status = 'failed'), MAX(created_at) FROM strategy_execution_logs WHERE user_id = $1 AND schedule_id = $2",
		uid, scheduleID,
	).Scan(&totalRuns, &successRuns, &failedRuns, &lastRunTime)
	if err != nil {
		return nil, err
	}
	if totalRuns > 0 {
		successRate = math.Round(float64(successRuns)/float64(totalRuns)*10000) / 100
	}

	// Latest error
	var latestErr *string
	_ = s.pg.QueryRow(ctx,
		"SELECT error_message FROM strategy_execution_logs WHERE user_id = $1 AND schedule_id = $2 AND status = 'failed' ORDER BY created_at DESC LIMIT 1",
		uid, scheduleID,
	).Scan(&latestErr)
	if latestErr != nil && *latestErr != "" {
		latestError = *latestErr
	}

	// Latest order info
	var latestOrderTicket int64
	var latestOrderProfit float64
	var hasLatestOrderProfit bool
	err = s.pg.QueryRow(ctx,
		"SELECT COALESCE(ticket, 0), COALESCE(profit, 0) FROM order_history WHERE user_id = $1 AND schedule_id = $2 AND close_time IS NOT NULL ORDER BY close_time DESC LIMIT 1",
		uid, scheduleID,
	).Scan(&latestOrderTicket, &latestOrderProfit)
	if err == nil && latestOrderTicket != 0 {
		hasLatestOrderProfit = true
	}

	// Grading config
	cfg := s.getGradingConfig(ctx)
	gradeLevel, gradeColor, gradeNoteCode := computeGrade(successRate, int(failedRuns), cfg)

	summary := &antv1.ScheduleHealthSummary{
		TotalRuns:            totalRuns,
		SuccessRuns:          successRuns,
		FailedRuns:           failedRuns,
		SuccessRate:          successRate,
		LastRunAt:            ts(lastRunTime),
		LatestError:          latestError,
		LatestOrderTicket:    strconv.FormatInt(latestOrderTicket, 10),
		LatestOrderProfit:    latestOrderProfit,
		HasLatestOrderProfit: hasLatestOrderProfit,
		GradeLevel:           gradeLevel,
		GradeColor:           gradeColor,
		GradeNoteCode:        gradeNoteCode,
		GreenSuccessRate:     cfg.GreenSuccessRate,
		GreenMaxFailedRuns:   int32(cfg.GreenMaxFailedRuns),
		YellowSuccessRate:    cfg.YellowSuccessRate,
		MinSampleSize:        int32(cfg.MinSampleSize),
	}

	// Run logs
	runRows, err := s.pg.Query(ctx,
		"SELECT id, status, COALESCE(signal_type, ''), COALESCE(execution_time_ms, 0), COALESCE(error_message, ''), created_at FROM strategy_execution_logs WHERE user_id = $1 AND schedule_id = $2 ORDER BY created_at DESC LIMIT $3",
		uid, scheduleID, runLimit,
	)
	if err != nil {
		return nil, err
	}
	defer runRows.Close()
	runLogs := make([]*antv1.ScheduleHealthRunLog, 0, runLimit)
	for runRows.Next() {
		var id, status, signalType, errMsg string
		var durationMs int64
		var createdAt time.Time
		if serr := runRows.Scan(&id, &status, &signalType, &durationMs, &errMsg, &createdAt); serr != nil {
			return nil, serr
		}
		runLogs = append(runLogs, &antv1.ScheduleHealthRunLog{
			Id: id, Status: status, SignalType: signalType, DurationMs: durationMs,
			ErrorMessage: errMsg, CreatedAt: timestamppb.New(createdAt),
		})
	}
	if runRows.Err() != nil {
		return nil, runRows.Err()
	}

	// Orders
	orderRows, err := s.pg.Query(ctx,
		"SELECT id::text, ticket, order_type, symbol, profit, open_time, close_time FROM order_history WHERE user_id = $1 AND schedule_id = $2 ORDER BY COALESCE(close_time, open_time) DESC LIMIT $3",
		uid, scheduleID, orderLimit,
	)
	if err != nil {
		return nil, err
	}
	defer orderRows.Close()
	orders := make([]*antv1.ScheduleHealthOrder, 0, orderLimit)
	for orderRows.Next() {
		var id, orderType, symbol string
		var ticket int64
		var profit float64
		var openTime, closeTime *time.Time
		if serr := orderRows.Scan(&id, &ticket, &orderType, &symbol, &profit, &openTime, &closeTime); serr != nil {
			return nil, serr
		}
		orders = append(orders, &antv1.ScheduleHealthOrder{
			Id: id, Ticket: ticket, OrderType: orderType, Symbol: symbol,
			Profit: profit, OpenTime: ts(openTime), CloseTime: ts(closeTime),
		})
	}
	if orderRows.Err() != nil {
		return nil, orderRows.Err()
	}

	return connect.NewResponse(&antv1.GetScheduleHealthResponse{
		Summary: summary,
		RunLogs: runLogs,
		Orders:  orders,
	}), nil
}
