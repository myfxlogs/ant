// live_runner_loop.go — Event loop and subscription helpers extracted from live_runner.go.
package strategy

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/mthub"
	"alphaforge/tools/mql2go"
)

func (s *StrategyExecutionServer) runLiveEventLoop(p liveEventLoopParams) {
	for {
		select {
		case <-p.runCtx.Done():
			s.log.Info("LiveStrategyRunner: context cancelled, exiting")
			return

		case bar, ok := <-p.barCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: bar channel closed, exiting")
				return
			}
			// LIVE-1: extra-symbol context windows also use finalized bars only.
			if bar.Closed && p.extraSymbolSet[bar.Symbol] && bar.Period == p.cfg.Timeframe {
				handleExtraSymbolBar(bar, p.extraBars)
				continue
			}
			if !shouldRunOnBar(bar, p.cfg.Symbol, p.cfg.Timeframe) {
				continue
			}
			// Per-bar entitlement revalidation for marketplace strategies (task 4).
			// Revoked → session self-terminates; positions are NOT auto-closed (license boundary).
			if p.cfg.EntitlementCheck != nil && !p.cfg.EntitlementCheck(p.runCtx) {
				s.log.Warn("LiveStrategyRunner: entitlement revoked, stopping session",
					zap.String("run_id", p.cfg.RunID.String()),
					zap.String("account_id", p.cfg.AccountID))
				return
			}
			s.handleBar(p.runCtx, p.cfg, bar, p.bars, p.session, p.firstBar, p.activeSess, p.extraBars)

		case tick, ok := <-p.tickCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: tick channel closed")
				p.tickCh = nil
				continue
			}
			if tick.Symbol != p.cfg.Symbol {
				continue
			}
			s.handleTick(p.runCtx, p.cfg, tick, p.session, p.firstBar, p.activeSess)

		case evt, ok := <-p.tradeCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: trade channel closed")
				p.tradeCh = nil
				continue
			}
			if evt.Symbol != p.cfg.Symbol {
				continue
			}
			s.handleTrade(p.runCtx, p.cfg, evt, p.session, p.firstBar, p.activeSess)
		}
	}
}

func (s *StrategyExecutionServer) detectExecModels(ctx context.Context, cfg LiveStrategyConfig) (needsTick, needsTrade bool) {
	needsTick = cfg.Models&ExecModelTick != 0
	needsTrade = cfg.Models&ExecModelTrade != 0
	if cfg.Models == 0 && cfg.Code != "" {
		var cachedBytecode []byte
		if cfg.StrategyID != "" && s.importedRepo != nil {
			if sid, parseErr := uuid.Parse(cfg.StrategyID); parseErr == nil {
				cachedBytecode, _ = s.importedRepo.GetBytecode(ctx, sid)
			}
		}
		probe, _, probeErr := mql2go.CompileMQLCached(cfg.Code, cachedBytecode)
		if probeErr == nil {
			needsTick = probe.HasOnTick()
			needsTrade = probe.Bytecode().OnTrade >= 0
		}
	}
	return
}

func (s *StrategyExecutionServer) subscribeTickUpdates(accountID string, needsTick bool) (<-chan *mthub.TickUpdate, func()) {
	if !needsTick || s.mtHub == nil {
		return nil, nil
	}
	ts, ok := interface{}(s.mtHub).(LiveTickSubscriber)
	if !ok {
		return nil, nil
	}
	return ts.SubscribeTickUpdates(accountID)
}

func (s *StrategyExecutionServer) subscribeTradeEvents(accountID string, needsTrade bool) (<-chan *mthub.BrokerTradeEvent, func()) {
	if !needsTrade || s.mtHub == nil {
		return nil, nil
	}
	ts, ok := interface{}(s.mtHub).(LiveTradeSubscriber)
	if !ok {
		return nil, nil
	}
	return ts.SubscribeTradeEvents(accountID)
}

func initExtraBars(cfg LiveStrategyConfig) (map[string][]liveBar, map[string]bool) {
	extraBars := make(map[string][]liveBar, len(cfg.ExtraSymbols))
	extraSymbolSet := make(map[string]bool, len(cfg.ExtraSymbols))
	for _, sym := range cfg.ExtraSymbols {
		if sym != "" && sym != cfg.Symbol {
			extraSymbolSet[sym] = true
			extraBars[sym] = make([]liveBar, 0, maxContextBars)
		}
	}
	return extraBars, extraSymbolSet
}

func handleExtraSymbolBar(bar *mthub.BarUpdate, extraBars map[string][]liveBar) {
	ew := extraBars[bar.Symbol]
	appendDedupBar(&ew, liveBar{
		open:     bar.Open.String(),
		high:     bar.High.String(),
		low:      bar.Low.String(),
		close:    bar.Close.String(),
		volume:   strconv.FormatFloat(bar.Volume, 'f', -1, 64),
		openTime: bar.OpenTime,
	})
	extraBars[bar.Symbol] = ew
}

// appendDedupBar appends a bar to the window with three-state deduplication:
//   - openTime < last → skip (out-of-order/replay)
//   - openTime == last → replace last bar (real-time stream is authoritative over backfill snapshot)
//   - openTime > last → append (new bar)
func appendDedupBar(bars *[]liveBar, b liveBar) {
	n := len(*bars)
	if n > 0 && b.openTime < (*bars)[n-1].openTime {
		return
	}
	if n > 0 && b.openTime == (*bars)[n-1].openTime {
		(*bars)[n-1] = b
		return
	}
	*bars = append(*bars, b)
	if len(*bars) > maxContextBars {
		*bars = (*bars)[len(*bars)-maxContextBars:]
	}
}
