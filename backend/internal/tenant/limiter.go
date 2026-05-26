package tenant

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
)

// tenantState tracks per-tenant rate limit counters with per-second windows.
type tenantState struct {
	signalWindow int64 // unix second
	signalCount  atomic.Int64
	orderWindow  int64
	orderCount   atomic.Int64
	chWindow     int64
	chBytes      atomic.Int64

	config   Config
	lastSeen atomic.Int64 // unix nano
}

// Limiter enforces per-tenant rate limits with LRU eviction.
// Thread-safe; designed for hot-path checks during signal generation and order placement.
type Limiter struct {
	mu          sync.Mutex
	entries     map[string]*list.Element
	lru         *list.List
	maxEntries  int            // LRU cap (default 1000)
	idleTimeout time.Duration  // idle eviction (default 1h)

	// Blocked counters (no tenant_id label — safe for Prometheus)
	signalsBlocked  atomic.Int64
	ordersBlocked   atomic.Int64
	chWritesBlocked atomic.Int64
	entriesEvicted  atomic.Int64
	activeTenants   atomic.Int64
}

// NewLimiter creates a per-tenant rate limiter.
func NewLimiter() *Limiter {
	return &Limiter{
		entries:     make(map[string]*list.Element),
		lru:         list.New(),
		maxEntries:  1000,
		idleTimeout: time.Hour,
	}
}

// AllowSignal returns true if the tenant is under its signal rate limit.
func (l *Limiter) AllowSignal(tenantID string, cfg Config) bool {
	ts := l.getOrCreate(tenantID, cfg)
	ts.lastSeen.Store(time.Now().UnixNano())

	nowSec := time.Now().Unix()
	limit := cfg.SignalMaxPerSec
	if limit <= 0 {
		limit = 500
	}

	for {
		old := ts.signalWindow
		if nowSec > old {
			if atomic.CompareAndSwapInt64(&ts.signalWindow, old, nowSec) {
				ts.signalCount.Store(0)
			}
		} else {
			break
		}
	}

	if ts.signalCount.Add(1) > limit {
		l.signalsBlocked.Add(1)
		ts.signalCount.Add(-1)
		return false
	}
	return true
}

// AllowOrder returns true if the tenant is under its order rate limit.
func (l *Limiter) AllowOrder(tenantID string, cfg Config) bool {
	ts := l.getOrCreate(tenantID, cfg)
	ts.lastSeen.Store(time.Now().UnixNano())

	nowSec := time.Now().Unix()
	limit := cfg.OrderMaxPerSec
	if limit <= 0 {
		limit = 50
	}

	for {
		old := ts.orderWindow
		if nowSec > old {
			if atomic.CompareAndSwapInt64(&ts.orderWindow, old, nowSec) {
				ts.orderCount.Store(0)
			}
		} else {
			break
		}
	}

	if ts.orderCount.Add(1) > limit {
		l.ordersBlocked.Add(1)
		ts.orderCount.Add(-1)
		return false
	}
	return true
}

// AllowCHWrite returns true if the tenant is under its CH write bandwidth limit.
func (l *Limiter) AllowCHWrite(tenantID string, cfg Config, bytes int64) bool {
	ts := l.getOrCreate(tenantID, cfg)
	ts.lastSeen.Store(time.Now().UnixNano())

	nowSec := time.Now().Unix()
	limit := cfg.CHWriteBytesPerSec
	if limit <= 0 {
		limit = 50_000_000
	}

	for {
		old := ts.chWindow
		if nowSec > old {
			if atomic.CompareAndSwapInt64(&ts.chWindow, old, nowSec) {
				ts.chBytes.Store(0)
			}
		} else {
			break
		}
	}

	if ts.chBytes.Add(bytes) > limit {
		l.chWritesBlocked.Add(1)
		ts.chBytes.Add(-bytes)
		return false
	}
	return true
}

// BlockedCounts returns global blocked counters (safe for Prometheus).
func (l *Limiter) BlockedCounts() (signals, orders, chWrites int64) {
	return l.signalsBlocked.Load(), l.ordersBlocked.Load(), l.chWritesBlocked.Load()
}

// ActiveTenants returns the number of tracked tenants.
func (l *Limiter) ActiveTenants() int64 { return l.activeTenants.Load() }

// EvictedCount returns total evicted entries.
func (l *Limiter) EvictedCount() int64 { return l.entriesEvicted.Load() }

// UpsertConfig updates or inserts a tenant's rate config. Thread-safe.
func (l *Limiter) UpsertConfig(cfg Config) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if elem, ok := l.entries[cfg.TenantID]; ok {
		ts := elem.Value.(*tenantState)
		ts.config = cfg
		return
	}

	if l.lru.Len() >= l.maxEntries {
		if tail := l.lru.Back(); tail != nil {
			ts := tail.Value.(*tenantState)
			delete(l.entries, ts.config.TenantID)
			l.lru.Remove(tail)
			l.entriesEvicted.Add(1)
		}
	}

	ts := &tenantState{config: cfg}
	ts.lastSeen.Store(time.Now().UnixNano())
	elem := l.lru.PushFront(ts)
	l.entries[cfg.TenantID] = elem
	l.activeTenants.Store(int64(len(l.entries)))
}

// EvictIdle removes tenants whose lastSeen is older than idleTimeout.
func (l *Limiter) EvictIdle() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-l.idleTimeout).UnixNano()
	var removed int64
	for e := l.lru.Back(); e != nil; {
		prev := e.Prev()
		ts := e.Value.(*tenantState)
		if ts.lastSeen.Load() < cutoff {
			l.lru.Remove(e)
			delete(l.entries, ts.config.TenantID)
			removed++
		}
		e = prev
	}
	l.entriesEvicted.Add(removed)
	l.activeTenants.Store(int64(len(l.entries)))
	return removed
}

func (l *Limiter) getOrCreate(tenantID string, cfg Config) *tenantState {
	l.mu.Lock()
	defer l.mu.Unlock()

	if elem, ok := l.entries[tenantID]; ok {
		l.lru.MoveToFront(elem)
		return elem.Value.(*tenantState)
	}

	if l.lru.Len() >= l.maxEntries {
		if tail := l.lru.Back(); tail != nil {
			ts := tail.Value.(*tenantState)
			delete(l.entries, ts.config.TenantID)
			l.lru.Remove(tail)
			l.entriesEvicted.Add(1)
		}
	}

	ts := &tenantState{config: cfg}
	nowUnix := time.Now().Unix()
	ts.signalWindow = nowUnix
	ts.orderWindow = nowUnix
	ts.chWindow = nowUnix
	ts.lastSeen.Store(time.Now().UnixNano())
	elem := l.lru.PushFront(ts)
	l.entries[tenantID] = elem
	l.activeTenants.Store(int64(len(l.entries)))
	return ts
}
