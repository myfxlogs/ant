package strategy

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
	"alphaforge/internal/service"
)

func TestLaunchEventSession_EmptyTemplateCode(t *testing.T) {
	schedule := makeTestSchedule(model.ScheduleTypeEvent, true)

	tplReader := &mockTemplateReader{
		getTemplate: func(ctx context.Context, id, userID uuid.UUID) (*service.TemplateRow, error) {
			return &service.TemplateRow{
				ID:   schedule.TemplateID,
				Code: "", // empty code
			}, nil
		},
	}

	lastRunErr := error(nil)
	repo := &mockScheduleRepo{
		getByID: func(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
			return schedule, nil
		},
		updateLastRun: func(ctx context.Context, id uuid.UUID, runErr error) error {
			lastRunErr = runErr
			return nil
		},
	}

	engine := &ScheduleEngine{
		repo:           repo,
		templateReader: tplReader,
		activeRuns:     make(map[uuid.UUID]*runHandle),
		notifyCh:       make(chan struct{}, 1),
		log:            zap.NewNop(),
		entitlementCheck: func(ctx context.Context, userID, strategyID string) bool {
			return true
		},
	}

	err := engine.StartSchedule(context.Background(), schedule.ID)
	if err == nil {
		t.Fatal("expected error for empty template code")
	}
	if lastRunErr == nil {
		t.Error("expected UpdateLastRun to be called with non-nil error")
	}
}

// TestLaunchEventSession_TemplateFetchError verifies that when the template
// reader returns an error, the launch is rejected.
func TestLaunchEventSession_TemplateFetchError(t *testing.T) {
	schedule := makeTestSchedule(model.ScheduleTypeEvent, true)

	tplReader := &mockTemplateReader{
		getTemplate: func(ctx context.Context, id, userID uuid.UUID) (*service.TemplateRow, error) {
			return nil, errors.New("database connection lost")
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
			return true
		},
	}

	err := engine.StartSchedule(context.Background(), schedule.ID)
	if err == nil {
		t.Fatal("expected error for template fetch failure")
	}
}

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
func TestDispatch_SharesBuildLiveRun_EntitlementDenied(t *testing.T) {
	schedule := makeTestSchedule(model.ScheduleTypeCron, true)

	tplCalled := false
	tplReader := &mockTemplateReader{
		getTemplate: func(ctx context.Context, id, userID uuid.UUID) (*service.TemplateRow, error) {
			tplCalled = true
			return nil, nil
		},
	}

	lastRunErr := error(nil)
	repo := &mockScheduleRepo{
		updateLastRun: func(ctx context.Context, id uuid.UUID, runErr error) error {
			lastRunErr = runErr
			return nil
		},
	}

	engine := &ScheduleEngine{
		repo:           repo,
		templateReader: tplReader,
		activeRuns:     make(map[uuid.UUID]*runHandle),
		notifyCh:       make(chan struct{}, 1),
		log:            zap.NewNop(),
		entitlementCheck: func(ctx context.Context, userID, strategyID string) bool {
			return false
		},
	}

	engine.dispatch(context.Background(), schedule)

	if tplCalled {
		t.Error("template reader should not be called when entitlement denied in dispatch")
	}
	if lastRunErr == nil {
		t.Error("expected UpdateLastRun to be called with entitlement error")
	}
}

// DEPLOY-LIVE-6: Verify dispatch path shares the same template validation
// gate as launchEventSession. Both paths must reject empty template code.
func TestDispatch_SharesBuildLiveRun_EmptyTemplate(t *testing.T) {
	schedule := makeTestSchedule(model.ScheduleTypeCron, true)

	tplReader := &mockTemplateReader{
		getTemplate: func(ctx context.Context, id, userID uuid.UUID) (*service.TemplateRow, error) {
			return &service.TemplateRow{Code: ""}, nil
		},
	}

	lastRunErr := error(nil)
	repo := &mockScheduleRepo{
		updateLastRun: func(ctx context.Context, id uuid.UUID, runErr error) error {
			lastRunErr = runErr
			return nil
		},
	}

	engine := &ScheduleEngine{
		repo:           repo,
		templateReader: tplReader,
		activeRuns:     make(map[uuid.UUID]*runHandle),
		notifyCh:       make(chan struct{}, 1),
		log:            zap.NewNop(),
		entitlementCheck: func(ctx context.Context, userID, strategyID string) bool {
			return true
		},
	}

	engine.dispatch(context.Background(), schedule)

	if lastRunErr == nil {
		t.Error("expected UpdateLastRun to be called with template error")
	}
	if engine.isRunning(schedule.ID) {
		t.Error("schedule should not be running after empty template rejection")
	}
}

// DEPLOY-LIVE-8: Adversarial proof — cancelled handler ctx must NOT cancel runCtx.
// buildLiveRun uses lifecycleCtx (engine ctx) as runCtx parent, not the caller's ctx.
// Revert: context.WithCancel(ctx) instead of lifecycleCtx → runCtx.Done() fires → RED.
func TestBuildLiveRun_CancelledHandlerCtx_RunCtxSurvives(t *testing.T) {
	schedule := makeTestSchedule(model.ScheduleTypeEvent, true)
	ownerID := schedule.UserID

	tplReader := &mockTemplateReader{
		getTemplate: func(ctx context.Context, id, userID uuid.UUID) (*service.TemplateRow, error) {
			return &service.TemplateRow{ID: schedule.TemplateID, UserID: &ownerID, Code: "// ok"}, nil
		},
	}

	repo := &mockScheduleRepo{
		getByID: func(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
			return schedule, nil
		},
	}

	engine := &ScheduleEngine{
		repo:             repo,
		templateReader:   tplReader,
		activeRuns:       make(map[uuid.UUID]*runHandle),
		notifyCh:         make(chan struct{}, 1),
		log:              zap.NewNop(),
		entitlementCheck: func(ctx context.Context, userID, strategyID string) bool { return true },
		lifecycleCtx:     context.Background(), // simulate engine started
	}

	// Use a cancelled ctx as the handler ctx.
	handlerCtx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	_, handle, runCtx, err := engine.buildLiveRun(handlerCtx, schedule, "test")
	if err != nil {
		t.Fatalf("buildLiveRun should succeed even with cancelled handler ctx: %v", err)
	}

	// runCtx must NOT be cancelled — it derives from lifecycleCtx, not handlerCtx.
	select {
	case <-runCtx.Done():
		t.Fatal("runCtx was cancelled when handler ctx cancelled — runCtx uses handler ctx instead of lifecycleCtx (DEPLOY-LIVE-8 regression)")
	default:
		// Good: runCtx is still alive
	}

	// Cleanup: cancel run and remove from activeRuns (runOne not launched
	// since we test buildLiveRun directly, so don't call StopSchedule which
	// waits on wg that will never Done).
	handle.cancel()
	engine.mu.Lock()
	delete(engine.activeRuns, schedule.ID)
	engine.mu.Unlock()
}

// DEPLOY-LIVE-8: nil lifecycleCtx guard — engine never Start()'ed must not panic.
func TestBuildLiveRun_NilLifecycleCtx_NoPanic(t *testing.T) {
	schedule := makeTestSchedule(model.ScheduleTypeEvent, true)
	ownerID := schedule.UserID

	tplReader := &mockTemplateReader{
		getTemplate: func(ctx context.Context, id, userID uuid.UUID) (*service.TemplateRow, error) {
			return &service.TemplateRow{ID: schedule.TemplateID, UserID: &ownerID, Code: "// ok"}, nil
		},
	}

	repo := &mockScheduleRepo{
		getByID: func(ctx context.Context, id uuid.UUID) (*model.StrategySchedule, error) {
			return schedule, nil
		},
	}

	engine := &ScheduleEngine{
		repo:             repo,
		templateReader:   tplReader,
		activeRuns:       make(map[uuid.UUID]*runHandle),
		notifyCh:         make(chan struct{}, 1),
		log:              zap.NewNop(),
		entitlementCheck: func(ctx context.Context, userID, strategyID string) bool { return true },
		// lifecycleCtx deliberately nil — simulate Start() not yet called
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("buildLiveRun panicked with nil lifecycleCtx: %v", r)
		}
	}()

	_, handle, _, err := engine.buildLiveRun(context.Background(), schedule, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Cleanup
	handle.cancel()
	engine.mu.Lock()
	delete(engine.activeRuns, schedule.ID)
	engine.mu.Unlock()
}
