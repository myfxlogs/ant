package strategy

// generateBacktestHarnessBase generates a backtest harness with the given
// strategy creation code and extra imports. The strategy creation code
// should define a `strategy` variable implementing sdk.Strategy.
//
// Compiled path: strategy := &TypeName{}
// ADR-0023: WASM interp path removed — all MQL strategies use VMRunner.
func generateBacktestHarnessBase(strategyCreation, extraImport string) string {
	return backtestHarnessPrelude + extraImport + backtestHarnessImports + strategyCreation + backtestHarnessBody
}

const backtestHarnessPrelude = `package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	`

const backtestHarnessImports = `
)

func mustDecimal(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func mustInt64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func mustInt32(s string) int32 {
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0
	}
	return int32(n)
}

func main() {
	`

const backtestHarnessBody = `

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "backtest: read request: %v\n", err)
		os.Exit(2)
	}
	var req antv1.ExecuteBacktestRequest
	if err := proto.Unmarshal(input, &req); err != nil {
		fmt.Fprintf(os.Stderr, "backtest: unmarshal request: %v\n", err)
		os.Exit(2)
	}

	bars := make([]sdk.Bar, len(req.Klines))
	for i, k := range req.Klines {
		bars[i] = sdk.Bar{
			Open:      mustDecimal(k.Open),
			High:      mustDecimal(k.High),
			Low:       mustDecimal(k.Low),
			Close:     mustDecimal(k.Close),
			Volume:    mustInt64(k.Volume),
			Timestamp: k.OpenTimeMs,
		}
	}

	cfg := backtest.Config{
		Symbol:         req.Symbol,
		Timeframe:      req.Timeframe,
		InitialCapital: mustDecimal(req.InitialCapital),
		Leverage:       mustInt32(req.Leverage),
		Commission:     mustDecimal(req.Commission),
		Slippage:       mustDecimal(req.SlippageRate),
		SwapRate:       mustDecimal(req.SwapRate),
		StrictMode:     req.StrictMode,
		Params:         req.StrategyParams,
	}
	if req.StartDateMs > 0 {
		cfg.StartDate = time.UnixMilli(req.StartDateMs)
	}
	if req.EndDateMs > 0 {
		cfg.EndDate = time.UnixMilli(req.EndDateMs)
	}

	engine := backtest.New(cfg, strategy, bars)
	result, err := engine.Run(context.Background())
	if err != nil {
		resp := &antv1.ExecuteBacktestResponse{
			Success: false,
			Error:   err.Error(),
		}
		out, _ := proto.Marshal(resp)
		os.Stdout.Write(out)
		return
	}

	resp := &antv1.ExecuteBacktestResponse{
		Success: true,
		Metrics: &antv1.ExecuteBacktestMetrics{
			TotalReturn:   result.Metrics.TotalReturn,
			AnnualReturn:  result.Metrics.AnnualReturn,
			MaxDrawdown:   result.Metrics.MaxDrawdown,
			SharpeRatio:   result.Metrics.SharpeRatio,
			WinRate:       result.Metrics.WinRate,
			ProfitFactor:  result.Metrics.ProfitFactor,
			TotalTrades:   result.Metrics.TotalTrades,
			WinningTrades: result.Metrics.WinningTrades,
			LosingTrades:  result.Metrics.LosingTrades,
		},
	}
	if result.Metrics != nil {
		tr, _ := decimal.NewFromString(result.Metrics.TotalReturn)
		totalPnl := mustDecimal(req.InitialCapital).Mul(tr)
		resp.Metrics.TotalPnlAbsolute = totalPnl.String()
	}

	for _, ep := range result.Equity {
		resp.EquityCurve = append(resp.EquityCurve, ep.Equity.String())
		resp.EquityTimesMs = append(resp.EquityTimesMs, ep.Time.UnixMilli())
	}

	for i, t := range result.Trades {
		side := "BUY"
		if t.Side == sdk.SideSell {
			side = "SELL"
		}
		resp.Trades = append(resp.Trades, &antv1.ExecuteBacktestTrade{
			Ticket:     int64(i + 1),
			Side:       side,
			Volume:     t.Volume.String(),
			OpenTsMs:   t.EntryTime.UnixMilli(),
			OpenPrice:  t.EntryPrice.String(),
			CloseTsMs:  t.ExitTime.UnixMilli(),
			ClosePrice: t.ExitPrice.String(),
			Pnl:        t.Profit.String(),
			Commission: t.Commission.String(),
			Reason:     t.Comment,
		})
	}

	// ADR-0023 §5.5 #14: ExecutionAssumptions — transparency panel.
	resp.ExecutionAssumptions = &antv1.ExecutionAssumptions{
		SimulationMode:   "KLINE_RANGE",
		SignalTiming:     "next_bar_open",
		FillRule:         "bar_close",
		ActualCommission: req.Commission,
		ActualSlippage:   req.SlippageRate,
		ActualLeverage:   fmt.Sprintf("%d", req.Leverage),
		TradeDirection:   "both",
	}
	if !req.StrictMode {
		resp.ExecutionAssumptions.SignalTiming = "same_bar_close"
		resp.ExecutionAssumptions.MtfFallbackReason = "strict_mode disabled"
	}

	out, err := proto.Marshal(resp)
	if err != nil {
		os.Exit(3)
	}
	os.Stdout.Write(out)
}
`
