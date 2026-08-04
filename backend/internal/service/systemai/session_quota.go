package systemai

import (
	"context"
	"sync"
)

// SessionQuotaKey is the exported context key for the session token counter.
// The agent loop uses this to inject the counter into per-call contexts.
var SessionQuotaKey = sessionQuotaKey{}

// sessionQuotaKey is a context key for carrying the session-scoped token counter.
type sessionQuotaKey struct{}

// sessionCounter tracks tokens consumed within a single agent session.
// The agent loop makes multiple LLM calls; each call's post-call billing
// adds to this counter so subsequent pre-checks see the cumulative session usage.
type sessionCounter struct {
	mu    sync.Mutex
	total int
}

// AddTokens adds the given token count to the session counter.
func (sc *sessionCounter) AddTokens(n int) {
	sc.mu.Lock()
	sc.total += n
	sc.mu.Unlock()
}

// Total returns the current session token total.
func (sc *sessionCounter) Total() int {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.total
}

// WithSessionQuota attaches a session-scoped token counter to the context.
// The agent loop calls this at session start; all LLM calls within the
// session share the same counter.
func WithSessionQuota(ctx context.Context) (context.Context, *sessionCounter) {
	sc := &sessionCounter{}
	return context.WithValue(ctx, sessionQuotaKey{}, sc), sc
}

// sessionCounterFromCtx returns the session counter from context, or nil.
func sessionCounterFromCtx(ctx context.Context) *sessionCounter {
	v, _ := ctx.Value(sessionQuotaKey{}).(*sessionCounter)
	return v
}
