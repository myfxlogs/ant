package interceptor

import (
	"net/http"
	"strings"
	"sync"
)

// SSEStreamLimitMiddleware enforces a per-user concurrent SSE stream limit.
// It activates for requests whose Accept or Content-Type header signals
// text/event-stream, and uses the authenticated user ID (or client IP) as key.
func SSEStreamLimitMiddleware(maxStreams int) func(http.Handler) http.Handler {
	if maxStreams <= 0 {
		maxStreams = 5
	}
	lim := &sseStreamLimiter{
		max:    maxStreams,
		active: make(map[string]int),
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isSSERequest(r) {
				next.ServeHTTP(w, r)
				return
			}
			key := sseOwnerKey(r)
			if !lim.acquire(key) {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("too many SSE streams"))
				return
			}
			defer lim.release(key)
			next.ServeHTTP(w, r)
		})
	}
}

type sseStreamLimiter struct {
	max    int
	mu     sync.Mutex
	active map[string]int
}

func (l *sseStreamLimiter) acquire(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.active[key] >= l.max {
		return false
	}
	l.active[key]++
	return true
}

func (l *sseStreamLimiter) release(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.active[key] > 0 {
		l.active[key]--
		if l.active[key] == 0 {
			delete(l.active, key)
		}
	}
}

func isSSERequest(r *http.Request) bool {
	h := strings.ToLower(r.Header.Get("Accept"))
	if strings.Contains(h, "text/event-stream") {
		return true
	}
	h = strings.ToLower(r.Header.Get("Content-Type"))
	return strings.Contains(h, "text/event-stream")
}

func sseOwnerKey(r *http.Request) string {
	if uid := GetUserID(r.Context()); uid != "" {
		return uid
	}
	if ip := GetClientIP(r.Context()); ip != "" {
		return ip
	}
	return "anon"
}
