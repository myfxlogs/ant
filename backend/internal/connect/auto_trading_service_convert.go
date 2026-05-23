package connect

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
)

func convertSchedule(schedule *model.StrategyScheduleLegacy) *v1.StrategySchedule {
	var lastRunAt, nextRunAt *timestamppb.Timestamp
	if schedule.LastRunAt != nil {
		lastRunAt = timestamppb.New(*schedule.LastRunAt)
	}
	if schedule.NextRunAt != nil {
		nextRunAt = timestamppb.New(*schedule.NextRunAt)
	}

	parameters := make(map[string]string)
	if len(schedule.Parameters) > 0 {
		_ = json.Unmarshal(schedule.Parameters, &parameters)
	}

	return &v1.StrategySchedule{
		Id:           schedule.ID.String(),
		UserId:       schedule.UserID.String(),
		TemplateId:   schedule.TemplateID.String(),
		AccountId:    schedule.AccountID.String(),
		Name:         schedule.Name,
		Symbol:       schedule.Symbol,
		Timeframe:    schedule.Timeframe,
		Parameters:   parameters,
		ScheduleType: schedule.ScheduleType,
		IsActive:     schedule.IsActive,
		LastRunAt:    lastRunAt,
		NextRunAt:    nextRunAt,
		RunCount:     int32(schedule.RunCount),
		LastError:    schedule.LastError,
		CreatedAt:    timestamppb.New(schedule.CreatedAt),
		UpdatedAt:    timestamppb.New(schedule.UpdatedAt),
	}
}
