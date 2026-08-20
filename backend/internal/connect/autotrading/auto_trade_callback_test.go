// auto_trade_callback_test.go — Tests for SCHEDULE-HOTLOOP-1 autoTrade cache
// coherence: ToggleAutoTrade and UpdateGlobalSettings must call the
// onAutoTradeChanged callback after DB success. UpdateGlobalSettings only
// when autoTradeEnabled actually changes.

package autotrading

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/model"
)

// contextWithUserID injects a userID into the context as the auth interceptor would.
func contextWithUserID(uid uuid.UUID) context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey, uid.String())
}

// mockAutoRepo tracks autoTrade changes for callback verification.
type mockAutoRepo struct {
	mu                sync.Mutex
	settings          map[uuid.UUID]*model.GlobalSettings
	updateAutoCount   int
	updateGlobalCount int
}

func newMockAutoRepo() *mockAutoRepo {
	return &mockAutoRepo{settings: make(map[uuid.UUID]*model.GlobalSettings)}
}

func (m *mockAutoRepo) GetGlobalSettingsByUserID(ctx context.Context, userID uuid.UUID) (*model.GlobalSettings, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.settings[userID]; ok {
		return s, nil
	}
	return nil, nil // not found → handler creates defaults
}

func (m *mockAutoRepo) CreateGlobalSettings(ctx context.Context, gs *model.GlobalSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.settings[gs.UserID] = gs
	return nil
}

func (m *mockAutoRepo) UpdateGlobalSettings(ctx context.Context, gs *model.GlobalSettings) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateGlobalCount++
	m.settings[gs.UserID] = gs
	return nil
}

func (m *mockAutoRepo) UpdateAutoTradeEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateAutoCount++
	if s, ok := m.settings[userID]; ok {
		s.AutoTradeEnabled = enabled
	}
	return nil
}

// Stub the remaining methods to satisfy the autoTradeStore interface (not used in these tests).
func (m *mockAutoRepo) GetRiskConfigByAccountID(ctx context.Context, accountID uuid.UUID) (*model.RiskConfig, error) {
	return nil, nil
}

func (m *mockAutoRepo) CreateRiskConfig(ctx context.Context, rc *model.RiskConfig) error {
	return nil
}

func (m *mockAutoRepo) UpdateRiskConfig(ctx context.Context, rc *model.RiskConfig) error {
	return nil
}

func (m *mockAutoRepo) CountActiveSchedules(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockAutoRepo) CountPendingExecutions(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockAutoRepo) CountTodayExecutionsByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockAutoRepo) GetTodayProfitByUser(ctx context.Context, userID uuid.UUID) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (m *mockAutoRepo) GetTradingLogs(ctx context.Context, userID uuid.UUID, params *model.LogListParams) ([]*model.TradingLog, int, error) {
	return nil, 0, nil
}

func (m *mockAutoRepo) GetRecentTradingLogs(ctx context.Context, userID uuid.UUID, limit int) ([]*model.TradingLog, error) {
	return nil, nil
}

// --- Tests ---

// TestSCHEDULE_HOTLOOP_1_ToggleAutoTradeCallsCallback: ToggleAutoTrade must
// call onAutoTradeChanged after DB success. Adversarial: delete callback call → RED.
func TestSCHEDULE_HOTLOOP_1_ToggleAutoTradeCallsCallback(t *testing.T) {
	repo := newMockAutoRepo()
	uid := uuid.New()
	gs := model.NewGlobalSettings(uid)
	gs.AutoTradeEnabled = false
	_ = repo.CreateGlobalSettings(context.Background(), gs)

	server := NewAutoTradingServer(nil, nil, zap.NewNop())
	// Replace autoRepo with our mock (field is unexported, so we construct differently).
	server.autoRepo = repo

	var callbackCalled atomic.Bool
	var callbackUserID uuid.UUID
	var callbackMu sync.Mutex
	server.SetOnAutoTradeChanged(func(userID uuid.UUID) {
		callbackMu.Lock()
		callbackUserID = userID
		callbackCalled.Store(true)
		callbackMu.Unlock()
	})

	// Inject userID into context.
	ctx := contextWithUserID(uid)

	_, err := server.ToggleAutoTrade(ctx, connect.NewRequest(&antv1.ToggleAutoTradeRequest{Enabled: true}))
	if err != nil {
		t.Fatalf("ToggleAutoTrade failed: %v", err)
	}

	if !callbackCalled.Load() {
		t.Fatal("onAutoTradeChanged callback was not called after ToggleAutoTrade")
	}
	if callbackUserID != uid {
		t.Errorf("callback called with wrong userID: got %v, want %v", callbackUserID, uid)
	}
}

// TestSCHEDULE_HOTLOOP_1_UpdateGlobalSettingsCallsCallbackOnChange:
// UpdateGlobalSettings must call callback only when autoTradeEnabled actually
// changes. Adversarial: delete the change-check → callback called even when
// unchanged → RED (or delete callback → not called when changed → RED).
func TestSCHEDULE_HOTLOOP_1_UpdateGlobalSettingsCallsCallbackOnChange(t *testing.T) {
	repo := newMockAutoRepo()
	uid := uuid.New()
	gs := model.NewGlobalSettings(uid)
	gs.AutoTradeEnabled = false
	_ = repo.CreateGlobalSettings(context.Background(), gs)

	server := NewAutoTradingServer(nil, nil, zap.NewNop())
	server.autoRepo = repo

	var callCount atomic.Int32
	server.SetOnAutoTradeChanged(func(userID uuid.UUID) {
		callCount.Add(1)
	})

	ctx := contextWithUserID(uid)

	// First call: autoTrade false → true (changed) → callback should fire.
	_, err := server.UpdateGlobalSettings(ctx, connect.NewRequest(&antv1.UpdateGlobalSettingsRequest{
		AutoTradeEnabled: proto.Bool(true),
	}))
	if err != nil {
		t.Fatalf("first UpdateGlobalSettings failed: %v", err)
	}
	if callCount.Load() != 1 {
		t.Errorf("after autoTrade change: callback called %d times, want 1", callCount.Load())
	}

	// Second call: autoTrade stays true (unchanged) → callback should NOT fire.
	_, err = server.UpdateGlobalSettings(ctx, connect.NewRequest(&antv1.UpdateGlobalSettingsRequest{
		AutoTradeEnabled: proto.Bool(true), // same as current
	}))
	if err != nil {
		t.Fatalf("second UpdateGlobalSettings failed: %v", err)
	}
	if callCount.Load() != 1 {
		t.Errorf("after no autoTrade change: callback called %d times total, want 1 (no new call)", callCount.Load())
	}

	// Third call: autoTrade true → false (changed) → callback should fire again.
	_, err = server.UpdateGlobalSettings(ctx, connect.NewRequest(&antv1.UpdateGlobalSettingsRequest{
		AutoTradeEnabled: proto.Bool(false),
	}))
	if err != nil {
		t.Fatalf("third UpdateGlobalSettings failed: %v", err)
	}
	if callCount.Load() != 2 {
		t.Errorf("after autoTrade change back: callback called %d times total, want 2", callCount.Load())
	}
}
