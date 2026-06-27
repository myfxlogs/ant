package strategy

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// findStrategyTypeName parses Go source code and returns the name of the
// type that implements sdk.Strategy (the type with an OnInit receiver method).
func findStrategyTypeName(code string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "strategy.go", code, 0)
	if err != nil {
		return "", fmt.Errorf("parse strategy code: %w", err)
	}
	for _, decl := range f.Decls {
		fnDecl, ok := decl.(*ast.FuncDecl)
		if !ok || fnDecl.Recv == nil || len(fnDecl.Recv.List) == 0 {
			continue
		}
		if fnDecl.Name.Name != "OnInit" {
			continue
		}
		switch t := fnDecl.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				return ident.Name, nil
			}
		case *ast.Ident:
			return t.Name, nil
		}
	}
	return "", fmt.Errorf("no type with OnInit method found in strategy code")
}

// generateBacktestHarness generates a Go source file (package main) that:
//   - reads ExecuteBacktestRequest from stdin (proto binary)
//   - instantiates the strategy by type name
//   - converts K-lines to []sdk.Bar
//   - runs backtest.Engine
//   - writes ExecuteBacktestResponse to stdout (proto binary)
//
// This file is compiled alongside the user's strategy code via `go run`.
func generateBacktestHarness(strategyTypeName string) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/strategy/backtest"
	"anttrader/strategy/sdk"
)

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(1)
	}
	var req antv1.ExecuteBacktestRequest
	if err := proto.Unmarshal(input, &req); err != nil {
		os.Exit(2)
	}

	bars := make([]sdk.Bar, len(req.Klines))
	for i, k := range req.Klines {
		bars[i] = sdk.Bar{
			Open:      decimal.NewFromFloat(k.Open),
			High:      decimal.NewFromFloat(k.High),
			Low:       decimal.NewFromFloat(k.Low),
			Close:     decimal.NewFromFloat(k.Close),
			Volume:    int64(k.Volume),
			Timestamp: k.OpenTimeMs,
		}
	}

	cfg := backtest.Config{
		Symbol:         req.Symbol,
		Timeframe:      req.Timeframe,
		InitialCapital: decimal.NewFromFloat(req.InitialCapital),
		Leverage:       int32(req.Leverage),
		Commission:     decimal.NewFromFloat(req.Commission),
		Slippage:       decimal.NewFromFloat(req.SlippageRate),
		SwapRate:       decimal.NewFromFloat(req.SwapRate),
		StrictMode:     req.StrictMode,
	}
	if req.StartDateMs > 0 {
		cfg.StartDate = time.UnixMilli(req.StartDateMs)
	}
	if req.EndDateMs > 0 {
		cfg.EndDate = time.UnixMilli(req.EndDateMs)
	}

	strategy := &%s{}

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
		resp.Metrics.TotalPnlAbsolute = result.Metrics.TotalReturn * req.InitialCapital
	}

	for _, ep := range result.Equity {
		val, _ := ep.Equity.Float64()
		resp.EquityCurve = append(resp.EquityCurve, val)
	}

	for i, t := range result.Trades {
		side := "BUY"
		if t.Side == sdk.SideSell {
			side = "SELL"
		}
		vol, _ := t.Volume.Float64()
		entryPrice, _ := t.EntryPrice.Float64()
		exitPrice, _ := t.ExitPrice.Float64()
		pnl, _ := t.Profit.Float64()
		comm, _ := t.Commission.Float64()
		resp.Trades = append(resp.Trades, &antv1.ExecuteBacktestTrade{
			Ticket:     int64(i + 1),
			Side:       side,
			Volume:     vol,
			OpenTsMs:   t.EntryTime.UnixMilli(),
			OpenPrice:  entryPrice,
			CloseTsMs:  t.ExitTime.UnixMilli(),
			ClosePrice: exitPrice,
			Pnl:        pnl,
			Commission: comm,
			Reason:     t.Comment,
		})
	}

	out, err := proto.Marshal(resp)
	if err != nil {
		os.Exit(3)
	}
	os.Stdout.Write(out)
}
`, strategyTypeName)
}

// generateLiveHarness generates a Go source file (package main) that:
//   - reads ExecuteLiveRequest from stdin (proto binary)
//   - instantiates the strategy by type name
//   - builds sdk.BarSeries from LiveStrategyContext OHLCV arrays
//   - calls OnInit then OnBar with the latest bar
//   - converts sdk.Signal to antv1.StrategySignal
//   - writes ExecuteLiveResponse to stdout (proto binary)
func generateLiveHarness(strategyTypeName string) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"io"
	"os"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/strategy/runner"
	"anttrader/strategy/sdk"
)

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(1)
	}
	var req antv1.ExecuteLiveRequest
	if err := proto.Unmarshal(input, &req); err != nil {
		os.Exit(2)
	}

	lctx := req.GetContext()
	if lctx == nil {
		resp := &antv1.ExecuteLiveResponse{Success: false, Error: "no context provided"}
		out, _ := proto.Marshal(resp)
		os.Stdout.Write(out)
		return
	}

	// Build bars from LiveStrategyContext OHLCV arrays.
	n := len(lctx.Close)
	bars := make([]sdk.Bar, n)
	for i := 0; i < n; i++ {
		bars[i] = sdk.Bar{
			Open:      decimal.NewFromFloat(lctx.Open[i]),
			High:      decimal.NewFromFloat(lctx.High[i]),
			Low:       decimal.NewFromFloat(lctx.Low[i]),
			Close:     decimal.NewFromFloat(lctx.Close[i]),
			Volume:    int64(lctx.Volume[i]),
			Timestamp: lctx.BarTimesMs[i],
		}
	}

	// Build params map from LiveParam list.
	params := make(map[string]string)
	for _, p := range lctx.GetParams() {
		params[p.GetKey()] = p.GetValue()
	}

	cfg := runner.Config{
		Symbol:    lctx.Symbol,
		Timeframe: lctx.Timeframe,
		Params:    params,
		Mode:      lctx.Mode,
	}

	r := runner.New(cfg)
	strategy := &%s{}
	r.SetStrategy(strategy)

	if err := r.Init(context.Background()); err != nil {
		resp := &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
		out, _ := proto.Marshal(resp)
		os.Stdout.Write(out)
		return
	}

	barSeries := sdk.BarsToSlice(bars)
	sig, err := r.OnBar(context.Background(), barSeries, lctx.Timeframe)
	if err != nil {
		resp := &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
		out, _ := proto.Marshal(resp)
		os.Stdout.Write(out)
		return
	}

	_ = r.Deinit(context.Background(), "bar_complete")

	resp := &antv1.ExecuteLiveResponse{Success: true}
	if sig != nil {
		ss := signalToProto(sig, lctx.Symbol)
		resp.Signal = ss
		resp.Signals = []*antv1.StrategySignal{ss}
	}

	out, err := proto.Marshal(resp)
	if err != nil {
		os.Exit(3)
	}
	os.Stdout.Write(out)
}

func signalToProto(sig *sdk.Signal, symbol string) *antv1.StrategySignal {
	if sig == nil {
		return nil
	}
	signalType := "hold"
	switch sig.Action {
	case sdk.ActionBuy:
		signalType = "buy"
	case sdk.ActionSell:
		signalType = "sell"
	case sdk.ActionBuyLimit:
		signalType = "buy_limit"
	case sdk.ActionSellLimit:
		signalType = "sell_limit"
	case sdk.ActionBuyStop:
		signalType = "buy_stop"
	case sdk.ActionSellStop:
		signalType = "sell_stop"
	case sdk.ActionClose:
		signalType = "close"
	case sdk.ActionModify:
		signalType = "modify"
	case sdk.ActionCancel:
		signalType = "cancel"
	case sdk.ActionCloseAll:
		signalType = "close_all"
	case sdk.ActionCancelAll:
		signalType = "cancel_all"
	}
	vol, _ := sig.Volume.Float64()
	price, _ := sig.Price.Float64()
	sl, _ := sig.StopLoss.Float64()
	tp, _ := sig.TakeProfit.Float64()
	sym := sig.Symbol
	if sym == "" {
		sym = symbol
	}
	return &antv1.StrategySignal{
		Symbol:         sym,
		SignalType:     signalType,
		Volume:         vol,
		Price:          price,
		StopLoss:       sl,
		TakeProfit:     tp,
		ExecutedTicket: sig.OrderTicket,
	}
}
`, strategyTypeName)
}
