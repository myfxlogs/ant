package marketplace

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// ── Admin: Batch Generation RPCs (Phase 2.2d) ────────────────────────────────

func (s *MarketplaceServer) ListAutoGenTasks(
	ctx context.Context,
	req *connect.Request[antv1.ListAutoGenTasksRequest],
) (*connect.Response[antv1.ListAutoGenTasksResponse], error) {
	if s.batch == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("batch generator not configured"))
	}
	if _, err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}

	limit := int(req.Msg.Limit)
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	tasks, err := s.batch.ListTasks(ctx, req.Msg.Status, limit, int(req.Msg.Offset))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list tasks: %w", err))
	}

	var infos []*antv1.AutoGenTaskInfo
	for _, t := range tasks {
		info := &antv1.AutoGenTaskInfo{
			Id:           t.ID.String(),
			Symbol:       t.Symbol,
			Timeframe:    t.Timeframe,
			StrategyType: t.StrategyType,
			RiskLevel:    t.RiskLevel,
			Status:       t.Status,
			CreatedAtMs:  t.CreatedAt.UnixMilli(),
		}
		if t.StrategyID != nil {
			info.StrategyId = t.StrategyID.String()
		}
		if t.QualityPassed != nil {
			info.QualityPassed = *t.QualityPassed
		}
		if t.ErrorMessage != nil {
			info.ErrorMessage = *t.ErrorMessage
		}
		if t.FinishedAt != nil {
			info.FinishedAtMs = t.FinishedAt.UnixMilli()
		}
		infos = append(infos, info)
	}

	return connect.NewResponse(&antv1.ListAutoGenTasksResponse{
		Tasks: infos,
		Total: int32(len(infos)),
	}), nil
}

func (s *MarketplaceServer) ApproveAutoGenTask(
	ctx context.Context,
	req *connect.Request[antv1.ApproveAutoGenTaskRequest],
) (*connect.Response[antv1.ApproveAutoGenTaskResponse], error) {
	if s.batch == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("batch generator not configured"))
	}
	if _, err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}

	taskID, err := uuid.Parse(req.Msg.TaskId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid task_id"))
	}

	if err := s.batch.ApproveTask(ctx, taskID, s.svc); err != nil {
		return connect.NewResponse(&antv1.ApproveAutoGenTaskResponse{
			Success: false,
			Error:   err.Error(),
		}), nil
	}

	return connect.NewResponse(&antv1.ApproveAutoGenTaskResponse{Success: true}), nil
}

func (s *MarketplaceServer) RejectAutoGenTask(
	ctx context.Context,
	req *connect.Request[antv1.RejectAutoGenTaskRequest],
) (*connect.Response[antv1.RejectAutoGenTaskResponse], error) {
	if s.batch == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("batch generator not configured"))
	}
	if _, err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}

	taskID, err := uuid.Parse(req.Msg.TaskId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid task_id"))
	}

	if err := s.batch.RejectTask(ctx, taskID, req.Msg.Reason); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.RejectAutoGenTaskResponse{Success: true}), nil
}

func (s *MarketplaceServer) TriggerBatchGeneration(
	ctx context.Context,
	req *connect.Request[antv1.TriggerBatchGenerationRequest],
) (*connect.Response[antv1.TriggerBatchGenerationResponse], error) {
	if s.batch == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("batch generator not configured"))
	}
	if _, err := s.checkAdmin(ctx); err != nil {
		return nil, err
	}

	msg := req.Msg
	risk := msg.RiskLevel
	if risk == "" {
		risk = "moderate"
	}
	count, err := s.batch.EnqueueBatch(ctx, msg.Symbols, msg.Timeframes, msg.StrategyTypes, risk)
	if err != nil {
		return connect.NewResponse(&antv1.TriggerBatchGenerationResponse{
			Enqueued: int32(count),
			Error:    err.Error(),
		}), nil
	}

	return connect.NewResponse(&antv1.TriggerBatchGenerationResponse{
		Enqueued: int32(count),
	}), nil
}

// ── ListStrategyTemplates handler (Phase 2.3) ────────────────────────────────

func (s *MarketplaceServer) ListStrategyTemplates(
	ctx context.Context,
	req *connect.Request[antv1.ListStrategyTemplatesRequest],
) (*connect.Response[antv1.ListStrategyTemplatesResponse], error) {
	if s.pgPool == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("database not configured"))
	}

	lang := langFromAccept(req.Header().Get("Accept-Language"))

	query := `SELECT id, template_key, template_type, name_i18n, description_i18n, parameters, default_risk_level, icon
	          FROM strategy_parameter_templates WHERE enabled = true`
	args := []any{}
	if tt := req.Msg.TemplateType; tt != "" {
		query += ` AND template_type = $1`
		args = append(args, tt)
	}
	query += ` ORDER BY sort_order`

	rows, err := s.pgPool.Query(ctx, query, args...)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("query templates: %w", err))
	}
	defer rows.Close()

	var templates []*antv1.StrategyParameterTemplate
	for rows.Next() {
		var id, key, tType, nameI18n, descI18n, paramsJSON, riskLevel, icon string
		if err := rows.Scan(&id, &key, &tType, &nameI18n, &descI18n, &paramsJSON, &riskLevel, &icon); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("scan template: %w", err))
		}
		templates = append(templates, &antv1.StrategyParameterTemplate{
			Id:               id,
			TemplateKey:      key,
			TemplateType:     tType,
			Name:             pickLocalized(nameI18n, lang),
			Description:      pickLocalized(descI18n, lang),
			ParametersSchema: paramsJSON,
			DefaultRiskLevel: riskLevel,
			Icon:             icon,
		})
	}

	return connect.NewResponse(&antv1.ListStrategyTemplatesResponse{Templates: templates}), nil
}

// ── i18n helpers ─────────────────────────────────────────────────────────────

func langFromAccept(acceptLang string) string {
	if acceptLang == "" {
		return "en"
	}
	if len(acceptLang) >= 2 && acceptLang[:2] == "zh" {
		return "zh-cn"
	}
	return "en"
}

// pickLocalized extracts the best matching language from a JSONB i18n map.
func pickLocalized(jsonStr string, lang string) string {
	key := `"` + lang + `":"`
	idx := strings.Index(jsonStr, key)
	if idx < 0 {
		key = `"en":"`
		idx = strings.Index(jsonStr, key)
		if idx < 0 {
			return ""
		}
	}
	start := idx + len(key)
	end := strings.Index(jsonStr[start:], `"`)
	if end < 0 {
		return ""
	}
	return jsonStr[start : start+end]
}
