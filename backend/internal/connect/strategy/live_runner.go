// live_runner.go — LiveStrategyRunner: subscribes to real-time bar stream,
// builds proto-native LiveStrategyContext per bar, calls ExecuteLive RPC,
// and dispatches signals to OMS.
//
// Architecture:
//   Bar stream (mthub/LiveSource) → buildContext() → ExecuteLive (Python LiveWorker pool)
//   → signal dispatch → OMS (live) or log (paper)
//
// Follows push-first architecture: bar events drive the loop; no polling.

package strategy

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/mthub"
)

const (
	// maxContextBars is the maximum number of bars included in the live context OHLCV arrays.
	// Balances signal quality (more bars = better indicator accuracy) against context size.
	maxContextBars = 500
)

// liveBar holds a single bar's OHLCV for the live context rolling window.
type liveBar struct {
	open, high, low, close, volume float64
	openTime                       int64
}

// LiveStrategyConfig holds the parameters for running a strategy in live/paper mode.
type LiveStrategyConfig struct {
	AccountID string
	Symbol    string
	Timeframe string
	Code      string
	Mode      string // "live" | "paper"
	Params    map[string]string
}

// RunLiveStrategy subscribes to real-time bar updates for the given account/symbol/timeframe,
// builds a proto-native LiveStrategyContext for each bar, calls the Python strategy engine
// via ExecuteLive, and dispatches the resulting signal.
//
// Blocks until ctx is cancelled. Callers should run this in a goroutine.
func (s *PythonStrategyServer) RunLiveStrategy(ctx context.Context, cfg LiveStrategyConfig) error {
	if s.barSource == nil {
		return fmt.Errorf("live strategy runner: no BarSource configured")
	}

	source, ok := s.barSource.(LiveBarSubscriber)
	if !ok {
		return fmt.Errorf("live strategy runner: BarSource does not support streaming (got %s)", s.barSource.Name())
	}

	barCh, cancel := source.Subscribe(cfg.AccountID)
	defer cancel()

	s.log.Info("LiveStrategyRunner: started",
		zap.String("account", cfg.AccountID),
		zap.String("symbol", cfg.Symbol),
		zap.String("timeframe", cfg.Timeframe),
		zap.String("mode", cfg.Mode),
	)

	// Accumulate bars for context OHLCV arrays (rolling window).
	bars := make([]liveBar, 0, maxContextBars)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("LiveStrategyRunner: context cancelled, exiting")
			return nil
		case bar, ok := <-barCh:
			if !ok {
				s.log.Warn("LiveStrategyRunner: bar channel closed, exiting")
				return nil
			}
			// Filter by symbol/timeframe.
			if bar.Symbol != cfg.Symbol || bar.Period != cfg.Timeframe {
				continue
			}
			// Append to rolling window.
			bars = append(bars, liveBar{
				open: bar.Open, high: bar.High, low: bar.Low,
				close: bar.Close, volume: bar.Volume,
				openTime: bar.OpenTime,
			})
			if len(bars) > maxContextBars {
				bars = bars[len(bars)-maxContextBars:]
			}

			// Build proto-native LiveStrategyContext.
			lctx := buildLiveContext(cfg, bars)

			// Call Python strategy via ConnectRPC.
			req := &antv1.ExecuteLiveRequest{StrategyCode: cfg.Code, Context: lctx}
			resp, err := s.ExecuteLive(ctx, connect.NewRequest(req))
			if err != nil {
				s.log.Warn("LiveStrategyRunner: ExecuteLive failed", zap.Error(err))
				continue
			}

			if !resp.Msg.GetSuccess() {
				s.log.Warn("LiveStrategyRunner: strategy returned error",
					zap.String("error", resp.Msg.GetError()))
				continue
			}

			sig := resp.Msg.GetSignal()
			if sig == nil || sig.GetSignalType() == "hold" || sig.GetSignalType() == "" {
				continue
			}

			// Dispatch signal.
			s.dispatchLiveSignal(ctx, cfg, sig)
		}
	}
}

// buildLiveContext constructs the proto-native LiveStrategyContext from accumulated bars.
func buildLiveContext(cfg LiveStrategyConfig, bars []liveBar) *antv1.LiveStrategyContext {
	n := len(bars)
	closeVals := make([]float64, n)
	openVals := make([]float64, n)
	highVals := make([]float64, n)
	lowVals := make([]float64, n)
	volVals := make([]float64, n)
	times := make([]int64, n)

	for i, b := range bars {
		closeVals[i] = b.close
		openVals[i] = b.open
		highVals[i] = b.high
		lowVals[i] = b.low
		volVals[i] = b.volume
		times[i] = b.openTime
	}

	ctx := &antv1.LiveStrategyContext{
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
	if n > 0 {
		ctx.CurrentPrice = closeVals[n-1]
	}
	return ctx
}

// buildLiveParams converts a string map to repeated LiveParam.
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

// dispatchLiveSignal routes the strategy signal to the appropriate destination.
// LIVE mode: signal → OMS → broker (via mthub)
// PAPER mode: signal → log (paper portfolio coming later)
func (s *PythonStrategyServer) dispatchLiveSignal(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
	s.log.Info("LiveStrategyRunner: signal",
		zap.String("account", cfg.AccountID),
		zap.String("symbol", cfg.Symbol),
		zap.String("type", sig.GetSignalType()),
		zap.Float64("volume", sig.GetVolume()),
		zap.Float64("sl", sig.GetStopLoss()),
		zap.Float64("tp", sig.GetTakeProfit()),
	)

	if cfg.Mode == "paper" {
		s.log.Info("LiveStrategyRunner: paper signal (not dispatched to broker)",
			zap.String("signal", sig.GetSignalType()),
			zap.Float64("volume", sig.GetVolume()))
		return
	}

	// LIVE mode: submit market order to broker via mthub.
	if s.mtHub == nil {
		s.log.Warn("LiveStrategyRunner: no MtHubService configured, cannot dispatch live order")
		return
	}

	side := signalToSide(sig.GetSignalType())
	if side == 0 { // neither buy nor sell — hold, close, pending orders handled by OMS
		return
	}

	req := &mthub.OrderRequest{
		AccountID: cfg.AccountID,
		Canonical: cfg.Symbol,
		Side:      side,
		OrderType: mthub.OrderMarket,
		Volume:    decimal.NewFromFloat(sig.GetVolume()),
	}
	if sig.GetStopLoss() > 0 {
		req.StopLoss = decimal.NewFromFloat(sig.GetStopLoss())
	}
	if sig.GetTakeProfit() > 0 {
		req.TakeProfit = decimal.NewFromFloat(sig.GetTakeProfit())
	}

	sideStr := sideToString(side)

	// Submit order asynchronously — don't block the bar loop.
	go func() {
		record, err := s.mtHub.PlaceOrder(context.Background(), req)
		if err != nil {
			s.log.Error("LiveStrategyRunner: order submission failed",
				zap.String("symbol", cfg.Symbol),
				zap.String("side", sideStr),
				zap.Error(err),
			)
			return
		}
		s.log.Info("LiveStrategyRunner: order submitted",
			zap.Int64("ticket", record.Ticket),
			zap.String("symbol", cfg.Symbol),
			zap.String("side", sideStr),
		)
	}()
}

// signalToSide maps a strategy signal action to mthub.Side. Returns 0 for non-directional signals.
func signalToSide(action string) mthub.Side {
	switch action {
	case "buy":
		return mthub.SideBuy
	case "sell":
		return mthub.SideSell
	default:
		return 0
	}
}

// sideToString returns a human-readable Side string for logging.
func sideToString(side mthub.Side) string {
	switch side {
	case mthub.SideBuy:
		return "buy"
	case mthub.SideSell:
		return "sell"
	default:
		return "unknown"
	}
}
