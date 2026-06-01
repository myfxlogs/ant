package mdgateway

import (
	"context"
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

	// No algorithmic fallback — raw symbol IS the canonical.
	// Suffix stripping caused mismatches: brokers use symbols like "XAUUSDm"
	// and don't recognize the stripped form "XAUUSD" for historical queries.
	canonical := raw
	n.cache[key] = canonical
	return canonical
}

// InvalidateCache removes a cached entry for (broker, symbol_raw).
// Called by NormalizerInvalidator on PG NOTIFY events (ADR-0011 §2.3).
func (n *Normalizer) InvalidateCache(broker, symbolRaw string) {
	key := broker + ":" + symbolRaw
	delete(n.cache, key)
}
