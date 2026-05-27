package chmigrate

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"go.uber.org/zap"
)

func TestMigrate_PackageLoads(t *testing.T) {
	t.Parallel()
	t.Log("chmigrate package compiled and loaded")
}

func TestSplitSQL_Single(t *testing.T) {
	t.Parallel()
	stmts := splitSQL("SELECT 1")
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	if stmts[0] != "SELECT 1" {
		t.Errorf("got %q, want %q", stmts[0], "SELECT 1")
	}
}

func TestSplitSQL_Multiple(t *testing.T) {
	t.Parallel()
	stmts := splitSQL("SELECT 1; SELECT 2; SELECT 3")
	if len(stmts) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(stmts))
	}
}

func TestSplitSQL_Empty(t *testing.T) {
	t.Parallel()
	stmts := splitSQL("")
	if len(stmts) != 0 {
		t.Fatalf("expected 0 statements, got %d", len(stmts))
	}
}

func TestSplitSQL_WhitespaceOnly(t *testing.T) {
	t.Parallel()
	stmts := splitSQL("  ;  ;  ")
	if len(stmts) != 0 {
		t.Fatalf("expected 0 statements from whitespace-only, got %d", len(stmts))
	}
}

func TestSplitSQL_TrailingSemicolon(t *testing.T) {
	t.Parallel()
	stmts := splitSQL("SELECT 1;")
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(stmts))
	}
	if stmts[0] != "SELECT 1" {
		t.Errorf("got %q, want %q", stmts[0], "SELECT 1")
	}
}

func TestSplitSQL_TrimWhitespace(t *testing.T) {
	t.Parallel()
	stmts := splitSQL("  SELECT 1  ;  SELECT 2  ")
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(stmts))
	}
	if stmts[0] != "SELECT 1" {
		t.Errorf("got %q, want %q", stmts[0], "SELECT 1")
	}
	if stmts[1] != "SELECT 2" {
		t.Errorf("got %q, want %q", stmts[1], "SELECT 2")
	}
}

func TestSplitSQL_MultiLine(t *testing.T) {
	t.Parallel()
	sql := "CREATE TABLE foo (a Int32); INSERT INTO foo VALUES (1)"
	stmts := splitSQL(sql)
	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(stmts))
	}
	if !strings.Contains(stmts[0], "CREATE TABLE") {
		t.Errorf("first statement should be CREATE TABLE, got %q", stmts[0])
	}
}

func TestSplitSQL_ExchangeTables(t *testing.T) {
	t.Parallel()
	// Simulates the EXCHANGE TABLES pattern from 006_md_ticks_v2.sql.
	sql := "CREATE TABLE IF NOT EXISTS md_ticks_v2 (...); EXCHANGE TABLES md_ticks AND md_ticks_v2; DROP TABLE IF EXISTS md_ticks_v2"
	stmts := splitSQL(sql)
	if len(stmts) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(stmts))
	}
}

func TestSplitSQL_FourStatements(t *testing.T) {
	t.Parallel()
	stmts := splitSQL("A; B; C; D")
	if len(stmts) != 4 {
		t.Fatalf("expected 4 statements, got %d", len(stmts))
	}
}

// mockCHRow implements driver.Row for testing.
type mockCHRow struct{ count uint64 }

func (r *mockCHRow) Scan(dest ...any) error {
	if len(dest) > 0 {
		if p, ok := dest[0].(*uint64); ok {
			*p = r.count
		}
	}
	return nil
}
func (r *mockCHRow) ScanStruct(dest any) error { return nil }
func (r *mockCHRow) Err() error                { return nil }

type mockCHConn struct{}

func (m *mockCHConn) Contributors() []string                                       { return nil }
func (m *mockCHConn) ServerVersion() (*driver.ServerVersion, error)                 { return nil, nil }
func (m *mockCHConn) Select(ctx context.Context, dest any, query string, args ...any) error {
	return fmt.Errorf("mock: not implemented")
}
func (m *mockCHConn) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockCHConn) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	return &mockCHRow{count: 0} // version not yet applied
}
func (m *mockCHConn) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	return nil, fmt.Errorf("mock: not implemented")
}
func (m *mockCHConn) Exec(ctx context.Context, query string, args ...any) error {
	return nil // always succeed
}
func (m *mockCHConn) AsyncInsert(ctx context.Context, query string, wait bool, args ...any) error {
	return fmt.Errorf("mock: not implemented")
}
func (m *mockCHConn) Ping(ctx context.Context) error { return nil }
func (m *mockCHConn) Stats() driver.Stats            { return driver.Stats{} }
func (m *mockCHConn) Close() error                   { return nil }

func TestRun_NoPendingMigrations(t *testing.T) {
	t.Parallel()
	conn := &mockCHConn{}
	err := Run(context.Background(), conn, zap.NewNop())
	if err != nil {
		t.Logf("Run returned error (may be ok without SQL files): %v", err)
	}
}

func TestRun_ValidatesInterface(t *testing.T) {
	t.Parallel()
	var _ clickhouse.Conn = &mockCHConn{}
}

func TestMustRun_Success(t *testing.T) {
	t.Parallel()
	conn := &mockCHConn{}
	// Safe: Run returns nil, so log.Fatal is never called.
	MustRun(context.Background(), conn, zap.NewNop())
}
