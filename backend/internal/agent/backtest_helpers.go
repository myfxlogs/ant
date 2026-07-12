package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/backtest"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go"
)

// runVMBacktest executes a VM backtest with the given runner and config.
// Shared by SubmitStrategy and Generator.
func runVMBacktest(
	ctx context.Context,
	runner *mql2go.VMRunner,
	cfg *antv1.AgentBacktestConfig,
	bars []sdk.Bar,
	params map[string]string,
) (*backtest.Result, error) {
	initialCapital := parseDecimalDefault(cfg.InitialCapital, "10000")
	leverage := int32(100)
	if cfg.Leverage != "" {
		if d, err := decimal.NewFromString(cfg.Leverage); err == nil {
			leverage = int32(d.IntPart())
		}
	}
	commission := parseDecimalDefault(cfg.Commission, "0.0003")
	slippage := parseDecimalDefault(cfg.Slippage, "0.00001")

	btConfig := backtest.Config{
		Symbol:         cfg.Symbol,
		Timeframe:      cfg.Timeframe,
		InitialCapital: initialCapital,
		Leverage:       leverage,
		Commission:     commission,
		Slippage:       slippage,
		SwapRate:       decimal.RequireFromString("0.00001"),
		StrictMode:     cfg.StrictMode,
		Params:         params,
	}

	backtest.DeriveSymbolInfoFromBars(&btConfig, bars)

	vmCtx, vmCancel := context.WithTimeout(ctx, 30*time.Second)
	defer vmCancel()

	engine := backtest.New(btConfig, runner, bars)
	result, err := engine.Run(vmCtx)
	if err != nil {
		return nil, fmt.Errorf("backtest engine: %w", err)
	}
	return result, nil
}

func buildBacktestResultProto(r *backtest.Result) *antv1.AgentBacktestResult {
	resp := &antv1.AgentBacktestResult{
		Success: true,
	}
	if r.Metrics != nil {
		resp.TotalReturn = r.Metrics.TotalReturn
		resp.AnnualReturn = r.Metrics.AnnualReturn
		resp.MaxDrawdown = r.Metrics.MaxDrawdown
		resp.SharpeRatio = r.Metrics.SharpeRatio
		resp.WinRate = r.Metrics.WinRate
		resp.ProfitFactor = r.Metrics.ProfitFactor
		resp.TotalTrades = r.Metrics.TotalTrades
		resp.WinningTrades = r.Metrics.WinningTrades
		resp.LosingTrades = r.Metrics.LosingTrades
		totalPnl := r.Config.InitialCapital.Mul(decimal.NewFromFloat(r.Metrics.TotalReturn)) // float64 boundary from backtest VM — fix upstream
		resp.TotalPnlAbsolute = totalPnl.String()
	}
	for _, ep := range r.Equity {
		resp.EquityCurve = append(resp.EquityCurve, ep.Equity.String())
		resp.EquityTimesMs = append(resp.EquityTimesMs, ep.Time.UnixMilli())
	}
	for i, t := range r.Trades {
		side := "BUY"
		if t.Side == sdk.SideSell {
			side = "SELL"
		}
		resp.Trades = append(resp.Trades, &antv1.AgentTrade{
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
	return resp
}
