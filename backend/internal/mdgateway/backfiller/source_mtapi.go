package backfiller

import (
	"context"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// SourceMTAPI fetches historical bars from the mtapi GetPriceHistory RPC.
// The mtapi adapter is injected at construction time.
type SourceMTAPI struct {
	// adapter is the mt4/mt5 adapter that implements the GetPriceHistory call.
	// Will be injected by runner.go when the mtapi connection is available.
	// adapter MTAPIBarSource
}

// MTAPIBarSource is the interface that mt4/mt5 adapters implement for bar fetching.
type MTAPIBarSource interface {
	GetPriceHistory(ctx context.Context, accountID, symbolRaw string, period string, from, to int64) ([]*mdtick.Bar, error)
}

// NewSourceMTAPI creates a new mtapi-backed bar source.
func NewSourceMTAPI() *SourceMTAPI {
	return &SourceMTAPI{}
}

// FetchBars satisfies the Source interface.
func (s *SourceMTAPI) FetchBars(ctx context.Context, req FetchReq) ([]*mdtick.Bar, error) {
	// Stub: will be wired by M10.2-4 runner integration.
	// The real implementation calls the adapter's GetPriceHistory.
	return nil, nil
}
