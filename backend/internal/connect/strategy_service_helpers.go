package connect

import (
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
)

func convertScheduleToPB(s *model.StrategySchedule) *v1.StrategySchedule {
	if s == nil {
		return &v1.StrategySchedule{}
	}
	paramsOut := map[string]string{}
	if p, err := s.GetParameters(); err == nil {
		for k, v := range p {
			paramsOut[k] = toString(v)
		}
	}
	confMap, _ := s.GetScheduleConfig()
	conf := scheduleConfigFromMap(confMap)

	out := &v1.StrategySchedule{
		Id:             s.ID.String(),
		UserId:         s.UserID.String(),
		TemplateId:     s.TemplateID.String(),
		AccountId:      s.AccountID.String(),
		Name:           s.Name,
		Symbol:         s.Symbol,
		Timeframe:      s.Timeframe,
		Parameters:     paramsOut,
		ScheduleType:   s.ScheduleType,
		ScheduleConfig: conf,
		IsActive:       s.IsActive,
		RunCount:       int32(s.RunCount),
		EnableCount:    int32(s.EnableCount),
		LastError:      s.LastError,
		CreatedAt:      timestamppb.New(s.CreatedAt),
		UpdatedAt:      timestamppb.New(s.UpdatedAt),
	}
	if s.LastRunAt != nil {
		out.LastRunAt = timestamppb.New(*s.LastRunAt)
	}
	if s.NextRunAt != nil {
		out.NextRunAt = timestamppb.New(*s.NextRunAt)
	}
	return out
}

func scheduleConfigToMap(cfg *v1.ScheduleConfig) map[string]interface{} {
	if cfg == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"cron_expression": cfg.CronExpression,
		"interval_ms":     cfg.IntervalMs,
		"event_trigger":   cfg.EventTrigger,
		"trigger_mode":    cfg.TriggerMode,
		"stable_override_interval_ms": cfg.StableOverrideIntervalMs,
		"hf_cooldown_ms":  cfg.HfCooldownMs,
	}
}

func scheduleConfigFromMap(m map[string]interface{}) *v1.ScheduleConfig {
	if m == nil {
		return &v1.ScheduleConfig{}
	}
	c := &v1.ScheduleConfig{}
	if v, ok := m["cron_expression"]; ok {
		c.CronExpression = toString(v)
	}
	if v, ok := m["interval_ms"]; ok {
		c.IntervalMs = toInt64(v)
	}
	if v, ok := m["event_trigger"]; ok {
		c.EventTrigger = toString(v)
	}
	if v, ok := m["trigger_mode"]; ok {
		c.TriggerMode = toString(v)
	}
	if v, ok := m["stable_override_interval_ms"]; ok {
		c.StableOverrideIntervalMs = toInt64(v)
	}
	if v, ok := m["hf_cooldown_ms"]; ok {
		c.HfCooldownMs = toInt64(v)
	}
	return c
}

func ensureNextRunAtForInterval(s *model.StrategySchedule) error {
	if s == nil {
		return nil
	}
	if !s.IsActive {
		s.NextRunAt = nil
		return nil
	}
	if toString(s.ScheduleType) == model.ScheduleTypeEvent {
		s.NextRunAt = nil
		return nil
	}
	conf, err := s.GetScheduleConfig()
	if err != nil {
		return err
	}
	if conf == nil {
		next := time.Now().Add(timeframeToDurationForSchedule(s.Timeframe))
		s.NextRunAt = &next
		return nil
	}
	intervalMs := toInt64(conf["stable_override_interval_ms"])
	if intervalMs <= 0 {
		intervalMs = toInt64(conf["interval_ms"])
	}
	if intervalMs <= 0 {
		next := time.Now().Add(timeframeToDurationForSchedule(s.Timeframe))
		s.NextRunAt = &next
		return nil
	}
	next := time.Now().Add(time.Duration(intervalMs) * time.Millisecond)
	s.NextRunAt = &next
	return nil
}

func timeframeToDurationForSchedule(tf string) time.Duration {
	switch strings.ToUpper(strings.TrimSpace(toString(tf))) {
	case "M1":
		return 1 * time.Minute
	case "M5":
		return 5 * time.Minute
	case "M15":
		return 15 * time.Minute
	case "M30":
		return 30 * time.Minute
	case "H1":
		return 1 * time.Hour
	case "H4":
		return 4 * time.Hour
	case "D1":
		return 24 * time.Hour
	default:
		return 5 * time.Minute
	}
}

func mustJSON(m map[string]interface{}) []byte {
	// model.JSONB is []byte; use model's helper if available, else json via SetScheduleConfig.
	s := &model.StrategySchedule{}
	_ = s.SetScheduleConfig(m)
	return s.ScheduleConfig
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return ""
}

func toInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	default:
		return 0
	}
}
