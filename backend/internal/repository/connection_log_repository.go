package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

func (r *LogRepository) CreateConnectionLog(ctx context.Context, log *model.AccountConnectionLog) error {
	query := `
		INSERT INTO account_connection_logs (
			id, user_id, account_id, event_type, status, message, error_detail,
			server_host, server_port, login_id, connection_duration_seconds, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err := r.db.Exec(ctx, query,
		log.ID, log.UserID, log.AccountID, log.EventType, log.Status, log.Message, log.ErrorDetail,
		log.ServerHost, log.ServerPort, log.LoginID, log.ConnectionDurationSecs, log.CreatedAt)
	return fmt.Errorf("create connection log: %w", err)
}

func (r *LogRepository) GetConnectionLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.AccountConnectionLog, int, error) {
	baseQuery := `FROM account_connection_logs WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	if params != nil {
		if params.AccountID != "" {
			baseQuery += fmt.Sprintf(` AND account_id = $%d`, argIndex)
			accountID, _ := uuid.Parse(params.AccountID)
			args = append(args, accountID)
			argIndex++
		}
		if params.Status != "" {
			baseQuery += fmt.Sprintf(` AND status = $%d`, argIndex)
			args = append(args, params.Status)
			argIndex++
		}
		if params.StartDate != "" {
			baseQuery += fmt.Sprintf(` AND created_at >= $%d`, argIndex)
			args = append(args, params.StartDate)
			argIndex++
		}
		if params.EndDate != "" {
			baseQuery += fmt.Sprintf(` AND created_at <= $%d`, argIndex)
			args = append(args, params.EndDate)
			argIndex++
		}
	}

	countQuery := `SELECT COUNT(*) ` + baseQuery
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	page := 1
	pageSize := 20
	if params != nil {
		if params.Page > 0 {
			page = params.Page
		}
		if params.PageSize > 0 {
			pageSize = params.PageSize
		}
	}

	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(`SELECT * %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, baseQuery, argIndex, argIndex+1)
	args = append(args, pageSize, offset)

	queryRows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer queryRows.Close()
	var logs []*model.AccountConnectionLog
	for queryRows.Next() {
		var log model.AccountConnectionLog
		if err := queryRows.Scan(&log.ID, &log.UserID, &log.AccountID, &log.EventType, &log.Status, &log.Message, &log.ErrorDetail, &log.ServerHost, &log.ServerPort, &log.LoginID, &log.ConnectionDurationSecs, &log.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &log)
	}
	if err := queryRows.Err(); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
