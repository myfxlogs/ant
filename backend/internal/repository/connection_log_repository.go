package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"alphaforge/internal/model"
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
	baseQ, args, argIdx := buildConnLogFilters(userID, params)
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) `+baseQ, args...).Scan(&total); err != nil { return nil, 0, err }
	page, pageSize := normalizeOpLogPagination(params)
	dataQ := fmt.Sprintf(`SELECT * %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, baseQ, argIdx, argIdx+1)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.Query(ctx, dataQ, args...)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var logs []*model.AccountConnectionLog
	for rows.Next() {
		var l model.AccountConnectionLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.AccountID, &l.EventType, &l.Status, &l.Message, &l.ErrorDetail, &l.ServerHost, &l.ServerPort, &l.LoginID, &l.ConnectionDurationSecs, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	return logs, total, rows.Err()
}

func buildConnLogFilters(userID uuid.UUID, params *model.LogQueryParams) (baseQ string, args []interface{}, idx int) {
	baseQ = `FROM account_connection_logs WHERE user_id = $1`
	args = []interface{}{userID}
	idx = 2
	if params == nil { return }
	addFilter := func(cond, val string) { baseQ += fmt.Sprintf(` AND %s = $%d`, cond, idx); args = append(args, val); idx++ }
	if params.AccountID != "" { aid, _ := uuid.Parse(params.AccountID); addFilter("account_id", ""); args[len(args)-1] = aid }
	if params.Status != "" { addFilter("status", params.Status) }
	if params.StartDate != "" { baseQ += fmt.Sprintf(` AND created_at >= $%d`, idx); args = append(args, params.StartDate); idx++ }
	if params.EndDate != "" { baseQ += fmt.Sprintf(` AND created_at <= $%d`, idx); args = append(args, params.EndDate); idx++ }
	return
}
