package model

import (
	"testing"

	"github.com/google/uuid"
)

func TestStrategyMagicDeterministic(t *testing.T) {
	id := uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	m1 := StrategyMagic(id)
	m2 := StrategyMagic(id)
	if m1 != m2 {
		t.Fatalf("StrategyMagic not deterministic: %d vs %d", m1, m2)
	}
	if m1 == 0 {
		t.Fatal("StrategyMagic of non-zero UUID should not be 0")
	}
}

func TestStrategyMagicNilUUID(t *testing.T) {
	if StrategyMagic(uuid.Nil) != 0 {
		t.Fatal("StrategyMagic of uuid.Nil should return 0")
	}
}

func TestStrategyMagicDifferentUUIDs(t *testing.T) {
	id1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	id2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	m1 := StrategyMagic(id1)
	m2 := StrategyMagic(id2)
	if m1 == m2 {
		t.Fatalf("different UUIDs should produce different magic: %d == %d", m1, m2)
	}
}
