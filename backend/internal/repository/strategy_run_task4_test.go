//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func getTestPGPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := "postgres://ant:ant@localhost:5432/ant?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping integration test: pg connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestTask4_CleanupStaleRuns_ExcludeIDs verifies that CleanupStaleRuns
// excludes the given run IDs. Insert 3 running rows, exclude 2 → only
// the 3rd is stopped. Remove the SQL exclusion clause → all 3 stopped → RED.
func TestTask4_CleanupStaleRuns_ExcludeIDs(t *testing.T) {
	pool := getTestPGPool(t)
	repo := NewStrategyRunRepository(pool)
	ctx := context.Background()

	// Insert 3 running rows.
	ids := make([]uuid.UUID, 3)
	for i := range ids {
		ids[i] = uuid.New()
		_, err := pool.Exec(ctx, `
			INSERT INTO strategy_runs (id, user_id, strategy_id, status, started_at, mode)
			VALUES ($1, $2, $3, 'running', NOW(), 'live')
		`, ids[i], uuid.New(), uuid.New())
		if err != nil {
			t.Fatalf("insert row %d: %v", i, err)
		}
	}
	defer func() {
		for _, id := range ids {
			_, _ = pool.Exec(ctx, "DELETE FROM strategy_runs WHERE id = $1", id)
		}
	}()

	// Exclude ids[0] and ids[1], clean up the rest.
	exclude := []uuid.UUID{ids[0], ids[1]}
	n, err := repo.CleanupStaleRuns(ctx, exclude)
	if err != nil {
		t.Fatalf("CleanupStaleRuns: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row cleaned, got %d", n)
	}

	// Verify excluded rows are still running.
	for i := 0; i < 2; i++ {
		var status string
		err := pool.QueryRow(ctx, "SELECT status FROM strategy_runs WHERE id = $1", ids[i]).Scan(&status)
		if err != nil {
			t.Fatalf("query row %d: %v", i, err)
		}
		if status != "running" {
			t.Fatalf("excluded row %d status = %s, want running", i, status)
		}
	}

	// Verify non-excluded row is stopped.
	var status string
	err = pool.QueryRow(ctx, "SELECT status FROM strategy_runs WHERE id = $1", ids[2]).Scan(&status)
	if err != nil {
		t.Fatalf("query row 2: %v", err)
	}
	if status != "stopped" {
		t.Fatalf("non-excluded row status = %s, want stopped", status)
	}
}

// TestTask4_CleanupStaleRuns_EmptyExclude verifies that empty excludeIDs
// cleans up all running rows (backward compatibility).
func TestTask4_CleanupStaleRuns_EmptyExclude(t *testing.T) {
	pool := getTestPGPool(t)
	repo := NewStrategyRunRepository(pool)
	ctx := context.Background()

	id := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO strategy_runs (id, user_id, strategy_id, status, started_at, mode)
		VALUES ($1, $2, $3, 'running', NOW(), 'live')
	`, id, uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM strategy_runs WHERE id = $1", id)
	}()

	n, err := repo.CleanupStaleRuns(ctx, nil)
	if err != nil {
		t.Fatalf("CleanupStaleRuns: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected at least 1 row cleaned, got %d", n)
	}

	var status string
	err = pool.QueryRow(ctx, "SELECT status FROM strategy_runs WHERE id = $1", id).Scan(&status)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "stopped" {
		t.Fatalf("status = %s, want stopped", status)
	}
}

// TestTask4_MarkRunning verifies that MarkRunning restores a stopped row
// to running status. Remove the MarkRunning call → row stays stopped → RED.
func TestTask4_MarkRunning(t *testing.T) {
	pool := getTestPGPool(t)
	repo := NewStrategyRunRepository(pool)
	ctx := context.Background()

	id := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO strategy_runs (id, user_id, strategy_id, status, started_at, mode, stopped_at, error)
		VALUES ($1, $2, $3, 'stopped', NOW(), 'live', NOW(), 'server restarted')
	`, id, uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM strategy_runs WHERE id = $1", id)
	}()

	err = repo.MarkRunning(ctx, id)
	if err != nil {
		t.Fatalf("MarkRunning: %v", err)
	}

	var status, stoppedAt, errMsg *string
	err = pool.QueryRow(ctx, "SELECT status, stopped_at::text, error FROM strategy_runs WHERE id = $1", id).
		Scan(&status, &stoppedAt, &errMsg)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if *status != "running" {
		t.Fatalf("status = %s, want running", *status)
	}
	if stoppedAt != nil {
		t.Fatalf("stopped_at should be NULL, got %s", *stoppedAt)
	}
	if errMsg != nil {
		t.Fatalf("error should be NULL, got %s", *errMsg)
	}
}

// Ensure time import is used (for potential future timing assertions).
var _ = time.Second
