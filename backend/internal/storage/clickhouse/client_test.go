package clickhouse_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"anttrader/internal/storage/clickhouse"
)

func testConfig() clickhouse.Config {
	host := os.Getenv("CH_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("CH_PORT")
	if port == "" {
		port = "9000"
	}
	return clickhouse.Config{
		Addr:     host + ":" + port,
		Database: "ant",
		User:     "default",
		Password: "clickhouse",
	}
}

func TestConnect_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := clickhouse.Connect(ctx, testConfig())
	require.NoError(t, err)
	defer client.Close()

	assert.NotNil(t, client.Conn())
}

func TestConnect_InvalidAddr(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := testConfig()
	cfg.Addr = "nonexistent:9999"
	_, err := clickhouse.Connect(ctx, cfg)
	assert.Error(t, err)
}

func TestConnect_EmptyAddr(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := clickhouse.Connect(ctx, clickhouse.Config{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Addr is required")
}

func TestPing_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := clickhouse.Connect(ctx, testConfig())
	require.NoError(t, err)
	defer client.Close()

	err = client.Ping(ctx)
	assert.NoError(t, err)
}

func TestPing_NotConnected(t *testing.T) {
	client := &clickhouse.Client{}
	err := client.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestPrepareBatch_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := clickhouse.Connect(ctx, testConfig())
	require.NoError(t, err)
	defer client.Close()

	// Create a test table
	conn := client.Conn()
	require.NoError(t, conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS _test_prepare_batch (
			id UInt64,
			name String
		) ENGINE = Memory
	`))
	defer conn.Exec(ctx, "DROP TABLE IF EXISTS _test_prepare_batch")

	batch, err := client.PrepareBatch(ctx, "INSERT INTO _test_prepare_batch (id, name)")
	require.NoError(t, err)

	require.NoError(t, batch.Append(uint64(1), "alpha"))
	require.NoError(t, batch.Append(uint64(2), "beta"))
	require.NoError(t, batch.Send())

	// Verify rows were inserted
	var count uint64
	require.NoError(t, conn.QueryRow(ctx, "SELECT count() FROM _test_prepare_batch").Scan(&count))
	assert.Equal(t, uint64(2), count)
}

func TestClose(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := clickhouse.Connect(ctx, testConfig())
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)

	// Ping after close should fail
	err = client.Ping(ctx)
	assert.Error(t, err)
}
