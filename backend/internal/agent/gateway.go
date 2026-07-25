package agent

import (
	"context"
	"fmt"
	"regexp"
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

	// ── Step 2: Persist to imported_strategies (before backtest — O1: backtest failure must not lose strategy) ──
	persisted := false
	if s.importedRepo != nil {
		uid, _ := uuid.Parse(userID)
		if uid == uuid.Nil {
			s.log.Warn("SubmitStrategy: skip persist, invalid userID", zap.String("userID", userID))
		} else {
			paramsRaw := interp.SerializeParams(runner.Bytecode().Params)
			row := &repository.ImportedStrategy{
				UserID:        uid,
				Name:          deriveNameFromSourceCode(sourceCode, coverage.Version),
				SourceLang:    coverage.Version,
				SourceCode:    sourceCode,
				Params:        paramsRaw,
				CoverageScore: coverage.Score,
			}
			if err := s.importedRepo.Create(ctx, row); err != nil {
				s.log.Warn("SubmitStrategy: persist failed", zap.Error(err))
			} else {
				strategyID = row.ID.String()
				persisted = true
				if s.versionRepo != nil {
					if _, vErr := s.versionRepo.CreateVersion(ctx, row.ID, uid, sourceCode, coverage.Version, "Agent import"); vErr != nil {
						s.log.Warn("SubmitStrategy: create version snapshot failed", zap.Error(vErr))
					}
				}
			}
		}
	}
	// F1: if persist failed, clear strategyID so frontend doesn't get an invalid ID
	if !persisted {
		strategyID = ""
	}

	// ── Steps 3-7: Backtest + LLM analysis + bridging (only when backtest_config provided) ──
	var btResult *backtestPipelineResult
	btCfg := msg.BacktestConfig
	if btCfg != nil {
		btResult, err = s.runBacktestPipeline(ctx, backtestPipelineInput{
			Runner:     runner,
			BtCfg:      btCfg,
			Params:     msg.Params,
			UserID:     userID,
			StrategyID: strategyID,
			SourceCode: sourceCode,
			Coverage:   coverage,
			Language:   language,
		})
		if err != nil {
			s.log.Warn("SubmitStrategy: backtest pipeline failed", zap.Error(err))
		}
	}

	// ── Build response ──
	resp := &antv1.SubmitStrategyResponse{
		StrategyId:     strategyID,
		CompileSuccess: true,
		Mode:           antv1.SubmitMode_SUBMIT_MODE_SYNC,
		CoverageScore:  coverage.Score,
	}
	if btResult != nil {
		resp.Profile = btResult.profile
		resp.Analysis = btResult.analysis
		resp.Result = btResult.btProto
		resp.SemanticDiff = btResult.semanticDiff
		resp.BridgeStatus = btResult.bridgeStatus
		resp.BridgedPythonSource = btResult.bridgedPython
		resp.BridgeCompileError = btResult.bridgeErr
	} else {
		resp.BridgeStatus = "not_attempted"
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
	return FetchBarsForBacktest(ctx, s.marketDataRepo, cfg)
}

// SettingsStore returns the shared settings store instance.
func (s *GatewayServer) SettingsStore() *SettingsStore { return s.settings }

// HookEngine returns the shared hook engine instance.
func (s *GatewayServer) HookEngine() *HookEngine { return s.hooks }

// Generator returns the strategy generator for Phase 2 marketplace integration.
func (s *GatewayServer) Generator() *Generator { return s.generator }

var (
	propertyNameRe = regexp.MustCompile(`#property\s+(?:indicator_name|expert|description)\s+"([^"]+)"`)
	classNameRe    = regexp.MustCompile(`class\s+(\w+)\s*(?::\s*public\s+\w+)?\s*\{`)
)

// deriveNameFromSourceCode extracts a human-friendly strategy name from MQL source.
// Priority: #property indicator_name/expert > class name > fallback "Imported <lang>".
func deriveNameFromSourceCode(source, lang string) string {
	if m := propertyNameRe.FindStringSubmatch(source); len(m) > 1 {
		return m[1]
	}
	if m := classNameRe.FindStringSubmatch(source); len(m) > 1 {
		return m[1]
	}
	return fmt.Sprintf("Imported %s", lang)
}
