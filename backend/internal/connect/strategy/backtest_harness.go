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
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/strategy/backtest"
	"anttrader/strategy/sdk"
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
`, strategyTypeName)
}

// generateLiveHarness generates a Go source file (package main) that:
//   - reads ExecuteLiveRequest from stdin (length-prefixed proto binary)
//   - instantiates the strategy by type name
//   - dispatches by request_type: BAR → OnBar, TICK → OnTick, TRADE → OnTrade, TIMER → OnTimer
//   - maintains its own bar window (delta-bar protocol for BAR requests)
//   - calls OnInit once at startup (initialized from first request's bar_context)
//   - writes length-prefixed ExecuteLiveResponse to stdout
//   - calls OnDeinit on EOF or error
func generateLiveHarness(strategyTypeName string) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/strategy/runner"
	"anttrader/strategy/sdk"
)

const maxBarWindow = 500

// barWindow persists across requests — the harness maintains its own rolling bar window.
var barWindow []sdk.Bar

// initRunner initializes the strategy runner from the first request's bar context.
// Returns the runner and config, or an error response.
func initRunner(strategy *%[1]s, req *antv1.ExecuteLiveRequest) (*runner.Runner, *antv1.ExecuteLiveResponse) {
	bctx := req.GetBarContext()
	if bctx == nil {
		return nil, &antv1.ExecuteLiveResponse{Success: false, Error: "first request must have bar_context for initialization"}
	}

	params := make(map[string]string)
	for _, p := range bctx.GetParams() {
		params[p.GetKey()] = p.GetValue()
	}

	cfg := runner.Config{
		Symbol:    bctx.Symbol,
		Timeframe: bctx.Timeframe,
		Params:    params,
		Mode:      bctx.Mode,
	}

	r := runner.New(cfg)
	r.SetStrategy(strategy)

	if err := r.Init(context.Background()); err != nil {
		return nil, &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
	}

	return r, nil
}

func main() {
	strategy := &%[1]s{}

	// Read the first request — must be BAR type for initialization.
	req, err := readRequest(os.Stdin)
	if err != nil {
		if err == io.EOF {
			return
		}
		fmt.Fprintf(os.Stderr, "live harness: read first request: %%v\n", err)
		os.Exit(1)
	}

	r, errResp := initRunner(strategy, req)
	if errResp != nil {
		writeResponse(os.Stdout, errResp)
		return
	}

	// Process events in a loop — state is preserved across events.
	// Dispatch by request_type.
	for {
		resp := dispatch(r, req)
		if err := writeResponse(os.Stdout, resp); err != nil {
			fmt.Fprintf(os.Stderr, "live harness: write response: %%v\n", err)
			break
		}

		// Read next request (blocks until host writes).
		req, err = readRequest(os.Stdin)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "live harness: read request: %%v\n", err)
			break
		}
	}

	_ = r.Deinit(context.Background(), "stream_end")
}

// dispatch routes a request to the appropriate handler based on request_type.
func dispatch(r *runner.Runner, req *antv1.ExecuteLiveRequest) *antv1.ExecuteLiveResponse {
	switch req.GetRequestType() {
	case antv1.RequestType_REQUEST_TYPE_BAR:
		return handleBar(r, req.GetBarContext())
	case antv1.RequestType_REQUEST_TYPE_TICK:
		return handleTick(r, req.GetTickContext())
	case antv1.RequestType_REQUEST_TYPE_TRADE:
		return handleTrade(r, req.GetTradeContext())
	case antv1.RequestType_REQUEST_TYPE_TIMER:
		return handleTimer(r, req.GetTimerContext())
	default:
		// Backward compat: if request_type is unset, try bar_context.
		if bctx := req.GetBarContext(); bctx != nil {
			return handleBar(r, bctx)
		}
		return &antv1.ExecuteLiveResponse{Success: false, Error: "unknown request type"}
	}
}

// ── BAR handler ─────────────────────────────────────────────────────

func handleBar(r *runner.Runner, lctx *antv1.LiveStrategyContext) *antv1.ExecuteLiveResponse {
	if lctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "bar_context missing"}
	}
	updateRunnerState(r, lctx)

	if len(lctx.DeltaBars) > 0 {
		// Delta-bar protocol: append new bar(s) to the rolling window.
		for _, db := range lctx.DeltaBars {
			barWindow = append(barWindow, sdk.Bar{
				Open:      mustDecimal(db.Open),
				High:      mustDecimal(db.High),
				Low:       mustDecimal(db.Low),
				Close:     mustDecimal(db.Close),
				Volume:    mustInt64(db.Volume),
				Timestamp: db.BarTimeMs,
			})
		}
		if len(barWindow) > maxBarWindow {
			barWindow = barWindow[len(barWindow)-maxBarWindow:]
		}
	} else {
		// Full OHLCV rebuild (first bar or backward compat).
		n := len(lctx.Close)
		barWindow = make([]sdk.Bar, n)
		for i := 0; i < n; i++ {
			barWindow[i] = sdk.Bar{
				Open:      mustDecimal(lctx.Open[i]),
				High:      mustDecimal(lctx.High[i]),
				Low:       mustDecimal(lctx.Low[i]),
				Close:     mustDecimal(lctx.Close[i]),
				Volume:    mustInt64(lctx.Volume[i]),
				Timestamp: lctx.BarTimesMs[i],
			}
		}
	}

	barSeries := sdk.BarsToSlice(barWindow)
	sig, err := r.OnBar(context.Background(), barSeries, lctx.Timeframe)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
	}
	return signalResponse(sig, lctx.Symbol)
}

// ── TICK handler ────────────────────────────────────────────────────

func handleTick(r *runner.Runner, tctx *antv1.TickContext) *antv1.ExecuteLiveResponse {
	if tctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "tick_context missing"}
	}
	updateAccountFromTick(r, tctx)

	bid := mustDecimal(tctx.Bid)
	ask := mustDecimal(tctx.Ask)
	sig, err := r.OnTick(context.Background(), bid, ask)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
	}
	return signalResponse(sig, tctx.Symbol)
}

// ── TRADE handler ───────────────────────────────────────────────────

func handleTrade(r *runner.Runner, evctx *antv1.TradeContext) *antv1.ExecuteLiveResponse {
	if evctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "trade_context missing"}
	}
	updateAccountFromTrade(r, evctx)

	side := sdk.SideBuy
	if evctx.Side == "sell" {
		side = sdk.SideSell
	}

	event := sdk.TradeEvent{
		Ticket:     evctx.Ticket,
		Symbol:     evctx.Symbol,
		EventType:  tradeEventType(evctx.EventType),
		Side:       side,
		Volume:     mustDecimal(evctx.Volume),
		Price:      mustDecimal(evctx.Price),
		StopLoss:   mustDecimal(evctx.StopLoss),
		TakeProfit: mustDecimal(evctx.TakeProfit),
		Profit:     mustDecimal(evctx.Profit),
		Commission: mustDecimal(evctx.Commission),
		Swap:       mustDecimal(evctx.Swap),
	}

	sig, err := r.OnTrade(context.Background(), event)
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
	}
	return signalResponse(sig, evctx.Symbol)
}

func tradeEventType(s string) sdk.TradeEventType {
	switch s {
	case "fill":
		return sdk.TradeFilled
	case "close":
		return sdk.TradeClosed
	case "modify":
		return sdk.TradeModified
	case "cancel":
		return sdk.TradeCancelled
	}
	return sdk.TradeFilled
}

// ── TIMER handler ───────────────────────────────────────────────────

func handleTimer(r *runner.Runner, tmctx *antv1.TimerContext) *antv1.ExecuteLiveResponse {
	if tmctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "timer_context missing"}
	}
	updateAccountFromTimer(r, tmctx)

	sig, err := r.OnTimerTick(context.Background())
	if err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}
	}
	return signalResponse(sig, tmctx.Symbol)
}

// ── Helpers ─────────────────────────────────────────────────────────

func updateRunnerState(r *runner.Runner, lctx *antv1.LiveStrategyContext) {
	if lctx == nil {
		return
	}
	positions := make([]sdk.Position, 0, len(lctx.Positions))
	for _, lp := range lctx.Positions {
		side := sdk.SideBuy
		if lp.Side == "sell" {
			side = sdk.SideSell
		}
		positions = append(positions, sdk.Position{
			Ticket:    lp.Ticket,
			Side:      side,
			Volume:    mustDecimal(lp.Volume),
			OpenPrice: mustDecimal(lp.OpenPrice),
		})
	}
	r.UpdateLiveState(lctx.Balance, lctx.Equity, positions)
}

func updateAccountFromTick(r *runner.Runner, tctx *antv1.TickContext) {
	r.UpdateLiveState(tctx.Balance, tctx.Equity, livePositionsToSdk(tctx.Positions))
	r.UpdateTickState(mustDecimal(tctx.Bid), mustDecimal(tctx.Ask))
}

func updateAccountFromTrade(r *runner.Runner, evctx *antv1.TradeContext) {
	r.UpdateLiveState(evctx.Balance, evctx.Equity, livePositionsToSdk(evctx.Positions))
}

func updateAccountFromTimer(r *runner.Runner, tmctx *antv1.TimerContext) {
	r.UpdateLiveState(tmctx.Balance, tmctx.Equity, livePositionsToSdk(tmctx.Positions))
}

func livePositionsToSdk(lps []*antv1.LivePosition) []sdk.Position {
	positions := make([]sdk.Position, 0, len(lps))
	for _, lp := range lps {
		side := sdk.SideBuy
		if lp.Side == "sell" {
			side = sdk.SideSell
		}
		positions = append(positions, sdk.Position{
			Ticket:    lp.Ticket,
			Side:      side,
			Volume:    mustDecimal(lp.Volume),
			OpenPrice: mustDecimal(lp.OpenPrice),
		})
	}
	return positions
}

func signalResponse(sig *sdk.Signal, symbol string) *antv1.ExecuteLiveResponse {
	resp := &antv1.ExecuteLiveResponse{Success: true}
	if sig != nil {
		ss := signalToProto(sig, symbol)
		resp.Signal = ss
		resp.Signals = []*antv1.StrategySignal{ss}
	}
	return resp
}

// readRequest reads a length-prefixed protobuf message from r.
func readRequest(r io.Reader) (*antv1.ExecuteLiveRequest, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	msgLen := binary.BigEndian.Uint32(lenBuf[:])
	buf := make([]byte, msgLen)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	var req antv1.ExecuteLiveRequest
	if err := proto.Unmarshal(buf, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func writeResponse(w io.Writer, resp *antv1.ExecuteLiveResponse) error {
	out, err := proto.Marshal(resp)
	if err != nil {
		return err
	}
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(out)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
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
	sym := sig.Symbol
	if sym == "" {
		sym = symbol
	}
	return &antv1.StrategySignal{
		Symbol:         sym,
		SignalType:     signalType,
		Volume:         sig.Volume.String(),
		Price:          sig.Price.String(),
		StopLoss:       sig.StopLoss.String(),
		TakeProfit:     sig.TakeProfit.String(),
		ExecutedTicket: sig.OrderTicket,
	}
}

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
`, strategyTypeName)
}
