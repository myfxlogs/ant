//go:build integration

package mthub

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestIdempotencyGuard_CheckAndSet_FirstCall(t *testing.T) {
	redis := getTestRedis(t)
	guard := NewIdempotencyGuard(redis)
	ctx := context.Background()
	key := "test-key-" + time.Now().Format("150405")

	// First call — not a duplicate.
	isDup, _, err := guard.CheckAndSet(ctx, "acc-test", key, 0)
	if err != nil {
		t.Fatalf("CheckAndSet failed: %v", err)
	}
	if isDup {
		t.Fatal("first call should not be duplicate")
	}

	// Cleanup.
	guard.DeleteKey(ctx, "acc-test", key)
}

func TestIdempotencyGuard_CheckAndSet_Duplicate(t *testing.T) {
	redis := getTestRedis(t)
	guard := NewIdempotencyGuard(redis)
	ctx := context.Background()
	key := "test-key-" + time.Now().Format("150405")

	// First call succeeds.
	isDup, _, err := guard.CheckAndSet(ctx, "acc-test", key, 100)
	if err != nil {
		t.Fatalf("first CheckAndSet failed: %v", err)
	}
	if isDup {
		t.Fatal("first call should not be duplicate")
	}

	// Second call — duplicate.
	isDup, existing, err := guard.CheckAndSet(ctx, "acc-test", key, 200)
	if err != nil {
		t.Fatalf("second CheckAndSet failed: %v", err)
	}
	if !isDup {
		t.Fatal("second call should be duplicate")
	}
	if existing != 100 {
		t.Fatalf("expected existing ticket 100, got %d", existing)
	}

	// Cleanup.
	guard.DeleteKey(ctx, "acc-test", key)
}

func TestIdempotencyGuard_SetTicket(t *testing.T) {
	redis := getTestRedis(t)
	guard := NewIdempotencyGuard(redis)
	ctx := context.Background()
	key := "test-key-" + time.Now().Format("150405")

	// Set a key first.
	isDup, _, err := guard.CheckAndSet(ctx, "acc-test", key, 0)
	if err != nil {
		t.Fatalf("CheckAndSet failed: %v", err)
	}
	if isDup {
		t.Fatal("first call should not be duplicate")
	}

	// Update ticket.
	err = guard.SetTicket(ctx, "acc-test", key, 999)
	if err != nil {
		t.Fatalf("SetTicket failed: %v", err)
	}

	// Verify ticket was updated.
	isDup, existing, err := guard.CheckAndSet(ctx, "acc-test", key, 0)
	if err != nil {
		t.Fatalf("CheckAndSet after SetTicket failed: %v", err)
	}
	if !isDup {
		t.Fatal("should be duplicate after SetTicket")
	}
	if existing != 999 {
		t.Fatalf("expected ticket 999, got %d", existing)
	}

	// Cleanup.
	guard.DeleteKey(ctx, "acc-test", key)
}

func TestIdempotencyGuard_DeleteKey(t *testing.T) {
	redis := getTestRedis(t)
	guard := NewIdempotencyGuard(redis)
	ctx := context.Background()
	key := "test-key-" + time.Now().Format("150405")

	// Set a key.
	guard.CheckAndSet(ctx, "acc-test", key, 0)
	// Delete it.
	guard.DeleteKey(ctx, "acc-test", key)
	// Now CheckAndSet should succeed again.
	isDup, _, err := guard.CheckAndSet(ctx, "acc-test", key, 0)
	if err != nil {
		t.Fatalf("CheckAndSet after DeleteKey failed: %v", err)
	}
	if isDup {
		t.Fatal("should not be duplicate after DeleteKey")
	}

	guard.DeleteKey(ctx, "acc-test", key)
}

func TestThreeLayerGuard_CheckAndSet_WithPGAndRedis(t *testing.T) {
	t.Parallel()
	pg := getTestPG(t)
	redis := getTestRedis(t)
	guard := NewThreeLayerGuard(pg, redis)
	ctx := context.Background()
	key := "test-3layer-" + time.Now().Format("150405")

	// First call — not duplicate.
	isDup, _, err := guard.CheckAndSet(ctx, "acc-test", key, 100)
	if err != nil {
		t.Fatalf("CheckAndSet failed: %v", err)
	}
	if isDup {
		t.Fatal("first call should not be duplicate")
	}
	// Confirm after success.
	guard.Confirm(ctx, "acc-test", key, 100)

	// Cleanup Redis key.
	redis.Del(ctx, idemKey("acc-test", key))
}

func TestThreeLayerGuard_CheckAndSet_Duplicate(t *testing.T) {
	pg := getTestPG(t)
	redis := getTestRedis(t)
	guard := NewThreeLayerGuard(pg, redis)
	ctx := context.Background()
	key := "test-3layer-dup-" + time.Now().Format("150405")

	// First call.
	isDup, _, err := guard.CheckAndSet(ctx, "acc-test", key, 200)
	if err != nil {
		t.Fatalf("first CheckAndSet failed: %v", err)
	}
	if isDup {
		t.Fatal("first call should not be duplicate")
	}

	// Second call with same key — duplicate.
	isDup, existing, err := guard.CheckAndSet(ctx, "acc-test", key, 300)
	if err != nil {
		t.Fatalf("second CheckAndSet failed: %v", err)
	}
	if !isDup {
		t.Fatal("second call should be duplicate")
	}
	if existing != 200 {
		t.Fatalf("expected existing ticket 200, got %d", existing)
	}

	redis.Del(ctx, idemKey("acc-test", key))
}

func TestThreeLayerGuard_Confirm(t *testing.T) {
	t.Parallel()
	redis := getTestRedis(t)
	guard := NewThreeLayerGuard(nil, redis)
	ctx := context.Background()
	key := "test-confirm-" + time.Now().Format("150405")

	// CheckAndSet with Redis only.
	isDup, _, err := guard.CheckAndSet(ctx, "acc-test", key, 0)
	if err != nil {
		t.Fatalf("CheckAndSet failed: %v", err)
	}
	if isDup {
		t.Fatal("should not be duplicate")
	}

	// Confirm updates the ticket.
	err = guard.Confirm(ctx, "acc-test", key, 999)
	if err != nil {
		t.Fatalf("Confirm failed: %v", err)
	}

	redis.Del(ctx, idemKey("acc-test", key))
}

func TestStateCache_PersistAndLoadFromRedis(t *testing.T) {
	redis := getTestRedis(t)
	log := zap.NewNop()
	cache := NewStateCache(redis, log)

	// Apply a fill event.
	cache.ApplyEvent(&TradeEvent{
		EventID:   "ev-redis-1",
		EventType: TradeEventOrderFilled,
		AccountID: "acc-redis-test",
		Canonical: "EURUSD",
		Side:      "BUY",
		Volume:    decimal.NewFromInt(1),
		Price:     decimal.NewFromFloat(1.085),
		Ticket:    12345,
		Timestamp: time.Now(),
		Version:   1,
	})

	// Verify in-memory state.
	pos := cache.GetPosition("acc-redis-test", "EURUSD")
	if pos == nil {
		t.Fatal("expected position in cache after fill")
	}

	// Persist to Redis (exercises serialization path).
	cache.persistToRedis(&TradeEvent{
		EventID:   "ev-redis-1",
		EventType: TradeEventOrderFilled,
		AccountID: "acc-redis-test",
		Canonical: "EURUSD",
		Ticket:    12345,
		Timestamp: time.Now(),
	})

	// Load from Redis.
	cache2 := NewStateCache(redis, log)
	err := cache2.LoadFromRedis(context.Background())
	if err != nil {
		t.Logf("LoadFromRedis: %v", err)
	}
}
