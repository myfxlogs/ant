package risksvc

import (
	"testing"
)

func TestCapabilityTier_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tier     CapabilityTier
		expected string
	}{
		{Tier0ViewOnly, "Tier0-ViewOnly"},
		{Tier1Paper, "Tier1-Paper"},
		{Tier2LiveLimited, "Tier2-LiveLimited"},
		{Tier3LiveFull, "Tier3-LiveFull"},
		{CapabilityTier(99), "Tier99-Unknown"},
	}
	for _, tc := range tests {
		got := tc.tier.String()
		if got != tc.expected {
			t.Fatalf("tier %d: want %s, got %s", int(tc.tier), tc.expected, got)
		}
	}
}

func TestCapabilityTier_EnumsExist(t *testing.T) {
	t.Parallel()
	tiers := []CapabilityTier{Tier0ViewOnly, Tier1Paper, Tier2LiveLimited, Tier3LiveFull}
	for _, tr := range tiers {
		if int(tr) < 0 || int(tr) > 3 {
			t.Fatalf("tier %d out of range", int(tr))
		}
	}
}

func TestCapability_TierCheck_Tier0Blocked(t *testing.T) {
	t.Parallel()
	c := &Capability{UserID: "u1", Tier: Tier0ViewOnly}
	r := c.TierCheck()
	if r.Allowed {
		t.Fatal("Tier0 should be blocked")
	}
}

func TestCapability_TierCheck_Tier1Allowed(t *testing.T) {
	t.Parallel()
	c := &Capability{UserID: "u1", Tier: Tier1Paper}
	r := c.TierCheck()
	if !r.Allowed {
		t.Fatal("Tier1 should be allowed")
	}
}

func TestCapability_TierCheck_Tier2Allowed(t *testing.T) {
	t.Parallel()
	c := &Capability{UserID: "u1", Tier: Tier2LiveLimited}
	r := c.TierCheck()
	if !r.Allowed {
		t.Fatal("Tier2 should be allowed")
	}
}

func TestCapability_TierCheck_Tier3Allowed(t *testing.T) {
	t.Parallel()
	c := &Capability{UserID: "u1", Tier: Tier3LiveFull}
	r := c.TierCheck()
	if !r.Allowed {
		t.Fatal("Tier3 should be allowed")
	}
}

func TestCapability_TierCheck_KillSwitch(t *testing.T) {
	t.Parallel()
	c := &Capability{UserID: "u1", Tier: Tier3LiveFull, KillSwitchOn: true}
	r := c.TierCheck()
	if r.Allowed {
		t.Fatal("killswitch should block even Tier3")
	}
}

func TestCapability_HasOrderType_Nil(t *testing.T) {
	t.Parallel()
	c := &Capability{UserID: "u1", Tier: Tier3LiveFull}
	if !c.HasOrderType("MARKET") {
		t.Fatal("nil order_types should allow all")
	}
}

func TestCapability_HasOrderType_Explicit(t *testing.T) {
	t.Parallel()
	c := &Capability{UserID: "u1", Tier: Tier2LiveLimited, OrderTypes: []string{"MARKET", "LIMIT"}}
	if !c.HasOrderType("MARKET") {
		t.Fatal("MARKET should be allowed")
	}
	if c.HasOrderType("STOP") {
		t.Fatal("STOP should not be allowed")
	}
}

func TestCapabilityStore_GetDefault(t *testing.T) {
	t.Parallel()
	s := NewCapabilityStore()
	c := s.Get("nonexistent")
	if c.Tier != Tier0ViewOnly {
		t.Fatalf("default tier should be 0 (Tier0ViewOnly, deny-by-default), got %d", c.Tier)
	}
}

func TestCapabilityStore_SetAndGet(t *testing.T) {
	t.Parallel()
	s := NewCapabilityStore()
	s.Set(&Capability{UserID: "u1", Tier: Tier3LiveFull})
	c := s.Get("u1")
	if c.Tier != Tier3LiveFull {
		t.Fatalf("want Tier3, got %d", c.Tier)
	}
}

func TestCapabilityStore_Count(t *testing.T) {
	t.Parallel()
	s := NewCapabilityStore()
	if s.Count() != 0 {
		t.Fatal("new store should be empty")
	}
	s.Set(&Capability{UserID: "u1", Tier: Tier3LiveFull})
	s.Set(&Capability{UserID: "u2", Tier: Tier1Paper})
	if s.Count() != 2 {
		t.Fatalf("want 2, got %d", s.Count())
	}
}
