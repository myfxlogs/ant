package strategy

// generateInterpBacktestHarness generates a Go source file (package main) that:
//   - reads serialized IR from stdin (u32 length prefix + IR bytes)
//   - reads ExecuteBacktestRequest from stdin (proto binary, after IR)
//   - deserializes IR via interp.DeserializeIR
//   - creates Interpreter via interp.NewInterpreter
//   - converts K-lines to []sdk.Bar
//   - runs backtest.Engine
//   - writes ExecuteBacktestResponse to stdout (proto binary)
//
// This harness is compiled to wasip1/wasm alongside the interp package.
// No user strategy code needed — the IR IS the strategy.
func generateInterpBacktestHarness() string {
	return `package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/strategy/backtest"
	"anttrader/strategy/sdk"
	"anttrader/tools/mql2go/interp"
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
	var irLenBuf [4]byte
	if _, err := io.ReadFull(os.Stdin, irLenBuf[:]); err != nil {
		fmt.Fprintf(os.Stderr, "interp backtest: read IR length: %v\n", err)
		os.Exit(1)
	}
	irLen := binary.LittleEndian.Uint32(irLenBuf[:])
	irData := make([]byte, irLen)
	if _, err := io.ReadFull(os.Stdin, irData); err != nil {
		fmt.Fprintf(os.Stderr, "interp backtest: read IR data: %v\n", err)
		os.Exit(1)
	}

	ir := interp.DeserializeIR(irData)
	if ir == nil {
		fmt.Fprintln(os.Stderr, "interp backtest: deserialize IR failed")
		os.Exit(1)
	}

	strategy := interp.NewInterpreter(ir)

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "interp backtest: read request: %v\n", err)
		os.Exit(2)
	}
	var req antv1.ExecuteBacktestRequest
	if err := proto.Unmarshal(input, &req); err != nil {
		fmt.Fprintf(os.Stderr, "interp backtest: unmarshal request: %v\n", err)
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
		totalPnl := mustDecimal(req.InitialCapital).Mul(decimal.NewFromFloat(result.Metrics.TotalReturn))
		resp.Metrics.TotalPnlAbsolute = totalPnl.String()
	}

	for _, ep := range result.Equity {
		resp.EquityCurve = append(resp.EquityCurve, ep.Equity.String())
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

	out, err := proto.Marshal(resp)
	if err != nil {
		os.Exit(3)
	}
	os.Stdout.Write(out)
}
`
}
