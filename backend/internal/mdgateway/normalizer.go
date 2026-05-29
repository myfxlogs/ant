package mdgateway

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CanonicalResolver resolves (broker, symbol_raw) -> canonical symbol.
// Implements mdtick.CanonicalResolver.
type Normalizer struct {
	pg    *pgxpool.Pool
	cache map[string]string // key: broker:raw
}

// NewNormalizer creates a new normalizer. cache is backed by LRU in production
// (via hashicorp/golang-lru/v2); simplified to map for now.
func NewNormalizer(pg *pgxpool.Pool) *Normalizer {
	return &Normalizer{pg: pg, cache: make(map[string]string)}
}

// Resolve resolves (broker, symbol_raw) → canonical symbol.
// Order: 1. in-memory cache  2. PG broker_symbols  3. algorithmic fallback.
func (n *Normalizer) Resolve(ctx context.Context, broker, raw string) string {
	key := broker + ":" + raw
	if v, ok := n.cache[key]; ok {
		return v
	}

	// Guard against unbounded cache growth: reset if exceeding 100k entries.
	const maxCacheSize = 100_000
	if len(n.cache) > maxCacheSize {
		n.cache = make(map[string]string, maxCacheSize)
	}

	// Try PG lookup
	if n.pg != nil {
		var canonical string
		queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		err := n.pg.QueryRow(queryCtx,
			"SELECT canonical FROM broker_symbols WHERE broker=$1 AND symbol_raw=$2 LIMIT 1",
			broker, raw,
		).Scan(&canonical)
		if err == nil && canonical != "" {
			n.cache[key] = canonical
			return canonical
		}
	}

	// Algorithmic fallback: strip common suffixes
	canonical := stripSuffix(raw)
	n.cache[key] = canonical
	return canonical
}

// InvalidateCache removes a cached entry for (broker, symbol_raw).
// Called by NormalizerInvalidator on PG NOTIFY events (ADR-0011 §2.3).
func (n *Normalizer) InvalidateCache(broker, symbolRaw string) {
	key := broker + ":" + symbolRaw
	delete(n.cache, key)
}

// stripSuffix removes known MT symbol suffixes per alfq Q-005 + Q-006.
func stripSuffix(raw string) string {
	s := strings.ToLower(raw)

	// Dot-delimited suffixes: e.g. "EURUSD.m" → "EURUSD"
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		base := s[:idx]
		suffix := s[idx+1:]
		switch suffix {
		case "m", "pro", "x", "c":
			return strings.ToUpper(base)
		default:
			return strings.ToUpper(s)
		}
	}

	// Known MT5 suffixes appended without delimiter (case-insensitive).
	// e.g. "XAUUSDm" → "XAUUSD", "BTCUSDpro" → "BTCUSD"
	// Sorted longest-first so "EURUSD_r" matches "_r" before "r".
	suffixes := []string{"_institutional", "_retail", "_i", "_r", "pro", "m", "x", "c", "t", "r"}
	for _, suf := range suffixes {
		if strings.HasSuffix(s, suf) {
			return strings.ToUpper(strings.TrimSuffix(s, suf))
		}
	}
	return strings.ToUpper(s)
}
