package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/repository"
	"alphaforge/internal/service/systemai"
	"alphaforge/strategy/sdk"
	"alphaforge/tools/mql2go"
	"alphaforge/tools/mql2go/interp"
)

// GatewayServer implements ant.v1.AgentGatewayServiceHandler.
// Synchronous strategy submission → compile → backtest → LLM analysis.
type GatewayServer struct {
	marketDataRepo repository.MarketDataStore
	log           *zap.Logger
	bridge         *Bridge    // reuse bridge instance
	profiler       *Profiler  // reuse profiler instance
	interpreter    *Interpreter // reuse interpreter instance
	generator      *Generator // strategy generation from natural language
	memory         *MemoryStore // ADR-0025 §4 three-layer memory
	hooks          *HookEngine  // ADR-0025 §8 lifecycle hooks
	settings       *SettingsStore // ADR-0025 §5 tiered settings
	permissions    *PermissionEngine // ADR-0025 §9 capability permissions
	importedRepo   *repository.ImportedStrategyRepository
	versionRepo    *repository.StrategyVersionRepository
}

var _ antv1c.AgentGatewayServiceHandler = (*GatewayServer)(nil)

// NewGatewayServer creates the Agent Gateway ConnectRPC handler.
func NewGatewayServer(pool *pgxpool.Pool, mdr repository.MarketDataStore, aiSvc *systemai.Service, log *zap.Logger) *GatewayServer {
	cache := NewLLCache(30 * time.Minute)
	profiler := NewProfiler(aiSvc, cache)
	interpreter := NewInterpreter(aiSvc, cache)
	hooks := NewHookEngine(log)
	memory := NewMemoryStore(pool, log)
	settings := NewSettingsStore(pool)
	// Build a minimal backtest repo for the Generator's tools.
	btRepo := repository.NewBacktestRunRepository(pool)
	memExec := func(ctx context.Context, sql string, args ...any) error {
		_, err := pool.Exec(ctx, sql, args...)
		return err
	}
	memQuery := func(ctx context.Context, sql string, args ...any) (string, error) {
		var result string
		err := pool.QueryRow(ctx, sql, args...).Scan(&result)
		return result, err
	}
	generator := NewGenerator(aiSvc, log, cache, memory, mdr, btRepo, memExec, memQuery)
	generator.SetConversationRepo(repository.NewAIConversationRepository(pool))
	return &GatewayServer{
		marketDataRepo: mdr,
		log:            log,
		bridge:         NewBridge(aiSvc, log, cache),
		profiler:       profiler,
		interpreter:    interpreter,
		generator:      generator,
		memory:         memory,
		hooks:          hooks,
		settings:       settings,
		permissions:    NewPermissionEngine(settings),
		importedRepo:   repository.NewImportedStrategyRepository(pool),
		versionRepo:    repository.NewStrategyVersionRepository(pool),
	}
}

// SubmitStrategy handles Agent strategy submission: compile MQL → VM backtest → profile + analysis.
// Synchronous mode only — blocks until backtest completes (<30s).
func (s *GatewayServer) SubmitStrategy(
	ctx context.Context,
	req *connect.Request[antv1.SubmitStrategyRequest],
) (*connect.Response[antv1.SubmitStrategyResponse], error) {
	msg := req.Msg
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}

	// SYNC only — async mode is Phase 3.
	if msg.Mode == antv1.SubmitMode_SUBMIT_MODE_ASYNC {
		return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("async mode not yet supported"))
	}

	sourceCode := msg.SourceCode
	if len(sourceCode) > mql2go.MaxSourceSize {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("source code exceeds max size %d bytes", mql2go.MaxSourceSize))
	}
	if sourceCode == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("source_code is required"))
	}

	strategyID := uuid.New().String()

	// ── ADR-0025 §8: PreStrategySubmit hook ──
	if s.hooks != nil && s.hooks.HasHandlers(HookPreStrategySubmit) {
		uid, _ := uuid.Parse(userID)
		result := s.hooks.Fire(ctx, &HookContext{
			Event:      HookPreStrategySubmit,
			UserID:     uid,
			StrategyID: strategyID,
			Source:     sourceCode,
		})
		if result.Abort {
			return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("strategy submission blocked: %s", result.Reason))
		}
	}

	// ── Step 1: Compile with coverage (language dispatch) ──
	language := msg.Language
	if language == "" {
		language = "mql4" // default
	}

	var runner *mql2go.VMRunner
	var coverage *mql2go.CoverageResult
	var err error

	switch language {
	case "python":
		runner, coverage, err = mql2go.CompilePythonWithCoverage(sourceCode)
	default:
		runner, coverage, err = mql2go.CompileMQLWithCoverage(sourceCode)
	}

	if err != nil {
		s.log.Warn("AgentGateway: compile failed", zap.String("strategyID", strategyID), zap.String("language", language), zap.Error(err))
		return connect.NewResponse(&antv1.SubmitStrategyResponse{
			StrategyId:    strategyID,
			CompileSuccess: false,
			CompileError:  err.Error(),
			Mode:          antv1.SubmitMode_SUBMIT_MODE_SYNC,
		}), nil
	}

	// ── Steps 2-6: Backtest + LLM analysis + bridging (only when backtest_config provided) ──
	var profile *antv1.StrategyProfile
	var analysis *antv1.BacktestAnalysis
	var semanticDiff *antv1.SemanticDiff
	var btProto *antv1.AgentBacktestResult
	bridgeStatus := "not_attempted"
	var bridgedPython string
	var bridgeCompileErr string

	btCfg := msg.BacktestConfig
	if btCfg != nil {
		// ── Step 2: Fetch market data bars ──
		bars, err := s.fetchBars(ctx, btCfg)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetch market data: %w", err))
		}
		if len(bars) < 2 {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("insufficient market data: %d bars (need ≥2)", len(bars)))
		}

		// ── Step 3: Run backtest via VM + SimBroker ──
		btResult, btErr := runVMBacktest(ctx, runner, btCfg, bars, msg.Params)

		// Convert to proto for response + LLM analysis.
		if btErr != nil {
			btProto = &antv1.AgentBacktestResult{Success: false, Error: btErr.Error()}
		} else {
			btProto = buildBacktestResultProto(btResult)
		}

		// ── ADR-0025 §8: PostBacktest hook ──
		if s.hooks != nil && s.hooks.HasHandlers(HookPostBacktest) {
			uid, _ := uuid.Parse(userID)
			s.hooks.Fire(ctx, &HookContext{
				Event:          HookPostBacktest,
				UserID:         uid,
				StrategyID:     strategyID,
				BacktestResult: btProto,
			})
		}

		// ── Step 4: Generate strategy profile (LLM injection point [1]) ──
		var profErr error
		profile, profErr = s.profiler.GenerateProfile(ctx, userID, sourceCode, coverage)
		if profErr != nil {
			s.log.Warn("AgentGateway: profile generation failed", zap.Error(profErr))
		}

		// ── Step 5: Generate backtest analysis (LLM injection point [4]) ──
		if btErr == nil {
			analysis, err = s.interpreter.AnalyzeBacktest(ctx, userID, btProto, profile)
			if err != nil {
				s.log.Warn("AgentGateway: analysis generation failed", zap.Error(err))
			}
		}

		// ── Step 6: Blind-spot bridge (ADR-0024) ──
		// When MQL has blind spots (coverage < 1.0), trigger LLM translation to Python subset.
		// Retry loop: LLM translate → compile → coverage check → backtest → error feedback → retry (max 3).
		if language != "python" && coverage.Score < 1.0 && len(coverage.BlindSpots) > 0 {
			validateBacktest := func(pyRunner *mql2go.VMRunner) error {
				_, btErr := runVMBacktest(ctx, pyRunner, btCfg, bars, msg.Params)
				return btErr
			}

			bridgeResult, bridgeErr := s.bridge.TranslateWithRetry(ctx, userID, sourceCode, coverage, profile, validateBacktest)
			if bridgeErr != nil {
				s.log.Warn("AgentGateway: bridge translation failed", zap.Error(bridgeErr))
				bridgeStatus = "bridge_failed"
				bridgeCompileErr = bridgeErr.Error()
				semanticDiff = buildBridgeFailureReport(coverage)
			} else {
				bridgeStatus = bridgeResult.Status
				bridgeCompileErr = bridgeResult.CompileError
				if bridgeResult.BacktestError != "" {
					bridgeCompileErr = bridgeResult.BacktestError
				}
				if bridgeResult.Status == "success" {
					bridgedPython = bridgeResult.PythonSource
					pyCov := bridgeResult.BridgedCov
					s.log.Info("AgentGateway: bridge translation successful",
						zap.Int("attempts", bridgeResult.Attempts),
						zap.Float64("original_coverage", coverage.Score),
						zap.Float64("bridged_coverage", pyCov.Score))
					if pyCov != nil {
						semanticDiff = &antv1.SemanticDiff{
							Changes:      buildBridgeChanges(coverage, pyCov),
							EffectSummary: fmt.Sprintf("覆盖率 %.0f%% → %.0f%%", coverage.Score*100, pyCov.Score*100),
						}
					}
				} else {
					// Degradation path — generate blind spot report + MT hosted suggestion
					s.log.Warn("AgentGateway: bridge failed after retries",
						zap.Int("attempts", bridgeResult.Attempts),
						zap.String("last_error", truncate(bridgeCompileErr, 200)))
					semanticDiff = buildBridgeFailureReport(coverage)
				}
			}
		}
	}

	// ── Step 7: Persist to imported_strategies ──
	if s.importedRepo != nil {
		uid, _ := uuid.Parse(userID)
		paramsRaw := interp.SerializeParams(runner.Bytecode().Params)
		row := &repository.ImportedStrategy{
			UserID:        uid,
			Name:          fmt.Sprintf("Imported %s", language),
			SourceLang:    language,
			SourceCode:    sourceCode,
			Params:        paramsRaw,
			CoverageScore: coverage.Score,
		}
		if err := s.importedRepo.Create(ctx, row); err != nil {
			s.log.Warn("SubmitStrategy: persist failed", zap.Error(err))
		} else {
			strategyID = row.ID.String()
			if s.versionRepo != nil {
				if _, vErr := s.versionRepo.CreateVersion(ctx, row.ID, uid, sourceCode, language, "Agent import"); vErr != nil {
					s.log.Warn("SubmitStrategy: create version snapshot failed", zap.Error(vErr))
				}
			}
		}
	}

	// ── Build response ──
	resp := &antv1.SubmitStrategyResponse{
		StrategyId:          strategyID,
		CompileSuccess:      true,
		Mode:                antv1.SubmitMode_SUBMIT_MODE_SYNC,
		Profile:             profile,
		Analysis:            analysis,
		CoverageScore:       coverage.Score,
		Result:              btProto,
		SemanticDiff:        semanticDiff,
		BridgeStatus:        bridgeStatus,
		BridgedPythonSource: bridgedPython,
		BridgeCompileError:  bridgeCompileErr,
	}
	for _, bs := range coverage.BlindSpots {
		resp.BlindSpots = append(resp.BlindSpots, &antv1.AgentBlindSpot{
			Builtin:  bs.Builtin,
			Severity: bs.Severity,
			Count:    int32(bs.Count),
		})
	}

	return connect.NewResponse(resp), nil
}

// GenerateStrategy generates a Python subset strategy from natural language via LLM,
// then compiles and backtests it in the VM. Streams progress chunks to the client.
func (s *GatewayServer) GenerateStrategy(
	ctx context.Context,
	req *connect.Request[antv1.AgentGenerateStrategyRequest],
	stream *connect.ServerStream[antv1.AgentGenerateStrategyChunk],
) error {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id: %w", err))
	}

	msg := req.Msg
	if msg.Message == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("message is required"))
	}

	err = s.generator.Generate(ctx, uid, msg, func(chunk *antv1.AgentGenerateStrategyChunk) error {
		return stream.Send(chunk)
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("generation failed: %w", err))
	}
	return nil
}

// ── Internal helpers ──────────────────────────────────────────────────

func (s *GatewayServer) fetchBars(ctx context.Context, cfg *antv1.AgentBacktestConfig) ([]sdk.Bar, error) {
	if s.marketDataRepo == nil {
		return nil, fmt.Errorf("market data store not configured")
	}
	var from, to *time.Time
	if cfg.StartDateMs > 0 {
		t := time.UnixMilli(cfg.StartDateMs)
		from = &t
	}
	if cfg.EndDateMs > 0 {
		t := time.UnixMilli(cfg.EndDateMs)
		to = &t
	}
	chBars, err := s.marketDataRepo.GetKlines(ctx, cfg.Symbol, "", cfg.Timeframe, from, to, 100000)
	if err != nil {
		return nil, err
	}
	// Convert to sdk.Bar (reverse to chronological order)
	bars := make([]sdk.Bar, 0, len(chBars))
	for i := len(chBars) - 1; i >= 0; i-- {
		b := chBars[i]
		bars = append(bars, sdk.Bar{
			Open:      b.Open,
			High:      b.High,
			Low:       b.Low,
			Close:     b.Close,
			Volume:    int64(b.Volume),
			Timestamp: int64(b.OpenTsUnixMs),
		})
	}
	return bars, nil
}

// SettingsStore returns the shared settings store instance.
func (s *GatewayServer) SettingsStore() *SettingsStore { return s.settings }

// HookEngine returns the shared hook engine instance.
func (s *GatewayServer) HookEngine() *HookEngine { return s.hooks }

// Generator returns the strategy generator for Phase 2 marketplace integration.
func (s *GatewayServer) Generator() *Generator { return s.generator }
