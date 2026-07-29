package agent

import (
	"context"
	"fmt"
	"strings"

	connectai "alphaforge/internal/connect/ai"
	systemai "alphaforge/internal/service/systemai"
	"alphaforge/tools/mql2go"
)

// readCurrentCodeTool returns the workspace strategy code with line numbers.
type readCurrentCodeTool struct {
	result *generateState
}

func (t *readCurrentCodeTool) Name() string { return "read_current_code" }

func (t *readCurrentCodeTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name:        "read_current_code",
			Description: "读取当前workspace中的策略代码（带行号）。修改代码前必须先调用此工具。",
			Parameters:  map[string]any{schemaKeyType: schemaTypeObject, schemaKeyProperties: map[string]any{}},
		},
	}
}

func (t *readCurrentCodeTool) Run(_ context.Context, in connectai.ToolInput) connectai.ToolOutput {
	code := in.Code
	if code == "" {
		code = t.result.PythonSource
	}
	if code == "" {
		return connectai.ToolOutput{Success: false, Error: "workspace has no code yet"}
	}
	lines := strings.Split(code, "\n")
	var sb strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&sb, "%4d | %s\n", i+1, line)
	}
	return connectai.ToolOutput{Success: true, Output: sb.String()}
}

// editCodeTool performs exact old_string→new_string replacement on workspace code,
// then re-compiles to verify correctness. Follows Claude Code Edit semantics.
type editCodeTool struct {
	result *generateState
}

func (t *editCodeTool) Name() string { return "edit_code" }

func (t *editCodeTool) Schema() systemai.ToolDefinition {
	return systemai.ToolDefinition{
		Type: "function",
		Function: systemai.ToolDefFunction{
			Name: "edit_code",
			Description: "精确编辑策略代码（old_string→new_string）。小改动用此工具；大改动请用write_strategy。old_string必须唯一匹配。",
			Parameters: map[string]any{
				schemaKeyType:     schemaTypeObject,
				"required": []string{"old_string", "new_string"},
				schemaKeyProperties: map[string]any{
					"old_string": map[string]any{schemaKeyType: schemaTypeString, "description": "要替换的原始代码片段（必须唯一匹配）"},
					"new_string": map[string]any{schemaKeyType: schemaTypeString, "description": "替换后的新代码片段"},
				},
			},
		},
	}
}

func (t *editCodeTool) Run(_ context.Context, in connectai.ToolInput) connectai.ToolOutput {
	oldStr, _ := in.RawArgs["old_string"].(string)
	newStr, _ := in.RawArgs["new_string"].(string)
	if oldStr == "" {
		return connectai.ToolOutput{Success: false, Error: "old_string is required"}
	}

	code := in.Code
	if code == "" {
		code = t.result.PythonSource
	}

	count := strings.Count(code, oldStr)
	if count == 0 {
		return connectai.ToolOutput{Success: false, Error: "old_string not found in code. Use read_current_code to verify the exact text."}
	}
	if count > 1 {
		return connectai.ToolOutput{Success: false, Error: fmt.Sprintf("old_string appears %d times — not unique. Add more surrounding context to make it unique.", count)}
	}

	newCode := strings.Replace(code, oldStr, newStr, 1)
	t.result.PythonSource = newCode

	_, cov, err := mql2go.CompilePythonWithCoverage(newCode)
	if err != nil {
		return connectai.ToolOutput{Success: false, Error: fmt.Sprintf("edit applied but compile failed: %v", err)}
	}
	return connectai.ToolOutput{
		Success: true,
		Output: map[string]string{
			"status":   "edited and compiled",
			"coverage": fmt.Sprintf("%.1f%%", cov.Score*100),
		},
	}
}
