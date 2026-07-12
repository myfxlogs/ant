package system

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/mdgateway/adapter/mdtick"
	"alphaforge/internal/mdgateway/indicator"
	"alphaforge/internal/repository"
)

// SetMarketDataRepo injects the market data repository for indicator streaming.
func (s *StreamServer) SetMarketDataRepo(repo repository.MarketDataStore) {
	s.marketDataRepo = repo
}

// SubscribeIndicators computes technical indicators server-side and streams results.
// Follows the push-first architecture: backend computes, frontend renders.
func (s *StreamServer) SubscribeIndicators(
	ctx context.Context,
	req *connect.Request[antv1.SubscribeIndicatorsRequest],
	stream *connect.ServerStream[antv1.IndicatorUpdateEvent],
) error {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}

	m := req.Msg
	if m.Symbol == "" || m.Timeframe == "" || len(m.IndicatorIds) == 0 {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("symbol, timeframe, and at least one indicator_id are required"))
	}

	if s.marketDataRepo == nil {
		return connect.NewError(connect.CodeUnavailable,
			fmt.Errorf("market data not available"))
	}

	// Fetch historical bars from ClickHouse.
	const defaultLimit = 500
	bars, err := s.marketDataRepo.GetKlines(ctx, m.Symbol, "" /*broker*/, m.Timeframe, nil, nil, defaultLimit)
	if err != nil {
		s.log.Warn("SubscribeIndicators: fetch bars failed",
			zap.String("symbol", m.Symbol),
			zap.String("timeframe", m.Timeframe),
			zap.Error(err))
		return connect.NewError(connect.CodeUnavailable,
			fmt.Errorf("failed to fetch market data: %w", err))
	}
	if len(bars) == 0 {
		return connect.NewError(connect.CodeNotFound,
			fmt.Errorf("no bar data for %s/%s", m.Symbol, m.Timeframe))
	}

	barCount := int32(len(bars))
	mdtickBars := klineBarsToMdtickBars(bars)

	// Compute and stream each requested indicator.
	for _, id := range m.IndicatorIds {
		entry, ok := indicator.Registry[id]
		if !ok {
			s.log.Warn("SubscribeIndicators: unknown indicator", zap.String("id", id))
			continue
		}

		// Merge defaults with optional per-indicator param overrides.
		params := mergeIndicatorParams(entry.Def.Defaults, m.Params, id)
		result, err := indicator.Compute(id, mdtickBars, params)
		if err != nil {
			s.log.Warn("SubscribeIndicators: compute failed",
				zap.String("id", id), zap.Error(err))
			continue
		}

		event := buildIndicatorEvent(m.Symbol, m.Timeframe, barCount, result, entry.Def.Kind)
		if err := stream.Send(event); err != nil {
			return err
		}
	}

	return nil
}

// buildIndicatorEvent converts an indicator.Result into a proto IndicatorUpdateEvent.
func buildIndicatorEvent(symbol, timeframe string, barCount int32, r *indicator.Result, kind indicator.Kind) *antv1.IndicatorUpdateEvent {
	event := &antv1.IndicatorUpdateEvent{
		Symbol:      symbol,
		Timeframe:   timeframe,
		IndicatorId: r.DefID,
		BarCount:    barCount,
		Pane:        string(kind),
	}

	if len(r.Lines) == 1 {
		// Single-line indicator: use values directly.
		for _, vals := range r.Lines {
			event.Values = decSliceToFloat64(vals)
		}
	} else {
		// Multi-line indicator: use series map.
		event.Series = make(map[string]*antv1.IndicatorSeries, len(r.Lines))
		for name, vals := range r.Lines {
			event.Series[name] = &antv1.IndicatorSeries{
				Values: decSliceToFloat64(vals),
			}
		}
	}

	return event
}

// klineBarsToMdtickBars converts repository.KlineBar (decimal OHLC) to mdtick.Bar (decimal.Decimal).
func klineBarsToMdtickBars(bars []repository.KlineBar) []mdtick.Bar {
	out := make([]mdtick.Bar, len(bars))
	for i, b := range bars {
		out[i] = mdtick.Bar{
			Open:   b.Open,
			High:   b.High,
			Low:    b.Low,
			Close:  b.Close,
			Volume: b.Volume,
		}
	}
	return out
}

// decSliceToFloat64 converts []decimal.Decimal to []float64 for proto transport.
// Precision loss ≤ 1e-15 — acceptable at the proto wire boundary.
func decSliceToFloat64(vals []decimal.Decimal) []float64 {
	out := make([]float64, len(vals))
	for i, v := range vals {
		out[i], _ = v.Float64()
	}
	return out
}

// mergeIndicatorParams merges indicator defaults with optional param overrides from the request.
// Params map format: {"RSI.period": "14"} → extracts "14" for indicator "RSI" key "period".
func mergeIndicatorParams(defaults map[string]float64, params map[string]string, indicatorID string) map[string]float64 {
	merged := make(map[string]float64, len(defaults)+1)
	for k, v := range defaults {
		merged[k] = v
	}
	if params == nil {
		return merged
	}
	prefix := indicatorID + "."
	for k, v := range params {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			paramKey := k[len(prefix):]
			if fv, err := parseFloat(v); err == nil {
				merged[paramKey] = fv
			}
		}
	}
	return merged
}

// parseFloat parses a string to float64, returning 0 on error.
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
