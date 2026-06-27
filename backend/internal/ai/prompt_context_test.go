package ai

import (
	"strings"
	"testing"
)

func TestClassifyIntent_RepairPriority(t *testing.T) {
	code := "package main\n\nfunc (s *MyStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {\n    return nil, nil\n}\n"

	tests := []struct {
		msg  string
		want InteractionMode
	}{
		{"报错了缺少 required 参数 stop_loss", ModeRepair},
		{"Traceback: AttributeError: 'NoneType' object", ModeRepair},
		{"验证失败: syntax error at line 5", ModeRepair},
		{"undefined variable 'price' — 错误", ModeRepair},
		{"这个止损逻辑对吗？", ModeDiscuss},
		{"加一个止盈条件", ModeRevise},
	}

	for _, tt := range tests {
		input := BuildContextInput{Code: code, Message: tt.msg}
		pc := BuildContext(input)
		if pc.Mode != tt.want {
			t.Errorf("msg=%q: got mode %d, want %d", tt.msg, pc.Mode, tt.want)
		}
	}
}

func TestRepairPrompt_CodeOnlyConstraints(t *testing.T) {
	input := BuildContextInput{
		Code:             "package main\n\nfunc (s *MyStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {\n    return nil, nil\n}",
		Message:          "报错: missing required param",
		ValidationErrors: []string{"missing stop_loss", "syntax error"},
	}
	pc := BuildContext(input)

	if pc.Mode != ModeRepair {
		t.Fatalf("expected ModeRepair, got %d", pc.Mode)
	}

	mustContain := []string{
		"CODE REPAIR EXPERT",
		"CRITICAL RULES",
		"NO markdown, NO explanations",
		"Start directly with import/type/func",
		"FIXME",
		"missing stop_loss",
	}
	for _, kw := range mustContain {
		if !strings.Contains(pc.SystemPrompt, kw) {
			t.Errorf("repair system prompt missing keyword: %q", kw)
		}
	}
}

func TestClassifyIntent_DiscussDoesNotTriggerRepair(t *testing.T) {
	code := "package main\n\nfunc (s *MyStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {\n    return nil, nil\n}\n"

	discussMsgs := []string{
		"这个止损逻辑对吗？",
		"为什么这里用sma而不是ema？",
		"帮我分析一下这个策略怎么样",
		"explain the entry logic",
	}

	for _, msg := range discussMsgs {
		mode := classifyIntent(code, msg)
		if mode != ModeDiscuss {
			t.Errorf("msg=%q: expected ModeDiscuss, got %d", msg, mode)
		}
	}
}

func TestClassifyIntent_ReviseIsDefault(t *testing.T) {
	code := "package main\n\nfunc (s *MyStrategy) OnBar(ctx sdk.Context, tf string) (*sdk.Signal, error) {\n    return nil, nil\n}\n"

	reviseMsgs := []string{
		"把sma改成ema",
		"加一个止盈",
		"降低仓位",
		"换品种到GBPUSD",
	}

	for _, msg := range reviseMsgs {
		mode := classifyIntent(code, msg)
		if mode != ModeRevise {
			t.Errorf("msg=%q: expected ModeRevise (default), got %d", msg, mode)
		}
	}
}
