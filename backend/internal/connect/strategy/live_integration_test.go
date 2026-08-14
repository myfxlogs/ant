package strategy

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mthub"
	"alphaforge/internal/risk"
)

// liveMockBarSource is a controllable bar source for live integration tests.
type liveMockBarSource struct {
	ch   chan *mthub.BarUpdate
	mu   sync.Mutex
	bars []*mthub.BarUpdate
}

func newMockBarSource() *liveMockBarSource {
	return &liveMockBarSource{ch: make(chan *mthub.BarUpdate, 4)}
}

func (m *liveMockBarSource) Name() string { return "mock" }

func (m *liveMockBarSource) Fetch(ctx context.Context, symbol, timeframe string, from, to *time.Time) ([]*antv1.ExecuteKlineBar, error) {
	return nil, nil
}

func (m *liveMockBarSource) Subscribe(accountID string) (<-chan *mthub.BarUpdate, func()) {
	return m.ch, func() {}
}

func (m *liveMockBarSource) push(bar *mthub.BarUpdate) {
	m.mu.Lock()
	m.bars = append(m.bars, bar)
	m.mu.Unlock()
	m.ch <- bar
}

func (m *liveMockBarSource) receivedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.bars)
}

// capturePaperEngine records paper orders placed by dispatchPaperSignal.
type capturePaperEngine struct {
	mu     sync.Mutex
	orders []capturePaperOrder
}

type capturePaperOrder struct {
	accountID, symbol, side string
	volume, bid, ask        decimal.Decimal
}

func (c *capturePaperEngine) PlacePaperOrder(ctx context.Context, accountID, symbol, side string, volume, bid, ask decimal.Decimal) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.orders = append(c.orders, capturePaperOrder{accountID, symbol, side, volume, bid, ask})
	return nil
}
func (c *capturePaperEngine) ClosePaperOrder(ctx context.Context, accountID, symbol string) error {
	return nil
}
func (c *capturePaperEngine) ModifyPaperOrder(ctx context.Context, accountID, symbol string, sl, tp decimal.Decimal) error {
	return nil
}
func (c *capturePaperEngine) CancelPaperOrder(ctx context.Context, accountID, symbol string) error {
	return nil
}
func (c *capturePaperEngine) PaperPnl(ctx context.Context, accountID, symbol string, bid, ask decimal.Decimal) (decimal.Decimal, error) {
	return decimal.Zero, nil
}

func (c *capturePaperEngine) orderCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.orders)
}

func (c *capturePaperEngine) first() (capturePaperOrder, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.orders) == 0 {
		return capturePaperOrder{}, false
	}
	return c.orders[0], true
}

// vmProbe compiles the eager MQL and runs a single VMLiveSession.Start with a
// closed bar. It returns the ExecuteLiveResponse so the test can assert on the
// VM signal output (steps ④-⑧) independently of the full runner plumbing.
func vmProbe(ctx context.Context, code string, bar *mthub.BarUpdate) (*antv1.ExecuteLiveResponse, error) {
	vmSess, err := NewVMLiveSession(code)
	if err != nil {
		return nil, fmt.Errorf("compile MQL: %w", err)
	}
	lctx := &antv1.LiveStrategyContext{
		Symbol:       bar.Symbol,
		Timeframe:    bar.Period,
		Mode:         "paper",
		Close:        []string{bar.Close.String()},
		Open:         []string{bar.Open.String()},
		High:         []string{bar.High.String()},
		Low:          []string{bar.Low.String()},
		Volume:       []string{decimal.NewFromFloat(bar.Volume).String()},
		BarTimesMs:   []int64{bar.OpenTime},
		CurrentPrice: bar.Close.String(),
	}
	req := &antv1.ExecuteLiveRequest{
		StrategyCode: code,
		RequestType:  antv1.RequestType_REQUEST_TYPE_BAR,
		BarContext:   lctx,
	}
	reqBytes, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	respBytes, err := vmSess.Start(ctx, reqBytes)
	if err != nil {
		return nil, fmt.Errorf("vm Start: %w", err)
	}
	var resp antv1.ExecuteLiveResponse
	if err := proto.Unmarshal(respBytes, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

// TestLivePath_E2E is the "net" that runs the full live path from a mock bar
// source to the paper engine, with step-by-step diagnostics. It first probes
// the VM in isolation (compile → OnBar → response signals) and then runs the
// full StrategyExecutionServer path. If any step is broken, the test fails with
// a precise message identifying the breakpoint.
func TestLivePath_E2E(t *testing.T) {
	const code = `void OnBar() { OrderSend(Symbol(), OP_BUY, 0.01, Ask, 3, 0, 0); }`
	bar := &mthub.BarUpdate{
		Symbol:   "BTCUSDm",
		Period:   "1m",
		Closed:   true,
		Open:     decimal.NewFromFloat(100.0),
		High:     decimal.NewFromFloat(101.0),
		Low:      decimal.NewFromFloat(99.0),
		Close:    decimal.NewFromFloat(100.5),
		Bid:      decimal.NewFromFloat(100.4),
		Ask:      decimal.NewFromFloat(100.5),
		Volume:   1000,
		OpenTime: 1786678260000,
	}

	// ② shouldRunOnBar passes for the closed matching bar.
	require.True(t, shouldRunOnBar(bar, "BTCUSDm", "1m"), "step ② broken: shouldRunOnBar should pass")

	// ④-⑧ VM isolation probe: compile, Start (OnBar), and inspect response.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := vmProbe(ctx, code, bar)
	require.NoError(t, err, "step ④/⑤ broken: compile or VM Start failed")
	require.True(t, resp.GetSuccess(), "step ⑤ broken: VM Start returned success=false: %s", resp.GetError())
	t.Logf("VM probe: success=%v error=%q signals=%d signal=%+v",
		resp.GetSuccess(), resp.GetError(), len(resp.GetSignals()), resp.GetSignal())

	// ⑦ OrderSend must produce a StrategySignal in the response.
	hasSignal := false
	if len(resp.GetSignals()) > 0 {
		hasSignal = true
	} else if sig := resp.GetSignal(); sig != nil && sig.GetSignalType() != "" && sig.GetSignalType() != "hold" {
		hasSignal = true
	}
	require.True(t, hasSignal, "step ⑦/⑧ broken: OnBar ran but ExecuteLiveResponse.Signals is empty — eager OrderSend did not generate a signal")

	// ⑩ Full RunLiveStrategy path: mock bar source → paper engine.
	barSource := newMockBarSource()
	paper := &capturePaperEngine{}
	srv := &StrategyExecutionServer{
		log:             zap.NewNop(),
		barSource:       barSource,
		paperEngine:     paper,
		gate:            risk.NewDefaultGate(),
		sessionRegistry: NewSessionRegistry(),
	}

	cfg := LiveStrategyConfig{
		AccountID:  "acct-1",
		UserID:     "user-1",
		Symbol:     "BTCUSDm",
		Timeframe:  "1m",
		Code:       code,
		Mode:       "paper",
		RunID:      uuid.New(),
		TickSeq:    new(atomic.Int64),
		ScheduleID: uuid.New(),
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	defer runCancel()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.RunLiveStrategy(runCtx, cfg) }()

	// ① Bar reaches RunLiveStrategy's barCh.
	require.Eventually(t, func() bool {
		// Once RunLiveStrategy has called source.Subscribe, it should be waiting
		// on barCh. Push the bar and verify it was received.
		barSource.push(bar)
		return barSource.receivedCount() == 1
	}, 2*time.Second, 50*time.Millisecond, "step ① broken: bar not received by runner")

	// ⑩ Paper engine receives the buy order.
	require.Eventually(t, func() bool {
		return paper.orderCount() > 0
	}, 5*time.Second, 100*time.Millisecond, "step ⑩ broken: paper engine got no order — signal dispatch chain broken after VM")

	order, _ := paper.first()
	require.Equal(t, "buy", order.side, "paper order side")
	require.Equal(t, "BTCUSDm", order.symbol, "paper order symbol")
	require.True(t, order.volume.Equal(decimal.NewFromFloat(0.01)), "paper order volume = 0.01")

	runCancel()
	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("RunLiveStrategy did not exit after context cancellation")
	}
}
