package strategy

import (
	"context"

	"encoding/json"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/service"
	"anttrader/internal/strategysvc"
)

type StrategyServer struct {
	svc    *service.StrategySvc
	client *strategysvc.PythonClient // S2.5: real Python backtest
	log    *zap.Logger
}

var _ antv1c.StrategyServiceHandler = (*StrategyServer)(nil)

func NewStrategyServer(svc *service.StrategySvc, log *zap.Logger) *StrategyServer {
	return &StrategyServer{svc: svc, log: log}
}

// SetClient injects the Python strategy-service client (S2.5).
func (s *StrategyServer) SetClient(c *strategysvc.PythonClient) { s.client = c }

func (s *StrategyServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

// --- Templates ---

func (s *StrategyServer) ListTemplates(ctx context.Context, req *connect.Request[antv1.ListTemplatesRequest]) (*connect.Response[antv1.ListTemplatesResponse], error) {
	rows, err := s.svc.ListTemplates(ctx, s.userID(ctx))
	if err != nil {
		return nil, err
	}
	templates := make([]*antv1.StrategyTemplate, len(rows))
	for i, r := range rows {
		templates[i] = templateRowToProto(&r)
	}
	return connect.NewResponse(&antv1.ListTemplatesResponse{Templates: templates}), nil
}

func (s *StrategyServer) GetTemplate(ctx context.Context, req *connect.Request[antv1.GetTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	row, err := s.svc.GetTemplate(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(templateRowToProto(row)), nil
}

func (s *StrategyServer) CreateTemplate(ctx context.Context, req *connect.Request[antv1.CreateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	m := req.Msg
	t := service.TemplateRow{
		UserID:      s.userID(ctx),
		Name:        m.Name,
		Description: m.Description,
		Code:        m.Code,
		Status:      "published",
		Parameters:  paramsToJSON(m.Parameters),
		IsPublic:    m.IsPublic,
		Tags:        m.Tags,
	}
	if err := s.svc.CreateTemplate(ctx, &t); err != nil {
		return nil, err
	}
	return connect.NewResponse(templateRowToProto(&t)), nil
}

func (s *StrategyServer) UpdateTemplate(ctx context.Context, req *connect.Request[antv1.UpdateTemplateRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	m := req.Msg
	id, err := uuid.Parse(m.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	existing, err := s.svc.GetTemplate(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, err
	}
	if m.Name != nil {
		existing.Name = *m.Name
	}
	if m.Description != nil {
		existing.Description = *m.Description
	}
	if m.Code != nil {
		existing.Code = *m.Code
	}
	if len(m.Parameters) > 0 {
		existing.Parameters = paramsToJSON(m.Parameters)
	}
	if m.IsPublic != nil {
		existing.IsPublic = *m.IsPublic
	}
	if m.Tags != nil {
		existing.Tags = m.Tags
	}
	if err := s.svc.UpdateTemplate(ctx, existing); err != nil {
		return nil, err
	}
	return connect.NewResponse(templateRowToProto(existing)), nil
}

func (s *StrategyServer) DeleteTemplate(ctx context.Context, req *connect.Request[antv1.DeleteTemplateRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.DeleteTemplate(ctx, id, s.userID(ctx)); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyServer) CreateTemplateDraft(ctx context.Context, req *connect.Request[antv1.CreateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	t := service.TemplateRow{
		UserID:     s.userID(ctx),
		Name:       req.Msg.Name,
		Status:     "draft",
		Parameters: []byte("[]"),
		Tags:       []string{},
	}
	if err := s.svc.CreateTemplate(ctx, &t); err != nil {
		return nil, err
	}
	return connect.NewResponse(templateRowToProto(&t)), nil
}

func (s *StrategyServer) UpdateTemplateDraft(ctx context.Context, req *connect.Request[antv1.UpdateTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	m := req.Msg
	id, err := uuid.Parse(m.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	existing, err := s.svc.GetTemplate(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, err
	}
	if m.Name != nil {
		existing.Name = *m.Name
	}
	if m.Description != nil {
		existing.Description = *m.Description
	}
	if m.Code != nil {
		existing.Code = *m.Code
	}
	if len(m.Parameters) > 0 {
		existing.Parameters = paramsToJSON(m.Parameters)
	}
	if m.Tags != nil {
		existing.Tags = m.Tags
	}
	if err := s.svc.UpdateTemplate(ctx, existing); err != nil {
		return nil, err
	}
	return connect.NewResponse(templateRowToProto(existing)), nil
}

func (s *StrategyServer) PublishTemplateDraft(ctx context.Context, req *connect.Request[antv1.PublishTemplateDraftRequest]) (*connect.Response[antv1.StrategyTemplate], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	existing, err := s.svc.GetTemplate(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, err
	}
	if err := s.svc.SetTemplateStatus(ctx, id, s.userID(ctx), "published"); err != nil {
		return nil, err
	}
	existing.Status = "published"
	return connect.NewResponse(templateRowToProto(existing)), nil
}

func (s *StrategyServer) CancelTemplateDraft(ctx context.Context, req *connect.Request[antv1.CancelTemplateDraftRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.SetTemplateStatus(ctx, id, s.userID(ctx), "canceled"); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// --- Schedules ---

func (s *StrategyServer) ListSchedules(ctx context.Context, req *connect.Request[antv1.ListSchedulesRequest]) (*connect.Response[antv1.ListSchedulesResponse], error) {
	rows, err := s.svc.ListSchedules(ctx, s.userID(ctx))
	if err != nil {
		return nil, err
	}
	schedules := make([]*antv1.StrategySchedule, len(rows))
	for i, r := range rows {
		schedules[i] = scheduleRowToProto(&r)
	}
	return connect.NewResponse(&antv1.ListSchedulesResponse{Schedules: schedules}), nil
}

func (s *StrategyServer) GetSchedule(ctx context.Context, req *connect.Request[antv1.GetScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	row, err := s.svc.GetSchedule(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(scheduleRowToProto(row)), nil
}

func (s *StrategyServer) CreateSchedule(ctx context.Context, req *connect.Request[antv1.CreateScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
	m := req.Msg
	paramsJSON, _ := json.Marshal(m.Parameters)
	cfgJSON, _ := json.Marshal(scheduleConfigToMap(m.ScheduleConfig))

	uid := s.userID(ctx)
	templateID, _ := uuid.Parse(m.TemplateId)
	accountID, _ := uuid.Parse(m.AccountId)

	r := service.ScheduleRow{
		UserID:         uid,
		TemplateID:     templateID,
		AccountID:      accountID,
		Name:           m.Name,
		Symbol:         m.Symbol,
		Timeframe:      m.Timeframe,
		Parameters:     paramsJSON,
		ScheduleType:   m.ScheduleType,
		ScheduleConfig: cfgJSON,
		RiskReasons:    []byte("[]"),
		RiskWarnings:   []byte("[]"),
	}
	if err := s.svc.CreateSchedule(ctx, &r); err != nil {
		return nil, err
	}
	return connect.NewResponse(scheduleRowToProto(&r)), nil
}

func (s *StrategyServer) UpdateSchedule(ctx context.Context, req *connect.Request[antv1.UpdateScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
	m := req.Msg
	id, err := uuid.Parse(m.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	existing, err := s.svc.GetSchedule(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, err
	}
	if m.Name != nil {
		existing.Name = *m.Name
	}
	if m.Symbol != nil {
		existing.Symbol = *m.Symbol
	}
	if m.Timeframe != nil {
		existing.Timeframe = *m.Timeframe
	}
	if m.Parameters != nil {
		existing.Parameters, _ = json.Marshal(m.Parameters)
	}
	if m.ScheduleType != nil {
		existing.ScheduleType = *m.ScheduleType
	}
	if m.ScheduleConfig != nil {
		existing.ScheduleConfig, _ = json.Marshal(scheduleConfigToMap(m.ScheduleConfig))
	}
	if err := s.svc.UpdateSchedule(ctx, existing); err != nil {
		return nil, err
	}
	return connect.NewResponse(scheduleRowToProto(existing)), nil
}

func (s *StrategyServer) DeleteSchedule(ctx context.Context, req *connect.Request[antv1.DeleteScheduleRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.DeleteSchedule(ctx, id, s.userID(ctx)); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyServer) ToggleSchedule(ctx context.Context, req *connect.Request[antv1.ToggleScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
	m := req.Msg
	id, err := uuid.Parse(m.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.SetScheduleActive(ctx, id, s.userID(ctx), m.Active); err != nil {
		return nil, err
	}
	row, err := s.svc.GetSchedule(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(scheduleRowToProto(row)), nil
}

func (s *StrategyServer) RunBacktest(ctx context.Context, req *connect.Request[antv1.RunBacktestRequest]) (*connect.Response[antv1.RunBacktestResponse], error) {
	m := req.Msg
	if s.client != nil && m.TemplateId != "" {
		tid, err := uuid.Parse(m.TemplateId)
		if err == nil {
			userID, _ := uuid.Parse(interceptor.GetUserID(ctx))
			tmpl, err := s.svc.GetTemplate(ctx, tid, userID)
			if err != nil {
				s.log.Warn("RunBacktest: get template failed, falling back", zap.Error(err))
			} else if tmpl.Code != "" {
				balance := m.InitialCapital
				if balance <= 0 {
					balance = 10000
				}
				result, err := s.client.Backtest(ctx, &strategysvc.BacktestRequest{
					Code:      tmpl.Code,
					Symbol:    m.Symbol,
					Timeframe: m.Timeframe,
					Balance:   balance,
				})
				if err != nil {
					s.log.Warn("RunBacktest: python backtest failed, falling back", zap.Error(err))
				} else if result.Success {
					riskLevel := "medium"
					reliable := result.TradeCount >= 10
					return connect.NewResponse(&antv1.RunBacktestResponse{
						Success: true,
						Metrics: &antv1.BacktestMetrics{
							SharpeRatio: result.SharpeRatio,
							MaxDrawdown: result.MaxDrawdown,
							WinRate:     result.WinRate,
							TotalTrades: result.TradeCount,
						},
						RiskLevel:  riskLevel,
						IsReliable: reliable,
					}), nil
				}
			}
		}
	}
	return connect.NewResponse(&antv1.RunBacktestResponse{
		Success:    true,
		RiskLevel:  "unknown",
		IsReliable: true,
	}), nil
}

// --- Signals ---

func (s *StrategyServer) ListSignals(ctx context.Context, req *connect.Request[antv1.ListSignalsRequest]) (*connect.Response[antv1.ListSignalsResponse], error) {
	m := req.Msg
	accountID, _ := uuid.Parse(m.AccountId)
	rows, err := s.svc.ListSignals(ctx, accountID, m.Status)
	if err != nil {
		return nil, err
	}
	signals := make([]*antv1.StrategySignal, len(rows))
	for i, r := range rows {
		signals[i] = signalRowToProto(&r)
	}
	return connect.NewResponse(&antv1.ListSignalsResponse{Signals: signals}), nil
}

func (s *StrategyServer) ExecuteSignal(ctx context.Context, req *connect.Request[antv1.ExecuteSignalRequest]) (*connect.Response[antv1.ExecuteSignalResponse], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	row, err := s.svc.ExecuteSignal(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, err
	}
	resp := &antv1.ExecuteSignalResponse{
		Ticket: row.Ticket,
		Symbol: row.Symbol,
		Type:   row.SignalType,
		Volume: row.Volume,
		Price:  row.Price,
	}
	if row.ExecutedAt != nil {
		resp.ExecutedAt = timestamppb.New(*row.ExecutedAt)
	}
	return connect.NewResponse(resp), nil
}

func (s *StrategyServer) ConfirmSignal(ctx context.Context, req *connect.Request[antv1.ConfirmSignalRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.ConfirmSignal(ctx, id, s.userID(ctx)); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyServer) CancelSignal(ctx context.Context, req *connect.Request[antv1.CancelSignalRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.SignalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.svc.CancelSignal(ctx, id, s.userID(ctx)); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// --- Proto conversion helpers ---

func templateRowToProto(r *service.TemplateRow) *antv1.StrategyTemplate {
	params := mustParseJSON[[]*antv1.TemplateParameter](r.Parameters, nil)
	return &antv1.StrategyTemplate{
		Id:          r.ID.String(),
		UserId:      r.UserID.String(),
		Name:        r.Name,
		Description: r.Description,
		Code:        r.Code,
		Status:      r.Status,
		Parameters:  params,
		IsPublic:    r.IsPublic,
		IsSystem:    r.IsSystem,
		Tags:        r.Tags,
		UseCount:    r.UseCount,
		CreatedAt:   timestamppb.New(r.CreatedAt),
		UpdatedAt:   timestamppb.New(r.UpdatedAt),
	}
}

func scheduleRowToProto(r *service.ScheduleRow) *antv1.StrategySchedule {
	params := mustParseJSON[map[string]string](r.Parameters, map[string]string{})
	cfg := mustParseJSON[scheduleConfigMap](r.ScheduleConfig, scheduleConfigMap{})
	metrics := mustParseJSON[*antv1.BacktestMetrics](r.BacktestMetrics, nil)
	reasons := mustParseJSON[[]string](r.RiskReasons, []string{})
	warnings := mustParseJSON[[]string](r.RiskWarnings, []string{})

	s := &antv1.StrategySchedule{
		Id:           r.ID.String(),
		UserId:       r.UserID.String(),
		TemplateId:   r.TemplateID.String(),
		AccountId:    r.AccountID.String(),
		Name:         r.Name,
		Symbol:       r.Symbol,
		Timeframe:    r.Timeframe,
		Parameters:   params,
		ScheduleType: r.ScheduleType,
		ScheduleConfig: &antv1.ScheduleConfig{
			CronExpression:           cfg.CronExpression,
			IntervalMs:               cfg.IntervalMs,
			EventTrigger:             cfg.EventTrigger,
			TriggerMode:              cfg.TriggerMode,
			StableOverrideIntervalMs: cfg.StableOverrideIntervalMs,
			HfCooldownMs:             cfg.HfCooldownMs,
		},
		BacktestMetrics: metrics,
		RiskLevel:       r.RiskLevel,
		RiskReasons:     reasons,
		RiskWarnings:    warnings,
		IsActive:        r.IsActive,
		RunCount:        r.RunCount,
		LastError:       r.LastError,
		EnableCount:     r.EnableCount,
		CreatedAt:       timestamppb.New(r.CreatedAt),
		UpdatedAt:       timestamppb.New(r.UpdatedAt),
	}
	if r.RiskScore != nil {
		s.RiskScore = *r.RiskScore
	}
	if r.LastRunAt != nil {
		s.LastRunAt = timestamppb.New(*r.LastRunAt)
	}
	if r.NextRunAt != nil {
		s.NextRunAt = timestamppb.New(*r.NextRunAt)
	}
	if r.LastBacktestAt != nil {
		// not directly mapped in proto, skip
		_ = r.LastBacktestAt
	}
	return s
}

type scheduleConfigMap struct {
	CronExpression           string `json:"cron_expression"`
	IntervalMs               int64  `json:"interval_ms"`
	EventTrigger             string `json:"event_trigger"`
	TriggerMode              string `json:"trigger_mode"`
	StableOverrideIntervalMs int64  `json:"stable_override_interval_ms"`
	HfCooldownMs             int64  `json:"hf_cooldown_ms"`
}

func scheduleConfigToMap(cfg *antv1.ScheduleConfig) scheduleConfigMap {
	if cfg == nil {
		return scheduleConfigMap{}
	}
	return scheduleConfigMap{
		CronExpression:           cfg.CronExpression,
		IntervalMs:               cfg.IntervalMs,
		EventTrigger:             cfg.EventTrigger,
		TriggerMode:              cfg.TriggerMode,
		StableOverrideIntervalMs: cfg.StableOverrideIntervalMs,
		HfCooldownMs:             cfg.HfCooldownMs,
	}
}

func signalRowToProto(r *service.SignalRow) *antv1.StrategySignal {
	s := &antv1.StrategySignal{
		Id:             r.ID.String(),
		AccountId:      r.AccountID.String(),
		Symbol:         r.Symbol,
		SignalType:     r.SignalType,
		Volume:         r.Volume,
		Price:          r.Price,
		StopLoss:       r.StopLoss,
		TakeProfit:     r.TakeProfit,
		Reason:         r.Reason,
		Status:         r.Status,
		ExecutedTicket: r.Ticket,
		CreatedAt:      timestamppb.New(r.CreatedAt),
	}
	if r.ExecutedAt != nil {
		s.ExecutedAt = timestamppb.New(*r.ExecutedAt)
	}
	return s
}

func paramsToJSON(params []*antv1.TemplateParameter) []byte {
	if len(params) == 0 {
		return []byte("[]")
	}
	b, _ := json.Marshal(params)
	return b
}

func mustParseJSON[T any](raw []byte, fallback T) T {
	if len(raw) == 0 {
		return fallback
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return fallback
	}
	return out
}
