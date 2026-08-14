package mthub

import (
	"context"
	"fmt"
	"time"
)

type symbolParamCacheEntry struct {
	param *SymbolParam
	at    time.Time
}

// CachedSymbolParam returns the trading parameters for a single symbol, using a
// short-lived in-memory cache to avoid a broker round-trip on every order.
// The canonical is the cache key; results are refreshed after 5 minutes.
func (s *MtHubService) CachedSymbolParam(ctx context.Context, accountID, canonical string) (*SymbolParam, error) {
	const ttl = 5 * time.Minute
	s.symbolParamMu.RLock()
	e, ok := s.symbolParamCache[canonical]
	s.symbolParamMu.RUnlock()
	if ok && Clk.Now().Sub(e.at) < ttl {
		return e.param, nil
	}

	params, err := s.SymbolParams(ctx, accountID, []string{canonical})
	if err != nil {
		return nil, err
	}
	if len(params) == 0 {
		return nil, fmt.Errorf("symbol %s not found", canonical)
	}
	p := params[0]
	s.symbolParamMu.Lock()
	s.symbolParamCache[canonical] = symbolParamCacheEntry{param: p, at: Clk.Now()}
	s.symbolParamMu.Unlock()
	return p, nil
}
