package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/service/systemai"
	"anttrader/strategy/sdk"
	"anttrader/tools/mql2go"
)

const (
	maxGenerateRetries = 3
	maxProfileRetries  = 2
	maxAnalysisRetries = 2
)

// costCeilingUSD is the hard LLM cost limit per strategy generation (ADR-0024 §D7).
const costCeilingUSD = 0.50

// Estimated cost per LLM call type (ADR-0024 §D7 cost model).
const (
	estCostCodeGen    = 0.004 // ~2000 in + ~1500 out tokens
	estCostProfile    = 0.0004 // ~500 in + ~200 out tokens
	estCostAnalysis   = 0.0006 // ~800 in + ~300 out tokens
)

// Generator orchestrates the strategy generation Agent loop.
// ADR-0024 Phase 3: natural language → strategy profile → LLM → Python subset → compile_py → VM backtest.
//
// Phase 3 simplification: the Agent loop runs in Go (not Python) with 3 retries (not 50 iterations).
// Phase 4 will migrate to a Python Agent process with pandas/optuna/pgvector per ADR-0024 §5.1.
type Generator struct {
	aiSvc       *systemai.Service
	log         *zap.Logger
	profiler    *Profiler
	interpreter *Interpreter
	cache       *LLCache
	memory      *MemoryStore
	retrospect  *RetrospectAgent
	hooks       *HookEngine
	settings    *SettingsStore
}

// NewGenerator creates the strategy generation orchestrator.
func NewGenerator(aiSvc *systemai.Service, log *zap.Logger, profiler *Profiler, interpreter *Interpreter, cache *LLCache, memory *MemoryStore, hooks *HookEngine, settings *SettingsStore) *Generator {
	return &Generator{
		aiSvc: aiSvc, log: log, profiler: profiler, interpreter: interpreter, cache: cache, memory: memory,
		retrospect: NewRetrospectAgent(aiSvc, memory, log),
		hooks:      hooks,
		settings:   settings,
	}
}

// generateState tracks mutable state across retry attempts within a single Generate call.
type generateState struct {
	PythonSource  string
	CompileError  string
	BacktestError string
}

// Generate runs the generation loop: LLM generate → compile → backtest → retry on failure.
// Streams progress chunks to the frontend via the stream callback.
func (g *Generator) Generate(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
	bars []sdk.Bar,
	stream func(*antv1.AgentGenerateStrategyChunk) error,
) error {
	btCfg := msg.BacktestConfig
	if btCfg == nil {
		btCfg = &antv1.AgentBacktestConfig{
			Symbol:    msg.Symbol,
			Timeframe: msg.Timeframe,
		}
	}

	result := &generateState{}
	var estCostUSD float64

	// Resolve managed settings for cost ceiling and max iterations (ADR-0025 §5.2).
	effectiveCostCeiling := costCeilingUSD // fallback to default constant
	effectiveMaxRetries := maxGenerateRetries
	if g.settings != nil {
		if rs, err := g.settings.ResolveSettings(ctx, userID); err == nil && rs.Loaded {
			if rs.Managed.MaxCostCeilingUSD > 0 {
				effectiveCostCeiling = rs.Managed.MaxCostCeilingUSD
			}
			if rs.Managed.MaxIterations > 0 {
				effectiveMaxRetries = rs.Managed.MaxIterations
			}
		}
	}

	// streamOrAbort sends a chunk and returns immediately if the client disconnected.
	streamOrAbort := func(chunk *antv1.AgentGenerateStrategyChunk) error {
		if err := stream(chunk); err != nil {
			g.log.Info("generator: client disconnected, aborting", zap.Error(err))
			return err
		}
		return nil
	}

	// ── Step 0: Generate strategy profile from NL (first attempt only) ──
	var preProfile *antv1.StrategyProfile
	if err := streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "planning"}); err != nil {
		return err
	}
	estCostUSD += estCostProfile
	pp, profErr := g.generateProfileFromNL(ctx, userID, msg)
	if profErr != nil {
		estCostUSD -= estCostProfile // failed call, don't count
		g.log.Warn("generator: pre-profile failed, proceeding without", zap.Error(profErr))
	} else {
		preProfile = pp
		if err := streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "planning", Profile: preProfile}); err != nil {
			return err
		}
	}

	// ── ADR-0025 §4.3: Load session memory for prompt injection ──
	var sessionMem *SessionMemory
	if g.memory != nil {
		mem, memErr := g.memory.LoadSessionMemory(ctx, userID, msg.Symbol, msg.Timeframe)
		if memErr != nil {
			g.log.Warn("generator: session memory load failed", zap.Error(memErr))
		} else {
			sessionMem = mem
		}
	}

	// ── ADR-0025 §3: Plan Mode — generate structured plan, wait for user confirmation ──
	planMode := msg.PlanMode
	if planMode == "" {
		planMode = "plan" // default to plan mode
	}

	if planMode == "plan" {
		plan, planErr := g.generatePlan(ctx, userID, msg, preProfile, sessionMem)
		if planErr != nil {
			g.log.Warn("generator: plan generation failed", zap.Error(planErr))
			_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{
				Phase: "done",
				Error: fmt.Sprintf("plan generation failed: %v", planErr),
			})
			return nil
		}
		_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{
			Phase: "done",
			Plan:  plan,
		})
		return nil
	}

	// planMode == "generate": use confirmed plan for code generation
	var confirmedPlan *antv1.StrategyPlan
	if msg.ConfirmedPlan != nil {
		confirmedPlan = msg.ConfirmedPlan
	}

	for attempt := 1; attempt <= effectiveMaxRetries; attempt++ {
		// Cost ceiling check (ADR-0024 §D7, ADR-0025 §5.2)
		if estCostUSD >= effectiveCostCeiling {
			g.log.Warn("generator: cost ceiling exceeded, stopping", zap.Float64("est_cost", estCostUSD), zap.Float64("ceiling", effectiveCostCeiling))
			_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{
				Phase:    "done",
				Error:    fmt.Sprintf("cost ceiling $%.2f exceeded (est $%.4f)", effectiveCostCeiling, estCostUSD),
				Attempts: int32(attempt - 1),
			})
			return nil
		}

		// ── LLM generation (streaming) ──
		var sysPrompt, userPrompt string
		if attempt == 1 {
			if confirmedPlan != nil {
				sysPrompt, userPrompt = buildGenerateFromPlanPrompt(msg, confirmedPlan, preProfile, sessionMem)
			} else {
				sysPrompt, userPrompt = buildGeneratePrompt(msg, preProfile, sessionMem)
			}
		} else {
			sysPrompt, userPrompt = buildGenerateRetryPrompt(msg, result.PythonSource, result.CompileError, result.BacktestError, preProfile, sessionMem)
		}

		if err := streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "generating", Attempts: int32(attempt)}); err != nil {
			return err
		}

		estCostUSD += estCostCodeGen

		llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		var codeBuf strings.Builder
		err := g.aiSvc.ChatCompletionStream(llmCtx, userID,
			[]systemai.ChatMessage{
				{Role: "system", Content: sysPrompt},
				{Role: "user", Content: userPrompt},
			},
			func(chunk systemai.ChatStreamChunk) error {
				codeBuf.WriteString(chunk.Content)
				return stream(&antv1.AgentGenerateStrategyChunk{Phase: "generating", Delta: chunk.Content})
			})
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				g.log.Info("generator: context canceled, aborting", zap.Error(ctx.Err()))
				return ctx.Err()
			}
			g.log.Warn("generator: LLM stream failed", zap.Int("attempt", attempt), zap.Error(err))
			result.CompileError = fmt.Sprintf("LLM call failed: %v", err)
			continue
		}

		pythonSource := stripMarkdownFences(codeBuf.String())
		result.PythonSource = pythonSource

		// ── Compile ──
		if err := streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "compiling", PythonSource: pythonSource}); err != nil {
			return err
		}

		runner, coverage, compileErr := mql2go.CompilePythonWithCoverage(pythonSource)
		if compileErr != nil {
			result.CompileError = compileErr.Error()
			result.BacktestError = ""
			g.log.Info("generator: compile failed", zap.Int("attempt", attempt), zap.String("error", truncate(compileErr.Error(), 200)))
			_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "compiling", CompileError: compileErr.Error()})
			continue
		}

		result.CompileError = ""

		// ── Backtest ──
		if err := streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "backtesting", CoverageScore: coverage.Score}); err != nil {
			return err
		}

		btResult, btErr := runVMBacktest(ctx, runner, btCfg, bars, msg.Params)
		if btErr != nil {
			result.BacktestError = btErr.Error()
			g.log.Info("generator: backtest failed", zap.Int("attempt", attempt), zap.String("error", truncate(btErr.Error(), 200)))
			_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "backtesting", BacktestError: btErr.Error()})
			continue
		}

		result.BacktestError = ""

		// ── Success: generate profile + analysis (with retry/degraded fallback) ──
		if err := streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "analyzing"}); err != nil {
			return err
		}

		btProto := buildBacktestResultProto(btResult)

		// Post-generation profile: retry max 2, degrade to nil on failure (ADR-0024 §5.4).
		var profile *antv1.StrategyProfile
		for pr := 1; pr <= maxProfileRetries; pr++ {
			estCostUSD += estCostProfile
			profile, profErr = g.profiler.GenerateProfile(ctx, userID.String(), pythonSource, coverage)
			if profErr == nil {
				break
			}
			estCostUSD -= estCostProfile // failed, don't count
			g.log.Warn("generator: post-profile attempt failed", zap.Int("attempt", pr), zap.Error(profErr))
		}
		if profile == nil {
			g.log.Warn("generator: post-profile degraded to nil after retries")
			g.fireDegradationAlert(ctx, userID, "profile generation failed after retries", stream)
		}

		// Analysis: retry max 2, degrade to template on failure (ADR-0024 §5.4).
		var analysis *antv1.BacktestAnalysis
		if profile != nil {
			for ar := 1; ar <= maxAnalysisRetries; ar++ {
				estCostUSD += estCostAnalysis
				analysis, err = g.interpreter.AnalyzeBacktest(ctx, userID.String(), btProto, profile)
				if err == nil {
					break
				}
				estCostUSD -= estCostAnalysis // failed, don't count
				g.log.Warn("generator: analysis attempt failed", zap.Int("attempt", ar), zap.Error(err))
			}
			if analysis == nil {
				analysis = degradedAnalysis(btProto)
				g.log.Warn("generator: analysis degraded to template")
				g.fireDegradationAlert(ctx, userID, "analysis generation failed after retries", stream)
			}
		}

		// ── Send final chunk ──
		chunk := &antv1.AgentGenerateStrategyChunk{
			Phase:         "done",
			PythonSource:  pythonSource,
			Result:        btProto,
			Profile:       profile,
			Analysis:      analysis,
			CoverageScore: coverage.Score,
			Attempts:      int32(attempt),
		}
		for _, bs := range coverage.BlindSpots {
			chunk.BlindSpots = append(chunk.BlindSpots, &antv1.AgentBlindSpot{
				Builtin:  bs.Builtin,
				Severity: bs.Severity,
				Count:    int32(bs.Count),
			})
		}
		if err := streamOrAbort(chunk); err != nil {
			return err
		}

		// ── ADR-0025 §6.2: PostStrategyGeneration hook — fires retrospect Agent via HookEngine ──
		if g.hooks != nil && g.hooks.HasHandlers(HookPostStrategyGen) {
			go g.hooks.Fire(context.Background(), &HookContext{
				Event:          HookPostStrategyGen,
				UserID:         userID,
				Profile:        profile,
				BacktestResult: btProto,
				Analysis:       analysis,
			})
		} else if g.retrospect != nil {
			go g.retrospect.Run(context.Background(), userID, retrospectInput{
				Message:        msg.Message,
				Symbol:         msg.Symbol,
				Timeframe:      msg.Timeframe,
				Profile:        profile,
				Plan:           confirmedPlan,
				BacktestResult: btProto,
				Analysis:       analysis,
				CoverageScore:  coverage.Score,
			})
		}

		return nil
	}

	// All retries exhausted
	finalChunk := &antv1.AgentGenerateStrategyChunk{
		Phase:          "done",
		PythonSource:   result.PythonSource,
		CompileError:   result.CompileError,
		BacktestError:  result.BacktestError,
		Attempts:       int32(effectiveMaxRetries),
		Error:          "generation failed after " + fmt.Sprintf("%d", effectiveMaxRetries) + " attempts",
	}
	_ = streamOrAbort(finalChunk)

	// ── ADR-0025 §6.2: Retrospect on failure too (via HookEngine if available) ──
	if g.hooks != nil && g.hooks.HasHandlers(HookPostStrategyGen) {
		go g.hooks.Fire(context.Background(), &HookContext{
			Event:          HookPostStrategyGen,
			UserID:         userID,
			BacktestResult: &antv1.AgentBacktestResult{Success: false, Error: result.BacktestError},
		})
	} else if g.retrospect != nil {
		go g.retrospect.Run(context.Background(), userID, retrospectInput{
			Message:        msg.Message,
			Symbol:         msg.Symbol,
			Timeframe:      msg.Timeframe,
			BacktestResult: &antv1.AgentBacktestResult{Success: false, Error: result.BacktestError},
			CoverageScore:  0,
		})
	}

	return nil
}

// generateProfileFromNL calls LLM to produce a strategy profile from the natural language
// description, without needing source code. This is the "策略画像" intermediate step
// (ADR-0024 Phase 3: NL → 策略画像 → Python 策略).
// Uses LLCache to avoid redundant LLM calls (ADR-0024 §5.3).
func (g *Generator) generateProfileFromNL(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
) (*antv1.StrategyProfile, error) {
	userPrompt := buildProfileFromNLPrompt(msg)
	cacheKey := msg.Message + "\x00" + msg.Symbol + "\x00" + msg.Timeframe
	for k, v := range msg.Params {
		cacheKey += "\x00" + k + "=" + v
	}

	if g.cache != nil {
		if cached, ok := g.cache.Get(cacheKey, userPrompt); ok {
			return parseProfileLines(cached), nil
		}
	}

	llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := g.aiSvc.ChatCompletion(llmCtx, userID, []systemai.ChatMessage{
		{Role: "system", Content: profileFromNLSystemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, fmt.Errorf("profile-from-NL LLM call: %w", err)
	}

	if g.cache != nil {
		g.cache.Set(cacheKey, userPrompt, resp)
	}

	return parseProfileLines(resp), nil
}
