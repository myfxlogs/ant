package strategy

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// validateBacktestRequest checks preconditions before creating a backtest run:
// decimal input validity, symbol availability, market data existence, and
// rejection of unimplemented execution config values.
func (s *StrategyExecutionServer) validateBacktestRequest(ctx context.Context, req *connect.Request[antv1.StartBacktestRunRequest]) error {
	// Empty strings cause "can't convert to decimal" panics deep in the engine.
	if req.Msg.InitialCapital != "" {
		if _, err := decimal.NewFromString(req.Msg.InitialCapital); err != nil {
			return connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("invalid initial_capital: %q", req.Msg.InitialCapital))
		}
	}
	if req.Msg.Symbol == "" {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("please select a symbol and timeframe from the chart before starting a backtest"))
	}
	// Whitelist validation for execution config values (reject unknown values).
	if cfg := req.Msg.GetExecutionConfig(); cfg != nil {
		validFillRules := map[string]bool{"": true, "bar_close": true, "market": true, "limit": true}
		if !validFillRules[cfg.GetFillRule()] {
			return connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("invalid fill_rule %q: must be one of bar_close, market, limit", cfg.GetFillRule()))
		}
		validSimModes := map[string]bool{"": true, "KLINE_RANGE": true, "OHLC_PATH": true}
		if !validSimModes[cfg.GetSimulationMode()] {
			if cfg.GetSimulationMode() == "DATASET" {
				return connect.NewError(connect.CodeInvalidArgument,
					fmt.Errorf("invalid simulation_mode %q: DATASET has been renamed to OHLC_PATH", cfg.GetSimulationMode()))
			}
			return connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("invalid simulation_mode %q: must be one of KLINE_RANGE, OHLC_PATH", cfg.GetSimulationMode()))
		}
		validSignalTimings := map[string]bool{"": true, "next_bar_open": true, "same_bar_close": true}
		if !validSignalTimings[cfg.GetSignalTiming()] {
			return connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("invalid signal_timing %q: must be one of next_bar_open, same_bar_close", cfg.GetSignalTiming()))
		}
	}
	if s.marketDataRepo == nil {
		return nil
	}
	tf := req.Msg.Timeframe
	if tf == "" {
		tf = "H1"
	}
	from, to := backtestDateRange(req.Msg)
	bars, _ := s.marketDataRepo.GetKlines(ctx, req.Msg.Symbol, "", tf, from, to, 2)
	if len(bars) >= 2 {
		return nil
	}
	available := s.availableSymbols(ctx, tf, nil)
	if available != "" {
		return connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("no market data for %s %s in the selected date range — available pairs with data: %s",
				req.Msg.Symbol, tf, available))
	}
	return connect.NewError(connect.CodeFailedPrecondition,
		fmt.Errorf("no market data available — please connect your MT account to stream quotes first"))
}

// backtestDateRange extracts the date range from the request, or returns nil,nil
// if not set (meaning the backtest worker will use the full available range).
func backtestDateRange(msg *antv1.StartBacktestRunRequest) (from, to *time.Time) {
	if msg.From != nil {
		t := msg.From.AsTime()
		from = &t
	}
	if msg.To != nil {
		t := msg.To.AsTime()
		to = &t
	}
	return
}

// availableSymbols returns a comma-separated list of pairs that have K-line data
// for the given timeframe, to help the user choose a valid symbol.
func (s *StrategyExecutionServer) availableSymbols(ctx context.Context, timeframe string, from *time.Time) string {
	candidates := []string{"EURUSDm", "GBPUSDm", "EURUSD", "XAUUSDm", "BTCUSDm", "USDJPYm"}
	var found []string
	for _, sym := range candidates {
		if bars, _ := s.marketDataRepo.GetKlines(ctx, sym, "", timeframe, from, nil, 1); len(bars) > 0 {
			found = append(found, sym)
		}
		if len(found) >= 5 {
			break
		}
	}
	if len(found) == 0 {
		return ""
	}
	result := ""
	for i, s := range found {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

// HasBacktestData checks whether the requested symbol+timeframe has K-line data.
// Exported for the frontend to pre-check before showing the backtest form.
func (s *StrategyExecutionServer) HasBacktestData(ctx context.Context, symbol, timeframe string) bool {
	if s.marketDataRepo == nil {
		return false
	}
	bars, _ := s.marketDataRepo.GetKlines(ctx, symbol, "", timeframe, nil, nil, 1)
	return len(bars) > 0
}
