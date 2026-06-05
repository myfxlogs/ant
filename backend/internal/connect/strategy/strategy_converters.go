package strategy

import (
	"encoding/json"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service"
)

// protoLog is a package-level logger for proto marshal/unmarshal diagnostics.
// Set via SetProtoLog during server init. If nil, errors are silent.
var protoLog *zap.Logger

func SetProtoLog(log *zap.Logger) { protoLog = log }

func logProtoErr(op, name string, err error) {
	if protoLog != nil {
		protoLog.Warn("strategy: proto "+op+" failed", zap.String("msg", name), zap.Error(err))
	}
}

// --- Proto conversion helpers ---

func templateRowToProto(r *service.TemplateRow) *antv1.StrategyTemplate {
	params := unmarshalTemplateParams(r.Parameters)
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
	// Parameters stored as proto binary (structpb.Struct → map[string]string).
	params := unmarshalScheduleParams(r.Parameters)
	// ScheduleConfig stored as proto binary.
	cfg := unmarshalScheduleConfig(r.ScheduleConfig)
	metrics := unmarshalProtoBacktestMetrics(r.BacktestMetrics)
	reasons := unmarshalProtoStringList(r.RiskReasons)
	warnings := unmarshalProtoStringList(r.RiskWarnings)

	s := &antv1.StrategySchedule{
		Id:             r.ID.String(),
		UserId:         r.UserID.String(),
		TemplateId:     r.TemplateID.String(),
		AccountId:      r.AccountID.String(),
		Name:           r.Name,
		Symbol:         r.Symbol,
		Timeframe:      r.Timeframe,
		Parameters:     params,
		ScheduleType:   r.ScheduleType,
		ScheduleConfig: cfg,
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
	return s
}

// --- Proto binary read helpers ---

// unmarshalScheduleParams unmarshals proto binary Parameters (stored as
// structpb.Struct) into a map[string]string.
func unmarshalScheduleParams(raw []byte) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	var sp structpb.Struct
	if err := proto.Unmarshal(raw, &sp); err != nil {
		logProtoErr("unmarshal", "schedule_params", err)
		return map[string]string{}
	}
	out := make(map[string]string, len(sp.Fields))
	for k, v := range sp.Fields {
		out[k] = v.GetStringValue()
	}
	return out
}

// unmarshalScheduleConfig unmarshals proto binary ScheduleConfig.
func unmarshalScheduleConfig(raw []byte) *antv1.ScheduleConfig {
	if len(raw) == 0 {
		return &antv1.ScheduleConfig{}
	}
	var cfg antv1.ScheduleConfig
	if err := proto.Unmarshal(raw, &cfg); err != nil {
		logProtoErr("unmarshal", "schedule_config", err)
		return &antv1.ScheduleConfig{}
	}
	return &cfg
}

// unmarshalTemplateParams unmarshals proto binary template parameters.
// Stored by marshaling a StrategyTemplate wrapper containing only the
// parameters field — decoding the wrapper recovers the repeated field.
func unmarshalTemplateParams(raw []byte) []*antv1.TemplateParameter {
	if len(raw) == 0 {
		return nil
	}
	// Try proto first (new format): unmarshal as StrategyTemplate wrapper.
	var wrapper antv1.StrategyTemplate
	if err := proto.Unmarshal(raw, &wrapper); err == nil && len(wrapper.Parameters) > 0 {
		return wrapper.Parameters
	} else if err != nil {
		logProtoErr("unmarshal", "template_params", err)
	}
	// Fallback: legacy JSON format.
	return unmarshalJSONTemplateParams(raw)
}

// unmarshalProtoBacktestMetrics unmarshals proto binary BacktestMetrics.
func unmarshalProtoBacktestMetrics(raw []byte) *antv1.BacktestMetrics {
	if len(raw) == 0 {
		return nil
	}
	var m antv1.BacktestMetrics
	if err := proto.Unmarshal(raw, &m); err != nil {
		logProtoErr("unmarshal", "backtest_metrics", err)
		return nil
	}
	return &m
}

// unmarshalProtoStringList unmarshals proto binary repeated string (stored as
// structpb.ListValue or JSON fallback).
func unmarshalProtoStringList(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	// Try proto: structpb.ListValue
	var lv structpb.ListValue
	if err := proto.Unmarshal(raw, &lv); err == nil {
		out := make([]string, 0, len(lv.Values))
		for _, v := range lv.Values {
			out = append(out, v.GetStringValue())
		}
		return out
	} else {
		logProtoErr("unmarshal", "string_list", err)
	}
	// Fallback: legacy JSON array.
	return mustParseJSON[[]string](raw, []string{})
}

// unmarshalJSONTemplateParams is a JSON fallback for legacy template parameter data.
func unmarshalJSONTemplateParams(raw []byte) []*antv1.TemplateParameter {
	return mustParseJSON[[]*antv1.TemplateParameter](raw, nil)
}

// mustParseJSON attempts to parse raw as JSON into T. Returns fallback on failure.
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

// paramsToProto marshals template parameters as proto binary.
// Uses a StrategyTemplate wrapper to encode the repeated field.
func templateParamsToProto(params []*antv1.TemplateParameter) []byte {
	if len(params) == 0 {
		return nil
	}
	wrapper := &antv1.StrategyTemplate{Parameters: params}
	b, err := proto.Marshal(wrapper)
	if err != nil {
		logProtoErr("marshal", "template_params", err)
		return nil
	}
	return b
}

// scheduleParamsToProto marshals a map[string]string as structpb.Struct proto binary.
func scheduleParamsToProto(params map[string]string) []byte {
	if len(params) == 0 {
		return nil
	}
	fields := make(map[string]*structpb.Value, len(params))
	for k, v := range params {
		fields[k] = structpb.NewStringValue(v)
	}
	b, err := proto.Marshal(&structpb.Struct{Fields: fields})
	if err != nil {
		logProtoErr("marshal", "schedule_params", err)
		return nil
	}
	return b
}

// stringListToProto marshals []string as structpb.ListValue proto binary.
func stringListToProto(items []string) []byte {
	if len(items) == 0 {
		return nil
	}
	vals := make([]*structpb.Value, len(items))
	for i, s := range items {
		vals[i] = structpb.NewStringValue(s)
	}
	b, err := proto.Marshal(&structpb.ListValue{Values: vals})
	if err != nil {
		logProtoErr("marshal", "string_list", err)
		return nil
	}
	return b
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
