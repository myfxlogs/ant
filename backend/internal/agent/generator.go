package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
	"anttrader/internal/service/systemai"
	"anttrader/strategy/sdk"
)


// Generator orchestrates the strategy generation Agent loop:
// intent → profile → plan → AgentLoop (generate/compile/backtest/fix).
type Generator struct {
	mkt       repository.MarketDataStore
	btRepo    *repository.BacktestRunRepository
	dbExec    func(ctx context.Context, sql string, args ...any) error
	dbQuery   func(ctx context.Context, sql string, args ...any) (string, error)
	aiSvc     *systemai.Service
	log       *zap.Logger
	cache     *LLCache
	memory    *MemoryStore
}

// NewGenerator creates the strategy generation orchestrator.
func NewGenerator(aiSvc *systemai.Service, log *zap.Logger, cache *LLCache, memory *MemoryStore, mkt repository.MarketDataStore, btRepo *repository.BacktestRunRepository, dbExec func(ctx context.Context, sql string, args ...any) error, dbQuery func(ctx context.Context, sql string, args ...any) (string, error)) *Generator {
	return &Generator{aiSvc: aiSvc, log: log, cache: cache, memory: memory, mkt: mkt, btRepo: btRepo, dbExec: dbExec, dbQuery: dbQuery}
}

// generateState tracks mutable state during AgentLoop execution — tools update
// compile/backtest results so runAgentLoop can inspect final state after completion.
type generateState struct {
	PythonSource  string
	CompileError  string
	BacktestError string
}

// Generate runs the LLM-driven agent loop: intent → profile → plan → AgentLoop (generate/compile/backtest/fix).
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
	pp, profErr := g.generateProfileFromNL(ctx, userID, msg)
	if profErr != nil {
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
	if err := streamOrAbort(&antv1.AgentGenerateStrategyChunk{Phase: "generating"}); err != nil {
		return err
	}

	return g.runAgentLoop(ctx, userID, msg, preProfile, sessionMem, confirmedPlan, streamOrAbort)
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

// generatePlan calls LLM to produce a structured StrategyPlan from NL + profile (ADR-0025 §3).
func (g *Generator) generatePlan(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
	profile *antv1.StrategyProfile,
	sessionMem *SessionMemory,
) (*antv1.StrategyPlan, error) {
	userPrompt := buildPlanPrompt(msg, profile, msg.PlanFeedback, sessionMem)

	llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	resp, err := g.aiSvc.ChatCompletion(llmCtx, userID, []systemai.ChatMessage{
		{Role: "system", Content: planSystemPrompt},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		return nil, fmt.Errorf("plan LLM call: %w", err)
	}

	return parsePlanResponse(resp), nil
}
