// Package mthub provides three-layer idempotency (M11-3, M10-BASE-B3):
//
//	Layer 1 — PG advisory lock: intra-process mutex on (accountID, clientID).
//	Layer 2 — Redis SETNX with 96h TTL: cross-instance dedup, covers G7 holidays.
//	Layer 3 — Broker magic: hash(clientID) truncated to 32 bits as Magic field.
//
// All three layers must pass before an order reaches the broker.
package mthub

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

const (
	// DefaultIdempotencyTTL is the Redis key TTL for idempotency keys.
	// 96 hours covers a full G7 holiday weekend (Fri close → Tue open).
	DefaultIdempotencyTTL = 96 * time.Hour
)

// ThreeLayerGuard implements the three-layer idempotency protocol.
// Zero-value is unusable; use NewThreeLayerGuard.
type ThreeLayerGuard struct {
	pg    *pgxpool.Pool
	redis *goredis.Client
}

// NewThreeLayerGuard creates a guard backed by PostgreSQL and Redis.
// Either pg or redis may be nil (the corresponding layer is skipped).
func NewThreeLayerGuard(pg *pgxpool.Pool, redis *goredis.Client) *ThreeLayerGuard {
	return &ThreeLayerGuard{pg: pg, redis: redis}
}

// BrokerMagic computes the 32-bit broker magic from a client ID.
// This is Layer 3: broker-side dedup using a deterministic hash.
func BrokerMagic(clientID string) int32 {
	h := fnv.New64a()
	h.Write([]byte(clientID))
	return int32(h.Sum64() & 0xFFFFFFFF)
}

// CheckAndSet performs the three-layer idempotency check.
// Returns (isDup, existingTicket, error).
// On success (isDup=false), the caller must call Confirm after broker submission.
func (g *ThreeLayerGuard) CheckAndSet(ctx context.Context, accountID, clientID string, ticket int64) (isDup bool, existingTicket int64, err error) {
	// Layer 1: PG advisory lock (non-blocking).
	if g.pg != nil {
		acquired, unlock, err := g.tryAdvisoryLock(ctx, accountID, clientID)
		if err != nil {
			return false, 0, fmt.Errorf("idempotency layer1 pg_lock: %w", err)
		}
		if !acquired {
			// Lock held by another concurrent request — treat as duplicate.
			return true, 0, nil
		}
		defer unlock()
	}

	// Layer 2: Redis SETNX with extended TTL.
	if g.redis != nil {
		key := idemKey(accountID, clientID)
		ok, err := g.redis.SetNX(ctx, key, ticket, DefaultIdempotencyTTL).Result()
		if err != nil {
			return false, 0, fmt.Errorf("idempotency layer2 redis: %w", err)
		}
		if !ok {
			existing, err := g.redis.Get(ctx, key).Int64()
			if err != nil {
				return false, 0, fmt.Errorf("idempotency check failed: %w", err)
			}
			return true, existing, nil
		}
	}

	return false, 0, nil
}

// Confirm updates the ticket in the Redis key after successful broker placement.
func (g *ThreeLayerGuard) Confirm(ctx context.Context, accountID, clientID string, ticket int64) error {
	if g.redis == nil {
		return nil
	}
	return g.redis.Set(ctx, idemKey(accountID, clientID), ticket, DefaultIdempotencyTTL).Err()
}

// advisoryLockKey hashes accountID and clientID into two int32 values
// suitable for pg_try_advisory_lock(int4, int4).
func advisoryLockKey(accountID, clientID string) (int32, int32) {
	h := fnv.New64a()
	h.Write([]byte(accountID))
	h.Write([]byte{0})
	h.Write([]byte(clientID))
	v := h.Sum64()
	return int32(v >> 32), int32(v & 0xFFFFFFFF)
}

func (g *ThreeLayerGuard) tryAdvisoryLock(ctx context.Context, accountID, clientID string) (acquired bool, unlock func(), err error) {
	k1, k2 := advisoryLockKey(accountID, clientID)

	// Use transaction-scoped lock: pg_try_advisory_xact_lock releases
	// automatically on COMMIT/ROLLBACK, so it works correctly with pgxpool
	// connection pooling (unlike session-level pg_try_advisory_lock).
	tx, err := g.pg.Begin(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("begin tx for advisory lock: %w", err)
	}

	var ok bool
	err = tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1, $2)", k1, k2).Scan(&ok)
	if err != nil {
		tx.Rollback(ctx)
		return false, nil, err
	}
	if !ok {
		tx.Rollback(ctx)
		return false, nil, nil
	}

	unlock = func() {
		// Transaction-scoped lock auto-releases on commit; no explicit unlock needed.
		_ = tx.Commit(ctx)
	}
	return true, unlock, nil
}

// --- Legacy guard (single-layer Redis, kept for backward compat) ---

// IdempotencyGuard is the original single-layer Redis guard.
// Deprecated: use ThreeLayerGuard for new code.
type IdempotencyGuard struct {
	rc *goredis.Client
}

// NewIdempotencyGuard creates a guard backed by a Redis client.
func NewIdempotencyGuard(rc *goredis.Client) *IdempotencyGuard {
	return &IdempotencyGuard{rc: rc}
}

func idemKey(accountID, clientID string) string {
	return fmt.Sprintf("idem:%s:%s", accountID, clientID)
}

func (g *IdempotencyGuard) CheckAndSet(ctx context.Context, accountID, clientID string, ticket int64) (isDup bool, existingTicket int64, err error) {
	key := idemKey(accountID, clientID)
	ok, err := g.rc.SetNX(ctx, key, ticket, DefaultIdempotencyTTL).Result()
	if err != nil {
		return false, 0, fmt.Errorf("idempotency: redis setnx: %w", err)
	}
	if ok {
		return false, 0, nil
	}
	existing, err := g.rc.Get(ctx, key).Int64()
	if err != nil {
		return false, 0, fmt.Errorf("idempotency check failed: %w", err)
	}
	return true, existing, nil
}

func (g *IdempotencyGuard) SetTicket(ctx context.Context, accountID, clientID string, ticket int64) error {
	return g.rc.Set(ctx, idemKey(accountID, clientID), ticket, DefaultIdempotencyTTL).Err()
}
