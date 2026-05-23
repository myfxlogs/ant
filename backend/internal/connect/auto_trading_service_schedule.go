package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/interceptor"
	"anttrader/internal/service"
)

func (s *AutoTradingService) CreateSchedule(ctx context.Context, req *connect.Request[v1.CreateScheduleRequest]) (*connect.Response[v1.StrategySchedule], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeWrite); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	accountID, err := uuid.Parse(req.Msg.AccountId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if s.schedule != nil {
		tplID, err := uuid.Parse(req.Msg.TemplateId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		params := make(map[string]interface{}, len(req.Msg.Parameters))
		for k, v := range req.Msg.Parameters {
			params[k] = v
		}
		conf := scheduleConfigToMap(req.Msg.ScheduleConfig)
		created, err := s.schedule.CreateSchedule(ctx, uid, &service.CreateScheduleRequest{
			TemplateID:     tplID,
			AccountID:      accountID,
			Name:           req.Msg.Name,
			Symbol:         req.Msg.Symbol,
			Timeframe:      req.Msg.Timeframe,
			Parameters:     params,
			ScheduleType:   req.Msg.ScheduleType,
			ScheduleConfig: conf,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		_ = ensureNextRunAtForInterval(created)
		_ = s.schedule.UpdateSchedule(ctx, created)
		return connect.NewResponse(convertScheduleToPB(created)), nil
	}

	return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("errors.schedule_service_not_available"))
}

func (s *AutoTradingService) GetSchedule(ctx context.Context, req *connect.Request[v1.GetScheduleRequest]) (*connect.Response[v1.StrategySchedule], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeRead); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	_, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if s.schedule != nil {
		item, err := s.schedule.GetSchedule(ctx, id)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return connect.NewResponse(convertScheduleToPB(item)), nil
	}
	return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("errors.schedule_service_not_available"))
}

func (s *AutoTradingService) UpdateSchedule(ctx context.Context, req *connect.Request[v1.UpdateScheduleRequest]) (*connect.Response[v1.StrategySchedule], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeWrite); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	_, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if s.schedule != nil {
		item, err := s.schedule.GetSchedule(ctx, id)
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
	return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("errors.schedule_service_not_available"))
}

func (s *AutoTradingService) DeleteSchedule(ctx context.Context, req *connect.Request[v1.DeleteScheduleRequest]) (*connect.Response[emptypb.Empty], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeWrite); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	_, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if s.schedule != nil {
		if err := s.schedule.DeleteSchedule(ctx, id); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&emptypb.Empty{}), nil
	}
	return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("errors.schedule_service_not_available"))
}

func (s *AutoTradingService) ToggleSchedule(ctx context.Context, req *connect.Request[v1.ToggleScheduleRequest]) (*connect.Response[v1.StrategySchedule], error) {
	if err := interceptor.RequireScopes(ctx, interceptor.ScopeTradeWrite); err != nil {
		return nil, err
	}

	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("errors.not_authenticated"))
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if s.schedule != nil {
		item, err := s.schedule.ToggleSchedule(ctx, uid, id, req.Msg.Active)
		if err != nil {
			if errors.Is(err, service.ErrActiveScheduleConflict) {
				return nil, connect.NewError(connect.CodeAlreadyExists, err)
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(convertScheduleToPB(item)), nil
	}
	return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("errors.schedule_service_not_available"))
}
