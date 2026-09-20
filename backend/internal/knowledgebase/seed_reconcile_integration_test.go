//go:build integration

package knowledgebase

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/pglisten"
	"alphaforge/tools/mql2go/interp"
)

// KB-SEED-DRIFT-1: kb_compat_fact seed rows are a snapshot of the Go
// registries taken at first seed; ON CONFLICT DO NOTHING meant later
// registry changes (e.g. VM-ENUM-NUMBERING-1 renumbering) never propagated,
// so the KB-first lookup kept serving stale values to every compile —
// SymbolInfoInteger(SYMBOL_TRADE_MODE) compiled to the dead prop 16 while
// the builtin only knows 22. Seed() must reconcile seed rows (update drifted
// values, prune removed identifiers) while preserving manual overrides.
//
// Adversarial: revert ON CONFLICT DO UPDATE → DO NOTHING, or drop
// pruneSeedRows → these tests go red on a stale/cruft row.

func startReconcileService(t *testing.T, ctx context.Context) *Service {
	t.Helper()
	pool := demandTestPool(t)
	listenCtx, cancel := context.WithCancel(ctx)
	s := New(pool, pglisten.New(pool, zap.NewNop()), zap.NewNop())
	if err := s.Start(listenCtx); err != nil {
		cancel()
		pool.Close()
		t.Fatalf("kb start: %v", err)
	}
	// cancel before Close: the listenLoop holds a dedicated conn; Close blocks
	// on it until the loop exits.
	t.Cleanup(func() { cancel(); pool.Close() })
	return s
}

// TestSeedReconcile_DriftedConstant asserts a seed row whose value_numeric
// drifted from interp.MQLConstants is corrected back by Start().
func TestSeedReconcile_DriftedConstant(t *testing.T) {
	pool := demandTestPool(t)
	defer pool.Close()
	ctx := context.Background()

	want := interp.MQLConstants["SYMBOL_TRADE_MODE"].ToInt()
	if _, err := pool.Exec(ctx,
		`UPDATE kb_compat_fact SET value_numeric = $1
		  WHERE identifier = 'SYMBOL_TRADE_MODE' AND kind = 'constant' AND source = 'seed'`,
		want+100,
	); err != nil {
		t.Fatalf("inject drift: %v", err)
	}

	s := startReconcileService(t, ctx)

	got, ok := s.LookupConstant("SYMBOL_TRADE_MODE")
	if !ok || got.ToInt() != want {
		t.Fatalf("SYMBOL_TRADE_MODE = %v (ok=%v), want %d — seed drift not reconciled", got, ok, want)
	}
	var dbVal int
	if err := pool.QueryRow(ctx,
		`SELECT value_numeric FROM kb_compat_fact
		  WHERE identifier = 'SYMBOL_TRADE_MODE' AND kind = 'constant'`,
	).Scan(&dbVal); err != nil {
		t.Fatalf("readback: %v", err)
	}
	if int32(dbVal) != want {
		t.Fatalf("db value_numeric = %d, want %d", dbVal, want)
	}
}

// TestSeedReconcile_PruneRemovedIdentifier asserts a seed row for an
// identifier absent from every built-in registry is deleted (removed
// constants must not resurrect via the KB-first lookup).
func TestSeedReconcile_PruneRemovedIdentifier(t *testing.T) {
	pool := demandTestPool(t)
	defer pool.Close()
	ctx := context.Background()

	name := fmt.Sprintf("KB_RECON_PRUNE_%s", uuid.NewString()[:8])
	if _, err := pool.Exec(ctx,
		`INSERT INTO kb_compat_fact (identifier, kind, status, severity, value_numeric, source)
		 VALUES ($1, 'constant', 'supported', 'info', 4242, 'seed')`,
		name,
	); err != nil {
		t.Fatalf("insert stale seed row: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM kb_compat_fact WHERE identifier = $1`, name)
	})

	s := startReconcileService(t, ctx)

	if _, ok := s.LookupConstant(name); ok {
		t.Fatalf("stale seed constant %q still resolvable — prune did not delete it", name)
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM kb_compat_fact WHERE identifier = $1`, name,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("stale seed row still in db (count=%d)", n)
	}
}

// TestSeedReconcile_ManualOverridePreserved asserts a manual-sourced row on a
// registry identifier is NOT overwritten by the reconcile — ops overrides are
// a deliberate feature of the KB-first lookup.
func TestSeedReconcile_ManualOverridePreserved(t *testing.T) {
	pool := demandTestPool(t)
	defer pool.Close()
	ctx := context.Background()

	// Replace the seed row with a manual override carrying a foreign value;
	// reconcile must leave it alone. The restore is deferred AFTER the pool
	// close defer above so LIFO runs restore while the pool is still open
	// (t.Cleanup runs after all defers — too late, the pool is closed).
	defer func() {
		c := context.Background()
		_, _ = pool.Exec(c,
			`DELETE FROM kb_compat_fact WHERE identifier = 'SYMBOL_TRADE_MODE' AND kind = 'constant'`)
		_, _ = pool.Exec(c,
			`INSERT INTO kb_compat_fact (identifier, kind, status, severity, value_numeric, source)
			 VALUES ('SYMBOL_TRADE_MODE', 'constant', 'supported', 'info', $1, 'seed')`,
			interp.MQLConstants["SYMBOL_TRADE_MODE"].ToInt())
	}()
	if _, err := pool.Exec(ctx,
		`DELETE FROM kb_compat_fact WHERE identifier = 'SYMBOL_TRADE_MODE' AND kind = 'constant'`,
	); err != nil {
		t.Fatalf("clear seed row: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO kb_compat_fact (identifier, kind, status, severity, value_numeric, source)
		 VALUES ('SYMBOL_TRADE_MODE', 'constant', 'supported', 'info', 9999, 'manual')`,
	); err != nil {
		t.Fatalf("insert manual override: %v", err)
	}

	s := startReconcileService(t, ctx)

	got, ok := s.LookupConstant("SYMBOL_TRADE_MODE")
	if !ok || got.ToInt() != 9999 {
		t.Fatalf("manual override = %v (ok=%v), want 9999 — reconcile clobbered a manual row", got, ok)
	}
}
