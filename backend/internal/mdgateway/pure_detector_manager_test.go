package mdgateway

import (
	"context"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestManager_Health_Empty(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	if len(mgr.Health()) != 0 {
		t.Error("Health should be empty for new manager")
	}
}

// --- runner.go drain ---

// --- quote_stuffing.go ---

func TestDefaultStuffingDetectorConfig(t *testing.T) {
	t.Parallel()
	cfg := DefaultStuffingDetectorConfig()
	if cfg.ZscoreThreshold != 4.0 {
		t.Errorf("ZscoreThreshold = %f, want 4.0", cfg.ZscoreThreshold)
	}
	if cfg.PauseDuration != 30*time.Second {
		t.Errorf("PauseDuration = %v, want 30s", cfg.PauseDuration)
	}
	if cfg.WindowSize != 50 {
		t.Errorf("WindowSize = %d, want 50", cfg.WindowSize)
	}
}

func TestNewStuffingDetector(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	if sd == nil {
		t.Fatal("NewStuffingDetector returned nil")
	}
}

func TestStuffingDetector_IsPaused_Empty(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	if sd.IsPaused("broker", "EURUSD") {
		t.Error("IsPaused should be false for new detector")
	}
}

func TestStuffingDetector_PausedSymbols_Empty(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	if len(sd.PausedSymbols()) != 0 {
		t.Error("PausedSymbols should be empty for new detector")
	}
}

func TestStuffingDetector_Observe_FirstTick(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	stuffed, z := sd.Observe("broker", "EURUSD")
	if stuffed {
		t.Error("first tick should not trigger stuffing")
	}
	if z != 0 {
		t.Errorf("zscore for first tick should be 0, got %f", z)
	}
}

func TestStuffingDetector_Observe_Multiple(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	for i := 0; i < 20; i++ {
		stuffed, _ := sd.Observe("broker", "EURUSD")
		if stuffed {
			t.Logf("stuffing detected at tick %d", i)
			break
		}
	}
}

func TestRemoveGateway_NotExist(t *testing.T) {
	t.Parallel()
	mgr := NewManager(ManagerDeps{})
	ctx := context.Background()
	err := mgr.RemoveGateway(ctx, "nonexistent")
	if err != nil {
		t.Errorf("RemoveGateway for nonexistent should not error: %v", err)
	}
}

// --- quality.go ---

func TestStuffingDetector_IsPaused_Expired(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	// Manually insert an expired pause entry (white-box).
	sd.mu.Lock()
	sd.pausedUntil["broker:EURUSD"] = time.Now().Add(-time.Hour)
	sd.mu.Unlock()
	if sd.IsPaused("broker", "EURUSD") {
		t.Error("expired pause should return false")
	}
	// Key should be cleaned up.
	sd.mu.Lock()
	_, exists := sd.pausedUntil["broker:EURUSD"]
	sd.mu.Unlock()
	if exists {
		t.Error("expired key should be deleted")
	}
}

func TestStuffingDetector_IsPaused_Active(t *testing.T) {
	t.Parallel()
	sd := NewStuffingDetector(DefaultStuffingDetectorConfig())
	sd.mu.Lock()
	sd.pausedUntil["broker:EURUSD"] = time.Now().Add(time.Hour)
	sd.mu.Unlock()
	if !sd.IsPaused("broker", "EURUSD") {
		t.Error("active pause should return true")
	}
}

// --- dlq_writer.go spillDLQ with spill ---

// --- quality.go check stale ---
