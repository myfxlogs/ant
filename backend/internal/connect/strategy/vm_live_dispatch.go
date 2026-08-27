package strategy

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/strategy/runner"
	"alphaforge/tools/mql2go"
)

// executeVMLive runs a single live event via the in-process Bytecode VM.
// MQL source → CompileMQLCached → VMRunner → runner.Runner → dispatch event → ExecuteLiveResponse.
// Uses bytecode cache from imported_strategies when strategy_id is available.
func (s *StrategyExecutionServer) executeVMLive(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error) {
	var cachedBytecode []byte
	if req.StrategyId != "" && s.importedRepo != nil {
		if sid, parseErr := uuid.Parse(req.StrategyId); parseErr == nil {
			cachedBytecode, _ = s.importedRepo.GetBytecode(ctx, sid)
		}
	}

	strategy, bcData, err := compileForLive(req.StrategyCode, cachedBytecode, false)
	if err != nil {
		return nil, fmt.Errorf("compile MQL: %w", err)
	}

	// Persist newly compiled bytecode for future runs.
	if bcData != nil && req.StrategyId != "" && s.importedRepo != nil {
		if sid, parseErr := uuid.Parse(req.StrategyId); parseErr == nil && sid != uuid.Nil {
			if saveErr := s.importedRepo.SaveBytecode(ctx, sid, bcData); saveErr != nil {
				s.log.Warn("executeVMLive: save bytecode cache failed", zap.Error(saveErr))
			}
		}
	}

	return s.dispatchVMLive(ctx, req, strategy)
}

// executePythonVMLive runs a single live event via the in-process Bytecode VM for Python strategies.
// Python source → CompilePython → VMRunner → runner.Runner → dispatch event → ExecuteLiveResponse.
// Uses bytecode cache from imported_strategies when strategy_id is available.
func (s *StrategyExecutionServer) executePythonVMLive(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error) {
	var cachedBytecode []byte
	if req.StrategyId != "" && s.importedRepo != nil {
		if sid, parseErr := uuid.Parse(req.StrategyId); parseErr == nil {
			cachedBytecode, _ = s.importedRepo.GetBytecode(ctx, sid)
		}
	}

	// VM-AUDIT-2026-08-27-6: use compileForLive which dispatches to
	// CompilePythonCached (SourceHash verification) for Python strategies.
	strategy, bcData, err := compileForLive(req.StrategyCode, cachedBytecode, true)
	if err != nil {
		return nil, fmt.Errorf("compile Python: %w", err)
	}

	// Persist newly compiled bytecode for future runs.
	// bcData is non-nil on both cache hit (returns cachedBytecode input) and
	// cold compile (fresh marshal); SaveBytecode is idempotent (mirrors MQL path).
	if bcData != nil && req.StrategyId != "" && s.importedRepo != nil {
		if sid, parseErr := uuid.Parse(req.StrategyId); parseErr == nil && sid != uuid.Nil {
			if saveErr := s.importedRepo.SaveBytecode(ctx, sid, bcData); saveErr != nil {
				s.log.Warn("executePythonVMLive: save bytecode cache failed", zap.Error(saveErr))
			}
		}
	}

	return s.dispatchVMLive(ctx, req, strategy)
}

// dispatchVMLive builds a runner from the compiled VMRunner and dispatches the live event.
// Shared by MQL and Python live execution paths.
func (s *StrategyExecutionServer) dispatchVMLive(ctx context.Context, req *antv1.ExecuteLiveRequest, strategy *mql2go.VMRunner) (*antv1.ExecuteLiveResponse, error) {
	bctx := req.GetBarContext()
	if bctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "first request must have bar_context for initialization"}, nil
	}

	params := make(map[string]string)
	for _, p := range bctx.GetParams() {
		params[p.GetKey()] = p.GetValue()
	}

	r := runner.New(runner.Config{
		Symbol:    bctx.Symbol,
		Timeframe: bctx.Timeframe,
		Params:    params,
		Mode:      bctx.Mode,
	})
	r.SetStrategy(strategy)

	// VM-TRADE-CONTEXT-6 S5: validate first bar context before Init.
	// Invalid OHLCV lengths or financial fields must be rejected before
	// OnInit executes — otherwise g_init=1 makes it impossible to distinguish
	// a valid init from a corrupt-data init.
	if err := validateFirstBarContext(bctx); err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "invalid first bar context: " + err.Error()}, nil
	}

	// VM-TRADE-CONTEXT-6 S6: set Login before Init so AccountNumber()
	// returns the authoritative value during OnInit.
	r.SetLogin(bctx.Login)

	if err := r.Init(ctx); err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: err.Error()}, nil
	}

	switch req.GetRequestType() {
	case antv1.RequestType_REQUEST_TYPE_BAR:
		return vmHandleBar(ctx, r, bctx), nil
	case antv1.RequestType_REQUEST_TYPE_TICK:
		return vmHandleTick(ctx, r, req.GetTickContext()), nil
	case antv1.RequestType_REQUEST_TYPE_TRADE:
		return vmHandleTrade(ctx, r, req.GetTradeContext()), nil
	case antv1.RequestType_REQUEST_TYPE_TIMER:
		return vmHandleTimer(ctx, r, req.GetTimerContext()), nil
	default:
		if bctx != nil {
			return vmHandleBar(ctx, r, bctx), nil
		}
		return &antv1.ExecuteLiveResponse{Success: false, Error: "unknown request type"}, nil
	}
}
