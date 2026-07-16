package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"alphaforge/internal/model"
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

	_, err := r.db.Exec(ctx, query,
		log.ID, log.UserID, log.OperationType, log.Module, log.ResourceType, log.ResourceID, log.Action,
		log.OldValue, log.NewValue, log.IPAddress, log.UserAgent, log.Status, log.ErrorMessage, log.DurationMs, log.CreatedAt)
	return fmt.Errorf("create operation log: %w", err)
}

func (r *LogRepository) GetOperationLogs(ctx context.Context, userID uuid.UUID, params *model.LogQueryParams) ([]*model.SystemOperationLog, int, error) {
	baseQuery, args, argIdx := buildOpLogFilters(userID, params)
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) `+baseQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	page, pageSize := normalizeOpLogPagination(params)
	dataQuery := fmt.Sprintf(`SELECT * %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, baseQuery, argIdx, argIdx+1)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var logs []*model.SystemOperationLog
	for rows.Next() {
		var l model.SystemOperationLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.OperationType, &l.Module, &l.ResourceType, &l.ResourceID, &l.Action, &l.OldValue, &l.NewValue, &l.IPAddress, &l.UserAgent, &l.Status, &l.ErrorMessage, &l.DurationMs, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	return logs, total, rows.Err()
}

func buildOpLogFilters(userID uuid.UUID, params *model.LogQueryParams) (string, []interface{}, int) {
	base := `FROM system_operation_logs WHERE user_id = $1`
	args := []interface{}{userID}
	idx := 2
	if params == nil { return base, args, idx }
	addFilter := func(cond, val string) { base += fmt.Sprintf(` AND %s = $%d`, cond, idx); args = append(args, val); idx++ }
	if params.Module != "" { addFilter("module", params.Module) }
	if params.Action != "" { addFilter("action", params.Action) }
	if params.Type != "" { addFilter("operation_type", params.Type) }
	if params.ResourceType != "" { addFilter("resource_type", params.ResourceType) }
	if params.ResourceID != "" { rid, _ := uuid.Parse(params.ResourceID); addFilter("resource_id", ""); args[len(args)-1] = rid }
	if params.Status != "" { addFilter("status", params.Status) }
	if params.StartDate != "" { addFilter("created_at >=", params.StartDate) }
	if params.EndDate != "" { addFilter("created_at <=", params.EndDate) }
	return base, args, idx
}

func normalizeOpLogPagination(params *model.LogQueryParams) (page, pageSize int) {
	page, pageSize = 1, 20
	if params != nil {
		if params.Page > 0 { page = params.Page }
		if params.PageSize > 0 { pageSize = params.PageSize }
	}
	return
}
