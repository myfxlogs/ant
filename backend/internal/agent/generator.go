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


// firePostGenHook dispatches the post-generation hook or retrospect agent (ADR-0025 §6, §8).
func (g *Generator) firePostGenHook(ctx context.Context, userID uuid.UUID, source string, profile *antv1.StrategyProfile, btProto *antv1.AgentBacktestResult, analysis *antv1.BacktestAnalysis) {
	if g.hooks != nil && g.hooks.HasHandlers(HookPostStrategyGen) {
		go g.hooks.Fire(ctx, &HookContext{
			Event:         HookPostStrategyGen,
			UserID:        userID,
			Source:        source,
			Profile:       profile,
			BacktestResult: btProto,
			Analysis:      analysis,
		})
	} else if g.retrospect != nil {
		go g.retrospect.Run(ctx, userID, retrospectInput{
			Profile:       profile,
			BacktestResult: btProto,
			Analysis:      analysis,
		})
	}
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

	// ── Intent classification: chat vs generate ──
	// Classify FIRST, before any profile/plan work, so chat messages don't
	// trigger unnecessary LLM calls that produce garbage output.
	planMode := msg.PlanMode
	if planMode == "" {
		planMode = "plan" // default to plan mode
	}
	if planMode == "plan" && msg.PlanFeedback == "" {
		if err := streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "planning"}); err != nil {
			return err
		}
		intent, intentErr := g.classifyIntent(ctx, userID, msg.Message)
		if intentErr != nil {
			g.log.Warn("generator: intent classification failed, defaulting to generate", zap.Error(intentErr))
		} else if intent == "chat" {
			return g.streamChatResponse(ctx, userID, msg, streamOrAbort)
		}
	}

	// ── Step 0: Generate strategy profile from NL (generate intent only) ──
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

	// ── Agent Loop: LLM drives generate → compile → backtest → fix → repeat ──
	_ = estCostUSD // retained for future cost tracking
	_ = effectiveMaxRetries
	_ = effectiveCostCeiling

	if err := streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "generating"}); err != nil {
		return err
	}

	return g.runAgentLoop(ctx, userID, msg, bars, btCfg, preProfile, sessionMem, confirmedPlan, streamOrAbort)
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

// classifyIntent determines whether the user message is a general discussion/question
// ("chat") or a strategy generation request ("generate"). Uses a fast LLM call.
func (g *Generator) classifyIntent(ctx context.Context, userID uuid.UUID, message string) (string, error) {
	llmCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := g.aiSvc.ChatCompletion(llmCtx, userID, []systemai.ChatMessage{
		{Role: "system", Content: intentClassificationPrompt},
		{Role: "user", Content: message},
	})
	if err != nil {
		return "", fmt.Errorf("intent classification LLM call: %w", err)
	}

	intent := strings.TrimSpace(strings.ToLower(resp))
	// Accept "chat", "generate", "discussion", "question", etc.
	switch {
	case strings.Contains(intent, "generate"), strings.Contains(intent, "strategy"), strings.Contains(intent, "code"):
		return "generate", nil
	default:
		return "chat", nil
	}
}

// streamChatResponse streams a natural LLM conversation response without going
// through the plan/generate/compile/backtest pipeline. This makes the AI chat
// feel natural — users can discuss strategies, ask questions, and get explanations.
func (g *Generator) streamChatResponse(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
	streamOrAbort func(*antv1.AgentGenerateStrategyChunk) error,
) error {
	// Send "chatting" phase so the frontend shows a thinking indicator
	if err := streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "chatting"}); err != nil {
		return err
	}

	// Build a conversational system prompt with trading context
	sysPrompt := chatSystemPrompt
	if msg.Symbol != "" || msg.Timeframe != "" {
		sysPrompt += fmt.Sprintf("\n\n## Current Context\nSymbol: %s\nTimeframe: %s", msg.Symbol, msg.Timeframe)
	}

	llmCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	err := g.aiSvc.ChatCompletionStream(llmCtx, userID,
		[]systemai.ChatMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: msg.Message},
		},
		func(chunk systemai.ChatStreamChunk) error {
			return streamOrAbort(&antv1.AgentGenerateStrategyChunk{
				Phase: "chatting",
				Delta: chunk.Content,
			})
		})
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		g.log.Warn("generator: chat stream failed", zap.Error(err))
		_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{
			Phase: "done",
			Error: fmt.Sprintf("chat failed: %v", err),
		})
		return nil
	}

	_ = streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "done"})
	return nil
}

// intentClassificationPrompt is a fast LLM prompt to classify user intent.
const intentClassificationPrompt = `You are an intent classifier for a trading strategy AI assistant. Classify the user's message into one of two categories:

- "generate": The user wants to CREATE, GENERATE, or BUILD a trading strategy/EA/robot. Keywords: 生成, 创建, 开发, 写一个, make, create, generate, build, write a strategy, 编写, 开发一套, 生成一套, 自动交易, EA.
- "chat": The user is asking a question, discussing, seeking advice, or wants an explanation. Keywords: 什么是, 怎么理解, 解释一下, 帮我分析, what is, how does, explain, 分析一下, 讨论, 建议, 看看, 评价.

Respond with ONLY one word: "generate" or "chat". No other text.`

// chatSystemPrompt is the system prompt for natural conversation mode.
const chatSystemPrompt = `You are AntTrader AI, a knowledgeable trading strategy assistant. You help users discuss trading strategies, explain technical concepts, analyze market conditions, and provide guidance on strategy development.

You are conversational, friendly, and professional. You can:
- Explain trading concepts (indicators, patterns, risk management)
- Discuss strategy ideas and their pros/cons
- Analyze market conditions and trends
- Suggest improvements to existing strategies
- Answer questions about the AntTrader platform

When the user wants to actually generate and backtest a strategy, suggest they describe the strategy in detail and you'll help create it.

Keep responses concise and focused. Use markdown formatting when helpful (code blocks for examples, bullet points for lists).`
