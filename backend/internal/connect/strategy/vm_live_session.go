package strategy

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/runner"
	"alphaforge/tools/mql2go"
)

// Session is the interface for live strategy sessions.
// VMLiveSession (in-process Bytecode VM) implements it.
type Session interface {
	Start(ctx context.Context, reqBytes []byte) ([]byte, error)
	SendEvent(ctx context.Context, reqBytes []byte) ([]byte, error)
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
	strategy, _, err := mql2go.CompileMQLCached(source, cachedBytecode)
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
// VM-CACHE-INTEGRITY-3: uses CompilePythonCached to verify source hash before
// accepting cached bytecode — no stale cache accepted.
// VM-CACHE-INTEGRITY-5: CompilePythonCached also restores CoverageResult on
// cache hit and verifies language (Version == "python"). If coverage restoration
// fails, the error is returned (not silently degraded).
func NewPythonVMLiveSessionCached(source string, cachedBytecode []byte) (*VMLiveSession, error) {
	if len(cachedBytecode) > 0 {
		// VM-CACHE-INTEGRITY-3/5: use CompilePythonCached which verifies
		// SourceHash + language + restores CoverageResult.
		// If this fails (hash mismatch, corrupt, wrong language, or coverage
		// restoration failure), fall through to cold compile.
		runner, _, err := mql2go.CompilePythonCached(source, cachedBytecode)
		if err == nil && runner != nil {
			runner.SetSignalMode(true)
			return &VMLiveSession{strategy: runner}, nil
		}
		// Cache miss — fall through to cold compile.
	}
	strategy, err := mql2go.CompilePython(source)
	if err != nil {
		return nil, fmt.Errorf("compile Python: %w", err)
	}
	strategy.SetSignalMode(true)
	return &VMLiveSession{strategy: strategy}, nil
}

func (s *VMLiveSession) Start(ctx context.Context, reqBytes []byte) ([]byte, error) {
	if s.started {
		return nil, fmt.Errorf("vm live session already started")
	}

	var req antv1.ExecuteLiveRequest
	if err := proto.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}

	bctx := req.GetBarContext()
	if bctx == nil {
		return nil, fmt.Errorf("first request must have bar_context for initialization")
	}

	// VM-TRADE-CONTEXT-6: validate the first bar context BEFORE Init so that
	// a bad request cannot execute OnInit and then fail mid-strategy.
	if err := validateFirstBarContext(bctx); err != nil {
		return nil, fmt.Errorf("first bar_context invalid: %w", err)
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

	// VM-TRADE-CONTEXT-5: inject account identity BEFORE Init so that
	// AccountNumber()/AccountCompany() are available during OnInit.
	s.runner.UpdateAccountIdentity(bctx.Login, bctx.Company)
	// VM-API-TRUTH-3: inject account status BEFORE Init so that
	// IsDemo()/IsConnected()/IsTradeAllowed() are available during OnInit.
	s.runner.UpdateAccountStatus(bctx.IsDemo, bctx.IsConnected, bctx.IsTradeAllowed)

	if err := s.runner.Init(ctx); err != nil {
		return nil, fmt.Errorf("init: %w", err)
	}

	s.started = true

	resp := s.dispatch(ctx, &req)
	return proto.Marshal(resp)
}

func (s *VMLiveSession) SendEvent(ctx context.Context, reqBytes []byte) ([]byte, error) {
	if !s.started {
		return nil, fmt.Errorf("vm live session not started")
	}

	var req antv1.ExecuteLiveRequest
	if err := proto.Unmarshal(reqBytes, &req); err != nil {
		return nil, fmt.Errorf("unmarshal request: %w", err)
	}

	resp := s.dispatch(ctx, &req)
	return proto.Marshal(resp)
}

// validateFirstBarContext validates the first bar context before OnInit.
// VM-TRADE-CONTEXT-6 round 4: delegates to the shared validateBarContext
// so VMLiveSession.Start and dispatchVMLive use the same validation logic.
func validateFirstBarContext(bctx *antv1.LiveStrategyContext) error {
	return validateBarContext(bctx)
}

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
		if bctx := req.GetBarContext(); bctx != nil {
			resp = vmHandleBar(ctx, s.runner, bctx)
			evalKind = evalKindBar
		} else {
			return &antv1.ExecuteLiveResponse{Success: false, Error: "unknown request type"}
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
