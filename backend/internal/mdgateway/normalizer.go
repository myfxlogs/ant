package mdgateway

import (
	"context"
	"strings"

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
func (n *Normalizer) Resolve(broker, raw string) string {
	key := broker + ":" + raw
	if v, ok := n.cache[key]; ok {
		return v
	}

	// Try PG lookup
	if n.pg != nil {
		var canonical string
		err := n.pg.QueryRow(context.Background(),
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
	s := raw
	// Remove trailing dot and everything after
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		base := s[:idx]
		suffix := strings.ToLower(s[idx+1:])
		// Known suffixes: m, pro, x, c
		switch suffix {
		case "m", "pro", "x", "c":
			return strings.ToUpper(base)
		default:
			return strings.ToUpper(s)
		}
	}
	// Remove _i / _r suffixes
	s = strings.TrimSuffix(s, "_i")
	s = strings.TrimSuffix(s, "_r")
	s = strings.TrimSuffix(s, "_institutional")
	s = strings.TrimSuffix(s, "_retail")
	return strings.ToUpper(s)
}
