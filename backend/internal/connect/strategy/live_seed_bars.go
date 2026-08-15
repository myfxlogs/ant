package strategy

import (
	"context"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// seedBarWindows pre-loads historical bars from PG (md_bars) into the live bar
// window so the strategy has a full context window from the first bar event.
// Uses MaxCloseTs + GetKlines with a `from` window to get the most recent bars
// (GetKlines is ASC ORDER BY + LIMIT — without `from` it returns the oldest).
//
// Broker identifier = mt_accounts.broker_company (the authoritative source for
// filtering md_bars rows to this account's data source). If broker is empty,
// MaxCloseTs returns 0, or klines are empty → log.Warn and skip (degrade to
// today's zero-start behavior; never block the run).
//
// Only closed bars are seeded (LIVE-1 consistency): bars with
// openTime+periodMs > now are discarded.
func (s *StrategyExecutionServer) seedBarWindows(
	ctx context.Context,
	cfg LiveStrategyConfig,
	bars *[]liveBar,
	extraBars map[string][]liveBar,
) {
	broker := s.resolveBrokerCompany(ctx, cfg)
	periodMs := periodToMs(cfg.Timeframe)
	if periodMs == 0 {
		s.log.Warn("seedBarWindows: unknown timeframe, skipping",
			zap.String("timeframe", cfg.Timeframe))
		return
	}
	seedSymbol(ctx, s, broker, cfg.Symbol, cfg.Timeframe, periodMs, bars)
	for _, sym := range cfg.ExtraSymbols {
		if sym == "" || sym == cfg.Symbol {
			continue
		}
		ew, ok := extraBars[sym]
		if !ok {
			continue
		}
		seedSymbol(ctx, s, broker, sym, cfg.Timeframe, periodMs, &ew)
		extraBars[sym] = ew
	}
}

func seedSymbol(
	ctx context.Context,
	s *StrategyExecutionServer,
	broker, symbol, timeframe string,
	periodMs int64,
	bars *[]liveBar,
) {
	if s.marketDataRepo == nil || broker == "" {
		return
	}
	maxTs, err := s.marketDataRepo.MaxCloseTs(ctx, broker, symbol, timeframe)
	if err != nil {
		s.log.Warn("seedBarWindows: MaxCloseTs failed",
			zap.String("symbol", symbol), zap.Error(err))
		return
	}
	if maxTs == 0 {
		s.log.Warn("seedBarWindows: no historical data, skipping",
			zap.String("symbol", symbol), zap.String("broker", broker))
		return
	}
	from := time.UnixMilli(maxTs - int64(maxContextBars)*periodMs + periodMs)
	klines, err := s.marketDataRepo.GetKlines(ctx, symbol, broker, timeframe, &from, nil, int32(maxContextBars))
	if err != nil {
		s.log.Warn("seedBarWindows: GetKlines failed",
			zap.String("symbol", symbol), zap.Error(err))
		return
	}
	now := time.Now().UnixMilli()
	seeded := 0
	for _, k := range klines {
		openMs := int64(k.OpenTsUnixMs)
		if openMs+periodMs > now {
			continue
		}
		*bars = append(*bars, liveBar{
			open:     k.Open.String(),
			high:     k.High.String(),
			low:      k.Low.String(),
			close:    k.Close.String(),
			volume:   strconv.FormatFloat(k.Volume, 'f', -1, 64),
			openTime: openMs,
		})
		seeded++
	}
	if len(*bars) > maxContextBars {
		*bars = (*bars)[len(*bars)-maxContextBars:]
	}
	s.log.Info("seedBarWindows: seeded historical bars",
		zap.String("symbol", symbol),
		zap.String("broker", broker),
		zap.Int("seeded", seeded),
		zap.Int("window_len", len(*bars)))
}

// resolveBrokerCompany fetches the broker_company from mt_accounts for the
// given config's account ID. Returns "" if unavailable (degrade gracefully).
func (s *StrategyExecutionServer) resolveBrokerCompany(ctx context.Context, cfg LiveStrategyConfig) string {
	if cfg.AccountID == "" {
		return ""
	}
	if s.brokerCompanyLookup == nil {
		return ""
	}
	return s.brokerCompanyLookup(ctx, cfg.AccountID)
}

// prefetchSymbolParam fetches symbol params once at startup with a 5s timeout.
// Builders read from cfg.SymbolParam — no per-event broker RPC (W2).
func (s *StrategyExecutionServer) prefetchSymbolParam(ctx context.Context, cfg *LiveStrategyConfig) {
	if s.mtHub == nil || cfg.AccountID == "" || cfg.Symbol == "" {
		return
	}
	fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	param, err := s.mtHub.CachedSymbolParam(fetchCtx, cfg.AccountID, cfg.Symbol)
	if err != nil || param == nil {
		s.log.Warn("LiveStrategyRunner: symbol param pre-fetch failed, will retry on first bar",
			zap.String("symbol", cfg.Symbol), zap.Error(err))
		return
	}
	cfg.SymbolParam = param
}
