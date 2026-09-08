package ai

import (
	"strings"
	"testing"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// FIX-2026-09-08-COMPILE-CTX 对抗证明。
//
// 业主报告：工作台选中编译失败的策略后直接发起 AI chat，上下文只带整份代码、
// 不带编译错误，Agent 不知道代码为什么坏。
// workspacePythonContext / pythonCompileErrors 服务端现场编译，把真实错误
// 注入上下文并指示优先修复。
//
// mutation: 还原 strategy_plan_context.go / strategy_plan_handler.go 注入 →
// 本文件编译失败（最强证明）。

// 平台 Python 子集的合法最小策略（实测 compile OK 的平铺函数形态）。
const validPy = `def run_dataframe(df, params):
    return df`

func TestWorkspacePythonContextValid(t *testing.T) {
	out := workspacePythonContext(validPy)
	if !strings.Contains(out, "当前策略代码") {
		t.Fatal("valid code must include the code section")
	}
	if strings.Contains(out, "编译失败") {
		t.Fatalf("valid code must not be flagged as compile failure: %s", out)
	}
}

func TestWorkspacePythonContextBroken(t *testing.T) {
	out := workspacePythonContext("def broken(:\n    pass")
	if !strings.Contains(out, "编译失败") {
		t.Fatalf("broken code must carry the compile-failure section: %s", out)
	}
	if !strings.Contains(out, "编译错误") || !strings.Contains(out, "优先修复") {
		t.Fatalf("must instruct fix-first: %s", out)
	}
	if workspacePythonContext("") != "" {
		t.Fatal("empty code must yield empty context")
	}
}

func TestBuildExecuteUserPromptCompileError(t *testing.T) {
	m := &antv1.ExecutePlanRequest{
		Plan:            "1. 修复",
		FeedbackMessage: "帮我修好",
		PreviousCode:    "def broken(:\n    pass",
	}
	out := buildExecuteUserPrompt(m)
	if !strings.Contains(out, "编译失败") || !strings.Contains(out, "优先修复") {
		t.Fatalf("execute prompt must surface compile errors: %s", out)
	}
	if strings.Contains(out, "```go") {
		t.Fatal("strategy code fence must be python, not legacy go")
	}

	valid := &antv1.ExecutePlanRequest{
		Plan:            "1. ok",
		FeedbackMessage: "继续",
		PreviousCode:    validPy,
	}
	if out := buildExecuteUserPrompt(valid); strings.Contains(out, "编译失败") {
		t.Fatalf("valid code must not be flagged: %s", out)
	}

	if out := buildExecuteUserPrompt(&antv1.ExecutePlanRequest{Plan: "P"}); out != "P" {
		t.Fatalf("no-feedback path must return plan verbatim, got %q", out)
	}
}

// analyze_mql 工具：编译失败即分析结果（错误文本点名盲区），不得当作工具崩溃。
func TestAnalyzeMQLToolBlindSpot(t *testing.T) {
	tool := &analyzeMQLChatTool{}
	if tool.Name() != "analyze_mql" {
		t.Fatal("tool name")
	}
	out := tool.Run(nil, ToolInput{Code: `int start() {
   double price[];
   ArrayResize(price, 5);
   return 0;
}`})
	if !out.Success {
		t.Fatalf("analysis must succeed even when code does not compile: %v", out.Error)
	}
	om, _ := out.Output.(map[string]any)
	if om == nil {
		t.Fatalf("analysis output must be a map, got %T", out.Output)
	}
	if v, _ := om["compiles"].(bool); v {
		t.Fatalf("local-array MQL is a known blind spot, must report compiles=false: %v", om)
	}
	if s, _ := om["error"].(string); !strings.Contains(s, "local arrays not supported") {
		t.Fatalf("error must name the blind spot: %v", om)
	}
	if empty := (&analyzeMQLChatTool{}).Run(nil, ToolInput{}); empty.Success {
		t.Fatal("empty input must fail")
	}
}
