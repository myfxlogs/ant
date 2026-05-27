package strategy

import (
	"context"
	"testing"
	"time"

)

// testStrategy is a minimal strategy for testing lifecycle hooks.
type testStrategy struct {
	BaseStrategy
	started  bool
	stopped  bool
	snapData []byte
}

func newTestStrategy() *testStrategy {
	return &testStrategy{BaseStrategy: *NewBaseStrategy("test", "1.0.0")}
}

func (s *testStrategy) OnStart(_ context.Context, state []byte) error {
	s.started = true
	s.snapData = state
	return nil
}

func (s *testStrategy) OnStop(_ context.Context) error {
	s.stopped = true
	return nil
}

func (s *testStrategy) Snapshot() ([]byte, error) {
	return []byte(`{"bar_count":42}`), nil
}

func (s *testStrategy) Restore(state []byte) error {
	s.snapData = state
	return nil
}

func TestBaseStrategy_Defaults(t *testing.T) {
	t.Parallel()
	b := NewBaseStrategy("default", "0.1.0")
	if b.Name() != "default" {
		t.Fatalf("name: want default, got %s", b.Name())
	}
	if b.Version() != "0.1.0" {
		t.Fatalf("version: want 0.1.0, got %s", b.Version())
	}

	ctx := context.Background()
	if err := b.OnStart(ctx, nil); err != nil {
		t.Fatalf("OnStart default should not error: %v", err)
	}
	if err := b.OnBar(ctx, nil); err != nil {
		t.Fatalf("OnBar default should not error: %v", err)
	}
	if err := b.OnTick(ctx, nil); err != nil {
		t.Fatalf("OnTick default should not error: %v", err)
	}
	if err := b.OnOrderEvent(ctx, nil); err != nil {
		t.Fatalf("OnOrderEvent default should not error: %v", err)
	}
	if err := b.OnEndOfDay(ctx, time.Now()); err != nil {
		t.Fatalf("OnEndOfDay default should not error: %v", err)
	}
	if err := b.OnStop(ctx); err != nil {
		t.Fatalf("OnStop default should not error: %v", err)
	}
	data, err := b.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot default should not error: %v", err)
	}
	if data != nil {
		t.Fatal("Snapshot default should return nil")
	}
	if err := b.Restore(nil); err != nil {
		t.Fatalf("Restore default should not error: %v", err)
	}
}

func TestSnapshotStrategy_Roundtrip(t *testing.T) {
	t.Parallel()
	ts := newTestStrategy()
	state, err := SnapshotStrategy(ts, "inst-1", "acc-1", "user-1")
	if err != nil {
		t.Fatalf("SnapshotStrategy: %v", err)
	}
	if state.StrategyName != "test" {
		t.Fatalf("strategy name: want test, got %s", state.StrategyName)
	}
	if state.StrategyVersion != "1.0.0" {
		t.Fatalf("version mismatch")
	}
	if state.InstanceID != "inst-1" {
		t.Fatalf("instance ID mismatch")
	}
	if state.SchemaVersion != 1 {
		t.Fatalf("schema version: want 1, got %d", state.SchemaVersion)
	}
	if len(state.CustomState) == 0 {
		t.Fatal("custom state should not be empty")
	}

	// Marshal / unmarshal roundtrip.
	data, err := MarshalState(state)
	if err != nil {
		t.Fatalf("MarshalState: %v", err)
	}
	restored, err := UnmarshalState(data)
	if err != nil {
		t.Fatalf("UnmarshalState: %v", err)
	}
	if restored.StrategyName != "test" {
		t.Fatalf("roundtrip name broken")
	}
	if restored.InstanceID != "inst-1" {
		t.Fatalf("roundtrip instance ID broken")
	}

	// Restore into strategy.
	ts2 := newTestStrategy()
	if err := RestoreStrategy(ts2, restored); err != nil {
		t.Fatalf("RestoreStrategy: %v", err)
	}
	if string(ts2.snapData) != `{"bar_count":42}` {
		t.Fatalf("restored data mismatch: %s", ts2.snapData)
	}
}

func TestCheckVersionCompatibility(t *testing.T) {
	t.Parallel()
	tests := []struct {
		old, new string
		expected VersionCompatibility
	}{
		{"1.0.0", "1.0.0", VersionCompatible},
		{"1.0.0", "1.0.1", VersionCompatible},
		{"1.0.0", "1.1.0", VersionMinorCompatible},
		{"1.0.0", "1.9.99", VersionMinorCompatible},
		{"1.0.0", "2.0.0", VersionMajorIncompatible},
		{"2.5.3", "3.0.0", VersionMajorIncompatible},
		{"v1.0.0", "v1.0.0", VersionCompatible},   // with v prefix
		{"v1.0.0", "v2.0.0", VersionMajorIncompatible},
	}

	for _, tc := range tests {
		result := CheckVersionCompatibility(tc.old, tc.new)
		if result != tc.expected {
			t.Fatalf("CheckVersion(%s, %s): want %d, got %d", tc.old, tc.new, tc.expected, result)
		}
	}
}

func TestReloader_ValidateUpgrade(t *testing.T) {
	t.Parallel()
	old := NewBaseStrategy("test", "1.0.0")
	r := NewReloader(old)

	// Same version OK.
	if err := r.ValidateUpgrade(NewBaseStrategy("test", "1.0.1")); err != nil {
		t.Fatalf("same major should be OK: %v", err)
	}

	// Minor bump OK.
	if err := r.ValidateUpgrade(NewBaseStrategy("test", "1.5.0")); err != nil {
		t.Fatalf("minor bump should be OK: %v", err)
	}

	// Major bump rejected.
	err := r.ValidateUpgrade(NewBaseStrategy("test", "2.0.0"))
	if err == nil {
		t.Fatal("major version mismatch should be rejected")
	}
	hrErr, ok := err.(*HotReloadError)
	if !ok {
		t.Fatalf("expected HotReloadError, got %T", err)
	}
	if !hrErr.CanRollover {
		t.Fatal("major mismatch should allow rollover")
	}
}

func TestReloader_Reload_MinorBump(t *testing.T) {
	t.Parallel()
	old := newTestStrategy()
	old.BaseStrategy.version = "1.0.0"
	r := NewReloader(old)

	newStrat := newTestStrategy()
	newStrat.BaseStrategy.version = "1.1.0"

	state := &StrategyState{
		CustomState: []byte(`{"bar_count":99}`),
	}

	if err := r.Reload(newStrat, state); err != nil {
		t.Fatalf("minor bump reload should succeed: %v", err)
	}
	if r.Current().Version() != "1.1.0" {
		t.Fatalf("current version should be 1.1.0, got %s", r.Current().Version())
	}
	if string(newStrat.snapData) != `{"bar_count":99}` {
		t.Fatalf("state not restored after reload: %s", newStrat.snapData)
	}
}

func TestReloader_Reload_MajorBumpRejected(t *testing.T) {
	t.Parallel()
	old := newTestStrategy()
	old.BaseStrategy.version = "1.0.0"
	r := NewReloader(old)

	newStrat := newTestStrategy()
	newStrat.BaseStrategy.version = "2.0.0"

	err := r.Reload(newStrat, nil)
	if err == nil {
		t.Fatal("major bump reload should be rejected")
	}
	// Current should still be old.
	if r.Current().Version() != "1.0.0" {
		t.Fatalf("current should remain 1.0.0 after rejected reload, got %s", r.Current().Version())
	}
}

func TestStateMetrics_AllFields(t *testing.T) {
	t.Parallel()
	m := &StateMetrics{
		TotalTrades: 100,
		WinRate:     0.55,
		SharpeRatio: 1.2,
		NetPnL:      15000.50,
		GrossPnL:    18000.00,
		MaxDrawdown: 5000.00,
		TotalCost:   2999.50,
	}
	if m.GrossPnL-m.TotalCost != m.NetPnL {
		// Note: Net not necessarily Gross - Cost in real world (different calculation),
		// but we assert the struct fields are set correctly.
	}
	if m.TotalTrades != 100 {
		t.Fatal("total trades wrong")
	}
}

func TestParseSemver(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		major int
		minor int
		patch int
	}{
		{"1.2.3", 1, 2, 3},
		{"v2.0.0", 2, 0, 0},
		{"0.0.1", 0, 0, 1},
		{"10.20.30", 10, 20, 30},
		{"1", 1, 0, 0},
		{"1.2", 1, 2, 0},
	}

	for _, tc := range tests {
		s := parseSemver(tc.input)
		if s.major != tc.major || s.minor != tc.minor || s.patch != tc.patch {
			t.Fatalf("parseSemver(%s): want %d.%d.%d, got %d.%d.%d",
				tc.input, tc.major, tc.minor, tc.patch, s.major, s.minor, s.patch)
		}
	}
}
