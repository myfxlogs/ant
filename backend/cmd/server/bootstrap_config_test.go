package main

import (
	"testing"

	"github.com/shopspring/decimal"
)

// bootstrap_config_test.go —— ENV-TO-PG-1：AI 阈值解析语义等价 pin。
// 原实现（env 直读）三分支逐一对齐：正十进制→其值；非正整数→其值；坏串/空→默认。
func TestParseAIDailyCostLimitSemantics(t *testing.T) {
	cases := []struct {
		v    string
		want string // decimal.String()
	}{
		{"45", "45"},     // 原链路：envInt=45 → decimal 45（decimal 覆写同值）
		{"45.5", "45.5"}, // 原链路：envInt 失败→base 50，decimal 覆写 45.5
		{"-3", "-3"},     // 原链路：envInt=-3 → base -3，decimal 覆写被 IsPositive 拒 → -3
		{"0", "0"},       // 原链路：base 0，覆写拒 → 0
		{"abc", "50"},    // 原链路：envInt 失败→50，decimal 解析失败→50
		{"", "50"},       // 原链路：env 缺→50
		{"1e3", "1000"},  // decimal 可解析科学计数法（正）→其值
	}
	for _, tc := range cases {
		got := parseAIDailyCostLimit(tc.v)
		if got.String() != tc.want {
			t.Fatalf("parseAIDailyCostLimit(%q) = %s, want %s", tc.v, got.String(), tc.want)
		}
	}
}

// 原语义：可解析且非负→其值；坏串/空/负→默认 1.0。
func TestParseAIMinBalanceSemantics(t *testing.T) {
	cases := []struct {
		v    string
		want string
	}{
		{"2.5", "2.5"},
		{"0", "0"},
		{"-5", "1"}, // 负→默认 1.0
		{"abc", "1"},
		{"", "1"},
	}
	for _, tc := range cases {
		got := parseAIMinBalance(tc.v)
		if !got.Equal(decimal.RequireFromString(tc.want)) {
			t.Fatalf("parseAIMinBalance(%q) = %s, want %s", tc.v, got.String(), tc.want)
		}
	}
}
