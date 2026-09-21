package strategy

import (
	"context"
	"fmt"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/runner"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go"
)

// Session is the interface for live strategy sessions.
// VMLiveSession (in-process Bytecode VM) implements it.
//
// FIX-2026-08-27-SESSION-PROTO-ROUNDTRIP: the interface passes
// *antv1.ExecuteLiveRequest / *antv1.ExecuteLiveResponse pointers instead of
// []byte. VMLiveSession is an in-process implementation, so proto
// marshal/unmarshal is unnecessary — and proto3 collapses empty repeated
// slices to nil on round-trip, which made "no open positions" (empty slice)
// indistinguishable from "data missing" (nil) and caused
// rejectNilRepeatedInLive to reject valid empty-position accounts. Passing
// the struct pointer preserves Go's empty-slice semantics (empty stays empty,
// never becomes nil).
type Session interface {
	Start(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error)
	SendEvent(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error)
	Close() error
}

// VMLiveSession manages a long-running in-process VM strategy instance.
// The MQL source is compiled once to a VMRunner; OnInit is called once;
// OnBar/OnTick/OnTrade are called for each streamed event.
type VMLiveSession struct {
	strategy *mql2go.VMRunner
	runner   *runner.Runner
	started  bool
	diag     *sessionDiag
	// syncDisp is non-nil in live mode once the event loop wires the
	// synchronous broker-mutation dispatcher (VM-LIVE-SYNC-DISPATCH-1).
	syncDisp bool
	// historyFn is the live order-history provider installed before Start;
	// applied to the runner once it is created (LIVE-HISTORY-POOL-1).
	historyFn func(ctx context.Context, from, to int64) ([]sdk.Position, error)
}

// NewVMLiveSession creates a VMLiveSession for an MQL strategy.
func NewVMLiveSession(source string) (*VMLiveSession, error) {
	strategy, err := mql2go.CompileMQL(source)
	if err != nil {
		return nil, fmt.Errorf("compile MQL: %w", err)
	}
	strategy.SetSignalMode(true)
	return &VMLiveSession{strategy: strategy}, nil
}

// NewVMLiveSessionCached creates a VMLiveSession using cached bytecode when available.
// Falls back to full compilation on cache miss or corruption.
func NewVMLiveSessionCached(source string, cachedBytecode []byte) (*VMLiveSession, error) {
	strategy, _, err := compileForLive(source, cachedBytecode, false)
	if err != nil {
		return nil, fmt.Errorf("compile MQL: %w", err)
	}
	strategy.SetSignalMode(true)
	return &VMLiveSession{strategy: strategy}, nil
}

// NewPythonVMLiveSession creates a VMLiveSession for a Python strategy.
// The VMRunner is language-agnostic — only the compilation front-end differs.
func NewPythonVMLiveSession(source string) (*VMLiveSession, error) {
	strategy, err := mql2go.CompilePython(source)
	if err != nil {
		return nil, fmt.Errorf("compile Python: %w", err)
	}
	strategy.SetSignalMode(true)
	return &VMLiveSession{strategy: strategy}, nil
}

// NewPythonVMLiveSessionCached creates a VMLiveSession for a Python strategy
// using cached bytecode when available. Falls back to full compilation on cache miss.
func NewPythonVMLiveSessionCached(source string, cachedBytecode []byte) (*VMLiveSession, error) {
	// VM-AUDIT-2026-08-27-6: use compileForLive which dispatches to
	// CompilePythonCached (SourceHash verification) for Python strategies.
	// bytecodeData is discarded — persistence is handled by the caller (initVMSession).
	runner, _, err := compileForLive(source, cachedBytecode, true)
	if err != nil {
		return nil, fmt.Errorf("compile Python: %w", err)
	}
	runner.SetSignalMode(true)
	return &VMLiveSession{strategy: runner}, nil
}

func (s *VMLiveSession) Start(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error) {
	if s.started {
		return nil, fmt.Errorf("vm live session already started")
	}

	bctx := req.GetBarContext()
	if bctx == nil {
		return nil, fmt.Errorf("first request must have bar_context for initialization")
	}

	params := make(map[string]string)
	for _, p := range bctx.GetParams() {
		params[p.GetKey()] = p.GetValue()
	}

	s.runner = runner.New(runner.Config{
		Symbol:    bctx.Symbol,
		Timeframe: bctx.Timeframe,
		Params:    params,
		Mode:      bctx.Mode,
	})
	s.runner.SetStrategy(s.strategy)
	// LIVE-HISTORY-POOL-1: the runner only exists now — apply the live
	// order-history provider that initVMSession staged on the session.
	if s.historyFn != nil {
		s.runner.SetHistoryProvider(s.historyFn)
	}

	// VM-TRADE-CONTEXT-6 S5: validate first bar context before Init.
	// Invalid OHLCV lengths or financial fields must be rejected before
	// OnInit executes — otherwise the strategy runs with corrupt data.
	if err := validateFirstBarContext(bctx); err != nil {
		return nil, fmt.Errorf("invalid first bar context: %w", err)
	}

	// VM-TRADE-CONTEXT-6 S6: set Login before Init so AccountNumber()
	// returns the authoritative value during OnInit.
	s.runner.SetLogin(bctx.Login)
	// VM-API-TRUTH-3: set account status before Init so IsConnected()/
	// IsDemo()/IsTradeAllowed() return authoritative values during OnInit.
	s.runner.SetAccountStatus(bctx.IsDemo, bctx.IsConnected, bctx.IsTradeAllowed)
	// LIVE-ACCOUNT-FIELDS-1: set account identity before Init (same point as
	// SetLogin/SetAccountStatus).
	s.runner.SetAccountIdentity(bctx.Leverage, bctx.Currency, bctx.Company, bctx.AccountMode)

	if err := s.runner.Init(ctx); err != nil {
		return nil, fmt.Errorf("init: %w", err)
	}

	s.started = true

	return s.dispatch(ctx, req), nil
}

func (s *VMLiveSession) SendEvent(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error) {
	if !s.started {
		return nil, fmt.Errorf("vm live session not started")
	}

	return s.dispatch(ctx, req), nil
}

// SetSyncDispatcher installs the synchronous broker-mutation dispatcher on
// the compiled VM (VM-LIVE-SYNC-DISPATCH-1, R1). The event loop calls this
// before each event in live mode; signal-mode trade builtins then execute
// mutations inside the event and return real broker outcomes.
// SetHistoryProvider stages the live order-history provider on the session;
// the runner only exists after Start, where it is applied
// (LIVE-HISTORY-POOL-1).
func (s *VMLiveSession) SetHistoryProvider(fn func(ctx context.Context, from, to int64) ([]sdk.Position, error)) {
	s.historyFn = fn
}

func (s *VMLiveSession) SetSyncDispatcher(fn func(*sdk.Signal) (int64, error)) {
	s.strategy.SetSyncDispatcher(fn)
	s.syncDisp = fn != nil
}

// SyncDispatched reports whether this session executes signals
// synchronously inside the VM event (live mode). dispatchResponse uses it
// to skip the post-event async dispatch — the mutation already ran.
func (s *VMLiveSession) SyncDispatched() bool { return s.syncDisp }

func (s *VMLiveSession) Close() error {
	if !s.started {
		return nil
	}
	s.started = false
	if s.runner != nil {
		_ = s.runner.Deinit(context.Background(), "session_close")
	}
	return nil
}

func (s *VMLiveSession) dispatch(ctx context.Context, req *antv1.ExecuteLiveRequest) *antv1.ExecuteLiveResponse {
	var resp *antv1.ExecuteLiveResponse
	var evalKind int
	switch req.GetRequestType() {
	case antv1.RequestType_REQUEST_TYPE_BAR:
		resp = vmHandleBar(ctx, s.runner, req.GetBarContext())
		evalKind = evalKindBar

	case antv1.RequestType_REQUEST_TYPE_TICK:
		resp = vmHandleTick(ctx, s.runner, req.GetTickContext())
		evalKind = evalKindTick

	case antv1.RequestType_REQUEST_TYPE_TRADE:
		resp = vmHandleTrade(ctx, s.runner, req.GetTradeContext())
		evalKind = evalKindTrade

	case antv1.RequestType_REQUEST_TYPE_TIMER:
		resp = vmHandleTimer(ctx, s.runner, req.GetTimerContext())
		evalKind = -1

	default:
		// VM-AUDIT-2026-08-27-5: unknown request types must not be silently
		// treated as bar events even if a stale BarContext is present. Return
		// an explicit error so the caller sees the unknown type instead of
		// executing the strategy on a misinterpreted request.
		return &antv1.ExecuteLiveResponse{
			Success: false,
			Error:   fmt.Sprintf("unknown request type: %s", req.GetRequestType()),
		}
	}

	if s.diag != nil && evalKind >= 0 {
		s.diag.RecordEval(evalKind)
		if resp.GetSuccess() {
			s.diag.RecordIndicators(s.strategy.LastIndicators(), s.strategy.OrdersTotal())
		}
	}

	return resp
}

// SetDiag attaches a sessionDiag for L1/L2 diagnostics capture.
// Called after session creation, once ActiveSession is available.
func (s *VMLiveSession) SetDiag(d *sessionDiag) {
	s.diag = d
}

// Ensure VMLiveSession implements Session.
var _ Session = (*VMLiveSession)(nil)
