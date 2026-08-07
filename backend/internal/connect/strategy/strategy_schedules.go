package strategy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
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

	// Reject system templates — user must save as their own first.
	if tpl, err := s.svc.GetTemplate(ctx, templateID, uid); err == nil && tpl.IsSystem {
		return nil, connect.NewError(connect.CodePermissionDenied,
			fmt.Errorf("system preset strategies cannot be scheduled directly — save as your own strategy first"))
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
	if m.AccountId != nil && *m.AccountId != existing.AccountID.String() {
		newAccountID, err := uuid.Parse(*m.AccountId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
		}
		// Verify the new account belongs to the user and is not frozen.
		var status string
		err = s.svc.DB().QueryRow(ctx,
			`SELECT account_status FROM mt_accounts WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
			newAccountID, s.userID(ctx)).Scan(&status)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("account not found or not owned by user"))
		}
		if status == "frozen" {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("target account is frozen"))
		}
		// Stop any running session for the old account before switching.
		if s.engine != nil {
			s.engine.StopSchedule(id)
		}
		existing.AccountID = newAccountID
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
			schedules[i] = scheduleRowToProto(&r)
		}

		if err := stream.Send(&antv1.WatchSchedulesEvent{Schedules: schedules}); err != nil {
			return err
		}
	}
}

// --- Schedule proto converters ---

func scheduleRowToProto(r *service.ScheduleRow) *antv1.StrategySchedule {
	if r == nil {
		return nil
	}
	s := &antv1.StrategySchedule{
		Id:           r.ID.String(),
		UserId:       r.UserID.String(),
		AccountId:    r.AccountID.String(),
		Name:         r.Name,
		Symbol:       r.Symbol,
		Timeframe:    r.Timeframe,
		ScheduleType: r.ScheduleType,
		IsActive:     r.IsActive,
		RunCount:     r.RunCount,
		LastError:    r.LastError,
		EnableCount:  r.EnableCount,
		CreatedAt:    timestamppb.New(r.CreatedAt),
		UpdatedAt:    timestamppb.New(r.UpdatedAt),
	}
	if r.TemplateID != uuid.Nil {
		s.TemplateId = r.TemplateID.String()
	}
	if r.RiskScore != nil {
		s.RiskScore = *r.RiskScore
	}
	if r.RiskLevel != "" {
		s.RiskLevel = r.RiskLevel
	}
	if len(r.Parameters) > 0 {
		var params antv1.StrategyParams
		if err := proto.Unmarshal(r.Parameters, &params); err == nil {
			s.Parameters = params.Values
		}
	}
	if len(r.ScheduleConfig) > 0 {
		var cfg antv1.ScheduleConfig
		if err := proto.Unmarshal(r.ScheduleConfig, &cfg); err == nil {
			s.ScheduleConfig = &cfg
		}
	}
	if len(r.BacktestMetrics) > 0 {
		var metrics antv1.BacktestMetrics
		if err := proto.Unmarshal(r.BacktestMetrics, &metrics); err == nil {
			s.BacktestMetrics = &metrics
		}
	}
	if len(r.RiskReasons) > 0 {
		var risk antv1.BacktestRisk
		if err := proto.Unmarshal(r.RiskReasons, &risk); err == nil {
			s.RiskReasons = risk.Reasons
		}
	}
	if len(r.RiskWarnings) > 0 {
		var risk antv1.BacktestRisk
		if err := proto.Unmarshal(r.RiskWarnings, &risk); err == nil {
			s.RiskWarnings = risk.Warnings
		}
	}
	if r.LastRunAt != nil {
		s.LastRunAt = timestamppb.New(*r.LastRunAt)
	}
	if r.NextRunAt != nil {
		s.NextRunAt = timestamppb.New(*r.NextRunAt)
	}
	return s
}

func scheduleParamsToProto(params map[string]string) []byte {
	data, _ := proto.Marshal(&antv1.StrategyParams{Values: params})
	return data
}

func stringListToProto(list []string) []byte {
	data, _ := proto.Marshal(&antv1.BacktestRisk{Reasons: list})
	return data
}
