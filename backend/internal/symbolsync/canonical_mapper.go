// Package symbolsync — canonical mapper: dict-first + rule-fallback.
// Ported from alfq. Resolves symbol_raw → canonical using the canonical_symbols
// dictionary with Canonicalize() as rule-based fallback.
package symbolsync

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
)

// CanonicalMapper resolves symbol_raw → canonical using the canonical_symbols dictionary
// with rule-based fallback for known suffix patterns.
type CanonicalMapper struct {
	db    *sqlx.DB
	mu    sync.RWMutex
	cache map[string]string // symbol_raw_upper → canonical
}

// NewCanonicalMapper creates a canonical mapper backed by the canonical_symbols PG dict.
func NewCanonicalMapper(db *sqlx.DB) *CanonicalMapper {
	return &CanonicalMapper{db: db, cache: make(map[string]string)}
}

// Resolve resolves a symbol_raw to canonical.
// Strategy: cache lookup → rule-based Canonicalize → dict validation.
func (m *CanonicalMapper) Resolve(ctx context.Context, symbolRaw string) (canonical string, partial bool) {
	upper := strings.ToUpper(symbolRaw)

	m.mu.RLock()
	if c, ok := m.cache[upper]; ok {
		m.mu.RUnlock()
		return c, c == ""
	}
	m.mu.RUnlock()

	canonical = Canonicalize(upper)

	if m.db != nil {
		var exists bool
		err := m.db.GetContext(ctx, &exists,
			`SELECT EXISTS(SELECT 1 FROM canonical_symbols WHERE canonical = $1 AND enabled = true)`,
			canonical,
		)
		if err == nil && exists {
			m.cacheSet(upper, canonical)
			return canonical, false
		}
		err = m.db.GetContext(ctx, &exists,
			`SELECT EXISTS(SELECT 1 FROM canonical_symbols WHERE canonical = $1 AND enabled = true)`,
			upper,
		)
		if err == nil && exists {
			m.cacheSet(upper, upper)
			return upper, false
		}
		m.cacheSet(upper, "")
		return canonical, true
	}

	return canonical, false
}

// ResolveOrDefault returns the canonical or the raw symbol as-is when dict lookup fails.
func (m *CanonicalMapper) ResolveOrDefault(ctx context.Context, symbolRaw string) string {
	canonical, partial := m.Resolve(ctx, symbolRaw)
	if partial {
		return strings.ToUpper(symbolRaw)
	}
	return canonical
}

// RefreshCache reloads the symbol_raw → canonical mapping from broker_symbols.
func (m *CanonicalMapper) RefreshCache(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("canonical mapper: db not available")
	}
	rows, err := m.db.QueryxContext(ctx,
		`SELECT symbol_raw, canonical FROM broker_symbols WHERE canonical IS NOT NULL AND canonical != ''`)
	if err != nil {
		return fmt.Errorf("canonical mapper: refresh: %w", err)
	}
	defer rows.Close()

	newCache := make(map[string]string)
	for rows.Next() {
		var raw, canon string
		if err := rows.Scan(&raw, &canon); err != nil {
			continue
		}
		newCache[strings.ToUpper(raw)] = canon
	}

	m.mu.Lock()
	m.cache = newCache
	m.mu.Unlock()
	return nil
}

func (m *CanonicalMapper) cacheSet(key, value string) {
	m.mu.Lock()
	m.cache[key] = value
	m.mu.Unlock()
}
