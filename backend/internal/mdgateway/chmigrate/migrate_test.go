package chmigrate_test

import (
	"context"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"anttrader/internal/mdgateway/chmigrate"
)

func connectCH(t *testing.T) clickhouse.Conn {
	t.Helper()
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"},
		Auth: clickhouse.Auth{
			Database: "ant",
			Username: "default",
			Password: "clickhouse",
		},
		DialTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestRun_AllMigrations(t *testing.T) {
	ctx := context.Background()
	conn := connectCH(t)
	log := zap.NewNop()

	err := chmigrate.Run(ctx, conn, log)
	require.NoError(t, err)

	// Verify all expected tables exist
	var count uint64
	require.NoError(t, conn.QueryRow(ctx,
		"SELECT count() FROM system.tables WHERE database='ant' AND name IN ('md_ticks','md_bars','factor_values','signals','_schema_migrations')",
	).Scan(&count))
	assert.Equal(t, uint64(5), count)
}

func TestRun_Idempotent(t *testing.T) {
	ctx := context.Background()
	conn := connectCH(t)
	log := zap.NewNop()

	// First run
	require.NoError(t, chmigrate.Run(ctx, conn, log))

	// Second run — should be idempotent (no error)
	err := chmigrate.Run(ctx, conn, log)
	assert.NoError(t, err, "second run must be idempotent")
}
