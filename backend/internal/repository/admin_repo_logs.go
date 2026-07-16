package repository

import (
	"context"
	"fmt"

	"alphaforge/internal/model"
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

	return r.db.QueryRow(ctx, query,
		log.AdminID, log.Module, log.ActionType, log.TargetType, log.TargetID,
		log.IPAddress, log.UserAgent, log.RequestMethod, log.RequestPath,
		log.Details, log.Success, log.ErrorMessage,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *AdminRepository) ListLogs(ctx context.Context, p *model.LogListParams) ([]*model.AdminLog, int64, error) {
	normalizeLogParams(p)
	countQuery, query, args := buildLogQueries(p)
	var total int64
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil { return nil, 0, err }
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, p.PageSize, (p.Page-1)*p.PageSize)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil { return nil, 0, err }
	defer rows.Close()
	var logs []*model.AdminLog
	for rows.Next() {
		var l model.AdminLog
		if err := rows.Scan(&l.ID, &l.AdminID, &l.Module, &l.ActionType, &l.TargetType,
			&l.TargetID, &l.IPAddress, &l.UserAgent, &l.RequestMethod,
			&l.RequestPath, &l.Details, &l.Success, &l.ErrorMessage, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, &l)
	}
	return logs, total, nil
}

func normalizeLogParams(p *model.LogListParams) {
	if p.Page <= 0 { p.Page = 1 }
	if p.PageSize <= 0 { p.PageSize = 20 }
	if p.PageSize > 100 { p.PageSize = 100 }
}

func buildLogQueries(p *model.LogListParams) (countQ, query string, args []interface{}) {
	countQ, query = `SELECT COUNT(*) FROM admin_logs WHERE 1=1`, `SELECT * FROM admin_logs WHERE 1=1`
	var idx int
	applyFilter := func(cond, val string) {
		countQ += fmt.Sprintf(" AND %s = $%d", cond, idx+1)
		query += fmt.Sprintf(" AND %s = $%d", cond, idx+1)
		args = append(args, val); idx++
	}
	if p.Module != "" { applyFilter("module", p.Module) }
	if p.ActionType != "" { applyFilter("action_type", p.ActionType) }
	if p.StartDate != "" { applyFilter("created_at >=", p.StartDate+" 00:00:00") }
	if p.EndDate != "" { applyFilter("created_at <=", p.EndDate+" 23:59:59") }
	if p.AdminID != "" { applyFilter("admin_id", p.AdminID) }
	return
}

func (r *AdminRepository) GetRiskMetricsWindows(ctx context.Context, windows []int, topN int) ([]RiskMetricsWindow, error) {
	if topN <= 0 { topN = 10 }
	out := make([]RiskMetricsWindow, 0, len(windows))
	for _, hours := range windows {
		if hours <= 0 { continue }
		item := RiskMetricsWindow{Window: fmt.Sprintf("%dh", hours), Hours: hours}
		if err := r.fetchWindowStats(ctx, hours, &item); err != nil { return nil, err }
		if err := r.fetchTopRejectCodes(ctx, hours, topN, &item); err != nil { return nil, err }
		out = append(out, item)
	}
	return out, nil
}

func (r *AdminRepository) fetchWindowStats(ctx context.Context, hours int, item *RiskMetricsWindow) error {
	return r.db.QueryRow(ctx, `
		SELECT COUNT(*),
			COUNT(*) FILTER (WHERE COALESCE(new_value->>'result','')='pass'),
			COUNT(*) FILTER (WHERE COALESCE(new_value->>'result','')='reject'),
			COUNT(*) FILTER (WHERE COALESCE(new_value->>'result','')='error'),
			COUNT(*) FILTER (WHERE COALESCE(new_value->>'action','')='order_send' AND COALESCE(new_value->>'result','')='pass'),
			COUNT(*) FILTER (WHERE COALESCE(new_value->>'action','')='order_send' AND COALESCE(new_value->>'result','') IN ('reject','error')),
			COUNT(*) FILTER (WHERE COALESCE(new_value->>'action','')='order_close' AND COALESCE(new_value->>'result','')='pass'),
			COUNT(*) FILTER (WHERE COALESCE(new_value->>'action','')='order_close' AND COALESCE(new_value->>'result','') IN ('reject','error'))
		FROM system_operation_logs
		WHERE module='trading_risk' AND action='pre_trade_validate' AND created_at>=NOW()-($1::int*INTERVAL '1 hour')
	`, hours).Scan(&item.RiskValidateTotal, &item.RiskValidatePass, &item.RiskValidateReject, &item.RiskValidateError,
		&item.OrderSendSuccess, &item.OrderSendFailed, &item.OrderCloseSuccess, &item.OrderCloseFailed)
}

func (r *AdminRepository) fetchTopRejectCodes(ctx context.Context, hours, topN int, item *RiskMetricsWindow) error {
	rows, err := r.db.Query(ctx, `
		SELECT COALESCE(NULLIF(new_value->>'risk_code',''),'(none)'), COUNT(*)
		FROM system_operation_logs
		WHERE module='trading_risk' AND action='pre_trade_validate'
		  AND created_at>=NOW()-($1::int*INTERVAL '1 hour') AND COALESCE(new_value->>'result','')='reject'
		GROUP BY risk_code ORDER BY cnt DESC, risk_code ASC LIMIT $2
	`, hours, topN)
	if err != nil { return err }
	defer rows.Close()
	for rows.Next() {
		var code RiskCodeCount
		if err := rows.Scan(&code.RiskCode, &code.Count); err != nil { return err }
		item.TopRejectRiskCodes = append(item.TopRejectRiskCodes, code)
	}
	return rows.Err()
}
