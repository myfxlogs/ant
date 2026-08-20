package strategy

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// makeAcceleratingBars generates n chronological bars with a quadratic uptrend:
// close[i] = base + i*linearStep + i²*quadStep. The accelerating slope keeps
// MACD main continuously rising above its signal line (main > signal), which
// a linear uptrend cannot guarantee — at convergence MACD main ≈ signal and
// the comparison becomes floating-point noise.
func makeAcceleratingBars(n int, base, linearStep, quadStep float64, startMs, periodMs int64) []liveBar {
	bars := make([]liveBar, n)
	for i := 0; i < n; i++ {
		fi := float64(i)
		c := base + fi*linearStep + fi*fi*quadStep
		bars[i] = liveBar{
			open:     decimal.NewFromFloat(c).String(),
			high:     decimal.NewFromFloat(c + 0.5).String(),
			low:      decimal.NewFromFloat(c - 0.5).String(),
			close:    decimal.NewFromFloat(c).String(),
			volume:   "1000",
			openTime: startMs + int64(i)*periodMs,
		}
	}
	return bars
}

// rollLiveBars drops the oldest bar and appends a newest bar, keeping length n.
func rollLiveBars(bars []liveBar, newBar liveBar) []liveBar {
	out := make([]liveBar, len(bars))
	copy(out, bars[1:])
	out[len(out)-1] = newBar
	return out
}

// buildLiveCtx builds a LiveStrategyContext from chronological bars (oldest first).
func buildLiveCtx(bars []liveBar, symbol, timeframe, mode string) *antv1.LiveStrategyContext {
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
		Symbol:     symbol,
		Timeframe:  timeframe,
		Mode:       mode,
	}
	if n > 0 {
		lctx.CurrentPrice = closeVals[n-1]
	}
	return lctx
}

// buildTickCtx builds a TickContext for a tick event.
func buildTickCtx(symbol string, bid, ask decimal.Decimal) *antv1.TickContext {
	return &antv1.TickContext{
		Symbol: symbol,
		Bid:    bid.String(),
		Ask:    ask.String(),
	}
}

// sendBarEvent sends a BAR request to the VMLiveSession and returns the response.
func sendBarEvent(ctx context.Context, sess *VMLiveSession, lctx *antv1.LiveStrategyContext) (*antv1.ExecuteLiveResponse, error) {
	req := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:  lctx,
	}
	reqBytes, _ := proto.Marshal(req)
	respBytes, err := sess.SendEvent(ctx, reqBytes)
	if err != nil {
		return nil, err
	}
	var resp antv1.ExecuteLiveResponse
	proto.Unmarshal(respBytes, &resp)
	return &resp, nil
}

// sendTickEvent sends a TICK request to the VMLiveSession and returns the response.
func sendTickEvent(ctx context.Context, sess *VMLiveSession, tctx *antv1.TickContext) (*antv1.ExecuteLiveResponse, error) {
	req := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_TICK,
		TickContext: tctx,
	}
	reqBytes, _ := proto.Marshal(req)
	respBytes, err := sess.SendEvent(ctx, reqBytes)
	if err != nil {
		return nil, err
	}
	var resp antv1.ExecuteLiveResponse
	proto.Unmarshal(respBytes, &resp)
	return &resp, nil
}

// startSession starts a VMLiveSession with the first BAR context.
func startSession(ctx context.Context, code string, lctx *antv1.LiveStrategyContext) (*VMLiveSession, *antv1.ExecuteLiveResponse, error) {
	sess, err := NewVMLiveSession(code)
	if err != nil {
		return nil, nil, err
	}
	req := &antv1.ExecuteLiveRequest{
		RequestType: antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:  lctx,
	}
	reqBytes, _ := proto.Marshal(req)
	respBytes, err := sess.Start(ctx, reqBytes)
	if err != nil {
		return nil, nil, err
	}
	var resp antv1.ExecuteLiveResponse
	proto.Unmarshal(respBytes, &resp)
	return sess, &resp, nil
}

// hasSellSignal returns true if the response contains a SELL signal.
func hasSellSignal(resp *antv1.ExecuteLiveResponse) bool {
	if resp == nil {
		return false
	}
	if sig := resp.GetSignal(); sig != nil && sig.GetSignalType() == "sell" {
		return true
	}
	for _, s := range resp.GetSignals() {
		if s.GetSignalType() == "sell" {
			return true
		}
	}
	return false
}

// macdBearishCrossoverCode is a legacy MQL4 start() strategy that sells when
// the MACD main line crosses below the signal line — the exact pattern that
// froze in production (LIVE-INDICATOR-1: MACD.main/signal stuck at 00:44 frame).
// Uses iMACD with MODE_MAIN (0) and MODE_SIGNAL (1), both cache-backed.
const macdBearishCrossoverCode = `
int start()
{
    double macdMain   = iMACD(NULL, 0, 12, 26, 9, PRICE_CLOSE, MODE_MAIN, 0);
    double macdSignal = iMACD(NULL, 0, 12, 26, 9, PRICE_CLOSE, MODE_SIGNAL, 0);
    if (macdMain < macdSignal) {
        OrderSend(Symbol(), OP_SELL, 0.01, Bid, 5, 0, 0);
    }
    return 0;
}`

// TestVMLiveSession_RollingWindow_IndicatorUnfreeze is the integration test for
// LIVE-INDICATOR-1. It reproduces legacy MQL4 start() semantics with a real
// MACD main/signal crossover — the exact indicator that froze in production:
//
//  1. First 500-bar BAR context initializes the session (uptrend, MACD main > signal).
//  2. First TICK runs start() → no SELL (no bearish crossover).
//  3. Second 500-bar BAR context (rolling window, sharp drop) → MACD main < signal.
//  4. Second TICK runs start() → MUST produce SELL signal.
//
// Without the revision fix, the indicator cache freezes at step 1's values,
// so step 4 sees the old MACD values and produces no signal.
//
// Adversarial proof: delete the r.barRev.Add(1) line in Runner.OnBar → step 4
// goes RED (no SELL signal because MACD is frozen at window-1 values).
func TestVMLiveSession_RollingWindow_IndicatorUnfreeze(t *testing.T) {
	const symbol = "TESTUSD"
	const timeframe = "1m"
	const periodMs = 60_000
	const startMs = 1_700_000_000_000
	const nBars = 500

	// Window 1: accelerating uptrend (quadratic, close goes from 100 to ~1350).
	// MACD main > signal because momentum is continuously accelerating —
	// a linear uptrend would converge MACD to a constant (main ≈ signal).
	bars1 := makeAcceleratingBars(nBars, 100.0, 0.5, 0.005, startMs, periodMs)

	// Window 2: drop oldest, append a bar with very low close (sharp reversal).
	// MACD main drops below signal because the newest bar's EMA12 falls fast
	// while the signal (EMA9 of MACD) lags.
	dropBar := liveBar{
		open:     "1.0",
		high:     "2.0",
		low:      "0.5",
		close:    "1.0",
		volume:   "1000",
		openTime: startMs + int64(nBars)*periodMs,
	}
	bars2 := rollLiveBars(bars1, dropBar)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Data verification: fresh session with window 2 must SELL (proves crossover).
	// This is a separate session — if it fails, the test data is invalid, not the fix.
	lctx2fresh := buildLiveCtx(bars2, symbol, timeframe, "paper")
	sessFresh, _, err := startSession(ctx, macdBearishCrossoverCode, lctx2fresh)
	require.NoError(t, err, "fresh session start failed")
	tctxFresh := buildTickCtx(symbol, decimal.NewFromFloat(1.0), decimal.NewFromFloat(1.5))
	respTickFresh, err := sendTickEvent(ctx, sessFresh, tctxFresh)
	require.NoError(t, err, "fresh tick failed")
	require.True(t, hasSellSignal(respTickFresh),
		"DATA VERIFICATION: fresh session with window 2 must SELL (MACD bearish crossover). Test data is invalid if this fails.")
	_ = sessFresh.Close()

	// Main test: reuse session across both windows.
	lctx1 := buildLiveCtx(bars1, symbol, timeframe, "paper")
	sess, resp1, err := startSession(ctx, macdBearishCrossoverCode, lctx1)
	require.NoError(t, err, "compile/start failed")
	require.True(t, resp1.GetSuccess(), "start response: %s", resp1.GetError())

	// Step 2: First TICK — start() runs, no bearish crossover → no SELL.
	tctx := buildTickCtx(symbol, decimal.NewFromFloat(1350.0), decimal.NewFromFloat(1350.5))
	respTick1, err := sendTickEvent(ctx, sess, tctx)
	require.NoError(t, err, "tick #1 failed")
	require.False(t, hasSellSignal(respTick1),
		"tick #1 should NOT produce SELL (no bearish crossover in window 1), got signal")

	// Step 3: Second 500-bar BAR event (rolling window with sharp drop).
	lctx2 := buildLiveCtx(bars2, symbol, timeframe, "paper")
	_, err = sendBarEvent(ctx, sess, lctx2)
	require.NoError(t, err, "bar #2 failed")

	// Step 4: Second TICK — start() runs with refreshed MACD → SELL.
	tctx2 := buildTickCtx(symbol, decimal.NewFromFloat(1.0), decimal.NewFromFloat(1.5))
	respTick2, err := sendTickEvent(ctx, sess, tctx2)
	require.NoError(t, err, "tick #2 failed")
	require.True(t, hasSellSignal(respTick2),
		"tick #2 MUST produce SELL (MACD bearish crossover in window 2) — indicator cache is frozen (LIVE-INDICATOR-1)")

	_ = sess.Close()
}

// TestVMLiveSession_RollingWindow_TickStability verifies that within the same
// bar (no new BAR event), multiple TICK events produce stable indicator values
// — the cache must NOT rebuild on ticks.
func TestVMLiveSession_RollingWindow_TickStability(t *testing.T) {
	const symbol = "TESTUSD"
	const timeframe = "1m"
	const periodMs = 60_000
	const startMs = 1_700_000_000_000
	const nBars = 500

	bars1 := makeAcceleratingBars(nBars, 100.0, 0.5, 0.005, startMs, periodMs)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	lctx1 := buildLiveCtx(bars1, symbol, timeframe, "paper")
	sess, _, err := startSession(ctx, macdBearishCrossoverCode, lctx1)
	require.NoError(t, err)

	// First TICK — no SELL (accelerating uptrend, MACD main > signal).
	tctx := buildTickCtx(symbol, decimal.NewFromFloat(1350.0), decimal.NewFromFloat(1350.5))
	resp1, err := sendTickEvent(ctx, sess, tctx)
	require.NoError(t, err)
	require.False(t, hasSellSignal(resp1), "first tick should not sell")

	// Multiple subsequent TICKs with same bar window — must NOT suddenly sell.
	// If the cache were rebuilding spuriously, MACD values could shift.
	for i := 0; i < 50; i++ {
		resp, err := sendTickEvent(ctx, sess, tctx)
		require.NoError(t, err)
		require.False(t, hasSellSignal(resp),
			"tick #%d produced unexpected SELL — indicators unstable within same bar", i+2)
	}

	_ = sess.Close()
}
