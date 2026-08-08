package agent

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
	"alphaforge/internal/service/systemai"
)

// Generator orchestrates the strategy generation Agent loop:
// intent → profile → plan → AgentLoop (generate/compile/backtest/fix).
type Generator struct {
	mkt              repository.MarketDataStore
	btRepo           *repository.BacktestRunRepository
	dbExec           func(ctx context.Context, sql string, args ...any) error
	dbQuery          func(ctx context.Context, sql string, args ...any) (string, error)
	aiSvc            *systemai.Service
	log              *zap.Logger
	cache            *LLCache
	memory           *MemoryStore
	conversationRepo *repository.AIConversationRepository
	creditSvc        CreditBilling
}

// CreditBilling is the interface for credit pre-hold and settlement.
// Implemented by *service.CreditService.
type CreditBilling interface {
	PreHold(ctx context.Context, userID uuid.UUID, sessionID, providerID, modelName string) error
	Settle(ctx context.Context, userID uuid.UUID, sessionID, providerID, modelName string, inputTokens, outputTokens int) error
	ReleaseHold(ctx context.Context, userID uuid.UUID, sessionID string) error
}

// SetConversationRepo injects the conversation store for multi-turn history.
func (g *Generator) SetConversationRepo(r *repository.AIConversationRepository) {
	g.conversationRepo = r
}

// SetCreditSvc injects the credit billing service for pre-hold/settlement.
func (g *Generator) SetCreditSvc(svc CreditBilling) {
	g.creditSvc = svc
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
	LastBacktest  *backtestSummary // captured on write_strategy success for persistent memory
	PlanSteps     []planStep       // plan-driven state machine (FEAT agent-rebuild Part 1)
}

// planStep represents a single step in the execution plan.
type planStep struct {
	Step   string `json:"step"`
	Status string `json:"status"` // pending | doing | done
}

// HasActivePlan returns true if a plan exists with incomplete steps.
func (g *generateState) HasActivePlan() bool {
	for _, s := range g.PlanSteps {
		if s.Status != "done" {
			return true
		}
	}
	return false
}

// CurrentStep returns the description of the current doing/pending step.
func (g *generateState) CurrentStep() string {
	for _, s := range g.PlanSteps {
		if s.Status == "doing" {
			return s.Step
		}
	}
	for _, s := range g.PlanSteps {
		if s.Status == "pending" {
			return s.Step
		}
	}
	return ""
}

// CompletedSteps returns the number of done steps.
func (g *generateState) CompletedSteps() int {
	n := 0
	for _, s := range g.PlanSteps {
		if s.Status == "done" {
			n++
		}
	}
	return n
}

// TotalSteps returns the total number of steps in the plan.
func (g *generateState) TotalSteps() int {
	return len(g.PlanSteps)
}

// Generate runs the unified AgentLoop — single path, full tools, no pre-processing.
// The LLM decides autonomously: discuss, query data, generate code, compile.
func (g *Generator) Generate(
	ctx context.Context,
	userID uuid.UUID,
	msg *antv1.AgentGenerateStrategyRequest,
	stream func(*antv1.AgentGenerateStrategyChunk) error,
) error {
	streamOrAbort := func(chunk *antv1.AgentGenerateStrategyChunk) error {
		if err := stream(chunk); err != nil {
			g.log.Info("generator: client disconnected, aborting", zap.Error(err))
			return err
		}
		return nil
	}
	return g.runAgentLoop(ctx, userID, msg, streamOrAbort)
}
