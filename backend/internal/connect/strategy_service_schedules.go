package connect

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	"anttrader/internal/service"
)

func (s *StrategyService) ListSchedules(ctx context.Context, req *connect.Request[v1.ListSchedulesRequest]) (*connect.Response[v1.ListSchedulesResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.schedule == nil {
		return connect.NewResponse(&v1.ListSchedulesResponse{Schedules: []*v1.StrategySchedule{}}), nil
	}
	items, err := s.schedule.GetSchedulesByUser(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.StrategySchedule, 0, len(items))
	for _, it := range items {
		out = append(out, convertScheduleToPB(it))
	}
	return connect.NewResponse(&v1.ListSchedulesResponse{Schedules: out}), nil
}

func (s *StrategyService) GetSchedule(ctx context.Context, req *connect.Request[v1.GetScheduleRequest]) (*connect.Response[v1.StrategySchedule], error) {
	if s.schedule == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("schedule service not available"))
	}
	scheduleID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	item, err := s.schedule.GetSchedule(ctx, scheduleID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(convertScheduleToPB(item)), nil
}

func (s *StrategyService) CreateSchedule(ctx context.Context, req *connect.Request[v1.CreateScheduleRequest]) (*connect.Response[v1.StrategySchedule], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if s.schedule == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("schedule service not available"))
	}

	tplID, err := uuid.Parse(req.Msg.TemplateId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	accID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	params := make(map[string]interface{}, len(req.Msg.Parameters))
	for k, v := range req.Msg.Parameters {
		params[k] = v
	}
	conf := scheduleConfigToMap(req.Msg.ScheduleConfig)

	created, err := s.schedule.CreateSchedule(ctx, userID, &service.CreateScheduleRequest{
		TemplateID:     tplID,
		AccountID:      accID,
		Name:           req.Msg.Name,
		Symbol:         req.Msg.Symbol,
		Timeframe:      req.Msg.Timeframe,
		Parameters:     params,
		ScheduleType:   req.Msg.ScheduleType,
		ScheduleConfig: conf,
	})
	if err != nil {
		if errors.Is(err, service.ErrScheduleAlreadyExists) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		if errors.Is(err, service.ErrTemplateNotPublished) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Note: schedule is created as inactive by default.
	// next_run_at will be set when the schedule is enabled.

	return connect.NewResponse(convertScheduleToPB(created)), nil
}

func (s *StrategyService) UpdateSchedule(ctx context.Context, req *connect.Request[v1.UpdateScheduleRequest]) (*connect.Response[v1.StrategySchedule], error) {
	if s.schedule == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("schedule service not available"))
	}
	scheduleID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	item, err := s.schedule.GetSchedule(ctx, scheduleID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if req.Msg.Name != nil {
		item.Name = *req.Msg.Name
	}
	if req.Msg.Symbol != nil {
		item.Symbol = *req.Msg.Symbol
	}
	if req.Msg.Timeframe != nil {
		item.Timeframe = *req.Msg.Timeframe
	}
	if req.Msg.ScheduleType != nil {
		item.ScheduleType = *req.Msg.ScheduleType
	}
	if req.Msg.ScheduleConfig != nil {
		item.ScheduleConfig = mustJSON(scheduleConfigToMap(req.Msg.ScheduleConfig))
	}
	if req.Msg.Parameters != nil {
		p := make(map[string]interface{}, len(req.Msg.Parameters))
		for k, v := range req.Msg.Parameters {
			p[k] = v
		}
		_ = item.SetParameters(p)
	}

	_ = ensureNextRunAtForInterval(item)
	if err := s.schedule.UpdateSchedule(ctx, item); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertScheduleToPB(item)), nil
}

func (s *StrategyService) DeleteSchedule(ctx context.Context, req *connect.Request[v1.DeleteScheduleRequest]) (*connect.Response[emptypb.Empty], error) {
	if s.schedule == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("schedule service not available"))
	}
	scheduleID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := s.schedule.DeleteSchedule(ctx, scheduleID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *StrategyService) ToggleSchedule(ctx context.Context, req *connect.Request[v1.ToggleScheduleRequest]) (*connect.Response[v1.StrategySchedule], error) {
	if s.schedule == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("schedule service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	scheduleID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	item, err := s.schedule.ToggleSchedule(ctx, userID, scheduleID, req.Msg.Active)
	if err != nil {
		if errors.Is(err, service.ErrActiveScheduleConflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if s.logSvc != nil {
		op := model.NewSystemOperationLog(userID, model.OperationTypeUpdate, "strategy_schedule", func() string {
			if req.Msg.Active {
				return "enable"
			}
			return "disable"
		}())
		op.ResourceType = "schedule"
		op.ResourceID = scheduleID
		op.Status = model.OperationStatusSuccess
		op.NewValue = map[string]interface{}{"active": req.Msg.Active}
		op.DurationMs = time.Since(start).Milliseconds()
		_ = s.logSvc.LogOperation(ctx, op)
	}
	return connect.NewResponse(convertScheduleToPB(item)), nil
}

func convertStrategyToSchedule(s *model.Strategy) *v1.StrategySchedule {
	isActive := s.Status == model.StrategyStatusActive
	return &v1.StrategySchedule{
		Id:        s.ID.String(),
		UserId:    s.UserID.String(),
		AccountId: s.AccountID.String(),
		Name:      s.Name,
		Symbol:    s.Symbol,
		IsActive:  isActive,
		CreatedAt: timestamppb.New(s.CreatedAt),
		UpdatedAt: timestamppb.New(s.UpdatedAt),
	}
}
