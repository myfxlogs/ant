package strategy

import (
	"context"
	"time"
	"google.golang.org/protobuf/proto"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service"
)

// --- Schedules ---

func (s *StrategyServer) ListSchedules(ctx context.Context, req *connect.Request[antv1.ListSchedulesRequest]) (*connect.Response[antv1.ListSchedulesResponse], error) {
	rows, err := s.svc.ListSchedules(ctx, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
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
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(scheduleRowToProto(row)), nil
}

func (s *StrategyServer) CreateSchedule(ctx context.Context, req *connect.Request[antv1.CreateScheduleRequest]) (*connect.Response[antv1.StrategySchedule], error) {
	m := req.Msg
	if strings.TrimSpace(m.Symbol) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("symbol is required"))
	}
	if strings.TrimSpace(m.Timeframe) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("timeframe is required"))
	}
	if !validScheduleType(m.ScheduleType) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid schedule type: %s", m.ScheduleType))
	}
	cfgBytes, err := proto.Marshal(m.ScheduleConfig)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal schedule config: %w", err))
	}

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
		Parameters:     scheduleParamsToProto(m.Parameters),
		ScheduleType:   m.ScheduleType,
		ScheduleConfig: cfgBytes,
		RiskReasons:    stringListToProto(nil),
		RiskWarnings:   stringListToProto(nil),
	}
	if err := s.svc.CreateSchedule(ctx, &r); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if s.engine != nil {
		s.engine.Notify()
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
		return nil, connect.NewError(connect.CodeInternal, err)
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
		existing.Parameters = scheduleParamsToProto(m.Parameters)
	}
	if m.ScheduleType != nil {
		existing.ScheduleType = *m.ScheduleType
	}
	if m.ScheduleConfig != nil {
		if b, err := proto.Marshal(m.ScheduleConfig); err == nil {
			existing.ScheduleConfig = b
		} else {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal schedule config: %w", err))
		}
	}
	if err := s.svc.UpdateSchedule(ctx, existing); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if s.engine != nil {
		s.engine.Notify()
	}
	return connect.NewResponse(scheduleRowToProto(existing)), nil
}

func (s *StrategyServer) DeleteSchedule(ctx context.Context, req *connect.Request[antv1.DeleteScheduleRequest]) (*connect.Response[emptypb.Empty], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if s.engine != nil {
		s.engine.StopSchedule(id)
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
	if s.engine != nil {
		if !m.Active {
			s.engine.StopSchedule(id)
		} else {
			s.engine.Notify()
		}
	}
	row, err := s.svc.GetSchedule(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(scheduleRowToProto(row)), nil
}

func validScheduleType(t string) bool {
	switch t {
	case "cron", "interval", "event":
		return true
	}
	return false
}

// WatchSchedules streams the full schedule list whenever it changes.
// Push-first architecture: replaces client-side polling with SSE stream.
func (s *StrategyServer) WatchSchedules(ctx context.Context, req *connect.Request[antv1.WatchSchedulesRequest], stream *connect.ServerStream[antv1.WatchSchedulesEvent]) error {
	uid := s.userID(ctx)
	var prevHash string
	notifCh, listenCancel, _ := s.pgListen.Listen(ctx, "schedule_change")
	if listenCancel != nil {
		defer listenCancel()
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-notifCh:
		case <-ticker.C:
		}

		rows, err := s.svc.ListSchedules(ctx, uid)
		if err != nil {
			continue
		}
		// Compute a simple hash to detect changes
		var sb strings.Builder
		for _, r := range rows {
			sb.WriteString(r.ID.String())
			if r.IsActive { sb.WriteByte('1') } else { sb.WriteByte('0') }
		}
		hash := sb.String()
		if hash == prevHash {
			continue
		}
		prevHash = hash

		schedules := make([]*antv1.StrategySchedule, len(rows))
		for i, r := range rows {
			schedules[i] = scheduleRowToProto(&r)
		}

		if err := stream.Send(&antv1.WatchSchedulesEvent{Schedules: schedules}); err != nil {
			return err
		}
	}
}
