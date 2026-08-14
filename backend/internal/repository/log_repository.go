package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type LogRepository struct {
	db *pgxpool.Pool
}

func NewLogRepository(db *pgxpool.Pool) *LogRepository {
	return &LogRepository{db: db}
}

type ScheduleRunLogRow struct {
	ID           uuid.UUID       `db:"id"`
	Kind         string          `db:"kind"`
	Action       string          `db:"action"`
	Status       string          `db:"status"`
	DurationMs   int64           `db:"duration_ms"`
	ErrorMessage string          `db:"error_message"`
	SignalType   string          `db:"signal_type"`
	SignalVolume decimal.Decimal `db:"signal_volume"`
	CreatedAt    time.Time       `db:"created_at"`
}

func (r *LogRepository) InsertScheduleRunLog(ctx context.Context, userID, scheduleID uuid.UUID, kind, action, status, errorMessage, signalType string, signalVolume decimal.Decimal) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO schedule_run_logs (user_id, schedule_id, kind, action, status, duration_ms, error_message, signal_type, signal_volume)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		userID, scheduleID, kind, action, status, 0, errorMessage, signalType, signalVolume)
	return err
}

func (r *LogRepository) GetScheduleRunLogs(ctx context.Context, userID uuid.UUID, scheduleID uuid.UUID, page, pageSize int) ([]*ScheduleRunLogRow, int, error) {
	baseQuery := `FROM schedule_run_logs WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	if scheduleID != uuid.Nil {
		baseQuery += fmt.Sprintf(` AND schedule_id = $%d`, argIndex)
		args = append(args, scheduleID)
		argIndex++
	}

	var total int
	countQuery := `SELECT COUNT(*) ` + baseQuery
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(
		`SELECT id, kind, action, status, duration_ms, error_message, signal_type, signal_volume, created_at %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		baseQuery, argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*ScheduleRunLogRow
	for rows.Next() {
		var l ScheduleRunLogRow
		if err := rows.Scan(&l.ID, &l.Kind, &l.Action, &l.Status, &l.DurationMs, &l.ErrorMessage, &l.SignalType, &l.SignalVolume, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
