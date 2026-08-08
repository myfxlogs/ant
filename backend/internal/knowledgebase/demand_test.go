package knowledgebase

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/pglisten"
)

// demandTestPool returns a PG pool for testing, skipping if DB is unavailable.
func demandTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://ant:QxhrPqrizFg0iTWNOnabaFvv@localhost:5433/ant?sslmode=disable")
	if err != nil {
		t.Skipf("no database: %v", err)
	}
	return pool
}

// TestRecordDemandSignal_HitCountIncrement is the adversarial proof for K3:
// recording the same builtin for the same user 3 times → hit_count=3.
// Delete the ON CONFLICT DO UPDATE (hit_count + 1) → count stays 1 → test red.
func TestRecordDemandSignal_HitCountIncrement(t *testing.T) {
	pool := demandTestPool(t)
	defer pool.Close()

	s := New(pool, pglisten.New(pool, zap.NewNop()), zap.NewNop())
	builtin := "iCustom_test"
	userID := uuid.New()

	_, _ = pool.Exec(context.Background(), `DELETE FROM kb_demand_signal WHERE builtin_name = $1`, builtin)

	for i := 0; i < 3; i++ {
		if err := s.RecordDemandSignal(context.Background(), builtin, userID); err != nil {
			t.Fatalf("record demand signal %d: %v", i+1, err)
		}
	}

	summary, err := s.GetDemandSummary(context.Background())
	if err != nil {
		t.Fatalf("get demand summary: %v", err)
	}

	found := false
	for _, row := range summary {
		if row.BuiltinName == builtin {
			found = true
			if row.TotalHits != 3 {
				t.Fatalf("expected hit_count=3, got %d — adversarial proof: ON CONFLICT DO UPDATE missing?", row.TotalHits)
			}
			if row.UniqueUsers != 1 {
				t.Fatalf("expected unique_users=1, got %d", row.UniqueUsers)
			}
		}
	}
	if !found {
		t.Fatal("iCustom_test not found in demand summary — recording failed")
	}

	_, _ = pool.Exec(context.Background(), `DELETE FROM kb_demand_signal WHERE builtin_name = $1`, builtin)
}

// TestRecordDemandSignal_UserDedup verifies that different users hitting
// the same builtin are counted separately (user_count dedup).
func TestRecordDemandSignal_UserDedup(t *testing.T) {
	pool := demandTestPool(t)
	defer pool.Close()

	s := New(pool, pglisten.New(pool, zap.NewNop()), zap.NewNop())
	builtin := "ObjectCreate_test"
	user1 := uuid.New()
	user2 := uuid.New()

	_, _ = pool.Exec(context.Background(), `DELETE FROM kb_demand_signal WHERE builtin_name = $1`, builtin)

	_ = s.RecordDemandSignal(context.Background(), builtin, user1)
	_ = s.RecordDemandSignal(context.Background(), builtin, user1)
	_ = s.RecordDemandSignal(context.Background(), builtin, user2)

	summary, err := s.GetDemandSummary(context.Background())
	if err != nil {
		t.Fatalf("get demand summary: %v", err)
	}

	for _, row := range summary {
		if row.BuiltinName == builtin {
			if row.TotalHits != 3 {
				t.Fatalf("expected total_hits=3, got %d", row.TotalHits)
			}
			if row.UniqueUsers != 2 {
				t.Fatalf("expected unique_users=2, got %d", row.UniqueUsers)
			}
			_, _ = pool.Exec(context.Background(), `DELETE FROM kb_demand_signal WHERE builtin_name = $1`, builtin)
			return
		}
	}
	t.Fatal("ObjectCreate_test not found in demand summary")
}
