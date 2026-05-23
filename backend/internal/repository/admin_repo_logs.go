package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"anttrader/internal/model"
)

type RiskCodeCount struct {
	RiskCode string
	Count    int64
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

func (r *AdminRepository) CreateLog(ctx context.Context, log *model.AdminLog) error {
	query := `
		INSERT INTO admin_logs (
			user_id, module, action_type, target_type, target_id,
			ip_address, user_agent, request_method, request_path,
			details, success, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`

	var detailsJSON []byte
	var err error
	if log.Details != nil {
		detailsJSON, err = json.Marshal(log.Details)
		if err != nil {
			return err
		}
	}

	return r.db.QueryRow(ctx, query,
		log.AdminID, log.Module, log.ActionType, log.TargetType, log.TargetID,
		log.IPAddress, log.UserAgent, log.RequestMethod, log.RequestPath,
		detailsJSON, log.Success, log.ErrorMessage,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *AdminRepository) ListLogs(ctx context.Context, params *model.LogListParams) ([]*model.AdminLog, int64, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	offset := (params.Page - 1) * params.PageSize

	countQuery := `SELECT COUNT(*) FROM admin_logs WHERE 1=1`
	query := `SELECT * FROM admin_logs WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if params.Module != "" {
		countQuery += fmt.Sprintf(" AND module = $%d", argIndex)
		query += fmt.Sprintf(" AND module = $%d", argIndex)
		args = append(args, params.Module)
		argIndex++
	}

	if params.ActionType != "" {
		countQuery += fmt.Sprintf(" AND action_type = $%d", argIndex)
		query += fmt.Sprintf(" AND action_type = $%d", argIndex)
		args = append(args, params.ActionType)
		argIndex++
	}

	if params.StartDate != "" {
		countQuery += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, params.StartDate+" 00:00:00")
		argIndex++
	}

	if params.EndDate != "" {
		countQuery += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, params.EndDate+" 23:59:59")
		argIndex++
	}

	if params.AdminID != "" {
		countQuery += fmt.Sprintf(" AND admin_id = $%d", argIndex)
		query += fmt.Sprintf(" AND admin_id = $%d", argIndex)
		args = append(args, params.AdminID)
		argIndex++
	}

	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, params.PageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*model.AdminLog
	for rows.Next() {
		var l model.AdminLog
		var detailsJSON []byte
		err := rows.Scan(
			&l.ID, &l.AdminID, &l.Module, &l.ActionType, &l.TargetType,
			&l.TargetID, &l.IPAddress, &l.UserAgent, &l.RequestMethod,
			&l.RequestPath, &detailsJSON, &l.Success, &l.ErrorMessage, &l.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		if detailsJSON != nil {
			json.Unmarshal(detailsJSON, &l.Details)
		}
		logs = append(logs, &l)
	}

	return logs, total, nil
}

func (r *AdminRepository) GetRiskMetricsWindows(ctx context.Context, windows []int, topN int) ([]RiskMetricsWindow, error) {
	if topN <= 0 {
		topN = 10
	}
	out := make([]RiskMetricsWindow, 0, len(windows))
	for _, hours := range windows {
		if hours <= 0 {
			continue
		}
		item := RiskMetricsWindow{
			Window: fmt.Sprintf("%dh", hours),
			Hours:  hours,
		}
		if err := r.db.QueryRow(ctx, `
			SELECT
				COUNT(*) AS risk_validate_total,
				COUNT(*) FILTER (WHERE COALESCE(new_value->>'result', '') = 'pass') AS risk_validate_pass,
				COUNT(*) FILTER (WHERE COALESCE(new_value->>'result', '') = 'reject') AS risk_validate_reject,
				COUNT(*) FILTER (WHERE COALESCE(new_value->>'result', '') = 'error') AS risk_validate_error,
				COUNT(*) FILTER (
					WHERE COALESCE(new_value->>'action', '') = 'order_send'
					  AND COALESCE(new_value->>'result', '') = 'pass'
				) AS order_send_success,
				COUNT(*) FILTER (
					WHERE COALESCE(new_value->>'action', '') = 'order_send'
					  AND COALESCE(new_value->>'result', '') IN ('reject', 'error')
				) AS order_send_failed,
				COUNT(*) FILTER (
					WHERE COALESCE(new_value->>'action', '') = 'order_close'
					  AND COALESCE(new_value->>'result', '') = 'pass'
				) AS order_close_success,
				COUNT(*) FILTER (
					WHERE COALESCE(new_value->>'action', '') = 'order_close'
					  AND COALESCE(new_value->>'result', '') IN ('reject', 'error')
				) AS order_close_failed
			FROM system_operation_logs
			WHERE module = 'trading_risk'
			  AND action = 'pre_trade_validate'
			  AND created_at >= NOW() - ($1::int * INTERVAL '1 hour')
		`, hours).Scan(
			&item.RiskValidateTotal,
			&item.RiskValidatePass,
			&item.RiskValidateReject,
			&item.RiskValidateError,
			&item.OrderSendSuccess,
			&item.OrderSendFailed,
			&item.OrderCloseSuccess,
			&item.OrderCloseFailed,
		); err != nil {
			return nil, err
		}

		rows, err := r.db.Query(ctx, `
			SELECT
				COALESCE(NULLIF(new_value->>'risk_code', ''), '(none)') AS risk_code,
				COUNT(*) AS cnt
			FROM system_operation_logs
			WHERE module = 'trading_risk'
			  AND action = 'pre_trade_validate'
			  AND created_at >= NOW() - ($1::int * INTERVAL '1 hour')
			  AND COALESCE(new_value->>'result', '') = 'reject'
			GROUP BY risk_code
			ORDER BY cnt DESC, risk_code ASC
			LIMIT $2
		`, hours, topN)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var code RiskCodeCount
			if err := rows.Scan(&code.RiskCode, &code.Count); err != nil {
				rows.Close()
				return nil, err
			}
			item.TopRejectRiskCodes = append(item.TopRejectRiskCodes, code)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
		out = append(out, item)
	}
	return out, nil
}
