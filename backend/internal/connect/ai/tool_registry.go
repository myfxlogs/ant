package ai

import (
	"context"

	"github.com/google/uuid"

	systemai "alphaforge/internal/service/systemai"
)

// ── Tool interface ──

// ToolInput is passed to each tool by the execution engine.
type ToolInput struct {
	Code      string
	Symbol    string
	Timeframe string
	Key       string // memory tools: key/name parameter
	Value     string // memory tools: value parameter
	UserID    uuid.UUID
	RawArgs   map[string]any // full LLM arguments — tools read what they need
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
	// Schema returns the JSON Schema definition for this tool (OpenAI function calling format).
	Schema() systemai.ToolDefinition
}

// ── Tool Registry ──

// ToolRegistry holds the ordered list of tools the AI agent can request.
type ToolRegistry struct {
	preTools []Tool
}

// NewEmptyToolRegistry creates a registry with no pre-loaded tools.
// Callers add tools via AddPreTool.
func NewEmptyToolRegistry() *ToolRegistry {
	return &ToolRegistry{}
}

// AddPreTool appends a tool to the pre-execution tool set.
func (r *ToolRegistry) AddPreTool(t Tool) {
	r.preTools = append(r.preTools, t)
}

// WireMemoryDB wires the PG pool for memory tools.
func (r *ToolRegistry) WireMemoryDB(execFn func(ctx context.Context, sql string, args ...any) error, queryFn func(ctx context.Context, sql string, args ...any) (string, error)) {
	rem := &rememberTool{execFn: execFn}
	rec := &recallTool{queryFn: queryFn}
	ls := &listStrategiesTool{queryFn: queryFn}
	sv := &saveStrategyTool{execFn: execFn}
	ld := &loadStrategyTool{queryFn: queryFn}
	r.preTools = append(r.preTools, rem, rec, ls, sv, ld)
}

// BuildToolSchemas returns JSON Schema definitions for all pre-execution tools.
// These are injected into LLM requests so the model can use native tool_use.
func (r *ToolRegistry) BuildToolSchemas() []systemai.ToolDefinition {
	schemas := make([]systemai.ToolDefinition, len(r.preTools))
	for i, t := range r.preTools {
		schemas[i] = t.Schema()
	}
	return schemas
}

// FindPreTool looks up a tool by name. Returns nil if not found.
func (r *ToolRegistry) FindPreTool(name string) Tool {
	for _, t := range r.preTools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}
