package systemai

import "testing"

// FIX-2026-09-08-BYOK-MODEL-PICKER 对抗证明。
//
// 用户在 /ai/settings 粘贴完整 endpoint URL（如 NOVA 文档给出的
// https://token.sensenova.cn/v1/chat/completions）时，chatEndpoint 必须产出
// 单一 /chat/completions 后缀；否则双路径 404 → BYOK 聊天全挂
// （生产日志: "discover models failed ... base_url must be openai-compatible"）。
//
// mutation: 还原 chatEndpoint 去掉 normalizeAPIBase → T-C1 RED。

func TestChatEndpointToleratesFullEndpointURL(t *testing.T) {
	got := chatEndpoint("openai_compatible_mts0f275", "https://token.sensenova.cn/v1/chat/completions")
	want := "https://token.sensenova.cn/v1/chat/completions"
	if got != want {
		t.Fatalf("chatEndpoint doubled path: got %q want %q", got, want)
	}
}

func TestChatEndpointRegularBaseURLUnchanged(t *testing.T) {
	cases := []struct{ provider, base, want string }{
		{"deepseek", "https://api.deepseek.com/v1", "https://api.deepseek.com/v1/chat/completions"},
		{"openai", "https://api.openai.com/v1/", "https://api.openai.com/v1/chat/completions"},
		{"zhipu", "https://open.bigmodel.cn/api/paas/v4", "https://open.bigmodel.cn/api/paas/v4/chat/completions"},
		{"openai_compatible_x", "https://llm.example.io/v1/models", "https://llm.example.io/v1/chat/completions"},
	}
	for _, c := range cases {
		if got := chatEndpoint(c.provider, c.base); got != c.want {
			t.Fatalf("chatEndpoint(%q,%q) = %q want %q", c.provider, c.base, got, c.want)
		}
	}
}

// resolveModel 契约：用户 primary 精确选择优先于 default_model/models[0]。
// mutation: 删除 primaryPID 匹配分支 → T-R1 RED。
func TestResolveModelPrimaryPickWins(t *testing.T) {
	if got := resolveModel("default-m", []string{"first-m"}, "zhipu", "zhipu", "picked-m"); got != "picked-m" {
		t.Fatalf("resolveModel primary pick: got %q want picked-m", got)
	}
	if got := resolveModel("default-m", []string{"first-m"}, "zhipu", "other", "picked-m"); got != "default-m" {
		t.Fatalf("resolveModel non-primary provider: got %q want default-m", got)
	}
	if got := resolveModel("", []string{"first-m"}, "zhipu", "other", ""); got != "first-m" {
		t.Fatalf("resolveModel fallback models[0]: got %q want first-m", got)
	}
}
