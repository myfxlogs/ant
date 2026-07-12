package mdgateway

import (
	"alphaforge/internal/factor"
)

// factorMarketStateAdapter adapts mdgateway.MarketStateTracker to factor.MarketStateReader.
type factorMarketStateAdapter struct {
	tracker *MarketStateTracker
}

// NewFactorMarketStateReader wraps a MarketStateTracker as a factor.MarketStateReader.
func NewFactorMarketStateReader(tracker *MarketStateTracker) factor.MarketStateReader {
	return &factorMarketStateAdapter{tracker: tracker}
}

func (a *factorMarketStateAdapter) Get(broker, canonical string) *factor.MarketStateSnapshot {
	ms := a.tracker.Get(broker, canonical)
	if ms == nil {
		return nil
	}
	return &factor.MarketStateSnapshot{IsTradeable: ms.IsTradeable}
}
