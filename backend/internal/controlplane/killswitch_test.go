package controlplane

import (
	"testing"
)

func TestKillSwitch_InitiallyDisengaged(t *testing.T) {
	t.Parallel()
	ks := NewKillSwitch()
	if ks.IsEngaged() {
		t.Fatal("kill switch should be disengaged by default")
	}
}

func TestKillSwitch_Engage(t *testing.T) {
	t.Parallel()
	ks := NewKillSwitch()
	ks.Engage("market crash", "admin-1")
	if !ks.IsEngaged() {
		t.Fatal("kill switch should be engaged after Engage()")
	}
	status := ks.Status()
	if !status.Engaged {
		t.Fatal("status should report engaged")
	}
	if status.Reason != "market crash" {
		t.Errorf("expected reason 'market crash', got %q", status.Reason)
	}
	if status.Operator != "admin-1" {
		t.Errorf("expected operator 'admin-1', got %q", status.Operator)
	}
	if status.EngagedAt == "" {
		t.Fatal("expected non-empty engaged_at timestamp")
	}
}

func TestKillSwitch_Disengage(t *testing.T) {
	t.Parallel()
	ks := NewKillSwitch()
	ks.Engage("test", "op")
	ks.Disengage()
	if ks.IsEngaged() {
		t.Fatal("kill switch should be disengaged after Disengage()")
	}
	status := ks.Status()
	if status.Engaged {
		t.Fatal("status should report disengaged")
	}
	if status.Reason != "" {
		t.Errorf("reason should be cleared, got %q", status.Reason)
	}
}

func TestKillSwitch_StatusWhenDisengaged(t *testing.T) {
	t.Parallel()
	ks := NewKillSwitch()
	s := ks.Status()
	if s.Engaged {
		t.Fatal("expected disengaged")
	}
	if s.EngagedAt != "" {
		t.Errorf("expected empty engaged_at when disengaged, got %q", s.EngagedAt)
	}
}
