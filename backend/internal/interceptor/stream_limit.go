// StreamLimitInterceptor caps concurrent streaming RPC handlers per key
// (G-POST2-1) and exports active/rejected stream metrics (G-POST2-2).
//
// It runs innermost in the withSency chain — after auth — so GetUserID /
// GetClientIP are already populated in the handler context. The previous
// SSEStreamLimitMiddleware could never do this: context values are injected
// by the auth Connect interceptor, which executes after HTTP middleware, so
// its keys collapsed onto "anon" — and its isSSERequest check never matched
// the frontend's application/connect+proto streams anyway.
package interceptor

import (
	"context"
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	connectrpc "connectrpc.com/connect"
)

var (
	// streamActiveGauge tracks in-flight streaming handlers by key type.
	streamActiveGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ant_stream_active_streams",
			Help: "Number of active streaming RPC handlers by key type (user/ip/anon).",
		},
		[]string{"key_type"},
	)

	// streamRejectionsTotal counts streaming handlers rejected by the cap.
	streamRejectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ant_stream_limit_rejections_total",
			Help: "Total streaming RPCs rejected by the per-key concurrency cap.",
		},
		[]string{"key_type"},
	)
)

// NewStreamLimitInterceptor returns a connect.Interceptor capping concurrent
// streaming handlers: userCap per authenticated user, ipCap per client IP
// (and for requests with neither — the shared "anon" key). With ~3-6 streams
// per tab, userCap=10 leaves headroom for two tabs; ipCap=30 covers NAT'd
// unauthenticated clients.
func NewStreamLimitInterceptor(userCap, ipCap int) connectrpc.Interceptor {
	return &streamLimitInterceptor{
		userCap: userCap,
		ipCap:   ipCap,
		active:  make(map[string]int),
	}
}

type streamLimitInterceptor struct {
	userCap int
	ipCap   int

	mu     sync.Mutex
	active map[string]int
}

func (l *streamLimitInterceptor) WrapUnary(next connectrpc.UnaryFunc) connectrpc.UnaryFunc {
	return next
}

func (l *streamLimitInterceptor) WrapStreamingClient(next connectrpc.StreamingClientFunc) connectrpc.StreamingClientFunc {
	return next
}

// WrapStreamingHandler acquires a slot before invoking the handler and
// releases it when the handler returns (= stream ended).
func (l *streamLimitInterceptor) WrapStreamingHandler(next connectrpc.StreamingHandlerFunc) connectrpc.StreamingHandlerFunc {
	return func(ctx context.Context, conn connectrpc.StreamingHandlerConn) error {
		keyType, key, limit := l.keyFor(ctx)
		if !l.acquire(key, limit) {
			streamRejectionsTotal.WithLabelValues(keyType).Inc()
			return connectrpc.NewError(connectrpc.CodeResourceExhausted, errors.New("too many concurrent streams"))
		}
		streamActiveGauge.WithLabelValues(keyType).Inc()
		defer func() {
			streamActiveGauge.WithLabelValues(keyType).Dec()
			l.release(key)
		}()
		return next(ctx, conn)
	}
}

// keyFor resolves the limit key: authenticated user first, then client IP,
// then the shared "anon" bucket. Keys are prefixed so a user id can never
// collide with an IP string in the shared map.
func (l *streamLimitInterceptor) keyFor(ctx context.Context) (keyType, key string, limit int) {
	if uid := GetUserID(ctx); uid != "" {
		return "user", "u:" + uid, l.userCap
	}
	if ip := GetClientIP(ctx); ip != "" {
		return "ip", "i:" + ip, l.ipCap
	}
	return "anon", "i:anon", l.ipCap
}

func (l *streamLimitInterceptor) acquire(key string, limit int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.active[key] >= limit {
		return false
	}
	l.active[key]++
	return true
}

func (l *streamLimitInterceptor) release(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.active[key] > 0 {
		l.active[key]--
		if l.active[key] == 0 {
			delete(l.active, key)
		}
	}
}
