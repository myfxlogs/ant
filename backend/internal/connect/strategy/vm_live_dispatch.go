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

	strategy, bcData, err := mql2go.CompileMQLCached(req.StrategyCode, cachedBytecode)
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
// Python source → CompilePythonCached → VMRunner → runner.Runner → dispatch event → ExecuteLiveResponse.
// Uses bytecode cache from imported_strategies when strategy_id is available.
// VM-CACHE-INTEGRITY-5: uses CompilePythonCached (not CompileMQLFromBytecode directly)
// to verify SourceHash + language (Version == "python") + restore CoverageResult.
func (s *StrategyExecutionServer) executePythonVMLive(ctx context.Context, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error) {
	var cachedBytecode []byte
	if req.StrategyId != "" && s.importedRepo != nil {
		if sid, parseErr := uuid.Parse(req.StrategyId); parseErr == nil {
			cachedBytecode, _ = s.importedRepo.GetBytecode(ctx, sid)
		}
	}

	// VM-CACHE-INTEGRITY-5: use CompilePythonCached which verifies SourceHash,
	// language (Version == "python"), and restores CoverageResult on cache hit.
	// Never use CompileMQLFromBytecode directly for Python — it would accept
	// MQL bytecode without language validation.
	strategy, _, err := mql2go.CompilePythonCached(req.StrategyCode, cachedBytecode)
	if err != nil {
		return nil, fmt.Errorf("compile Python: %w", err)
	}

	// Persist newly compiled bytecode for future runs.
	if req.StrategyId != "" && s.importedRepo != nil {
		if sid, parseErr := uuid.Parse(req.StrategyId); parseErr == nil && sid != uuid.Nil {
			if bcData, mErr := mql2go.MarshalBytecode(strategy.Bytecode()); mErr == nil {
				if saveErr := s.importedRepo.SaveBytecode(ctx, sid, bcData); saveErr != nil {
					s.log.Warn("executePythonVMLive: save bytecode cache failed", zap.Error(saveErr))
				}
			}
		}
	}

	return s.dispatchVMLive(ctx, req, strategy)
}

// dispatchVMLive builds a runner from the compiled VMRunner and dispatches the live event.
// Shared by MQL and Python live execution paths.
// VM-TRADE-CONTEXT-6 round 4: validates the first bar context BEFORE r.Init
// so that a bad request cannot execute OnInit and then fail mid-strategy.
func (s *StrategyExecutionServer) dispatchVMLive(ctx context.Context, req *antv1.ExecuteLiveRequest, strategy *mql2go.VMRunner) (*antv1.ExecuteLiveResponse, error) {
	bctx := req.GetBarContext()
	if bctx == nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: "first request must have bar_context for initialization"}, nil
	}

	// VM-TRADE-CONTEXT-6 round 4: validate ALL fields before Init.
	// Previously r.Init was called before vmHandleBar validation, allowing
	// OnInit to execute with invalid data (g_init would be set even with
	// bad decimals). Now validation happens first.
	if err := validateBarContext(bctx); err != nil {
		return &antv1.ExecuteLiveResponse{Success: false, Error: fmt.Sprintf("first bar_context invalid: %v", err)}, nil
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

	// VM-TRADE-CONTEXT-5: inject account identity BEFORE Init.
	r.UpdateAccountIdentity(bctx.Login, bctx.Company)
	// VM-API-TRUTH-3: inject account status BEFORE Init.
	r.UpdateAccountStatus(bctx.IsDemo, bctx.IsConnected, bctx.IsTradeAllowed)

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
