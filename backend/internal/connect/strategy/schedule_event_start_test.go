package strategy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/service"
)

// TestStartSchedule_EventType_EntitlementDenied verifies that an event-type
// schedule with no active entitlement is blocked by the entitlement gate
// and never reaches RunLiveStrategy.
func TestStartSchedule_EventType_EntitlementDenied(t *testing.T) {
	schedule := makeTestSchedule(model.ScheduleTypeEvent, true)

	repo := &mockScheduleRepo{
		getByID: func(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
			return schedule, nil
		},
		updateLastRun: func(ctx context.Context, id uuid.UUID, runErr error) error {
			if runErr == nil {
				t.Error("expected non-nil runErr in UpdateLastRun")
			}
			return nil
		},
	}

	tplReader := &mockTemplateReader{
		getTemplate: func(ctx context.Context, id, userID uuid.UUID) (*service.TemplateRow, error) {
			t.Error("template reader should not be called when entitlement denied")
			return nil, nil
		},
	}

	engine := &ScheduleEngine{
		repo:           repo,
		templateReader: tplReader,
		activeRuns:     make(map[uuid.UUID]*runHandle),
		notifyCh:       make(chan struct{}, 1),
		log:            zap.NewNop(),
		entitlementCheck: func(ctx context.Context, userID, strategyID string) bool {
			return false // denied
		},
	}

	err := engine.StartSchedule(context.Background(), schedule.ID)
	if err == nil {
		t.Fatal("expected error for denied entitlement")
	}
}

// TestStartSchedule_EventType_EntitlementGranted verifies that an event-type
// schedule with valid entitlement proceeds to template loading.
// The runner is nil so runOne will record an error, but the gates should pass.

// TestStartSchedule_EventType_EntitlementGranted verifies that an event-type
// schedule with valid entitlement proceeds to template loading.
// The runner is nil so runOne will record an error, but the gates should pass.
func TestStartSchedule_EventType_EntitlementGranted(t *testing.T) {
	schedule := makeTestSchedule(model.ScheduleTypeEvent, true)
	ownerID := schedule.UserID

	tplCalled := false
	tplReader := &mockTemplateReader{
		getTemplate: func(ctx context.Context, id, userID uuid.UUID) (*service.TemplateRow, error) {
			tplCalled = true
			return &service.TemplateRow{
				ID:     schedule.TemplateID,
				UserID: &ownerID,
				Code:   "// valid strategy code",
			}, nil
		},
	}

	repo := &mockScheduleRepo{
		getByID: func(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
			return schedule, nil
		},
	}

	engine := &ScheduleEngine{
		repo:           repo,
		templateReader: tplReader,
		activeRuns:     make(map[uuid.UUID]*runHandle),
		notifyCh:       make(chan struct{}, 1),
		log:            zap.NewNop(),
		entitlementCheck: func(ctx context.Context, userID, strategyID string) bool {
			return true // granted
		},
	}

	// runner is nil → launchEventSession will pass gates, load template,
	// but skip quota check (e.runner == nil) and skip run record creation.
	// runOne will record "strategy runner not configured" error.
	_ = engine.StartSchedule(context.Background(), schedule.ID)

	// Wait for goroutine to finish
	time.Sleep(100 * time.Millisecond)

	if !tplCalled {
		t.Error("template reader should have been called when entitlement granted")
	}
}

// TestStartSchedule_TimerType_NotifiesEngine verifies that a timer-type
// schedule does not launch an event session but notifies the timer loop.

// TestStartSchedule_TimerType_NotifiesEngine verifies that a timer-type
// schedule does not launch an event session but notifies the timer loop.
func TestStartSchedule_TimerType_NotifiesEngine(t *testing.T) {
	schedule := makeTestSchedule(model.ScheduleTypeCron, true)

	repo := &mockScheduleRepo{
		getByID: func(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
			return schedule, nil
		},
	}

	engine := &ScheduleEngine{
		repo:       repo,
		activeRuns: make(map[uuid.UUID]*runHandle),
		notifyCh:   make(chan struct{}, 1),
		log:        zap.NewNop(),
	}

	err := engine.StartSchedule(context.Background(), schedule.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify notify was sent
	select {
	case <-engine.notifyCh:
		// good
	case <-time.After(100 * time.Millisecond):
		t.Error("expected notifyCh signal for timer-type schedule")
	}
}

// TestStartSchedule_InactiveSchedule verifies that an inactive schedule
// is rejected.

// TestStartSchedule_InactiveSchedule verifies that an inactive schedule
// is rejected.
func TestStartSchedule_InactiveSchedule(t *testing.T) {
	schedule := makeTestSchedule(model.ScheduleTypeEvent, false)

	repo := &mockScheduleRepo{
		getByID: func(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
			return schedule, nil
		},
	}

	engine := &ScheduleEngine{
		repo:       repo,
		activeRuns: make(map[uuid.UUID]*runHandle),
		notifyCh:   make(chan struct{}, 1),
		log:        zap.NewNop(),
	}

	err := engine.StartSchedule(context.Background(), schedule.ID)
	if err == nil {
		t.Fatal("expected error for inactive schedule")
	}
}

// TestStartSchedule_AlreadyRunning verifies that starting an already-running
// schedule is a no-op.

// TestStartSchedule_AlreadyRunning verifies that starting an already-running
// schedule is a no-op.
func TestStartSchedule_AlreadyRunning(t *testing.T) {
	schedule := makeTestSchedule(model.ScheduleTypeEvent, true)

	repo := &mockScheduleRepo{
		getByID: func(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
			return schedule, nil
		},
	}

	engine := &ScheduleEngine{
		repo:       repo,
		activeRuns: make(map[uuid.UUID]*runHandle),
		notifyCh:   make(chan struct{}, 1),
		log:        zap.NewNop(),
	}

	// Simulate already running
	engine.activeRuns[schedule.ID] = &runHandle{
		cancel: func() {},
	}

	err := engine.StartSchedule(context.Background(), schedule.ID)
	if err != nil {
		t.Fatalf("expected nil error for already-running schedule, got: %v", err)
	}
}

// TestLaunchEventSession_OwnerSkipsEntitlementRevalidation verifies that
// the per-bar EntitlementCheck is nil for owner's own strategies (no
// revalidation needed) and non-nil for marketplace strategies.

// TestStartSchedule_NotFound verifies that a non-existent schedule
// returns an error.
func TestStartSchedule_NotFound(t *testing.T) {
	repo := &mockScheduleRepo{
		getByID: func(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
			return nil, errors.New("not found")
		},
	}

	engine := &ScheduleEngine{
		repo:       repo,
		activeRuns: make(map[uuid.UUID]*runHandle),
		notifyCh:   make(chan struct{}, 1),
		log:        zap.NewNop(),
	}

	err := engine.StartSchedule(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent schedule")
	}
}

// DEPLOY-LIVE-6: Verify dispatch path shares the same entitlement gate as
// launchEventSession via buildLiveRun. If the common function is bypassed,
// this test fails — proving both paths use buildLiveRun.
