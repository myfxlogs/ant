package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/tools/mql2go"
)

// backtestPipelineResult holds outputs from the backtest+LLM+bridge pipeline (Steps 3-7 of SubmitStrategy).
type backtestPipelineResult struct {
	profile       *antv1.StrategyProfile
	analysis      *antv1.BacktestAnalysis
	semanticDiff  *antv1.SemanticDiff
	btProto       *antv1.AgentBacktestResult
	bridgeStatus  string
	bridgedPython string
	bridgeErr     string
}

// backtestPipelineInput holds all parameters for the backtest pipeline.
type backtestPipelineInput struct {
	Runner     *mql2go.VMRunner
	BtCfg      *antv1.AgentBacktestConfig
	Params     map[string]string
	UserID     string
	StrategyID string
	SourceCode string
	Coverage   *mql2go.CoverageResult
	Language   string
}

// runBacktestPipeline executes Steps 3-7: fetch bars → backtest → profile → analysis → bridge.
// Errors from individual steps are logged but do not abort the pipeline; only fetchBars
// failures are returned as an error (caller treats it as non-fatal — strategy is already persisted).
func (s *GatewayServer) runBacktestPipeline(
	ctx context.Context,
	in backtestPipelineInput,
) (*backtestPipelineResult, error) {
	// Step 3: Fetch market data bars
	bars, err := s.fetchBars(ctx, in.BtCfg)
	if err != nil {
		return nil, fmt.Errorf("fetch market data: %w", err)
	}
	if len(bars) < 2 {
		return nil, fmt.Errorf("insufficient market data: %d bars (need ≥2)", len(bars))
	}

	result := &backtestPipelineResult{bridgeStatus: "not_attempted"}

	// Step 4: Run backtest via VM + SimBroker
	btResult, btErr := runVMBacktest(ctx, in.Runner, in.BtCfg, bars, in.Params)
	if btErr != nil {
		result.btProto = &antv1.AgentBacktestResult{Success: false, Error: btErr.Error()}
	} else {
		result.btProto = buildBacktestResultProto(btResult)
	}

	// PostBacktest hook (ADR-0025 §8)
	if s.hooks != nil && s.hooks.HasHandlers(HookPostBacktest) {
		uid, _ := uuid.Parse(in.UserID)
		s.hooks.Fire(ctx, &HookContext{
			Event:          HookPostBacktest,
			UserID:         uid,
			StrategyID:     in.StrategyID,
			BacktestResult: result.btProto,
		})
	}

	// Step 5: Generate strategy profile (LLM injection point [1])
	profile, profErr := s.profiler.GenerateProfile(ctx, in.UserID, in.SourceCode, in.Coverage)
	if profErr != nil {
		s.log.Warn("AgentGateway: profile generation failed", zap.Error(profErr))
	}
	result.profile = profile

	// Step 6: Generate backtest analysis (LLM injection point [4])
	if btErr == nil {
		analysis, analysisErr := s.interpreter.AnalyzeBacktest(ctx, in.UserID, result.btProto, profile)
		if analysisErr != nil {
			s.log.Warn("AgentGateway: analysis generation failed", zap.Error(analysisErr))
		}
		result.analysis = analysis
	}

	// Step 7: Blind-spot bridge (ADR-0024)
	if in.Language != "python" && in.Coverage.Score < 1.0 && len(in.Coverage.BlindSpots) > 0 {
		validateBacktest := func(pyRunner *mql2go.VMRunner) error {
			_, btErr := runVMBacktest(ctx, pyRunner, in.BtCfg, bars, in.Params)
			return btErr
		}

		bridgeResult, bridgeErr := s.bridge.TranslateWithRetry(ctx, in.UserID, in.SourceCode, in.Coverage, profile, validateBacktest)
		if bridgeErr != nil {
			s.log.Warn("AgentGateway: bridge translation failed", zap.Error(bridgeErr))
			result.bridgeStatus = "bridge_failed"
			result.bridgeErr = bridgeErr.Error()
			result.semanticDiff = buildBridgeFailureReport(in.Coverage)
		} else {
			result.bridgeStatus = bridgeResult.Status
			result.bridgeErr = bridgeResult.CompileError
			if bridgeResult.BacktestError != "" {
				result.bridgeErr = bridgeResult.BacktestError
			}
			if bridgeResult.Status == "success" {
				result.bridgedPython = bridgeResult.PythonSource
				pyCov := bridgeResult.BridgedCov
				s.log.Info("AgentGateway: bridge translation successful",
					zap.Int("attempts", bridgeResult.Attempts),
					zap.Float64("original_coverage", in.Coverage.Score),
					zap.Float64("bridged_coverage", pyCov.Score))
				if pyCov != nil {
					result.semanticDiff = &antv1.SemanticDiff{
						Changes:       buildBridgeChanges(in.Coverage, pyCov),
						EffectSummary: fmt.Sprintf("覆盖率 %.0f%% → %.0f%%", in.Coverage.Score*100, pyCov.Score*100),
					}
				}
			} else {
				s.log.Warn("AgentGateway: bridge failed after retries",
					zap.Int("attempts", bridgeResult.Attempts),
					zap.String("last_error", truncate(result.bridgeErr, 200)))
				result.semanticDiff = buildBridgeFailureReport(in.Coverage)
			}
		}
	}

	return result, nil
}
