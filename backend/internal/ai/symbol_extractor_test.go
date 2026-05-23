package ai

import (
	"context"
	"testing"
)

func TestSymbolExtractor_Chinese(t *testing.T) {
	e := NewSymbolExtractor()
	results := e.Extract(context.Background(), "帮我写一个交易比特币的策略")
	if len(results) == 0 {
		t.Fatal("expected at least BTCUSD from '比特币'")
	}
	found := false
	for _, r := range results {
		if r.Canonical == "BTCUSD" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected BTCUSD in results, got %v", results)
	}
}

func TestSymbolExtractor_English(t *testing.T) {
	e := NewSymbolExtractor()
	results := e.Extract(context.Background(), "trade EURUSD and GBPJPY with MACD crossover")
	if len(results) < 2 {
		t.Errorf("expected at least 2 symbols, got %d: %v", len(results), results)
	}
}

func TestSymbolExtractor_MultipleAliases(t *testing.T) {
	e := NewSymbolExtractor()
	// "比特币" + "btc" both map to BTCUSD — should deduplicate
	results := e.Extract(context.Background(), "交易比特币 btc 策略")
	btcCount := 0
	for _, r := range results {
		if r.Canonical == "BTCUSD" {
			btcCount++
		}
	}
	if btcCount != 1 {
		t.Errorf("expected exactly 1 BTCUSD (deduplicated), got %d", btcCount)
	}
}

func TestSymbolExtractor_UpperPattern(t *testing.T) {
	e := NewSymbolExtractor()
	results := e.Extract(context.Background(), "关注 XAUUSD 的走势")
	found := false
	for _, r := range results {
		if r.Canonical == "XAUUSD" {
			found = true
			if r.Confidence < 0.8 {
				t.Errorf("expected confidence ≥0.8 for direct pattern match, got %.2f", r.Confidence)
			}
			break
		}
	}
	if !found {
		t.Error("expected XAUUSD from direct pattern match")
	}
}

func TestSymbolExtractor_NoMatch(t *testing.T) {
	e := NewSymbolExtractor()
	results := e.Extract(context.Background(), "今天天气不错")
	if len(results) != 0 {
		t.Errorf("expected no symbols from unrelated text, got %v", results)
	}
}
