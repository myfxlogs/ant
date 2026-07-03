package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
	"anttrader/internal/service/systemai"
	"anttrader/strategy/sdk"
	"anttrader/tools/mql2go"
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
}

var _ antv1c.AgentGatewayServiceHandler = (*GatewayServer)(nil)

// NewGatewayServer creates the Agent Gateway ConnectRPC handler.
func NewGatewayServer(pool *pgxpool.Pool, mdr repository.MarketDataStore, aiSvc *systemai.Service, log *zap.Logger) *GatewayServer {
	cache := NewLLCache(30 * time.Minute)
	profiler := NewProfiler(aiSvc, cache)
	interpreter := NewInterpreter(aiSvc, cache)
	return &GatewayServer{
		marketDataRepo: mdr,
		log:            log,
		bridge:         NewBridge(aiSvc, log, cache),
		profiler:       profiler,
		interpreter:    interpreter,
		generator:      NewGenerator(aiSvc, log, profiler, interpreter, cache, NewMemoryStore(pool)),
		memory:         NewMemoryStore(pool),
		hooks:          NewHookEngine(log),
		settings:       NewSettingsStore(pool),
		permissions:    NewPermissionEngine(NewSettingsStore(pool)),
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
	if msg.Mode == antv1.SubmitMode_SUBMIT_ASYNC {
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
			Mode:          antv1.SubmitMode_SUBMIT_SYNC,
		}), nil
	}

	// ── Step 2: Fetch market data bars ──
	btCfg := msg.BacktestConfig
	if btCfg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("backtest_config is required"))
	}
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
	var btProto *antv1.AgentBacktestResult
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
	profile, profErr := s.profiler.GenerateProfile(ctx, userID, sourceCode, coverage)
	if profErr != nil {
		s.log.Warn("AgentGateway: profile generation failed", zap.Error(profErr))
	}

	// ── Step 5: Generate backtest analysis (LLM injection point [4]) ──
	var analysis *antv1.BacktestAnalysis
	if btErr == nil {
		analysis, err = s.interpreter.AnalyzeBacktest(ctx, userID, btProto, profile)
		if err != nil {
			s.log.Warn("AgentGateway: analysis generation failed", zap.Error(err))
		}
	}

	// ── Step 6: Blind-spot bridge (ADR-0024) ──
	// When MQL has blind spots (coverage < 1.0), trigger LLM translation to Python subset.
	// Retry loop: LLM translate → compile → coverage check → backtest → error feedback → retry (max 3).
	var semanticDiff *antv1.SemanticDiff
	bridgeStatus := "not_attempted"
	var bridgedPython string
	var bridgeCompileErr string

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

	// ── Build response ──
	resp := &antv1.SubmitStrategyResponse{
		StrategyId:          strategyID,
		CompileSuccess:      true,
		Mode:                antv1.SubmitMode_SUBMIT_SYNC,
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

	btCfg := msg.BacktestConfig
	if btCfg == nil {
		btCfg = &antv1.AgentBacktestConfig{
			Symbol:    msg.Symbol,
			Timeframe: msg.Timeframe,
		}
	}

	bars, err := s.fetchBars(ctx, btCfg)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("fetch market data: %w", err))
	}
	if len(bars) < 2 {
		return connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("insufficient market data: %d bars (need ≥2)", len(bars)))
	}

	err = s.generator.Generate(ctx, uid, msg, bars, func(chunk *antv1.AgentGenerateStrategyChunk) error {
		return stream.Send(chunk)
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("generation failed: %w", err))
	}
	return nil
}

// SearchExperience searches the knowledge base for similar experiences (ADR-0025 §4).
func (s *GatewayServer) SearchExperience(
	ctx context.Context,
	req *connect.Request[antv1.SearchExperienceRequest],
) (*connect.Response[antv1.SearchExperienceResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user ID"))
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.SearchExperienceResponse{}), nil
	}

	entries, err := s.memory.SearchExperiences(ctx, uid, req.Msg.Query, req.Msg.Category, int(req.Msg.Limit))
	if err != nil {
		s.log.Warn("SearchExperience failed", zap.Error(err))
		return connect.NewResponse(&antv1.SearchExperienceResponse{}), nil
	}
	return connect.NewResponse(&antv1.SearchExperienceResponse{Entries: entries}), nil
}

// StoreExperience stores an experience to the knowledge base (ADR-0025 §6.2).
func (s *GatewayServer) StoreExperience(
	ctx context.Context,
	req *connect.Request[antv1.StoreExperienceRequest],
) (*connect.Response[antv1.StoreExperienceResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user ID"))
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.StoreExperienceResponse{
			Id:     uuid.New().String(),
			Success: false,
		}), nil
	}

	id, err := s.memory.StoreExperience(ctx, uid, req.Msg.Category, req.Msg.Content, req.Msg.Fingerprint, nil, "")
	if err != nil {
		s.log.Warn("StoreExperience failed", zap.Error(err))
		return connect.NewResponse(&antv1.StoreExperienceResponse{
			Success: false,
		}), nil
	}
	return connect.NewResponse(&antv1.StoreExperienceResponse{
		Id:     id,
		Success: true,
	}), nil
}

// ListMemory returns user templates and agent experiences (ADR-0025 §4.4).
func (s *GatewayServer) ListMemory(
	ctx context.Context,
	req *connect.Request[antv1.ListMemoryRequest],
) (*connect.Response[antv1.ListMemoryResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user ID"))
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.ListMemoryResponse{}), nil
	}

	templates, err := s.memory.ListUserTemplates(ctx, uid)
	if err != nil {
		s.log.Warn("ListMemory: templates failed", zap.Error(err))
	}
	experiences, err := s.memory.ListAgentExperiences(ctx, uid)
	if err != nil {
		s.log.Warn("ListMemory: experiences failed", zap.Error(err))
	}
	return connect.NewResponse(&antv1.ListMemoryResponse{
		Templates:   templates,
		Experiences: experiences,
	}), nil
}

// SaveUserTemplate saves a user strategy template (ADR-0025 §4.4).
func (s *GatewayServer) SaveUserTemplate(
	ctx context.Context,
	req *connect.Request[antv1.SaveUserTemplateRequest],
) (*connect.Response[antv1.SaveUserTemplateResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user ID"))
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.SaveUserTemplateResponse{Success: false}), nil
	}

	err = s.memory.SaveUserTemplate(ctx, uid, req.Msg.Name, req.Msg.Content, req.Msg.ScopeJson)
	if err != nil {
		s.log.Warn("SaveUserTemplate failed", zap.Error(err))
		return connect.NewResponse(&antv1.SaveUserTemplateResponse{Success: false}), nil
	}
	return connect.NewResponse(&antv1.SaveUserTemplateResponse{Success: true}), nil
}

// DeleteUserTemplate deletes a user strategy template (ADR-0025 §4.4).
func (s *GatewayServer) DeleteUserTemplate(
	ctx context.Context,
	req *connect.Request[antv1.DeleteUserTemplateRequest],
) (*connect.Response[antv1.DeleteUserTemplateResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user ID"))
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.DeleteUserTemplateResponse{Success: false}), nil
	}

	err = s.memory.DeleteUserTemplate(ctx, uid, req.Msg.Id)
	if err != nil {
		s.log.Warn("DeleteUserTemplate failed", zap.Error(err))
		return connect.NewResponse(&antv1.DeleteUserTemplateResponse{Success: false}), nil
	}
	return connect.NewResponse(&antv1.DeleteUserTemplateResponse{Success: true}), nil
}

// DeleteAgentExperience deletes an agent experience (ADR-0025 §4.4).
func (s *GatewayServer) DeleteAgentExperience(
	ctx context.Context,
	req *connect.Request[antv1.DeleteAgentExperienceRequest],
) (*connect.Response[antv1.DeleteAgentExperienceResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user ID"))
	}

	if s.memory == nil {
		return connect.NewResponse(&antv1.DeleteAgentExperienceResponse{Success: false}), nil
	}

	err = s.memory.DeleteAgentExperience(ctx, uid, req.Msg.Id)
	if err != nil {
		s.log.Warn("DeleteAgentExperience failed", zap.Error(err))
		return connect.NewResponse(&antv1.DeleteAgentExperienceResponse{Success: false}), nil
	}
	return connect.NewResponse(&antv1.DeleteAgentExperienceResponse{Success: true}), nil
}

// GetAgentSettings returns resolved settings for the authenticated user (ADR-0025 §5).
func (s *GatewayServer) GetAgentSettings(
	ctx context.Context,
	req *connect.Request[antv1.GetAgentSettingsRequest],
) (*connect.Response[antv1.GetAgentSettingsResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user ID"))
	}

	if s.settings == nil {
		return connect.NewResponse(&antv1.GetAgentSettingsResponse{}), nil
	}

	settings, err := s.settings.GetSettings(ctx, uid)
	if err != nil {
		s.log.Warn("GetAgentSettings failed", zap.Error(err))
		return connect.NewResponse(&antv1.GetAgentSettingsResponse{}), nil
	}

	var entries []*antv1.AgentSettingEntry
	for k, v := range settings {
		entries = append(entries, &antv1.AgentSettingEntry{Key: k, Value: v, Tier: "resolved"})
	}
	return connect.NewResponse(&antv1.GetAgentSettingsResponse{Settings: entries}), nil
}

// SetUserSetting sets a user-level setting override (ADR-0025 §5).
func (s *GatewayServer) SetUserSetting(
	ctx context.Context,
	req *connect.Request[antv1.SetUserSettingRequest],
) (*connect.Response[antv1.SetUserSettingResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user ID"))
	}

	if s.settings == nil {
		return connect.NewResponse(&antv1.SetUserSettingResponse{Success: false}), nil
	}

	err = s.settings.SetUserSetting(ctx, uid, req.Msg.Key, req.Msg.Value)
	if err != nil {
		s.log.Warn("SetUserSetting failed", zap.Error(err))
		return connect.NewResponse(&antv1.SetUserSettingResponse{Success: false}), nil
	}
	return connect.NewResponse(&antv1.SetUserSettingResponse{Success: true}), nil
}

// DeleteUserSetting removes a user-level setting override (ADR-0025 §5).
func (s *GatewayServer) DeleteUserSetting(
	ctx context.Context,
	req *connect.Request[antv1.DeleteUserSettingRequest],
) (*connect.Response[antv1.DeleteUserSettingResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user ID"))
	}

	if s.settings == nil {
		return connect.NewResponse(&antv1.DeleteUserSettingResponse{Success: false}), nil
	}

	err = s.settings.DeleteUserSetting(ctx, uid, req.Msg.Key)
	if err != nil {
		s.log.Warn("DeleteUserSetting failed", zap.Error(err))
		return connect.NewResponse(&antv1.DeleteUserSettingResponse{Success: false}), nil
	}
	return connect.NewResponse(&antv1.DeleteUserSettingResponse{Success: true}), nil
}

// GetCapabilities returns the capability permissions for the authenticated user (ADR-0025 §9).
func (s *GatewayServer) GetCapabilities(
	ctx context.Context,
	req *connect.Request[antv1.GetCapabilitiesRequest],
) (*connect.Response[antv1.GetCapabilitiesResponse], error) {
	userID := interceptor.GetUserID(ctx)
	if userID == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("not authenticated"))
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user ID"))
	}

	if s.permissions == nil {
		return connect.NewResponse(&antv1.GetCapabilitiesResponse{}), nil
	}

	caps := s.permissions.CapabilitiesForUser(ctx, uid)
	var entries []*antv1.CapabilityEntry
	for cap, allowed := range caps {
		entries = append(entries, &antv1.CapabilityEntry{
			Capability: string(cap),
			Allowed:    allowed,
		})
	}
	return connect.NewResponse(&antv1.GetCapabilitiesResponse{Capabilities: entries}), nil
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
