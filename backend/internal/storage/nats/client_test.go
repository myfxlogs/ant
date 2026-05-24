package nats_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"anttrader/internal/storage/nats"
)

func testConfig() nats.Config {
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = "nats://127.0.0.1:4222"
	}
	return nats.Config{URL: url}
}

func TestConnect_Success(t *testing.T) {
	client, err := nats.Connect(context.Background(), testConfig())
	require.NoError(t, err)
	defer client.Close()

	assert.True(t, client.IsConnected())
	assert.NotNil(t, client.JetStream())
	assert.NotNil(t, client.Conn())
}

func TestConnect_EmptyURL(t *testing.T) {
	_, err := nats.Connect(context.Background(), nats.Config{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "URL is required")
}

func TestConnect_InvalidURL(t *testing.T) {
	_, err := nats.Connect(context.Background(), nats.Config{
		URL: "nats://nonexistent:9999",
	})
	assert.Error(t, err)
}

func TestEnsureStream_NewStream(t *testing.T) {
	client, err := nats.Connect(context.Background(), testConfig())
	require.NoError(t, err)
	defer client.Close()

	sc := nats.StreamConfig{
		Name:     "_test_stream_new",
		Subjects: []string{"_test.>"},
		MaxAge:   1 * time.Minute,
		MaxBytes: 1024 * 1024,
	}

	err = client.EnsureStream(context.Background(), sc)
	require.NoError(t, err)

	// Delete after test
	require.NoError(t, client.JetStream().DeleteStream("_test_stream_new"))
}

func TestEnsureStream_Idempotent(t *testing.T) {
	client, err := nats.Connect(context.Background(), testConfig())
	require.NoError(t, err)
	defer client.Close()

	sc := nats.StreamConfig{
		Name:     "_test_stream_idem",
		Subjects: []string{"_test_idem.>"},
		MaxAge:   1 * time.Minute,
		MaxBytes: 1024 * 1024,
	}

	err = client.EnsureStream(context.Background(), sc)
	require.NoError(t, err)

	// Second call with same config should succeed (no-op)
	err = client.EnsureStream(context.Background(), sc)
	assert.NoError(t, err)

	require.NoError(t, client.JetStream().DeleteStream("_test_stream_idem"))
}

func TestEnsureStream_ConfigMismatch(t *testing.T) {
	client, err := nats.Connect(context.Background(), testConfig())
	require.NoError(t, err)
	defer client.Close()

	sc := nats.StreamConfig{
		Name:     "_test_stream_mismatch",
		Subjects: []string{"_test_mm.>"},
		MaxAge:   1 * time.Minute,
		MaxBytes: 1024 * 1024,
	}

	err = client.EnsureStream(context.Background(), sc)
	require.NoError(t, err)

	// Same name, different config
	sc2 := nats.StreamConfig{
		Name:     "_test_stream_mismatch",
		Subjects: []string{"_test_mm.>"},
		MaxAge:   2 * time.Minute, // different
		MaxBytes: 1024 * 1024,
	}
	err = client.EnsureStream(context.Background(), sc2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MaxAge mismatch")

	require.NoError(t, client.JetStream().DeleteStream("_test_stream_mismatch"))
}

func TestEnsureAllStreams(t *testing.T) {
	client, err := nats.Connect(context.Background(), testConfig())
	require.NoError(t, err)
	defer client.Close()

	// MDStreams creates MD_EVENTS and OMS_EVENTS
	streams := nats.MDStreams()
	assert.Len(t, streams, 2)
	assert.Equal(t, "MD_EVENTS", streams[0].Name)
	assert.Equal(t, "OMS_EVENTS", streams[1].Name)

	err = client.EnsureAllStreams(context.Background())
	require.NoError(t, err)

	// Verify both exist
	for _, sc := range streams {
		info, err := client.JetStream().StreamInfo(sc.Name)
		require.NoError(t, err, "stream %s should exist", sc.Name)
		assert.Equal(t, sc.MaxAge, info.Config.MaxAge)
		assert.Equal(t, sc.MaxBytes, info.Config.MaxBytes)
	}
}

func TestIsConnected(t *testing.T) {
	client, err := nats.Connect(context.Background(), testConfig())
	require.NoError(t, err)

	assert.True(t, client.IsConnected())

	client.Close()
	assert.False(t, client.IsConnected())
}
