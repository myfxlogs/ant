package systemai

import (
	"strings"
	"testing"
)

// FIX-2026-09-08-CURL-IMPORT 对抗证明。
//
// 业主要求：新手把厂商文档的 curl 示例原样粘贴即可导入 provider 配置，
// 系统识别 URL / API Key / model（解析放后端，前端零信任）。
//
// mutation: 还原 curl_import.go → 本文件编译失败（最强证明）；
// 行为用例覆盖占位符 key、x-api-key、转义双引号 body、无 URL 报错。

// 业主提供的 NOVA/sensenova 原样示例（多行 + 续行符 + 占位符 key）。
const sensenovaCurl = `curl https://token.sensenova.cn/v1/chat/completions \
-H "Authorization: Bearer {your_key}" \
-H "Content-Type: application/json" \
-d '{
    "model": "kimi-k3",
    "messages": [
      { "role": "system", "content": "你是资深全栈开发工程师" },
      { "role": "user", "content": "用Go语言实现高并发快速排序算法" }
    ],
    "stream": false
}'`

func TestParseProviderCurlSensenovaExample(t *testing.T) {
	res, err := ParseProviderCurlRaw(sensenovaCurl, false)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if res.BaseURL != "https://token.sensenova.cn/v1" {
		t.Fatalf("BaseURL = %q, want endpoint suffix stripped", res.BaseURL)
	}
	if res.APIKey != "" {
		t.Fatalf("placeholder key must not be imported verbatim, got %q", res.APIKey)
	}
	if res.DefaultModel != "kimi-k3" || len(res.Models) != 1 || res.Models[0] != "kimi-k3" {
		t.Fatalf("model extraction wrong: default=%q models=%v", res.DefaultModel, res.Models)
	}
	if !strings.Contains(res.NameHint, "sensenova") {
		t.Fatalf("NameHint = %q, want sensanova hint", res.NameHint)
	}
	joined := strings.Join(res.Warnings, "\n")
	if !strings.Contains(joined, "占位") {
		t.Fatalf("placeholder warning missing: %v", res.Warnings)
	}
}

func TestParseProviderCurlRealKeyAndVariants(t *testing.T) {
	res, err := ParseProviderCurlRaw(`curl https://api.deepseek.com/v1/chat/completions -H "Authorization: Bearer sk-real-abc123" -H "Content-Type: application/json" -d '{"model":"deepseek-v4","messages":[{"role":"user","content":"hi"}],"stream":false}'`, false)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if res.APIKey != "sk-real-abc123" {
		t.Fatalf("APIKey = %q", res.APIKey)
	}
	if res.BaseURL != "https://api.deepseek.com/v1" || res.DefaultModel != "deepseek-v4" {
		t.Fatalf("url=%q model=%q", res.BaseURL, res.DefaultModel)
	}
	if strings.Contains(strings.Join(res.Warnings, "\n"), "占位") {
		t.Fatalf("real key must not warn placeholder: %v", res.Warnings)
	}

	res2, err := ParseProviderCurlRaw(`curl https://h.example.com/v1/chat/completions -H "x-api-key: my-anthropic-key" --data-raw "{\"model\":\"m1\"}"`, false)
	if err != nil {
		t.Fatalf("parse x-api-key: %v", err)
	}
	if res2.APIKey != "my-anthropic-key" || res2.DefaultModel != "m1" {
		t.Fatalf("x-api-key parse: key=%q model=%q", res2.APIKey, res2.DefaultModel)
	}
}

func TestParseProviderCurlFailClosed(t *testing.T) {
	if _, err := ParseProviderCurlRaw("echo hello world", false); err == nil {
		t.Fatal("no-URL input must error")
	}
	res, err := ParseProviderCurlRaw("curl https://only-url.example.com/v1/chat/completions", false)
	if err != nil {
		t.Fatalf("url-only parse: %v", err)
	}
	if res.BaseURL == "" || res.DefaultModel != "" {
		t.Fatalf("url-only: base=%q model=%q", res.BaseURL, res.DefaultModel)
	}
	joined := strings.Join(res.Warnings, "\n")
	if !strings.Contains(joined, "model") {
		t.Fatalf("model-missing warning missing: %v", res.Warnings)
	}
}

// 已保存过 Key 的厂商：导入不再提示任何 Key 相关告警（占位符/缺失都静默），
// 且占位符场景只产生一条告警而非两条。
func TestParseProviderCurlKeyWarningsMutedBySavedKey(t *testing.T) {
	res, err := ParseProviderCurlRaw(sensenovaCurl, true)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	joined := strings.Join(res.Warnings, "\n")
	if strings.Contains(joined, "API Key") {
		t.Fatalf("saved-key provider must not show key warnings: %v", res.Warnings)
	}

	res2, err := ParseProviderCurlRaw(sensenovaCurl, false)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	keyWarns := 0
	for _, w := range res2.Warnings {
		if strings.Contains(w, "API Key") {
			keyWarns++
		}
	}
	if keyWarns != 1 {
		t.Fatalf("placeholder case must yield exactly 1 key warning, got %d: %v", keyWarns, res2.Warnings)
	}
}
