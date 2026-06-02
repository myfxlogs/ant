package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"anttrader/internal/model"
)

func (r *LogRepository) CreateOperationLog(ctx context.Context, log *model.SystemOperationLog) error {
	query := `
		INSERT INTO system_operation_logs (
			id, user_id, operation_type, module, resource_type, resource_id, action,
			old_value, new_value, ip_address, user_agent, status, error_message, duration_ms, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			error_message = EXCLUDED.error_message,
			duration_ms = EXCLUDED.duration_ms,
			new_value = EXCLUDED.new_value,
			created_at = EXCLUDED.created_at`

	var oldValue, newValue []byte
	if log.OldValue != nil {
		oldValue, _ = json.Marshal(log.OldValue)
	}
	if log.NewValue != nil {
		newValue, _ = json.Marshal(log.NewValue)
	}

	_, err := r.db.Exec(ctx, query,
		log.ID, log.UserID, log.OperationType, log.Module, log.ResourceType, log.ResourceID, log.Action,
		oldValue, newValue, log.IPAddress, log.UserAgent, log.Status, log.ErrorMessage, log.DurationMs, log.CreatedAt)
	return fmt.Errorf("create operation log: %w", err)
}

func (r *LogRepository) GetOperationLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.SystemOperationLog, int, error) {
	baseQuery := `FROM system_operation_logs WHERE user_id = $1`
	args := []interface{}{userID}
	argIndex := 2

	if params != nil {
		if params.Module != "" {
			baseQuery += fmt.Sprintf(` AND module = $%d`, argIndex)
			args = append(args, params.Module)
			argIndex++
		}
		if params.Action != "" {
			baseQuery += fmt.Sprintf(` AND action = $%d`, argIndex)
			args = append(args, params.Action)
			argIndex++
		}
		if params.Type != "" {
			baseQuery += fmt.Sprintf(` AND operation_type = $%d`, argIndex)
			args = append(args, params.Type)
			argIndex++
		}
		if params.ResourceType != "" {
			baseQuery += fmt.Sprintf(` AND resource_type = $%d`, argIndex)
			args = append(args, params.ResourceType)
			argIndex++
		}
		if params.ResourceID != "" {
			baseQuery += fmt.Sprintf(` AND resource_id = $%d`, argIndex)
			rid, _ := uuid.Parse(params.ResourceID)
			args = append(args, rid)
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
	var logs []*model.SystemOperationLog
	for queryRows.Next() {
		var log model.SystemOperationLog
		if err := queryRows.Scan(&log.ID, &log.UserID, &log.OperationType, &log.Module, &log.ResourceType, &log.ResourceID, &log.Action, &log.OldValue, &log.NewValue, &log.IPAddress, &log.UserAgent, &log.Status, &log.ErrorMessage, &log.DurationMs, &log.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &log)
	}
	if err := queryRows.Err(); err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
