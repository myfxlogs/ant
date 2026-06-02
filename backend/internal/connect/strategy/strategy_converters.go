package strategy

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service"
)

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
