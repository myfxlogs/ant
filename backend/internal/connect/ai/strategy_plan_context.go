package ai

import (
	mql2go "alphaforge/tools/mql2go"
)

// strategy_plan_context.go — workspace code context for strategy-chat prompts.
//
// 设计（业主报告 2026-09-08）：用户在工作台选中编译失败的策略后直接发起 AI chat，
// 上下文此前只带整份策略代码、不带编译错误 —— Agent 不知道代码为什么坏。
// 服务端现场编译（零信任），把真实编译错误注入上下文并指示 Agent 优先修复。

// pythonCompileErrors returns the compiler error text for workspace Python
// code, "" when it compiles. Server-side truth: the agent sees the real
// compile failure instead of re-deriving it from raw code.
func pythonCompileErrors(code string) string {
	if _, err := mql2go.CompilePython(code); err != nil {
		return err.Error()
	}
	return ""
}

// workspacePythonContext builds the "当前策略代码" prompt section and, when the
// code fails to compile, appends the compiler errors with a fix-first
// directive so the agent prioritizes them over unrelated refactoring.
func workspacePythonContext(code string) string {
	if code == "" {
		return ""
	}
	out := "\n\n## 当前策略代码\n```python\n" + code + "\n```"
	if errs := pythonCompileErrors(code); errs != "" {
		out += "\n\n## ⚠ 当前工作区代码编译失败\n```\n" + errs + "\n```\n" +
			"用户工作区中的策略当前无法编译。若用户请求涉及修改策略，必须优先修复以上编译错误；修复完成前不要做无关重构，也不要建议改写为其他语言。"
	}
	return out
}
