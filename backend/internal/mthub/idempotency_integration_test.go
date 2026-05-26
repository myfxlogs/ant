//go:build integration

package mthub

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

func getTestPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		dsn = "postgres://ant:ant@localhost:5432/ant?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping integration test: pg connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func getTestRedis(t *testing.T) *goredis.Client {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	rc := goredis.NewClient(&goredis.Options{Addr: addr})
	if err := rc.Ping(context.Background()).Err(); err != nil {
		t.Skipf("skipping integration test: redis ping: %v", err)
	}
	t.Cleanup(func() { rc.Close() })
	return rc
}

func TestThreeLayerIdempotency_FirstCall(t *testing.T) {
	pg := getTestPG(t)
	redis := getTestRedis(t)
	g := NewThreeLayerGuard(pg, redis)

	// Clean up after test.
	defer redis.Del(context.Background(), idemKey("acc-int-1", "client-int-1"))

	isDup, _, err := g.CheckAndSet(context.Background(), "acc-int-1", "client-int-1", 0)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if isDup {
		t.Fatal("first call should not be duplicate")
	}
}

func TestThreeLayerIdempotency_DuplicateCall(t *testing.T) {
	pg := getTestPG(t)
	redis := getTestRedis(t)
	g := NewThreeLayerGuard(pg, redis)

	key := idemKey("acc-int-2", "client-int-2")
	defer redis.Del(context.Background(), key)

	// First call succeeds.
	isDup, _, err := g.CheckAndSet(context.Background(), "acc-int-2", "client-int-2", 100)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if isDup {
		t.Fatal("first call should not be duplicate")
	}

	// Second call (same account+client) should be duplicate.
	isDup, existing, err := g.CheckAndSet(context.Background(), "acc-int-2", "client-int-2", 200)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if !isDup {
		t.Fatal("second call should be duplicate")
	}
	if existing != 100 {
		t.Fatalf("existing ticket should be 100, got %d", existing)
	}
}

func TestThreeLayerIdempotency_Confirm(t *testing.T) {
	pg := getTestPG(t)
	redis := getTestRedis(t)
	g := NewThreeLayerGuard(pg, redis)

	key := idemKey("acc-int-3", "client-int-3")
	defer redis.Del(context.Background(), key)

	// CheckAndSet with placeholder ticket.
	_, _, err := g.CheckAndSet(context.Background(), "acc-int-3", "client-int-3", 0)
	if err != nil {
		t.Fatalf("CheckAndSet: %v", err)
	}

	// Confirm with real ticket after broker placement.
	if err := g.Confirm(context.Background(), "acc-int-3", "client-int-3", 999); err != nil {
		t.Fatalf("Confirm: %v", err)
	}

	// Verify ticket was updated in Redis.
	val, err := redis.Get(context.Background(), key).Int64()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 999 {
		t.Fatalf("ticket should be 999 after confirm, got %d", val)
	}
}

func TestThreeLayerIdempotency_TTL_96Hours(t *testing.T) {
	pg := getTestPG(t)
	redis := getTestRedis(t)
	g := NewThreeLayerGuard(pg, redis)

	key := idemKey("acc-int-ttl", "client-int-ttl")
	defer redis.Del(context.Background(), key)

	_, _, err := g.CheckAndSet(context.Background(), "acc-int-ttl", "client-int-ttl", 0)
	if err != nil {
		t.Fatalf("CheckAndSet: %v", err)
	}

	ttl, err := redis.TTL(context.Background(), key).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}
	// Should be close to 96h (allow some drift for execution time).
	if ttl < 95*time.Hour || ttl > 97*time.Hour {
		t.Fatalf("TTL should be ~96h, got %v", ttl)
	}
}

func TestThreeLayerIdempotency_Concurrent(t *testing.T) {
	pg := getTestPG(t)
	redis := getTestRedis(t)
	key := idemKey("acc-int-conc", "client-int-conc")
	defer redis.Del(context.Background(), key)

	var wg sync.WaitGroup
	success := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g := NewThreeLayerGuard(pg, redis)
			isDup, _, err := g.CheckAndSet(context.Background(), "acc-int-conc", "client-int-conc", 0)
			if err != nil {
				t.Logf("concurrent error: %v", err)
				return
			}
			success <- !isDup
		}()
	}
	wg.Wait()
	close(success)

	// Exactly one call should succeed.
	var succeeded int
	for s := range success {
		if s {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("exactly 1 of 10 concurrent calls should succeed, got %d", succeeded)
	}
}

func TestThreeLayerIdempotency_CrossWeekend(t *testing.T) {
	// Simulates the G7 holiday scenario: key must survive from
	// Friday 17:00 EST to Tuesday 09:00 EST (~88h).
	// Our 96h TTL provides sufficient coverage.
	pg := getTestPG(t)
	redis := getTestRedis(t)
	g := NewThreeLayerGuard(pg, redis)

	key := idemKey("acc-int-weekend", "client-int-weekend")
	defer redis.Del(context.Background(), key)

	_, _, err := g.CheckAndSet(context.Background(), "acc-int-weekend", "client-int-weekend", 0)
	if err != nil {
		t.Fatalf("CheckAndSet: %v", err)
	}

	ttl, err := redis.TTL(context.Background(), key).Result()
	if err != nil {
		t.Fatalf("TTL: %v", err)
	}

	// 96h > 88h (Fri 17:00 → Tue 09:00).
	if ttl < 88*time.Hour {
		t.Fatalf("TTL %v insufficient for G7 weekend (needs >88h)", ttl)
	}
}

func TestThreeLayerIdempotency_DifferentAccounts(t *testing.T) {
	pg := getTestPG(t)
	redis := getTestRedis(t)
	g := NewThreeLayerGuard(pg, redis)

	key1 := idemKey("acc-int-a", "client-int-x")
	key2 := idemKey("acc-int-b", "client-int-x")
	defer redis.Del(context.Background(), key1, key2)

	// Same client ID, different accounts — both should succeed.
	_, _, err := g.CheckAndSet(context.Background(), "acc-int-a", "client-int-x", 0)
	if err != nil {
		t.Fatalf("account A: %v", err)
	}
	_, _, err = g.CheckAndSet(context.Background(), "acc-int-b", "client-int-x", 0)
	if err != nil {
		t.Fatalf("account B: %v", err)
	}
}
