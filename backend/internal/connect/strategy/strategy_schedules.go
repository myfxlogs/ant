package strategy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/pglisten"
	"alphaforge/internal/service"
)

// --- Schedules ---

func (s *StrategyServer) ListSchedules(ctx context.Context, req *connect.Request[antv1.ListSchedulesRequest]) (*connect.Response[antv1.ListSchedulesResponse], error) {
	rows, err := s.svc.ListSchedules(ctx, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	schedules := make([]*antv1.StrategySchedule, len(rows))
	for i, r := range rows {
		schedules[i] = s.scheduleRowToProto(&r)
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
	return connect.NewResponse(s.scheduleRowToProto(row)), nil
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

	// Reject system templates — user must save as their own first.
	if tpl, err := s.svc.GetTemplate(ctx, templateID, uid); err == nil && tpl.IsSystem {
		return nil, connect.NewError(connect.CodePermissionDenied,
			fmt.Errorf("system preset strategies cannot be scheduled directly — save as your own strategy first"))
	}

	// LEAKAGE-1: Enforce tier-based account binding limit.
	if s.boundSvc != nil && accountID != uuid.Nil {
		if err := s.boundSvc.EnsureBoundAccount(ctx, uid, accountID); err != nil {
			if errors.Is(err, service.ErrAccountLimitExceeded) {
				return nil, connect.NewError(connect.CodePermissionDenied, err)
			}
			if errors.Is(err, service.ErrAccountNotOwned) {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	// D2: Validate parameters against template schema (REUSE validateScheduleParams — same as Update).
	if m.Parameters != nil && templateID != uuid.Nil {
		if err := s.validateScheduleParams(ctx, templateID, m.Parameters); err != nil {
			return nil, err
		}
	}

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
	s.notifyScheduleChange(ctx)
	return connect.NewResponse(s.scheduleRowToProto(&r)), nil
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

	// E3: Validate parameters against template schema before saving.
	if m.Parameters != nil && existing.TemplateID != uuid.Nil {
		if err := s.validateScheduleParams(ctx, existing.TemplateID, m.Parameters); err != nil {
			return nil, err
		}
	}

	substantiveChanged, err := s.applyScheduleUpdates(ctx, id, m, existing)
	if err != nil {
		return nil, err
	}
	if err := s.svc.UpdateSchedule(ctx, existing); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// E2: Auto-restart running session if substantive fields changed.
	// Pattern copied from ToggleSchedule — StopSchedule + StartSchedule.
	s.maybeRestartSchedule(ctx, id, substantiveChanged)
	s.notifyScheduleChange(ctx)
	return connect.NewResponse(s.scheduleRowToProto(existing)), nil
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
	s.notifyScheduleChange(ctx)
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
			// Event-type schedules need StartSchedule to launch a streaming session;
			// timer-type schedules just need Notify to recompute the timer.
			if row, err := s.svc.GetSchedule(ctx, id, s.userID(ctx)); err == nil && row.ScheduleType == "event" {
				_ = s.engine.StartSchedule(ctx, id)
			} else {
				s.engine.Notify()
			}
		}
	}
	row, err := s.svc.GetSchedule(ctx, id, s.userID(ctx))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.notifyScheduleChange(ctx)
	return connect.NewResponse(s.scheduleRowToProto(row)), nil
}

func (s *StrategyServer) notifyScheduleChange(ctx context.Context) {
	if s.pgListen != nil {
		pglisten.Notify(ctx, s.svc.DB(), "schedule_change", "")
	}
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
			s.log.Warn("WatchSchedules: list failed", zap.Error(err))
			continue
		}
		// Compute a simple hash to detect changes
		var sb strings.Builder
		for _, r := range rows {
			sb.WriteString(r.ID.String())
			if r.IsActive {
				sb.WriteByte('1')
			} else {
				sb.WriteByte('0')
			}
		}
		hash := sb.String()
		if hash == prevHash {
			continue
		}
		prevHash = hash

		schedules := make([]*antv1.StrategySchedule, len(rows))
		for i, r := range rows {
			schedules[i] = s.scheduleRowToProto(&r)
		}

		if err := stream.Send(&antv1.WatchSchedulesEvent{Schedules: schedules}); err != nil {
			return err
		}
	}
}
