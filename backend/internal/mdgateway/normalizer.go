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

// stripSuffix removes known broker symbol suffixes and uppercases the result.
// Known dot suffixes: .m, .pro, .x, .c (stripped).
// Known underscore suffixes: _i, _r, _institutional, _retail (stripped).
// Unknown dot suffixes are preserved but uppercased.
// This is intentionally NOT used in Resolve() — raw broker symbols are canonical.
func stripSuffix(raw string) string {
	if raw == "" {
		return ""
	}
	upper := func(s string) string {
		b := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			b[i] = c
		}
		return string(b)
	}

	// Known dot suffixes to strip.
	dotSuffixes := []string{".m", ".pro", ".x", ".c"}
	upperRaw := upper(raw)
	for _, s := range dotSuffixes {
		if len(upperRaw) > len(s) && upperRaw[len(upperRaw)-len(s):] == upper(s) {
			return upperRaw[:len(upperRaw)-len(s)]
		}
	}

	// Unknown dot suffix: preserve but uppercase.
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i] == '.' {
			return upperRaw
		}
	}

	// Known underscore suffixes to strip.
	underscoreSuffixes := []string{"_i", "_r", "_institutional", "_retail"}
	for _, s := range underscoreSuffixes {
		if len(upperRaw) > len(s) && upperRaw[len(upperRaw)-len(s):] == upper(s) {
			return upperRaw[:len(upperRaw)-len(s)]
		}
	}

	return upperRaw
}

// InvalidateCache removes a cached entry for (broker, symbol_raw).
// Called by NormalizerInvalidator on PG NOTIFY events (ADR-0011 §2.3).
func (n *Normalizer) InvalidateCache(broker, symbolRaw string) {
	key := broker + ":" + symbolRaw
	delete(n.cache, key)
}
