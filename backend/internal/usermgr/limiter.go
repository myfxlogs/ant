// Package usermgr provides per-user rate limiting and context helpers (M10-BASE-A4).
//
// Key constraint: Prometheus metrics must NOT carry user_id label (cardinality).
// Logs, traces, and LRU bookkeeping may carry user_id.
package usermgr

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"

	"anttrader/internal/clock"
)

// Clk is the package-level clock for deterministic testing (M10-BASE-A5).
var Clk clock.Clock = clock.NewRealClock()

// RateLimit tracks per-user rate limit counters with per-second windows.
type RateLimit struct {
	signalWindow int64 // unix second of current signal window
	signalCount  atomic.Int64
	orderWindow  int64 // unix second of current order window
	orderCount   atomic.Int64
	chWindow     int64 // unix second of current CH write window
	chBytes      atomic.Int64

	LastAccess atomic.Int64 // unix nano
	userID     string
}

// UserLimiter enforces per-user and global rate limits with LRU eviction.
// Thread-safe; all methods may be called from multiple goroutines.
type UserLimiter struct {
	cfg Config

	mu         sync.Mutex
	entries    map[string]*list.Element
	lru        *list.List
	maxEntries int

	// Global CH write ceiling (per-second window)
	globalChWindow  atomic.Int64 // unix second
	globalChBytes   atomic.Int64 // bytes written in current second

	// Counters (no user_id label — safe for Prometheus)
	signalsBlocked  atomic.Int64
	ordersBlocked   atomic.Int64
	chWritesBlocked atomic.Int64
	entriesEvicted  atomic.Int64

	activeUsers atomic.Int64
}

// Config holds UserLimiter settings.
type Config struct {
	MaxEntries        int           // LRU cap (default 50000)
	IdleTimeout       time.Duration // idle eviction (default 30 days)
	GlobalCHCeiling   int64         // global CH write bytes/sec ceiling
	SignalPerUserMax  int64         // signals per user per second
	OrderPerUserMax   int64         // orders per user per second
	CHWritePerUserMax int64         // CH write bytes per user per second
}

// DefaultConfig returns M10 defaults.
func DefaultConfig() Config {
	return Config{
		MaxEntries:        50000,
		IdleTimeout:       30 * 24 * time.Hour,
		GlobalCHCeiling:   100_000_000, // 100 MB/s
		SignalPerUserMax:  100,
		OrderPerUserMax:   10,
		CHWritePerUserMax: 10_000_000, // 10 MB/s
	}
}

// NewUserLimiter creates a per-user rate limiter.
func NewUserLimiter(cfg Config) *UserLimiter {
	return &UserLimiter{
		cfg:        cfg,
		entries:    make(map[string]*list.Element),
		lru:        list.New(),
		maxEntries: cfg.MaxEntries,
	}
}

// AllowSignal returns true if the user is under their signal rate limit.
func (l *UserLimiter) AllowSignal(userID string) bool {
	rl := l.getOrCreate(userID)
	rl.LastAccess.Store(time.Now().UnixNano())

	nowSec := time.Now().Unix()
	limit := l.cfg.SignalPerUserMax
	if limit <= 0 {
		limit = 100
	}

	for {
		old := rl.signalWindow
		if nowSec > old {
			// Try to advance the window.
			if atomic.CompareAndSwapInt64(&rl.signalWindow, old, nowSec) {
				rl.signalCount.Store(0)
			}
		} else {
			break
		}
	}

	current := rl.signalCount.Add(1)
	if current > limit {
		l.signalsBlocked.Add(1)
		rl.signalCount.Add(-1)
		return false
	}
	return true
}

// AllowOrder returns true if the user is under their order rate limit.
func (l *UserLimiter) AllowOrder(userID string) bool {
	rl := l.getOrCreate(userID)
	rl.LastAccess.Store(time.Now().UnixNano())

	nowSec := time.Now().Unix()
	limit := l.cfg.OrderPerUserMax
	if limit <= 0 {
		limit = 10
	}

	for {
		old := rl.orderWindow
		if nowSec > old {
			if atomic.CompareAndSwapInt64(&rl.orderWindow, old, nowSec) {
				rl.orderCount.Store(0)
			}
		} else {
			break
		}
	}

	current := rl.orderCount.Add(1)
	if current > limit {
		l.ordersBlocked.Add(1)
		rl.orderCount.Add(-1)
		return false
	}
	return true
}

// AllowCHWrite returns true if the user is under their CH write bandwidth limit
// AND the global CH write ceiling is not exceeded.
func (l *UserLimiter) AllowCHWrite(userID string, bytes int64) bool {
	// Global ceiling check first (fast path — no per-user work needed if blocked).
	if !l.allowGlobalCHWrite(bytes) {
		l.chWritesBlocked.Add(1)
		return false
	}

	rl := l.getOrCreate(userID)
	rl.LastAccess.Store(time.Now().UnixNano())

	nowSec := time.Now().Unix()
	limit := l.cfg.CHWritePerUserMax
	if limit <= 0 {
		limit = 10_000_000
	}

	for {
		old := rl.chWindow
		if nowSec > old {
			if atomic.CompareAndSwapInt64(&rl.chWindow, old, nowSec) {
				rl.chBytes.Store(0)
			}
		} else {
			break
		}
	}

	current := rl.chBytes.Add(bytes)
	if current > limit {
		l.chWritesBlocked.Add(1)
		rl.chBytes.Add(-bytes)
		return false
	}
	return true
}

func (l *UserLimiter) allowGlobalCHWrite(bytes int64) bool {
	ceiling := l.cfg.GlobalCHCeiling
	if ceiling <= 0 {
		return true
	}

	nowSec := time.Now().Unix()

	for {
		old := l.globalChWindow.Load()
		if nowSec > old {
			if l.globalChWindow.CompareAndSwap(old, nowSec) {
				l.globalChBytes.Store(0)
			}
		} else {
			break
		}
	}

	current := l.globalChBytes.Add(bytes)
	return current <= ceiling
}

// BlockedCounts returns global blocked counters (safe for Prometheus).
func (l *UserLimiter) BlockedCounts() (signals, orders, chWrites int64) {
	return l.signalsBlocked.Load(), l.ordersBlocked.Load(), l.chWritesBlocked.Load()
}

// ActiveUsers returns the current number of tracked users.
func (l *UserLimiter) ActiveUsers() int64 {
	return l.activeUsers.Load()
}

// EvictedCount returns the total number of LRU-evicted entries.
func (l *UserLimiter) EvictedCount() int64 {
	return l.entriesEvicted.Load()
}

// EvictIdle removes entries whose LastAccess is older than the idle timeout.
func (l *UserLimiter) EvictIdle() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	idleTimeout := l.cfg.IdleTimeout
	if idleTimeout <= 0 {
		idleTimeout = 30 * 24 * time.Hour
	}
	cutoff := time.Now().Add(-idleTimeout).UnixNano()
	var removed int64
	for e := l.lru.Back(); e != nil; {
		prev := e.Prev()
		rl := e.Value.(*RateLimit)
		if rl.LastAccess.Load() < cutoff {
			l.lru.Remove(e)
			delete(l.entries, rl.userID)
			removed++
		}
		e = prev
	}
	l.entriesEvicted.Add(removed)
	l.activeUsers.Store(int64(len(l.entries)))
	return removed
}

func (l *UserLimiter) getOrCreate(userID string) *RateLimit {
	l.mu.Lock()
	defer l.mu.Unlock()

	if elem, ok := l.entries[userID]; ok {
		l.lru.MoveToFront(elem)
		return elem.Value.(*RateLimit)
	}

	if l.lru.Len() >= l.maxEntries {
		if tail := l.lru.Back(); tail != nil {
			rl := tail.Value.(*RateLimit)
			delete(l.entries, rl.userID)
			l.lru.Remove(tail)
			l.entriesEvicted.Add(1)
		}
	}

	nowUnix := time.Now().Unix()
	rl := &RateLimit{
		userID:       userID,
		signalWindow: nowUnix,
		orderWindow:  nowUnix,
		chWindow:     nowUnix,
	}
	rl.LastAccess.Store(time.Now().UnixNano())
	elem := l.lru.PushFront(rl)
	l.entries[userID] = elem
	l.activeUsers.Store(int64(len(l.entries)))
	return rl
}
