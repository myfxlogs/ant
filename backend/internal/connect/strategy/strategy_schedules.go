package strategy

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/connect/ai"
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

// applyScheduleUpdates mutates existing with all fields from the request.
// Returns true if a substantive field (Symbol/Timeframe/Parameters/AccountID) changed.
func (s *StrategyServer) applyScheduleUpdates(ctx context.Context, id uuid.UUID, m *antv1.UpdateScheduleRequest, existing *service.ScheduleRow) (bool, error) {
	substantiveChanged := false
	if m.Name != nil {
		existing.Name = *m.Name
	}
	if m.Symbol != nil {
		if existing.Symbol != *m.Symbol {
			substantiveChanged = true
		}
		existing.Symbol = *m.Symbol
	}
	if m.Timeframe != nil {
		if existing.Timeframe != *m.Timeframe {
			substantiveChanged = true
		}
		existing.Timeframe = *m.Timeframe
	}
	if m.Parameters != nil {
		substantiveChanged = true
		existing.Parameters = scheduleParamsToProto(m.Parameters)
	}
	if m.ScheduleType != nil {
		existing.ScheduleType = *m.ScheduleType
	}
	if m.ScheduleConfig != nil {
		if b, err := proto.Marshal(m.ScheduleConfig); err == nil {
			existing.ScheduleConfig = b
		} else {
			return false, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal schedule config: %w", err))
		}
	}
	if m.AccountId != nil && *m.AccountId != existing.AccountID.String() {
		substantiveChanged = true
		if err := s.applyAccountSwitch(ctx, id, *m.AccountId, existing); err != nil {
			return false, err
		}
	}
	return substantiveChanged, nil
}

// maybeRestartSchedule stops and restarts a running session if substantive fields changed.
// Falls back to Notify for non-substantive changes or non-running schedules.
func (s *StrategyServer) maybeRestartSchedule(ctx context.Context, id uuid.UUID, substantiveChanged bool) {
	if s.engine == nil {
		return
	}
	if substantiveChanged && s.engine.isRunning(id) {
		s.engine.StopSchedule(id)
		if err := s.engine.StartSchedule(ctx, id); err != nil {
			if s.log != nil {
				s.log.Warn("UpdateSchedule: restart failed (params persisted, next start will pick up)",
					zap.String("scheduleId", id.String()), zap.Error(err))
			}
		}
	} else {
		s.engine.Notify()
	}
}

func (s *StrategyServer) applyAccountSwitch(ctx context.Context, id uuid.UUID, accountIDStr string, existing *service.ScheduleRow) error {
	newAccountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid account_id: %w", err))
	}
	var status string
	err = s.svc.DB().QueryRow(ctx,
		`SELECT account_status FROM mt_accounts WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		newAccountID, s.userID(ctx)).Scan(&status)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("account not found or not owned by user"))
	}
	if status == "frozen" {
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("target account is frozen"))
	}
	if s.boundSvc != nil {
		if err := s.boundSvc.EnsureBoundAccount(ctx, s.userID(ctx), newAccountID); err != nil {
			if errors.Is(err, service.ErrAccountLimitExceeded) {
				return connect.NewError(connect.CodePermissionDenied, err)
			}
			if errors.Is(err, service.ErrAccountNotOwned) {
				return connect.NewError(connect.CodeNotFound, err)
			}
			return connect.NewError(connect.CodeInternal, err)
		}
	}
	if s.engine != nil {
		s.engine.StopSchedule(id)
	}
	existing.AccountID = newAccountID
	return nil
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

// legacyDeadKeys are the 5 historical risk-parameter keys with zero consumers.
// They are silently stripped (not rejected) for backward compatibility.
var legacyDeadKeys = map[string]bool{
	"__risk.default_volume":           true,
	"__risk.max_positions":            true,
	"__risk.stop_loss_price_offset":   true,
	"__risk.take_profit_price_offset": true,
	"__risk.max_drawdown_pct":         true,
}

// validateScheduleParams validates incoming parameters against the template's
// declared input schema. REUSE: ai.ExtractParams (no regex duplication).
// - Unknown keys → 400 with key name in error message
// - Type mismatch (e.g. "abc" for int) → 400
// - Legacy dead keys → silently stripped (not rejected, not persisted)
// - Template not found → skip validation (degrade to allow, log.Warn)
func (s *StrategyServer) validateScheduleParams(ctx context.Context, templateID uuid.UUID, params map[string]string) error {
	tpl, err := s.svc.GetTemplate(ctx, templateID, s.userID(ctx))
	if err != nil {
		if s.log != nil {
			s.log.Warn("UpdateSchedule: template not found, skipping param validation",
				zap.String("templateId", templateID.String()))
		}
		return nil
	}
	declared := ai.ExtractParams(tpl.Code)
	if len(declared) == 0 {
		return nil
	}
	declaredMap := make(map[string]string, len(declared))
	for _, e := range declared {
		declaredMap[e.Name] = e.Type
	}
	var unknown []string
	for key, val := range params {
		if legacyDeadKeys[key] || strings.HasPrefix(key, "__schedule.") {
			continue
		}
		typ, ok := declaredMap[key]
		if !ok {
			unknown = append(unknown, key)
			continue
		}
		if err := validateParamType(typ, val); err != nil {
			return connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("parameter %q: %w", key, err))
		}
	}
	if len(unknown) > 0 {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unknown parameter(s): %s", strings.Join(unknown, ", ")))
	}
	// Strip legacy dead keys so they don't persist.
	for key := range params {
		if legacyDeadKeys[key] {
			delete(params, key)
		}
	}
	return nil
}

func validateParamType(typ, val string) error {
	switch typ {
	case "int":
		if _, err := strconv.ParseInt(val, 10, 64); err != nil {
			return fmt.Errorf("expected integer, got %q", val)
		}
	case "float":
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			return fmt.Errorf("expected number, got %q", val)
		}
	case "bool":
		if val != "true" && val != "false" {
			return fmt.Errorf("expected true/false, got %q", val)
		}
	case "str":
		// any string is valid
	}
	return nil
}
