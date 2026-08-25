package strategy

import (
	"context"
	"errors"
	"strings"
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// newExecuteLiveServer creates a StrategyExecutionServer with minimal fields for mode validation tests.
func newExecuteLiveServer() *StrategyExecutionServer {
	return &StrategyExecutionServer{log: zap.NewNop()}
}

// newAuthCtx returns a context with a valid user ID for userIDRequire.
func newAuthCtx() context.Context {
	return context.WithValue(context.Background(), interceptor.UserIDKey, uuid.New().String())
}

// newExecuteLiveRequest constructs a connect.Request with the given contexts and strategy code.
func newExecuteLiveRequest(code string, contexts ...func(*antv1.ExecuteLiveRequest)) *connect.Request[antv1.ExecuteLiveRequest] {
	req := &antv1.ExecuteLiveRequest{StrategyCode: code}
	for _, fn := range contexts {
		fn(req)
	}
	return connect.NewRequest(req)
}

func withBarContext(mode string) func(*antv1.ExecuteLiveRequest) {
	return func(r *antv1.ExecuteLiveRequest) {
		r.BarContext = &antv1.LiveStrategyContext{Mode: mode}
	}
}

func withTickContext(mode string) func(*antv1.ExecuteLiveRequest) {
	return func(r *antv1.ExecuteLiveRequest) {
		r.TickContext = &antv1.TickContext{Mode: mode}
	}
}

func withTradeContext(mode string) func(*antv1.ExecuteLiveRequest) {
	return func(r *antv1.ExecuteLiveRequest) {
		r.TradeContext = &antv1.TradeContext{Mode: mode}
	}
}

func withTimerContext(mode string) func(*antv1.ExecuteLiveRequest) {
	return func(r *antv1.ExecuteLiveRequest) {
		r.TimerContext = &antv1.TimerContext{Mode: mode}
	}
}

// connectCode extracts the connect.Code from an error returned by connect.NewError.
func connectCode(err error) connect.Code {
	var cerr *connect.Error
	if errors.As(err, &cerr) {
		return cerr.Code()
	}
	return connect.CodeUnknown
}

// TestExecuteLive_RejectsLiveMode_BeforeCompile (T1)
// Live mode in BarContext must be rejected with CodeInvalidArgument before
// compilation begins. The MQL source is deliberately syntactically invalid to
// prove the rejection happens before compile — if we get a mode error (not a
// compile error), mode validation ran first.
func TestExecuteLive_RejectsLiveMode_BeforeCompile(t *testing.T) {
	t.Parallel()
	srv := newExecuteLiveServer()
	req := newExecuteLiveRequest("this is not valid MQL at all !!!",
		withBarContext(modeLive),
	)
	_, err := srv.ExecuteLive(newAuthCtx(), req)
	if err == nil {
		t.Fatal("expected error for live mode, got nil")
	}
	if code := connectCode(err); code != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v (err=%v)", code, err)
	}
	if !strings.Contains(err.Error(), "live mode is not supported") {
		t.Fatalf("error must contain 'live mode is not supported', got: %v", err)
	}
}

// TestExecuteLive_RejectsUnknownAndEmptyMode (T2)
// Table-driven: empty, "LIVE", "backtest", "foo" must all be rejected.
func TestExecuteLive_RejectsUnknownAndEmptyMode(t *testing.T) {
	t.Parallel()
	srv := newExecuteLiveServer()
	badModes := []string{"", "LIVE", "backtest", "foo"}
	for _, mode := range badModes {
		label := mode
		if label == "" {
			label = "<empty>"
		}
		t.Run(label, func(t *testing.T) {
			req := newExecuteLiveRequest("void OnInit(){}",
				withBarContext(mode),
			)
			_, err := srv.ExecuteLive(newAuthCtx(), req)
			if err == nil {
				t.Fatalf("expected error for mode %q, got nil", mode)
			}
			if code := connectCode(err); code != connect.CodeInvalidArgument {
				t.Fatalf("mode %q: expected CodeInvalidArgument, got %v", mode, code)
			}
		})
	}
}

// TestExecuteLive_RejectsLiveModeInNonBarContexts (T3)
// Live mode in TickContext, TradeContext, or TimerContext (without BarContext)
// must also be rejected — all four context types are checked.
func TestExecuteLive_RejectsLiveModeInNonBarContexts(t *testing.T) {
	t.Parallel()
	srv := newExecuteLiveServer()
	cases := []struct {
		name string
		ctx  func(*antv1.ExecuteLiveRequest)
	}{
		{"tick_context", withTickContext(modeLive)},
		{"trade_context", withTradeContext(modeLive)},
		{"timer_context", withTimerContext(modeLive)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := newExecuteLiveRequest("void OnInit(){}", tc.ctx)
			_, err := srv.ExecuteLive(newAuthCtx(), req)
			if err == nil {
				t.Fatalf("expected error for live mode in %s, got nil", tc.name)
			}
			if code := connectCode(err); code != connect.CodeInvalidArgument {
				t.Fatalf("%s: expected CodeInvalidArgument, got %v", tc.name, code)
			}
			if !strings.Contains(err.Error(), "live mode is not supported") {
				t.Fatalf("%s: error must contain 'live mode is not supported', got: %v", tc.name, err)
			}
		})
	}
}

// TestExecuteLive_AllowsPaperMode (T4)
// Paper mode must not be rejected for mode reasons. It may fail for other
// reasons (e.g. compilation), but the error must not mention "mode".
func TestExecuteLive_AllowsPaperMode(t *testing.T) {
	t.Parallel()
	srv := newExecuteLiveServer()
	req := newExecuteLiveRequest("void OnInit(){}",
		withBarContext(modePaper),
	)
	_, err := srv.ExecuteLive(newAuthCtx(), req)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "mode") {
		t.Fatalf("paper mode should not be rejected for mode reasons, got: %v", err)
	}
}

// TestExecuteLive_RejectsNoContext (T5 regression)
// A request with zero contexts must be rejected (fail-closed).
func TestExecuteLive_RejectsNoContext(t *testing.T) {
	t.Parallel()
	srv := newExecuteLiveServer()
	req := newExecuteLiveRequest("void OnInit(){}")
	_, err := srv.ExecuteLive(newAuthCtx(), req)
	if err == nil {
		t.Fatal("expected error for no context, got nil")
	}
	if code := connectCode(err); code != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v (err=%v)", code, err)
	}
}

// TestVMLiveSession_LiveModeStillWorks (T5 — live dispatch path regression)
// VMLiveSession is the production scheduled-live dispatch path and must
// continue to work with live mode. This test directly constructs a session
// with Mode:"live" + non-empty authoritative financial fields, walks through
// Start → SendEvent, and asserts success — proving the mode rejection in the
// public ExecuteLive RPC handler does NOT affect the internal dispatch path.
//
// Adversarial proof (R1-P1, as independently re-verified): adding the S3
// mode rejection to validateBarContextWithMode — the shared validation path
// used by BOTH VMLiveSession.Start (Start → validateFirstBarContext →
// validateBarContext) and dispatchVMLive — makes this test RED, proving the
// test walks the real Start → SendEvent path. (Moving the S3 call into
// executeVMLive does NOT affect this test: VMLiveSession.Start dispatches
// internally via vm_live_session.go dispatch() and never calls
// executeVMLive — verified by mutation during audit.)
func TestVMLiveSession_LiveModeStillWorks(t *testing.T) {
	t.Parallel()

	// Minimal MQL4 strategy that does nothing in start() — sufficient to
	// prove Start/SendEvent succeed without depending on indicator logic.
	const noopMQL = `
int start()
{
    return 0;
}
`

	// Build a 1-bar live context with non-empty authoritative financial
	// fields. validateBarContextWithMode (vm_live_validators.go:187)
	// requires Balance/Equity/Margin/FreeMargin non-empty in live mode;
	// missing them would fail-start for VMLiveSession's own validator,
	// NOT because of P1's mode rejection.
	bars := []liveBar{
		{
			open: "100.0", high: "101.0", low: "99.0", close: "100.5",
			volume: "1000", openTime: 1_700_000_000_000,
		},
	}
	lctx := buildLiveCtx(bars, "TESTUSD", "1m", modeLive)
	lctx.Balance = "10000"
	lctx.Equity = "10000"
	lctx.Margin = "0"
	lctx.FreeMargin = "10000"

	ctx := context.Background()
	sess, resp, err := startSession(ctx, noopMQL, lctx)
	if err != nil {
		t.Fatalf("VMLiveSession.Start with live mode must succeed, got error: %v", err)
	}
	if sess == nil {
		t.Fatal("VMLiveSession.Start returned nil session")
	}
	defer sess.Close()
	if resp == nil {
		t.Fatal("VMLiveSession.Start returned nil response")
	}

	// Send a follow-up TICK event to prove SendEvent also works in live mode.
	tickResp, err := sendTickEvent(ctx, sess,
		buildTickCtx("TESTUSD", decimal.NewFromFloat(100.5), decimal.NewFromFloat(100.6)))
	if err != nil {
		t.Fatalf("VMLiveSession.SendEvent (tick) with live mode must succeed, got error: %v", err)
	}
	if tickResp == nil {
		t.Fatal("VMLiveSession.SendEvent returned nil response")
	}
}
