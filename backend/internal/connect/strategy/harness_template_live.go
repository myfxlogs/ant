package strategy

// generateLiveHarnessBase generates a live harness with the given
// strategy creation code and extra imports. The strategy creation code
// should define a `strategy` variable implementing sdk.Strategy.
//
// Compiled path: strategy := &TypeName{}
// VM path:        strategy := mql2go.NewVMRunner(bc)
func generateLiveHarnessBase(strategyCreation, extraImport string) string {
	return `package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/runner"
	"alphaforge/strategy/sdk"
	` + extraImport + `
)

const maxBarWindow = 500

var barWindow []sdk.Bar

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

func main() {
	` + strategyCreation + `

	req, err := readRequest(os.Stdin)
	if err != nil {
		if err == io.EOF {
			return
		}
		fmt.Fprintf(os.Stderr, "live harness: read first request: %v\n", err)
		os.Exit(1)
	}

	r, errResp := initRunner(strategy, req)
	if errResp != nil {
		writeResponse(os.Stdout, errResp)
		return
	}

	for {
		resp := dispatch(r, req)
		if err := writeResponse(os.Stdout, resp); err != nil {
			fmt.Fprintf(os.Stderr, "live harness: write response: %v\n", err)
			break
		}
		req, err = readRequest(os.Stdin)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "live harness: read request: %v\n", err)
			break
		}
	}

	_ = r.Deinit(context.Background(), "stream_end")
}

func initRunner(strategy sdk.Strategy, req *antv1.ExecuteLiveRequest) (*runner.Runner, *antv1.ExecuteLiveResponse) {
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
		if bctx := req.GetBarContext(); bctx != nil {
			return handleBar(r, bctx)
		}
		return &antv1.ExecuteLiveResponse{Success: false, Error: "unknown request type"}
	}
}

func handleBar(r *runner.Runner, lctx *antv1.LiveStrategyContext) *antv1.ExecuteLiveResponse {
	if lctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "bar_context missing"}
	}
	updateRunnerState(r, lctx)

	if len(lctx.DeltaBars) > 0 {
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

func updateRunnerState(r *runner.Runner, lctx *antv1.LiveStrategyContext) {
	if lctx == nil {
		return
	}
	r.UpdateLiveState(lctx.Balance, lctx.Equity, livePositionsToSdk(lctx.Positions))
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
`
}
