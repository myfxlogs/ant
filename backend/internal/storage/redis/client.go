// Package redis provides the Redis storage client for ant v2.
//
// It wraps go-redis/v9 with connection lifecycle management,
// used by IdempotencyGuard, distributed locks, and caching.
package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Config holds connection parameters for Redis.
type Config struct {
	Host               string
	Port               int
	Password           string
	DB                 int
	PoolSize           int
	MinIdleConns       int
	MaxRetries         int
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	PoolTimeout        time.Duration
	IdleTimeout        time.Duration
	IdleCheckFrequency time.Duration
}

// Addr returns the host:port connection string.
func (c Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Client wraps a go-redis client.
type Client struct {
	rc *goredis.Client
}

// Connect establishes a Redis connection and pings to verify.
func Connect(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Addr() == ":" {
		return nil, fmt.Errorf("redis: host is required")
	}

	opts := &goredis.Options{
		Addr:            cfg.Addr(),
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		MaxRetries:      cfg.MaxRetries,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolTimeout:     cfg.PoolTimeout,
		ConnMaxIdleTime: cfg.IdleTimeout,
	}

	if opts.PoolSize == 0 {
		opts.PoolSize = 10
	}
	if opts.DialTimeout == 0 {
		opts.DialTimeout = 5 * time.Second
	}

	rc := goredis.NewClient(opts)

	if err := rc.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	return &Client{rc: rc}, nil
}

// Client returns the underlying go-redis client.
func (c *Client) Client() *goredis.Client {
	return c.rc
}

// Ping checks whether the Redis connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	return c.rc.Ping(ctx).Err()
}

// IsConnected reports whether the Redis connection is alive.
func (c *Client) IsConnected() bool {
	return c.rc != nil && c.rc.Ping(context.Background()).Err() == nil
}

// Close releases the Redis connection pool.
func (c *Client) Close() error {
	if c.rc != nil {
		return c.rc.Close()
	}
	return nil
}
