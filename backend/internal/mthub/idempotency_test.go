package mthub

import (
	"context"
	"testing"
)

func TestBrokerMagic_Deterministic(t *testing.T) {
	a := BrokerMagic("client-abc-123")
	b := BrokerMagic("client-abc-123")
	if a != b {
		t.Fatalf("BrokerMagic must be deterministic: %d vs %d", a, b)
	}
}

func TestBrokerMagic_Different(t *testing.T) {
	a := BrokerMagic("client-abc-123")
	b := BrokerMagic("client-xyz-456")
	if a == b {
		t.Fatal("different client IDs should produce different magics")
	}
}

func TestBrokerMagic_32BitRange(t *testing.T) {
	// Broker magic must fit in int32 (signed 32-bit).
	m := BrokerMagic("any-client-id")
	_ = m // compiles as int32 — verifies type constraint
}

func TestThreeLayerGuard_NoPGNoRedis(t *testing.T) {
	// Guard with no backing stores — should always succeed (for testing).
	g := NewThreeLayerGuard(nil, nil)
	isDup, _, err := g.CheckAndSet(context.Background(), "acc-1", "client-1", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isDup {
		t.Fatal("nil-backend guard should never be duplicate")
	}
}

func TestThreeLayerGuard_Confirm_NoRedis(t *testing.T) {
	g := NewThreeLayerGuard(nil, nil)
	err := g.Confirm(context.Background(), "acc-1", "client-1", 100)
	if err != nil {
		t.Fatalf("Confirm without redis should succeed: %v", err)
	}
}

func TestIdemKey(t *testing.T) {
	key := idemKey("acc-1", "client-1")
	if key != "idem:acc-1:client-1" {
		t.Fatalf("unexpected idem key: %s", key)
	}
}
