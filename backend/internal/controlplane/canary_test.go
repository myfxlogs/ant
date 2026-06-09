package controlplane

import (
	"testing"
)

func TestCanaryManager_SetAndGet(t *testing.T) {
	t.Parallel()
	m := NewCanaryManager()
	m.Set(CanaryConfig{
		StrategyID:   "strat-1",
		VersionTag:   "v2.0-beta",
		AccountIDs:   []string{"acc-1"},
		DurationDays: 7,
	})
	cfg := m.Get("strat-1")
	if cfg == nil {
		t.Fatal("expected non-nil canary config")
	}
	if cfg.VersionTag != "v2.0-beta" {
		t.Errorf("expected v2.0-beta, got %s", cfg.VersionTag)
	}
	if cfg.StartAt == "" {
		t.Fatal("expected auto-filled StartAt")
	}
}

func TestCanaryManager_GetMissing(t *testing.T) {
	t.Parallel()
	m := NewCanaryManager()
	if m.Get("nonexistent") != nil {
		t.Fatal("expected nil for missing strategy")
	}
}

func TestCanaryManager_List(t *testing.T) {
	t.Parallel()
	m := NewCanaryManager()
	m.Set(CanaryConfig{StrategyID: "a", DurationDays: 3})
	m.Set(CanaryConfig{StrategyID: "b", DurationDays: 5})
	list := m.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 canaries, got %d", len(list))
	}
}

func TestCanaryManager_Remove(t *testing.T) {
	t.Parallel()
	m := NewCanaryManager()
	m.Set(CanaryConfig{StrategyID: "to-remove", DurationDays: 1})
	m.Remove("to-remove")
	if m.Get("to-remove") != nil {
		t.Fatal("expected nil after remove")
	}
}

func TestCanaryManager_Promote(t *testing.T) {
	t.Parallel()
	m := NewCanaryManager()
	m.Set(CanaryConfig{StrategyID: "to-promote", DurationDays: 3})
	m.Promote("to-promote")
	cfg := m.Get("to-promote")
	if !cfg.Promoted {
		t.Fatal("expected canary to be promoted")
	}
}
