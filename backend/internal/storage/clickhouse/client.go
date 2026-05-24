// Package clickhouse provides the ClickHouse storage client for ant v2.
//
// It wraps the ClickHouse Go driver with connection lifecycle management,
// health checks (Ping), and batch insert support (PrepareBatch).
package clickhouse

import (
	"context"
	"fmt"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Config holds connection parameters for a ClickHouse instance.
type Config struct {
	Addr     string // host:port
	Database string
	User     string
	Password string
}

// Client wraps a ClickHouse connection with lifecycle methods.
type Client struct {
	cfg  Config
	conn ch.Conn
}

// Connect establishes a ClickHouse connection and verifies it with a ping.
// It returns an error if the connection or ping fails.
func Connect(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Addr == "" {
		return nil, fmt.Errorf("clickhouse: Addr is required")
	}
	if cfg.Database == "" {
		cfg.Database = "default"
	}

	conn, err := ch.Open(&ch.Options{
		Addr: []string{cfg.Addr},
		Auth: ch.Auth{
			Database: cfg.Database,
			Username: cfg.User,
			Password: cfg.Password,
		},
		DialTimeout:      10 * time.Second,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: ch.ConnOpenInOrder,
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse: open: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("clickhouse: ping: %w", err)
	}

	return &Client{cfg: cfg, conn: conn}, nil
}

// Ping checks whether the ClickHouse connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	if c.conn == nil {
		return fmt.Errorf("clickhouse: not connected")
	}
	return c.conn.Ping(ctx)
}

// PrepareBatch returns a batch prepared statement for bulk inserts.
// The query should be an INSERT statement with placeholder values.
func (c *Client) PrepareBatch(ctx context.Context, query string) (driver.Batch, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("clickhouse: not connected")
	}
	return c.conn.PrepareBatch(ctx, query)
}

// Conn returns the underlying ClickHouse connection for advanced usage.
func (c *Client) Conn() ch.Conn {
	return c.conn
}

// Close shuts down the ClickHouse connection.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
