//go:build integration

package knowledgebase

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/pglisten"
	"alphaforge/tools/mql2go/interp"
)

// TestC1_NotifyCacheRefresh_E2E is the integration test for the real PG
// LISTEN/NOTIFY cache invalidation path. Unlike the unit tests which call
// loadFromDB manually, this test:
//  1. Records a new constant fact (RecordFact → PG INSERT + pg_notify).
//  2. Polls LookupConstant until the cache refreshes via listenLoop (max 5s).
//  3. Asserts the constant is now resolvable — proving the NOTIFY delivery
//     works end-to-end and the cache is not silently stale.
//
// Adversarial: if listenLoop is broken (e.g. wrong channel, dropped notify,
// cache not reloaded), this test times out → red.
func TestC1_NotifyCacheRefresh_E2E(t *testing.T) {
	pool := demandTestPool(t)

	ctx, cancel := context.WithCancel(context.Background())

	s := New(pool, pglisten.New(pool, zap.NewNop()), zap.NewNop())
	if err := s.Start(ctx); err != nil {
		t.Fatalf("kb start: %v", err)
	}
	defer func() {
		cancel()
		pool.Close()
	}()

	testConst := "MY_C1_NOTIFY_" + uuid.New().String()[:8]
	valNum := int32(99)

	// Clean up.
	_, _ = pool.Exec(context.Background(), `DELETE FROM kb_compat_fact WHERE identifier = $1`, testConst)
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM kb_compat_fact WHERE identifier = $1`, testConst)
	}()

	// Record a new constant — this writes to PG + sends pg_notify.
	err := s.RecordFact(context.Background(), FactRecord{
		Identifier:   testConst,
		Kind:         "constant",
		Status:       "supported",
		Severity:     "info",
		ValueNumeric: &valNum,
		Source:       "c1-notify-test",
	})
	if err != nil {
		t.Fatalf("RecordFact: %v", err)
	}

	// Poll LookupConstant until the cache refreshes via LISTEN/NOTIFY.
	// Max wait 5 seconds — NOTIFY delivery is typically <100ms but CI may be slower.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if v, ok := s.LookupConstant(testConst); ok {
			if v.Kind == interp.ValInt && v.Int == 99 {
				t.Logf("C1 NOTIFY e2e: cache refreshed via LISTEN/NOTIFY in <5s — compound interest works end-to-end!")
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("C1 NOTIFY e2e: constant not in cache after 5s — LISTEN/NOTIFY delivery may be broken (silent stale cache)")
}
