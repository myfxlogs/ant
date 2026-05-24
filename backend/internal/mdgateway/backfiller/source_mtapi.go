package backfiller

import (
	"context"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// SourceMTAPI fetches historical bars from the mtapi GetPriceHistory RPC.
// The mtapi adapter is injected at construction time via runner.go.
type SourceMTAPI struct {
	adapter MTAPIBarSource
}

// MTAPIBarSource is the interface that mt4/mt5 adapters implement for bar fetching.
type MTAPIBarSource interface {
	GetPriceHistory(ctx context.Context, accountID, symbolRaw string, period string, from, to int64) ([]*mdtick.Bar, error)
}

// NewSourceMTAPI creates a new mtapi-backed bar source.
func NewSourceMTAPI(adapter MTAPIBarSource) *SourceMTAPI {
	return &SourceMTAPI{adapter: adapter}
}

// FetchBars satisfies the Source interface.
func (s *SourceMTAPI) FetchBars(ctx context.Context, req FetchReq) ([]*mdtick.Bar, error) {
	if s.adapter == nil {
		return nil, nil
	}
	return s.adapter.GetPriceHistory(ctx, req.AccountID, req.SymbolRaw, req.Period, req.From, req.To)
}
