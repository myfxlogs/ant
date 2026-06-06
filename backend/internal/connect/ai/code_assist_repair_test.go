package ai

import (
	"testing"
)

func TestExtractCodeFromRepair_FencedCode(t *testing.T) {
	// 模拟 LLM 返回"解释 + fenced code"
	raw := "Here is the fixed code:\n\n```python\nimport pandas as pd\n\ndef run(ctx):\n    return {'signal': 'hold'}\n```\n\nThis fixes the error."
	code := extractCodeFromRepair(raw)
	if code == "" {
		t.Fatal("expected code extraction")
	}
	if code != "import pandas as pd\n\ndef run(ctx):\n    return {'signal': 'hold'}" {
		t.Errorf("unexpected code: %q", code)
	}
}

func TestExtractCodeFromRepair_Heuristic(t *testing.T) {
	// 模拟 LLM 返回 code-first（无 fence）
	raw := "import numpy as np\n\ndef run(ctx):\n    return {'signal': 'buy'}\n\n以上是修正后的代码"
	code := extractCodeFromRepair(raw)
	if code == "" {
		t.Fatal("expected heuristic extraction")
	}
}

func TestExtractCodeFromRepair_NoCode(t *testing.T) {
	// 模拟 LLM 返回纯解释文字
	raw := "这个策略的止损设置合理，但是建议增加止盈条件。可以考虑在盈利达到2%时自动止盈。"
	code := extractCodeFromRepair(raw)
	if code != "" {
		t.Errorf("expected empty, got: %q", code)
	}
}

func TestExtractCodeFromRepair_GenericFence(t *testing.T) {
	// 模拟 LLM 返回 generic ``` fence（无 python 标记）
	raw := "Fixed version:\n```\ndef run(ctx):\n    pass\n```"
	code := extractCodeFromRepair(raw)
	if code == "" {
		t.Fatal("expected extraction from generic fence")
	}
}
