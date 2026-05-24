// Package nats provides the NATS/JetStream storage client for ant v2.
//
// It wraps nats.go with connection lifecycle management and ensures
// the required JetStream streams exist on startup (idempotent).
package nats

import (
	"context"
	"fmt"
	"time"

	natsgo "github.com/nats-io/nats.go"
)

// Config holds connection parameters for NATS.
type Config struct {
	URL      string // nats://host:port
	CredsFile string // optional credentials file
}

// Client wraps a NATS connection with JetStream access.
type Client struct {
	cfg  Config
	conn *natsgo.Conn
	js   natsgo.JetStreamContext
}

// StreamConfig defines a JetStream stream to ensure on startup.
type StreamConfig struct {
	Name     string
	Subjects []string
	MaxAge   time.Duration
	MaxBytes int64
}

// MDStreams returns the required JetStream stream configs for ant v2 market data.
// See docs/spec/11-mdgateway.md §8, §13.5.
func MDStreams() []StreamConfig {
	return []StreamConfig{
		{
			Name:     "MD_EVENTS",
			Subjects: []string{"md.tick.>", "md.bar.>", "md.factor.>"},
			MaxAge:   24 * time.Hour,
			MaxBytes: 8 * 1024 * 1024 * 1024, // 8 GiB
		},
		{
			Name:     "OMS_EVENTS",
			Subjects: []string{"oms.order.>", "oms.signal.>"},
			MaxAge:   7 * 24 * time.Hour,
			MaxBytes: 5 * 1024 * 1024 * 1024, // 5 GiB
		},
	}
}

// Connect establishes a NATS connection and enables JetStream.
func Connect(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("nats: URL is required")
	}

	opts := []natsgo.Option{
		natsgo.Name("ant-mdgateway"),
		natsgo.Timeout(10 * time.Second),
		natsgo.ReconnectWait(2 * time.Second),
		natsgo.MaxReconnects(-1),
	}

	if cfg.CredsFile != "" {
		opts = append(opts, natsgo.UserCredentials(cfg.CredsFile))
	}

	conn, err := natsgo.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats: connect: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("nats: jetstream: %w", err)
	}

	return &Client{cfg: cfg, conn: conn, js: js}, nil
}

// EnsureStream creates the JetStream stream if it doesn't exist.
// If the stream already exists with matching config, it is a no-op.
// If the stream exists but config differs, it returns an error.
func (c *Client) EnsureStream(ctx context.Context, sc StreamConfig) error {
	stream, err := c.js.StreamInfo(sc.Name)
	if err == nil {
		// Stream exists — verify key fields match
		if stream.Config.MaxAge != sc.MaxAge {
			return fmt.Errorf("nats: stream %s MaxAge mismatch: have %v, want %v",
				sc.Name, stream.Config.MaxAge, sc.MaxAge)
		}
		if stream.Config.MaxBytes != sc.MaxBytes {
			return fmt.Errorf("nats: stream %s MaxBytes mismatch: have %d, want %d",
				sc.Name, stream.Config.MaxBytes, sc.MaxBytes)
		}
		return nil // already correct
	}

	// Stream doesn't exist — create it
	_, err = c.js.AddStream(&natsgo.StreamConfig{
		Name:      sc.Name,
		Subjects:  sc.Subjects,
		Storage:   natsgo.FileStorage,
		Retention: natsgo.LimitsPolicy,
		MaxAge:    sc.MaxAge,
		MaxBytes:  sc.MaxBytes,
		Discard:   natsgo.DiscardOld,
	})
	if err != nil {
		return fmt.Errorf("nats: add stream %s: %w", sc.Name, err)
	}
	return nil
}

// EnsureAllStreams calls EnsureStream for each config in MDStreams().
func (c *Client) EnsureAllStreams(ctx context.Context) error {
	for _, sc := range MDStreams() {
		if err := c.EnsureStream(ctx, sc); err != nil {
			return err
		}
	}
	return nil
}

// JetStream returns the JetStream context for publishing/subscribing.
func (c *Client) JetStream() natsgo.JetStreamContext {
	return c.js
}

// Conn returns the underlying NATS connection.
func (c *Client) Conn() *natsgo.Conn {
	return c.conn
}

// IsConnected reports whether the NATS connection is alive.
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// Close drains and closes the NATS connection.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
