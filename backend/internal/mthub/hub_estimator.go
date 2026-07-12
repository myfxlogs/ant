// hub_estimator.go — CostEstimator backed by live MT gateway SymbolParams.
//
// Lives in mthub (not costsvc) to avoid an import cycle: it needs both
// the Hub (to fetch symbol params) and costsvc.CostModel (to produce estimates).

package mthub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"alphaforge/internal/costsvc"
)

// HubCostEstimator resolves per-symbol cost models from the MT gateway.
// It implements costsvc.CostEstimator and is safe for concurrent use.
type HubCostEstimator struct {
	hub    *Hub
	log    *zap.Logger
	mu     sync.RWMutex
	cache  map[string]*costsvc.CostModel
	def    *costsvc.CostModel
	maxAge time.Duration
}

// NewHubCostEstimator creates a cost estimator that queries the MT gateway.
func NewHubCostEstimator(hub *Hub, defaultModel *costsvc.CostModel, log *zap.Logger) *HubCostEstimator {
	if log == nil {
		log = zap.NewNop()
	}
	return &HubCostEstimator{
		hub:    hub,
		log:    log,
		cache:  make(map[string]*costsvc.CostModel),
		def:    defaultModel,
		maxAge: 5 * time.Minute,
	}
}

// Estimate returns a cost breakdown for the given parameters.
func (e *HubCostEstimator) Estimate(ctx context.Context, params costsvc.EstimateParams) costsvc.CostBreakdown {
	model := e.getOrFetch(ctx, params.Symbol)
	return model.Estimate(params)
}

func (e *HubCostEstimator) getOrFetch(ctx context.Context, symbol string) *costsvc.CostModel {
	e.mu.RLock()
	if m, ok := e.cache[symbol]; ok {
		e.mu.RUnlock()
		return m
	}
	e.mu.RUnlock()

	e.mu.Lock()
	defer e.mu.Unlock()

	if m, ok := e.cache[symbol]; ok {
		return m
	}

	m, err := e.fetchSymbolModel(ctx, symbol)
	if err != nil {
		e.log.Warn("HubCostEstimator: fetch failed, using default",
			zap.String("symbol", symbol), zap.Error(err))
		e.cache[symbol] = e.def
		return e.def
	}

	e.cache[symbol] = m
	return m
}

func (e *HubCostEstimator) fetchSymbolModel(ctx context.Context, symbol string) (*costsvc.CostModel, error) {
	for _, accountID := range e.hub.ActiveAccountIDs() {
		exec := e.hub.Get(accountID)
		if exec == nil {
			continue
		}

		params, err := exec.FetchSymbolParams(ctx, symbolVariants(symbol))
		if err != nil {
			continue
		}

		model := paramsToCostModel(symbol, params)
		if model != nil {
			return model, nil
		}
	}
	return nil, fmt.Errorf("symbol %q not found on any connected account", symbol)
}

func paramsToCostModel(canonical string, params []*SymbolParam) *costsvc.CostModel {
	var best *SymbolParam
	for _, p := range params {
		if p.Canonical == canonical {
			best = p
			break
		}
	}
	if best == nil {
		for _, p := range params {
			if p.Canonical == canonical+"m" || p.Canonical == canonical {
				best = p
				break
			}
		}
	}
	if best == nil {
		return nil
	}

	pointVal, _ := best.PointValue.Float64()
	if pointVal <= 0 {
		pointVal = 0.10
	}

	// Pip size: 10 points for 5-digit, 1 point for 4-digit.
	pipInPoints := 1.0
	if best.Digits >= 5 {
		pipInPoints = 10.0
	} else if best.Digits == 3 {
		pipInPoints = 10.0 // JPY pairs: 3 digits
	}

	pipSize := mathPow10(float64(-best.Digits)) * pipInPoints
	pipValue := pointVal * pipInPoints

	return &costsvc.CostModel{
		Symbol:           canonical,
		SpreadPips:       1.0,
		PipSize:          pipSize,
		PipValue:         pipValue,
		CommissionPerLot: 0,
	}
}

func mathPow10(n float64) float64 {
	return float64(decimal.NewFromFloat(10).Pow(decimal.NewFromFloat(n)).InexactFloat64())
}

func symbolVariants(symbol string) []string {
	return []string{symbol, symbol + "m", symbol + "x", symbol + ".", symbol + ".."}
}

// Refresh clears the cache for a symbol.
func (e *HubCostEstimator) Refresh(symbol string) {
	e.mu.Lock()
	delete(e.cache, symbol)
	e.mu.Unlock()
}
