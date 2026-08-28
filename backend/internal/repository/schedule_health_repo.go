package repository

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/encoding/prototext"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// ScheduleHealthRepository provides read access to schedule execution logs and order history.
type ScheduleHealthRepository struct {
	db *pgxpool.Pool
}

// NewScheduleHealthRepository creates a schedule health repository.
func NewScheduleHealthRepository(db *pgxpool.Pool) *ScheduleHealthRepository {
	return &ScheduleHealthRepository{db: db}
}

// HealthGradingConfig defines thresholds for health grading.
type HealthGradingConfig = antv1.HealthGradingConfig

// DefaultHealthGradingConfig returns sensible defaults.
func DefaultHealthGradingConfig() *HealthGradingConfig {
	return &HealthGradingConfig{
		GreenSuccessRate:   90,
		GreenMaxFailedRuns: 1,
		YellowSuccessRate:  60,
		MinSampleSize:      1,
	}
}

// ScheduleHealthSummary holds aggregated schedule health statistics.
type ScheduleHealthSummary struct {
	TotalRuns            int32
	SuccessRuns          int32
	FailedRuns           int32
	SuccessRate          float64
	LastRunAt            *time.Time
	LatestError          string
	LatestOrderTicket    int64
	LatestOrderProfit    decimal.Decimal
	HasLatestOrderProfit bool
	GradeLevel           string
	GradeColor           string
	GradeNoteCode        string
	GreenSuccessRate     float64
	GreenMaxFailedRuns   int32
	YellowSuccessRate    float64
	MinSampleSize        int32
}

// ScheduleHealthRunLog is a single execution log entry.
type ScheduleHealthRunLog struct {
	ID           string
	Status       string
	SignalType   string
	DurationMs   int64
	ErrorMessage string
	CreatedAt    time.Time
}

// ScheduleHealthOrder is an order from history.
type ScheduleHealthOrder struct {
	ID        string
	Ticket    int64
	OrderType string
	Symbol    string
	Profit    decimal.Decimal
	OpenTime  *time.Time
	CloseTime *time.Time
}

// GetGradingConfig reads the health grading configuration from system_config.
func (r *ScheduleHealthRepository) GetGradingConfig(ctx context.Context) *HealthGradingConfig {
	cfg := DefaultHealthGradingConfig()
	var raw string
	err := r.db.QueryRow(ctx,
		"SELECT value FROM system_config WHERE key = 'strategy.schedule.health_grading_config' AND enabled = TRUE",
	).Scan(&raw)
	if err != nil || raw == "" {
		return cfg
	}
	var parsed antv1.HealthGradingConfig
	if err := prototext.Unmarshal([]byte(raw), &parsed); err != nil {
		// Legacy JSON fallback for old DB records.
		return cfg
	}
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
	return cfg
}

// GetScheduleStats returns summary statistics for a schedule's execution history.
func (r *ScheduleHealthRepository) GetScheduleStats(ctx context.Context, userID, scheduleID uuid.UUID) (totalRuns, successRuns, failedRuns int32, successRate float64, lastRunAt *time.Time, latestError string, err error) {
	err = r.db.QueryRow(ctx,
		"SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'success'), COUNT(*) FILTER (WHERE status = 'failed'), MAX(created_at) FROM schedule_run_logs WHERE user_id = $1 AND schedule_id = $2",
		userID, scheduleID,
	).Scan(&totalRuns, &successRuns, &failedRuns, &lastRunAt)
	if err != nil {
		return
	}
	if totalRuns > 0 {
		successRate = math.Round(float64(successRuns)/float64(totalRuns)*10000) / 100
	}
	var latestErr *string
	_ = r.db.QueryRow(ctx,
		"SELECT error_message FROM schedule_run_logs WHERE user_id = $1 AND schedule_id = $2 AND status = 'failed' ORDER BY created_at DESC LIMIT 1",
		userID, scheduleID,
	).Scan(&latestErr)
	if latestErr != nil && *latestErr != "" {
		latestError = *latestErr
	}
	return
}

// GetLatestOrderProfit returns the most recent closed order's ticket and profit.
func (r *ScheduleHealthRepository) GetLatestOrderProfit(ctx context.Context, userID, scheduleID uuid.UUID) (ticket int64, profit decimal.Decimal, hasData bool) {
	err := r.db.QueryRow(ctx,
		"SELECT COALESCE(ticket, 0), COALESCE(profit, 0) FROM trade_records WHERE user_id = $1 AND schedule_id = $2 AND close_time IS NOT NULL ORDER BY close_time DESC LIMIT 1",
		userID, scheduleID,
	).Scan(&ticket, &profit)
	return ticket, profit, err == nil && ticket != 0
}

// ListRunLogs returns recent execution logs for a schedule.
func (r *ScheduleHealthRepository) ListRunLogs(ctx context.Context, userID, scheduleID uuid.UUID, limit int) ([]ScheduleHealthRunLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.Query(ctx,
		"SELECT id, status, COALESCE(signal_type, ''), COALESCE(duration_ms, 0), COALESCE(error_message, ''), created_at FROM schedule_run_logs WHERE user_id = $1 AND schedule_id = $2 ORDER BY created_at DESC LIMIT $3",
		userID, scheduleID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []ScheduleHealthRunLog
	for rows.Next() {
		var l ScheduleHealthRunLog
		if err := rows.Scan(&l.ID, &l.Status, &l.SignalType, &l.DurationMs, &l.ErrorMessage, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// ListOrders returns recent orders for a schedule.
func (r *ScheduleHealthRepository) ListOrders(ctx context.Context, userID, scheduleID uuid.UUID, limit int) ([]ScheduleHealthOrder, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.db.Query(ctx,
		"SELECT id::text, ticket, order_type, symbol, profit, open_time, close_time FROM trade_records WHERE user_id = $1 AND schedule_id = $2 ORDER BY COALESCE(close_time, open_time) DESC LIMIT $3",
		userID, scheduleID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []ScheduleHealthOrder
	for rows.Next() {
		var o ScheduleHealthOrder
		if err := rows.Scan(&o.ID, &o.Ticket, &o.OrderType, &o.Symbol, &o.Profit, &o.OpenTime, &o.CloseTime); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}
