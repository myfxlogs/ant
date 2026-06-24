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
	"anttrader/internal/interceptor"
	"anttrader/internal/mthub"
	"anttrader/internal/risk"
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
	AccountID           string            // trading account (live MT4 ID or paper account ID)
	DataSourceAccountID string            // bar data source account (optional; defaults to AccountID)
	UserID              string            // user ID for auth context (paper mode requires this)
	Symbol              string
	Timeframe           string
	Code                string
	Mode                string            // "live" | "paper"
	Params              map[string]string
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

	// D6-A: Gate is mandatory for live trading. Fail-stop if not injected.
	if s.gate == nil {
		return fmt.Errorf("live strategy runner: risk.Gate not injected — live trading blocked per D6-A")
	}

	source, ok := s.barSource.(LiveBarSubscriber)
	if !ok {
		return fmt.Errorf("live strategy runner: BarSource does not support streaming (got %s)", s.barSource.Name())
	}

	// Use DataSourceAccountID for bar subscription if set (paper mode uses real MT account for data).
	barAccountID := cfg.AccountID
	if cfg.DataSourceAccountID != "" {
		barAccountID = cfg.DataSourceAccountID
	}

	s.log.Info("LiveStrategyRunner: subscribing to bars",
		zap.String("trading_account", cfg.AccountID),
		zap.String("bar_source_account", barAccountID),
		zap.String("symbol", cfg.Symbol),
		zap.String("timeframe", cfg.Timeframe),
	)

	barCh, cancel := source.Subscribe(barAccountID)
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

			// Build proto-native LiveStrategyContext (T3.1: now a method).
			lctx := s.buildLiveContext(cfg, bars)

			// Call Python strategy via ConnectRPC.
			// Inject user ID for paper/internal mode (auth context required).
			callCtx := ctx
			if cfg.UserID != "" {
				callCtx = context.WithValue(ctx, interceptor.UserIDKey, cfg.UserID)
			}
			req := &antv1.ExecuteLiveRequest{StrategyCode: cfg.Code, Context: lctx}
			resp, err := s.ExecuteLive(callCtx, connect.NewRequest(req))
			if err != nil {
				s.log.Warn("LiveStrategyRunner: ExecuteLive failed", zap.Error(err))
				continue
			}

			if !resp.Msg.GetSuccess() {
				s.log.Warn("LiveStrategyRunner: strategy returned error",
					zap.String("error", resp.Msg.GetError()))
				continue
			}

			// Dispatch all signals (multi-intent per bar).
			signals := resp.Msg.GetSignals()
			if len(signals) == 0 {
				sig := resp.Msg.GetSignal()
				if sig != nil && sig.GetSignalType() != "" && sig.GetSignalType() != "hold" {
					signals = []*antv1.StrategySignal{sig}
				}
			}
			for _, sig := range signals {
				if sig.GetSignalType() == "hold" || sig.GetSignalType() == "" {
					continue
				}
				s.dispatchLiveSignal(ctx, cfg, bar, sig)
			}
		}
	}
}

// buildLiveContext constructs the proto-native LiveStrategyContext from accumulated bars.
// T3.1: now also backfills position/positions/equity/balance/margin (hard injury ① fix).
func (s *PythonStrategyServer) buildLiveContext(cfg LiveStrategyConfig, bars []liveBar) *antv1.LiveStrategyContext {
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
	if n > 0 {
		lctx.CurrentPrice = closeVals[n-1]
	}

	// T3.1: Backfill live account/position state (hard injury ①).
	if s.mtHub != nil {
		s.backfillLiveState(context.Background(), cfg.AccountID, lctx)
	} else {
		s.log.Debug("buildLiveContext: mtHub is nil, skipping position backfill")
	}

	return lctx
}

// backfillLiveState queries the AccountStateProvider for real account data (T3.2b / D6-A).
// If the provider is not yet connected, equity-dependent gate rules will fail-closed
// (see risk.Gate.Evaluate: nil AccountState → equity rules deny).
func (s *PythonStrategyServer) backfillLiveState(ctx context.Context, accountID string, lctx *antv1.LiveStrategyContext) {
	if s.accountProvider == nil {
		// T3.2b fail-closed: no provider → zero equity → equity-dependent rules deny.
		// Set sentinel values so the gate can distinguish "not connected" from "zero balance".
		lctx.Equity = -1.0  // sentinel: account provider not connected
		lctx.Balance = -1.0
		s.log.Debug("backfillLiveState: no AccountStateProvider — equity rules fail-closed",
			zap.String("account", accountID))
		return
	}

	state, err := s.accountProvider.GetAccountState(ctx, accountID)
	if err != nil {
		s.log.Warn("backfillLiveState: AccountStateProvider failed — equity rules fail-closed",
			zap.String("account", accountID), zap.Error(err))
		lctx.Equity = -1.0
		lctx.Balance = -1.0
		return
	}

	if state == nil {
		lctx.Equity = -1.0
		lctx.Balance = -1.0
		return
	}

	equity, _ := state.Equity.Float64()
	balance, _ := state.Balance.Float64()
	lctx.Equity = equity
	lctx.Balance = balance

	// Backfill live positions from MT4 gateway so SDK strategies can
	// query self.broker.positions() and close/modify positions.
	if s.mtHub != nil {
		orders, err := s.mtHub.OpenedOrders(ctx, accountID)
		if err != nil {
			s.log.Warn("backfillLiveState: OpenedOrders failed", zap.String("account", accountID), zap.Error(err))
		} else {
			positions := make([]*antv1.LivePosition, 0, len(orders))
			for _, o := range orders {
				side := "buy"
				if o.Side == mthub.SideSell {
					side = "sell"
				}
				lp := &antv1.LivePosition{
					Ticket:    o.Ticket,
					Side:      side,
					Volume:    o.Volume.InexactFloat64(),
					OpenPrice: o.OpenPrice.InexactFloat64(),
				}
				positions = append(positions, lp)
			}
			s.log.Info("backfillLiveState: positions backfilled", zap.String("account", accountID), zap.Int("count", len(positions)))
			lctx.Positions = positions
			if len(positions) > 0 {
				lctx.Position = positions[0]
			}
		}
	}
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
//
// T3.1 (hard injury ②): expanded action set from buy/sell only to full broker semantics:
//
//	Market:    buy, sell         → PlaceOrder(market)
//	Pending:   buy_limit, sell_limit, buy_stop, sell_stop,
//	           buy_stop_limit, sell_stop_limit → PlaceOrder(limit/stop/stop_limit)
//	Close:     close, close_all  → CloseOrder
//	Modify:    modify            → ModifyOrder
//	Cancel:    cancel            → CancelPending
func (s *PythonStrategyServer) dispatchLiveSignal(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, sig *antv1.StrategySignal) {
	action := sig.GetSignalType()
	s.log.Info("LiveStrategyRunner: signal",
		zap.String("account", cfg.AccountID),
		zap.String("symbol", cfg.Symbol),
		zap.String("type", action),
		zap.Float64("volume", sig.GetVolume()),
		zap.Float64("sl", sig.GetStopLoss()),
		zap.Float64("tp", sig.GetTakeProfit()),
	)

	if cfg.Mode == "paper" {
		s.dispatchPaperSignal(ctx, cfg, bar, sig)
		return
	}

	if s.mtHub == nil {
		s.log.Warn("LiveStrategyRunner: no MtHubService configured, cannot dispatch live order")
		return
	}

	// T3.1: dispatch based on expanded action set.
	switch action {
	case "buy", "sell":
		s.dispatchMarketOrder(ctx, cfg, sig)
	case "buy_limit", "sell_limit", "buy_stop", "sell_stop",
		"buy_stop_limit", "sell_stop_limit":
		s.dispatchPendingOrder(ctx, cfg, sig)
	case "close":
		s.dispatchCloseOrder(ctx, cfg, sig)
	case "close_all":
		s.dispatchCloseAll(ctx, cfg)
	case "modify":
		s.dispatchModifyOrder(ctx, cfg, sig)
	case "cancel":
		s.dispatchCancelOrder(ctx, cfg, sig)
	default:
		// hold, unknown — no-op.
	}
}

// ── T3.1 action dispatchers ──────────────────────────────────────────

func (s *PythonStrategyServer) dispatchMarketOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
	side := signalToSide(sig.GetSignalType())
	if side == 0 {
		return
	}
	s.submitOrder(ctx, cfg, side, mthub.OrderMarket, sig)
}

func (s *PythonStrategyServer) dispatchPendingOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
	side := signalToSide(sig.GetSignalType())
	if side == 0 {
		return
	}
	var orderType mthub.OrderType
	switch sig.GetSignalType() {
	case "buy_limit", "sell_limit":
		orderType = mthub.OrderLimit
	case "buy_stop", "sell_stop":
		orderType = mthub.OrderStop
	case "buy_stop_limit", "sell_stop_limit":
		orderType = mthub.OrderStopLimit
	default:
		orderType = mthub.OrderLimit
	}
	s.submitOrder(ctx, cfg, side, orderType, sig)
}

// dispatchCloseAll closes all open positions for the account.
func (s *PythonStrategyServer) dispatchCloseAll(ctx context.Context, cfg LiveStrategyConfig) {
	if s.mtHub == nil {
		s.log.Warn("LiveStrategyRunner: dispatchCloseAll: no MtHubService")
		return
	}
	// Detach from parent cancellation but preserve values (userID, auth).
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		orders, err := s.mtHub.OpenedOrders(bgCtx, cfg.AccountID)
		if err != nil {
			s.log.Error("LiveStrategyRunner: dispatchCloseAll: OpenedOrders failed",
				zap.String("account", cfg.AccountID), zap.Error(err))
			return
		}
		closed := 0
		for _, o := range orders {
			if err := s.mtHub.CloseOrder(bgCtx, cfg.AccountID, o.Ticket, o.Volume); err != nil {
				s.log.Warn("LiveStrategyRunner: dispatchCloseAll: CloseOrder failed",
					zap.Int64("ticket", o.Ticket), zap.Error(err))
				continue
			}
			closed++
		}
		s.log.Info("LiveStrategyRunner: dispatchCloseAll complete",
			zap.String("account", cfg.AccountID),
			zap.Int("closed", closed),
			zap.Int("total", len(orders)),
		)
	}()
}

func (s *PythonStrategyServer) dispatchCloseOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: close order without ticket")
		return
	}
	// D6-A: gate moved to MtHubService.CloseOrder (single chokepoint).
	volume := decimal.NewFromFloat(sig.GetVolume())
	go func() {
		if err := s.mtHub.CloseOrder(context.Background(), cfg.AccountID, ticket, volume); err != nil {
			s.log.Error("LiveStrategyRunner: CloseOrder failed",
				zap.Int64("ticket", ticket), zap.Error(err))
			return
		}
		s.log.Info("LiveStrategyRunner: position closed", zap.Int64("ticket", ticket))
	}()
}

func (s *PythonStrategyServer) dispatchModifyOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: modify order without ticket")
		return
	}
	sl := decimal.NewFromFloat(sig.GetStopLoss())
	tp := decimal.NewFromFloat(sig.GetTakeProfit())
	price := decimal.NewFromFloat(sig.GetPrice())

	go func() {
		if err := s.mtHub.ModifyOrder(ctx, cfg.AccountID, ticket, sl, tp, price); err != nil {
			s.log.Error("LiveStrategyRunner: ModifyOrder failed",
				zap.Int64("ticket", ticket), zap.Error(err))
		}
	}()
}

func (s *PythonStrategyServer) dispatchCancelOrder(ctx context.Context, cfg LiveStrategyConfig, sig *antv1.StrategySignal) {
	ticket := sig.GetExecutedTicket()
	if ticket == 0 {
		s.log.Warn("LiveStrategyRunner: cancel order without ticket")
		return
	}
	// MT5 OrderClose handles both market positions AND pending orders.
	// With lots=decimal.Zero (no volume), the gateway cancels the pending
	// order instead of closing a position. See mt5.proto OrderClose docs.
	bgCtx := context.WithoutCancel(ctx)
	go func() {
		if err := s.mtHub.CloseOrder(bgCtx, cfg.AccountID, ticket, decimal.Zero); err != nil {
			s.log.Error("LiveStrategyRunner: CancelOrder failed",
				zap.Int64("ticket", ticket), zap.Error(err))
			return
		}
		s.log.Info("LiveStrategyRunner: pending order cancelled", zap.Int64("ticket", ticket))
	}()
}

func (s *PythonStrategyServer) dispatchPaperSignal(ctx context.Context, cfg LiveStrategyConfig, bar *mthub.BarUpdate, sig *antv1.StrategySignal) {
	if s.paperEngine == nil {
		s.log.Warn("LiveStrategyRunner: no PaperEngine, dropping paper signal")
		return
	}

	action := sig.GetSignalType()

		// Close/modify/cancel/close_all: paper engine needs PaperOrder.StopLoss/TakeProfit
	// fields + repo.GetOrder/UpdateOrder methods before these can be implemented.
	// TODO(M12-PAPER): add close/modify/cancel to paper engine.
	switch action {
	case "close", "close_all", "modify", "cancel":
		s.log.Debug("LiveStrategyRunner: paper close/modify/cancel — logged",
			zap.String("action", action))
		return
	}

	bid := bar.Bid
	ask := bar.Ask
	if err := s.paperEngine.PlacePaperOrder(ctx, cfg.AccountID, cfg.Symbol,
		action, decimal.NewFromFloat(sig.GetVolume()), bid, ask); err != nil {
		s.log.Warn("LiveStrategyRunner: paper order failed", zap.Error(err))
	}
}

// submitOrder is the common order submission helper (T3.1 / D6-A).
// Every order MUST pass through Gate.Evaluate() before reaching mthub.
func (s *PythonStrategyServer) submitOrder(ctx context.Context, cfg LiveStrategyConfig, side mthub.Side, orderType mthub.OrderType, sig *antv1.StrategySignal) {
	req := &mthub.OrderRequest{
		AccountID: cfg.AccountID,
		Canonical: cfg.Symbol,
		Side:      side,
		OrderType: orderType,
		Volume:    decimal.NewFromFloat(sig.GetVolume()),
	}
	if sig.GetStopLoss() > 0 {
		req.StopLoss = decimal.NewFromFloat(sig.GetStopLoss())
	}
	if sig.GetTakeProfit() > 0 {
		req.TakeProfit = decimal.NewFromFloat(sig.GetTakeProfit())
	}
	if sig.GetPrice() > 0 {
		req.Price = decimal.NewFromFloat(sig.GetPrice())
	}

	sideStr := sideToString(side)

	// D6-A: gate moved to MtHubService.PlaceOrder (single chokepoint).
	// Pass user ID in context for mthub capability check.
	placeCtx := context.WithoutCancel(ctx)
	if cfg.UserID != "" {
		placeCtx = context.WithValue(context.Background(), interceptor.UserIDKey, cfg.UserID)
	}
	go func() {
		record, err := s.mtHub.PlaceOrder(placeCtx, req)
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

// signalToOrderIntent converts a StrategySignal to an OrderIntent proto for gate evaluation (D6-A).
func signalToOrderIntent(sig *antv1.StrategySignal, cfg LiveStrategyConfig) *antv1.OrderIntent {
	return &antv1.OrderIntent{
		UserId:    "", // filled from auth context where available
		AccountId: cfg.AccountID,
		Symbol:    cfg.Symbol,
		Side:      sig.GetSignalType(),
		Volume:    fmt.Sprintf("%.5f", sig.GetVolume()),
		Type:      sig.GetSignalType(),
		Price:     fmt.Sprintf("%.5f", sig.GetPrice()),
		Sl:        fmt.Sprintf("%.5f", sig.GetStopLoss()),
		Tp:        fmt.Sprintf("%.5f", sig.GetTakeProfit()),
		Magic:     sig.GetExecutedTicket(),
		Source:    antv1.OrderIntentSource_ORDER_INTENT_SOURCE_LIVE,
	}
}

// getAccountState fetches live account state for gate evaluation (T3.2b).
// Returns nil if provider not connected — gate rules fail-closed on nil state.
func (s *PythonStrategyServer) getAccountState(ctx context.Context, accountID string) *risk.AccountState {
	if s.accountProvider == nil {
		return nil // fail-closed: equity-dependent gate rules will deny
	}
	state, err := s.accountProvider.GetAccountState(ctx, accountID)
	if err != nil {
		s.log.Debug("getAccountState: provider error", zap.String("account", accountID), zap.Error(err))
		return nil
	}
	return state
}

// signalToSide maps a strategy signal action to mthub.Side. Returns 0 for non-directional signals.
// T3.1: expanded to cover all buy_* / sell_* variants (hard injury ②).
func signalToSide(action string) mthub.Side {
	switch action {
	case "buy", "buy_limit", "buy_stop", "buy_stop_limit":
		return mthub.SideBuy
	case "sell", "sell_limit", "sell_stop", "sell_stop_limit":
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
