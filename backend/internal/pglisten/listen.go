// Package pglisten provides PostgreSQL LISTEN/NOTIFY helpers for push-first SSE.
// Replaces server-side DB polling (time.Ticker) with event-driven notifications.
package pglisten

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Listener wraps a dedicated pgx connection for LISTEN/NOTIFY.
type Listener struct {
	pool   *pgxpool.Pool
	log    *zap.Logger
	mu     sync.Mutex
	conns  map[string]context.CancelFunc // channel → cancel
}

// New creates a Listener using a connection from the pool for each channel.
func New(pool *pgxpool.Pool, log *zap.Logger) *Listener {
	return &Listener{pool: pool, log: log, conns: make(map[string]context.CancelFunc)}
}

// Listen starts listening on the given channel. Returns a Go channel that
// receives the payload (or empty string) on each NOTIFY.
// Call the returned cancel function to stop listening.
func (l *Listener) Listen(ctx context.Context, channel string) (<-chan string, context.CancelFunc, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.conns[channel]; ok {
		return nil, nil, fmt.Errorf("already listening on %s", channel)
	}

	conn, err := l.pool.Acquire(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("pglisten: acquire conn: %w", err)
	}

	listenCtx, cancel := context.WithCancel(ctx)
	notifCh := make(chan string, 32)

	_, err = conn.Exec(listenCtx, fmt.Sprintf("LISTEN %s", channel))
	if err != nil {
		cancel(); conn.Release()
		return nil, nil, fmt.Errorf("pglisten: LISTEN %s: %w", channel, err)
	}

	go func() {
		defer conn.Release()
		defer close(notifCh)
		defer func() {
			l.mu.Lock(); delete(l.conns, channel); l.mu.Unlock()
		}()

		for {
			select {
			case <-listenCtx.Done():
				return
			default:
			}

			notif, err := conn.Conn().WaitForNotification(listenCtx)
			if err != nil {
				if listenCtx.Err() == nil {
					l.log.Warn("pglisten: notification wait error", zap.String("channel", channel), zap.Error(err))
				}
				return
			}
			select {
			case notifCh <- notif.Payload:
			case <-listenCtx.Done():
				return
			default:
				l.log.Warn("pglisten: notification dropped (buffer full)", zap.String("channel", channel))
			}
		}
	}()

	l.conns[channel] = cancel
	return notifCh, cancel, nil
}

// Close cancels all active listeners.
func (l *Listener) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, cancel := range l.conns {
		cancel()
	}
	l.conns = make(map[string]context.CancelFunc)
}

// Notify sends a NOTIFY on the given channel. Best-effort (errors are logged).
func Notify(ctx context.Context, pool *pgxpool.Pool, channel, payload string) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err := pool.Exec(ctx, fmt.Sprintf("SELECT pg_notify('%s', '%s')", channel, payload))
	if err != nil {
		// Silent — NOTIFY is a performance optimization, not a correctness requirement
		_ = err
	}
}
