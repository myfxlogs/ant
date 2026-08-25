// live_context_build.go — LiveStrategyContext builder and helpers extracted from live_context.go.
package strategy

import (
	"context"
	"fmt"
	"time"

	antv1 "alphaforge/gen/proto/ant/v1"
)

func (s *StrategyExecutionServer) buildLiveContext(ctx context.Context, cfg LiveStrategyConfig, bars []liveBar, extraBars map[string][]liveBar) (*antv1.LiveStrategyContext, error) {
	n := len(bars)
	closeVals := make([]string, n)
	openVals := make([]string, n)
	highVals := make([]string, n)
	lowVals := make([]string, n)
	volVals := make([]string, n)
	times := make([]int64, n)
	for i, b := range bars {
		closeVals[i] = b.close
		openVals[i] = b.open
		highVals[i] = b.high
		lowVals[i] = b.low
		volVals[i] = b.volume
		times[i] = b.openTime
	}
	lctx := &antv1.LiveStrategyContext{
		Close:      closeVals,
		Open:       openVals,
		High:       highVals,
		Low:        lowVals,
		Volume:     volVals,
		BarTimesMs: times,
		Symbol:     cfg.Symbol,
		Timeframe:  cfg.Timeframe,
		Mode:       cfg.Mode,
		Params:     buildLiveParams(cfg.Params),
	}
	// VM-TRADE-CONTEXT-6: inject account identity from authoritative source
	// (mt_accounts table) so AccountNumber()/AccountCompany() work in OnInit.
	// VM-API-TRUTH-3: inject IsDemo/IsConnected/IsTradeAllowed from authoritative
	// account_status/account_type columns, not hardcoded constants.
	// VM-API-TRUTH-3 round 4: lookups return (value, error) — DB query errors
	// must propagate and block execution (fail-closed), not be confused with
	// real false/0/"". Investor accounts get IsTradeAllowed=false even if
	// account_status == 'connected'.
	if cfg.Mode == "live" {
		if s.accountLoginLookup == nil {
			return nil, fmt.Errorf("live mode: accountLoginLookup not configured")
		}
		login, err := s.accountLoginLookup(ctx, cfg.AccountID)
		if err != nil {
			return nil, fmt.Errorf("live mode: account login lookup failed for account %s: %w", cfg.AccountID, err)
		}
		if login == 0 {
			return nil, fmt.Errorf("live mode: account login lookup returned 0 for account %s", cfg.AccountID)
		}
		lctx.Login = login
		company, err := s.resolveBrokerCompanyErr(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("live mode: broker company lookup failed for account %s: %w", cfg.AccountID, err)
		}
		if company == "" {
			return nil, fmt.Errorf("live mode: broker company lookup returned empty for account %s", cfg.AccountID)
		}
		lctx.Company = company
		if s.accountIsDemoLookup == nil {
			return nil, fmt.Errorf("live mode: accountIsDemoLookup not configured")
		}
		isDemo, err := s.accountIsDemoLookup(ctx, cfg.AccountID)
		if err != nil {
			return nil, fmt.Errorf("live mode: account isDemo lookup failed for account %s: %w", cfg.AccountID, err)
		}
		lctx.IsDemo = isDemo
		// VM-API-TRUTH-3: IsConnected from authoritative account_status column.
		if s.accountConnectedLookup == nil {
			return nil, fmt.Errorf("live mode: accountConnectedLookup not configured")
		}
		isConnected, err := s.accountConnectedLookup(ctx, cfg.AccountID)
		if err != nil {
			return nil, fmt.Errorf("live mode: account connected lookup failed for account %s: %w", cfg.AccountID, err)
		}
		lctx.IsConnected = isConnected
		// VM-API-TRUTH-3 round 4: IsTradeAllowed must account for is_investor.
		// Investor/read-only accounts cannot trade even when connected.
		if s.accountTradeAllowedLookup == nil {
			return nil, fmt.Errorf("live mode: accountTradeAllowedLookup not configured")
		}
		tradeAllowed, err := s.accountTradeAllowedLookup(ctx, cfg.AccountID)
		if err != nil {
			return nil, fmt.Errorf("live mode: account trade allowed lookup failed for account %s: %w", cfg.AccountID, err)
		}
		// VM-API-TRUTH-3 round 5: accountIsInvestorLookup is REQUIRED in live
		// mode. Without it, investor accounts could bypass the trade
		// permission gate. Investor accounts get IsTradeAllowed=false even
		// if tradeAllowed=true.
		if s.accountIsInvestorLookup == nil {
			return nil, fmt.Errorf("live mode: accountIsInvestorLookup not configured (required for trade permission safety)")
		}
		isInvestor, invErr := s.accountIsInvestorLookup(ctx, cfg.AccountID)
		if invErr != nil {
			return nil, fmt.Errorf("live mode: account is_investor lookup failed for account %s: %w", cfg.AccountID, invErr)
		}
		if isInvestor {
			tradeAllowed = false
		}
		lctx.IsTradeAllowed = tradeAllowed
	} else {
		// Backtest/paper: lookups are optional; use whatever is available.
		// Errors are non-fatal in simulation mode (fail-open for testing).
		if s.accountLoginLookup != nil {
			if login, err := s.accountLoginLookup(ctx, cfg.AccountID); err == nil {
				lctx.Login = login
			}
		}
		lctx.Company = s.resolveBrokerCompany(ctx, cfg)
		if s.accountIsDemoLookup != nil {
			if isDemo, err := s.accountIsDemoLookup(ctx, cfg.AccountID); err == nil {
				lctx.IsDemo = isDemo
			}
		}
		if s.accountConnectedLookup != nil {
			if isConnected, err := s.accountConnectedLookup(ctx, cfg.AccountID); err == nil {
				lctx.IsConnected = isConnected
			}
		}
		if s.accountTradeAllowedLookup != nil {
			if tradeAllowed, err := s.accountTradeAllowedLookup(ctx, cfg.AccountID); err == nil {
				if s.accountIsInvestorLookup != nil {
					if isInvestor, invErr := s.accountIsInvestorLookup(ctx, cfg.AccountID); invErr == nil && isInvestor {
						tradeAllowed = false
					}
				}
				lctx.IsTradeAllowed = tradeAllowed
			}
		}
	}
	if n > 0 {
		lctx.CurrentPrice = closeVals[n-1]
	}
	if err := s.backfillContextStrings(cfg.AccountID, &lctx.Equity, &lctx.Balance, &lctx.Margin, &lctx.FreeMargin, &lctx.Positions, &lctx.PendingOrders); err != nil && cfg.Mode == "live" {
		return nil, err
	}
	s.backfillSymbolInfo(cfg, lctx)
	lctx.Symbols = buildSymbolSeries(extraBars)
	return lctx, nil
}

func buildLiveParams(params map[string]string) []*antv1.LiveParam {
	if len(params) == 0 {
		return nil
	}
	out := make([]*antv1.LiveParam, 0, len(params))
	for k, v := range params {
		out = append(out, &antv1.LiveParam{Key: k, Value: v})
	}
	return out
}

func buildSymbolSeries(extraBars map[string][]liveBar) []*antv1.LiveSymbolSeries {
	if len(extraBars) == 0 {
		return nil
	}
	out := make([]*antv1.LiveSymbolSeries, 0, len(extraBars))
	for sym, bars := range extraBars {
		if len(bars) == 0 {
			continue
		}
		n := len(bars)
		closeVals := make([]string, n)
		openVals := make([]string, n)
		highVals := make([]string, n)
		lowVals := make([]string, n)
		volVals := make([]string, n)
		for i, b := range bars {
			closeVals[i] = b.close
			openVals[i] = b.open
			highVals[i] = b.high
			lowVals[i] = b.low
			volVals[i] = b.volume
		}
		out = append(out, &antv1.LiveSymbolSeries{
			Symbol: sym,
			Close:  closeVals,
			Open:   openVals,
			High:   highVals,
			Low:    lowVals,
			Volume: volVals,
		})
	}
	return out
}

// backfillSymbolInfo populates Point/Digits/ContractSize/StopsLevel on
// LiveStrategyContext from the pre-fetched symbol params (W2: no per-event RPC).
// Falls back to a one-shot 5s-timeout fetch if startup pre-fetch failed.
func (s *StrategyExecutionServer) backfillSymbolInfo(cfg LiveStrategyConfig, lctx *antv1.LiveStrategyContext) {
	param := cfg.SymbolParam
	if param == nil && s.mtHub != nil && cfg.AccountID != "" && cfg.Symbol != "" {
		fetchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		param, _ = s.mtHub.CachedSymbolParam(fetchCtx, cfg.AccountID, cfg.Symbol)
		cancel()
	}
	if param == nil {
		return
	}
	lctx.Point = param.PointValue.String()
	lctx.Digits = param.Digits
	lctx.ContractSize = param.ContractSize.String()
	lctx.StopsLevel = param.StopLevel
}

// backfillTickSymbolInfo populates Point/Digits/ContractSize/StopsLevel on
// TickContext from the pre-fetched symbol params (W2: no per-event RPC).
// Falls back to a one-shot 5s-timeout fetch if startup pre-fetch failed.
func (s *StrategyExecutionServer) backfillTickSymbolInfo(cfg LiveStrategyConfig, tctx *antv1.TickContext) {
	param := cfg.SymbolParam
	if param == nil && s.mtHub != nil && cfg.AccountID != "" && cfg.Symbol != "" {
		fetchCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		param, _ = s.mtHub.CachedSymbolParam(fetchCtx, cfg.AccountID, cfg.Symbol)
		cancel()
	}
	if param == nil {
		return
	}
	tctx.Point = param.PointValue.String()
	tctx.Digits = param.Digits
	tctx.ContractSize = param.ContractSize.String()
	tctx.StopsLevel = param.StopLevel
}
