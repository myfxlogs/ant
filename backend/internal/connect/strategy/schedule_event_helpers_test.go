package strategy

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"alphaforge/internal/model"
	"alphaforge/internal/service"
)

type mockScheduleRepo struct {
	getByID            func(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error)
	getActiveSchedules func(ctx context.Context) ([]*model.StrategySchedule, error)
	updateLastRun      func(ctx context.Context, id uuid.UUID, runErr error) error
	updateNextRunAt    func(ctx context.Context, id uuid.UUID, next time.Time) error
	clearNextRunAt     func(ctx context.Context, id uuid.UUID) error
}

func (m *mockScheduleRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
	if m.getByID != nil {
		return m.getByID(ctx, id)
	}
	return nil, errors.New("not found")
}

func (m *mockScheduleRepo) GetActiveSchedules(ctx context.Context) ([]*model.StrategySchedule, error) {
	if m.getActiveSchedules != nil {
		return m.getActiveSchedules(ctx)
	}
	return nil, nil
}

func (m *mockScheduleRepo) GetDueSchedules(ctx context.Context, now time.Time) ([]*model.StrategySchedule, error) {
	return nil, nil
}

func (m *mockScheduleRepo) GetEarliestNextRunAt(ctx context.Context) (time.Time, error) {
	return time.Time{}, nil
}

func (m *mockScheduleRepo) UpdateLastRun(ctx context.Context, id uuid.UUID, runErr error) error {
	if m.updateLastRun != nil {
		return m.updateLastRun(ctx, id, runErr)
	}
	return nil
}

func (m *mockScheduleRepo) UpdateNextRunAt(ctx context.Context, id uuid.UUID, next time.Time) error {
	if m.updateNextRunAt != nil {
		return m.updateNextRunAt(ctx, id, next)
	}
	return nil
}

func (m *mockScheduleRepo) ClearNextRunAt(ctx context.Context, id uuid.UUID) error {
	if m.clearNextRunAt != nil {
		return m.clearNextRunAt(ctx, id)
	}
	return nil
}

func (m *mockScheduleRepo) ClearEventNextRunAt(ctx context.Context) (int, error) {
	return 0, nil
}

type mockTemplateReader struct {
	getTemplate func(ctx context.Context, id, userID uuid.UUID) (*service.TemplateRow, error)
}

func (m *mockTemplateReader) GetTemplate(ctx context.Context, id, userID uuid.UUID) (*service.TemplateRow, error) {
	if m.getTemplate != nil {
		return m.getTemplate(ctx, id, userID)
	}
	return nil, errors.New("not found")
}

// --- Helpers ---

func makeTestSchedule(scheduleType string, active bool) *model.StrategySchedule {
	return &model.StrategySchedule{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		TemplateID:   uuid.New(),
		AccountID:    uuid.New(),
		Symbol:       "EURUSD",
		Timeframe:    "1h",
		ScheduleType: scheduleType,
		IsActive:     active,
	}
}

// --- Tests ---

// TestStartSchedule_EventType_EntitlementDenied verifies that an event-type
// schedule with no active entitlement is blocked by the entitlement gate
// and never reaches RunLiveStrategy.
