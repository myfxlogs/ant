package strategy

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/connect/ai"
	"alphaforge/internal/service"
)

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
			s.log.Warn("validateScheduleParams: template not found, skipping param validation",
				zap.String("templateId", templateID.String()))
		}
		return nil
	}
	declared := ai.ExtractParams(tpl.Code)
	if len(declared) == 0 {
		return nil
	}
	cleaned, err := validateParamsAgainstSchema(declared, params)
	if err != nil {
		return err
	}
	// Replace params with cleaned copy (legacy dead keys stripped).
	for k := range params {
		delete(params, k)
	}
	for k, v := range cleaned {
		params[k] = v
	}
	return nil
}

// validateParamsAgainstSchema is a pure function that validates params against
// declared parameter entries. Returns a cleaned copy with legacy dead keys
// stripped (does NOT mutate the input map). Returns error for unknown keys
// or type mismatches.
func validateParamsAgainstSchema(declared []*antv1.ParameterEntry, params map[string]string) (map[string]string, error) {
	declaredMap := make(map[string]string, len(declared))
	for _, e := range declared {
		declaredMap[e.Name] = e.Type
	}
	cleaned := make(map[string]string, len(params))
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
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("parameter %q: %w", key, err))
		}
		cleaned[key] = val
	}
	if len(unknown) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unknown parameter(s): %s", strings.Join(unknown, ", ")))
	}
	return cleaned, nil
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
