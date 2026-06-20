package ai

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/ai"
	"anttrader/internal/repository"
)

// ── Tool interface ──

// ToolInput is passed to each tool by the execution engine.
type ToolInput struct {
	Code      string
	Symbol    string
	Timeframe string
	UserID    uuid.UUID
}

// ToolOutput is the structured result returned by each tool.
type ToolOutput struct {
	Success bool
	Output  any    // tool-specific struct (will be JSON-marshalled)
	Error   string
}

// Tool is a single step in the execution pipeline.
type Tool interface {
	Name() string
	Run(ctx context.Context, in ToolInput) ToolOutput
}

// ── Tool Registry ──

// ToolRegistry holds the ordered list of tools to execute after code generation.
type ToolRegistry struct {
	tools []Tool
}

// NewToolRegistry creates a registry with the standard tool set.
func NewToolRegistry(backtestRepo *repository.BacktestRunRepository) *ToolRegistry {
	return &ToolRegistry{
		tools: []Tool{
			&complianceTool{},
			&backtestTool{repo: backtestRepo},
		},
	}
}

// Execute runs all registered tools in order, streaming results via the callback.
func (r *ToolRegistry) Execute(ctx context.Context, in ToolInput, send func(*antv1.ExecutePlanChunk) error) {
	for _, t := range r.tools {
		callID := "call_" + t.Name()
		_ = send(&antv1.ExecutePlanChunk{
			Phase: "tool_call",
			ToolCall: &antv1.ToolCall{
				CallId: callID, Name: t.Name(), ParamsJson: `{}`,
			},
		})

		out := t.Run(ctx, in)
		outJSON, _ := json.Marshal(out.Output)
		_ = send(&antv1.ExecutePlanChunk{
			Phase: "tool_result",
			ToolResult: &antv1.ToolResult{
				CallId: callID, Name: t.Name(),
				Success: out.Success, OutputJson: string(outJSON), Error: out.Error,
			},
		})
	}
}

// ── compliance_check tool ──

type complianceTool struct{}

func (t *complianceTool) Name() string { return "compliance_check" }

func (t *complianceTool) Run(_ context.Context, in ToolInput) ToolOutput {
	blocks, warns := ai.NewCodeComplianceScanner().Scan(in.Code)
	allIssues := append(blocks, warns...)
	var issueProtos []*antv1.ComplianceIssue
	for _, iss := range allIssues {
		issueProtos = append(issueProtos, &antv1.ComplianceIssue{
			Rule: iss.RuleName, Message: iss.Message, Severity: iss.Severity, Line: int32(iss.Line),
		})
	}
	return ToolOutput{
		Success: len(blocks) == 0,
		Output:  &antv1.ComplianceResult{Passed: len(blocks) == 0, Issues: issueProtos},
	}
}

// ── backtest tool ──

type backtestTool struct{ repo *repository.BacktestRunRepository }

func (t *backtestTool) Name() string { return "backtest" }

func (t *backtestTool) Run(ctx context.Context, in ToolInput) ToolOutput {
	runID, err := CreateBacktestRun(ctx, t.repo, in.UserID, in.Code, in.Symbol, in.Timeframe)
	if err != nil {
		return ToolOutput{Success: false, Error: err.Error()}
	}
	return ToolOutput{
		Success: true,
		Output:  map[string]string{"run_id": runID},
	}
}
