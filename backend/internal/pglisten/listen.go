// Package pglisten provides PostgreSQL LISTEN/NOTIFY helpers for push-first SSE.
// Uses a shared-listener fan-out pattern: one PG connection per channel,
// broadcasting notifications to all SSE subscribers via Go channels.
package pglisten

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// validChannel matches PostgreSQL identifier rules for LISTEN channel names.
var validChannel = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// channelState holds the dedicated PG connection and subscriber set for one channel.
type channelState struct {
	cancel  context.CancelFunc
	subs    map[int]chan string
	nextSub int
}

// Listener wraps a dedicated pgx connection per channel for LISTEN/NOTIFY,
// fanning out notifications to multiple SSE subscribers without per-stream
// connection acquisition.
type Listener struct {
	pool     *pgxpool.Pool
	log      *zap.Logger
	mu       sync.Mutex
	channels map[string]*channelState
}

// New creates a Listener that shares one PG connection per channel across
// all SSE subscribers.
func New(pool *pgxpool.Pool, log *zap.Logger) *Listener {
	return &Listener{pool: pool, log: log, channels: make(map[string]*channelState)}
}

// Listen subscribes to the given channel. Returns a Go channel that receives
// the NOTIFY payload on each notification. Call the returned cancel function
// to unsubscribe.
//
// Multiple subscribers can listen on the same channel; they share one PG
// connection. When the last subscriber cancels, the PG connection is released.
func (l *Listener) Listen(ctx context.Context, channel string) (<-chan string, context.CancelFunc, error) {
	if !validChannel.MatchString(channel) {
		return nil, nil, fmt.Errorf("pglisten: invalid channel name %q", channel)
	}

	l.mu.Lock()
	st, exists := l.channels[channel]
	if !exists {
		st = &channelState{subs: make(map[int]chan string)}
		l.channels[channel] = st

		conn, err := l.pool.Acquire(ctx)
		if err != nil {
			l.mu.Unlock()
			return nil, nil, fmt.Errorf("pglisten: acquire conn: %w", err)
		}

		listenCtx, cancel := context.WithCancel(context.Background())
		_, err = conn.Exec(listenCtx, fmt.Sprintf("LISTEN %s", channel))
		if err != nil {
			l.mu.Unlock()
			cancel()
			conn.Release()
			delete(l.channels, channel)
			return nil, nil, fmt.Errorf("pglisten: LISTEN %s: %w", channel, err)
		}

		st.cancel = cancel
		l.mu.Unlock()

		go l.readLoop(listenCtx, channel, st, conn)
	} else {
		l.mu.Unlock()
	}

	l.mu.Lock()
	subID := st.nextSub
	st.nextSub++
	notifCh := make(chan string, 32)
	st.subs[subID] = notifCh
	l.mu.Unlock()

	cancel := func() {
		l.mu.Lock()
		if st, ok := l.channels[channel]; ok {
			if ch, ok := st.subs[subID]; ok {
				delete(st.subs, subID)
				close(ch)
			}
			if len(st.subs) == 0 {
				st.cancel()
				delete(l.channels, channel)
			}
		}
		l.mu.Unlock()
	}

	return notifCh, cancel, nil
}

// readLoop is the single goroutine per channel that reads PG notifications
// and fans them out to all subscribers.
func (l *Listener) readLoop(ctx context.Context, channel string, st *channelState, conn *pgxpool.Conn) {
	defer conn.Release()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		notif, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() == nil {
				l.log.Warn("pglisten: notification wait error",
					zap.String("channel", channel),
					zap.Error(err),
				)
			}
			return
		}

		l.mu.Lock()
		for _, sub := range st.subs {
			select {
			case sub <- notif.Payload:
			default:
				l.log.Warn("pglisten: notification dropped (buffer full)",
					zap.String("channel", channel),
				)
			}
		}
		l.mu.Unlock()
	}
}

// Close cancels all active listeners and releases all PG connections.
func (l *Listener) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, st := range l.channels {
		st.cancel()
		for _, sub := range st.subs {
			close(sub)
		}
	}
	l.channels = make(map[string]*channelState)
}

// Notify sends a NOTIFY on the given channel. Best-effort (errors are silent).
func Notify(ctx context.Context, pool *pgxpool.Pool, channel, payload string) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err := pool.Exec(ctx, "SELECT pg_notify($1, $2)", channel, payload)
	if err != nil {
		log.Printf("pglisten: notify failed on channel %s: %v", channel, err)
	}
}
