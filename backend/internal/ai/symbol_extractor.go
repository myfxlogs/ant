// Package ai — SymbolExtractor: natural language → canonical symbol (M4-1).
// Given user input like "帮我写一个交易比特币的策略", extracts canonical symbols
// using a lightweight AI prompt or deterministic canonical mapping.
package ai

import (
	"context"
	"regexp"
	"strings"
)

// ExtractedSymbol represents a canonical symbol found in user input.
type ExtractedSymbol struct {
	Canonical  string  `json:"canonical"`   // e.g. BTCUSD
	RawText    string  `json:"raw_text"`    // the original text fragment
	Confidence float64 `json:"confidence"`  // 0.0-1.0
}

// SymbolExtractor extracts canonical symbols from natural language text.
type SymbolExtractor struct {
	// knownSymbols is a lookup from Chinese/English aliases to canonical names
	knownSymbols map[string]string
}

// NewSymbolExtractor creates a symbol extractor with the standard alias dictionary.
func NewSymbolExtractor() *SymbolExtractor {
	return &SymbolExtractor{knownSymbols: symbolAliases()}
}

// Extract finds canonical symbols in user text using deterministic keyword matching.
// For full AI-powered extraction, use ExtractWithAI (M4-3).
func (e *SymbolExtractor) Extract(ctx context.Context, text string) []ExtractedSymbol {
	textLower := strings.ToLower(text)
	var results []ExtractedSymbol
	seen := make(map[string]bool)

	for alias, canonical := range e.knownSymbols {
		aliasLower := strings.ToLower(alias)
		if strings.Contains(textLower, aliasLower) && !seen[canonical] {
			seen[canonical] = true
			results = append(results, ExtractedSymbol{
				Canonical:  canonical,
				RawText:    alias,
				Confidence: confidenceForMatch(alias, text),
			})
		}
	}

	// Also try pattern match: 6-letter uppercase pairs like EURUSD, BTCUSD
	pattern := regexp.MustCompile(`\b([A-Z]{6,7})\b`)
	for _, m := range pattern.FindAllStringSubmatch(text, -1) {
		candidate := strings.ToUpper(m[1])
		if !seen[candidate] && e.isKnownCanonical(candidate) {
			seen[candidate] = true
			results = append(results, ExtractedSymbol{
				Canonical:  candidate,
				RawText:    m[1],
				Confidence: 0.85,
			})
		}
	}

	return results
}

// isKnownCanonical checks if a string is a known canonical symbol.
func (e *SymbolExtractor) isKnownCanonical(s string) bool {
	for _, v := range e.knownSymbols {
		if v == s {
			return true
		}
	}
	return false
}

// confidenceForMatch estimates extraction confidence.
func confidenceForMatch(alias, text string) float64 {
	if strings.Contains(strings.ToLower(text), strings.ToLower(alias)) {
		return 0.8
	}
	return 0.5
}

// symbolAliases returns the Chinese/English alias → canonical symbol map.
// Covers ~50 mainstream symbols from internal/symbol/canonical.go.
func symbolAliases() map[string]string {
	return map[string]string{
		// Forex majors (Chinese + English aliases)
		"欧元": "EURUSD", "eurusd": "EURUSD", "欧元美元": "EURUSD",
		"英镑": "GBPUSD", "gbpusd": "GBPUSD", "英镑美元": "GBPUSD",
		"日元": "USDJPY", "usdjpy": "USDJPY", "美日": "USDJPY", "美元日元": "USDJPY",
		"瑞郎": "USDCHF", "usdchf": "USDCHF", "美瑞": "USDCHF",
		"澳元": "AUDUSD", "audusd": "AUDUSD", "澳美": "AUDUSD", "澳元美元": "AUDUSD",
		"加元": "USDCAD", "usdcad": "USDCAD", "美加": "USDCAD",
		"纽元": "NZDUSD", "nzdusd": "NZDUSD", "纽美": "NZDUSD",
		"欧镑": "EURGBP", "eurgbp": "EURGBP",
		"欧日": "EURJPY", "eurjpy": "EURJPY",
		"镑日": "GBPJPY", "gbpjpy": "GBPJPY",

		// Commodities
		"黄金": "XAUUSD", "xauusd": "XAUUSD", "gold": "XAUUSD",
		"白银": "XAGUSD", "xagusd": "XAGUSD", "silver": "XAGUSD",
		"原油": "XTIUSD", "xtiusd": "XTIUSD", "wti": "XTIUSD", "oil": "XTIUSD",
		"布伦特": "XBRUSD", "xbrusd": "XBRUSD", "brent": "XBRUSD",
		"天然气": "XNGUSD", "xngusd": "XNGUSD",

		// Indices
		"道琼斯": "US30", "us30": "US30", "dow": "US30",
		"纳斯达克": "US100", "us100": "US100", "nasdaq": "US100", "纳指": "US100",
		"标普": "US500", "us500": "US500", "sp500": "US500", "标普500": "US500", "spx": "US500",
		"德国dax": "GER40", "ger40": "GER40", "dax": "GER40",
		"富时100": "UK100", "uk100": "UK100", "富时": "UK100",
		"日经": "JPN225", "jpn225": "JPN225", "日经225": "JPN225", "nikkei": "JPN225",
		"恒生": "HK50", "hk50": "HK50", "恒指": "HK50",

		// Crypto
		"比特币": "BTCUSD", "btcusd": "BTCUSD", "bitcoin": "BTCUSD", "btc": "BTCUSD",
		"以太坊": "ETHUSD", "ethusd": "ETHUSD", "ethereum": "ETHUSD", "eth": "ETHUSD",
		"瑞波": "XRPUSD", "xrpusd": "XRPUSD", "ripple": "XRPUSD", "xrp": "XRPUSD",
		"莱特币": "LTCUSD", "ltcusd": "LTCUSD",
		"狗狗币": "DOGEUSD", "dogeusd": "DOGEUSD", "doge": "DOGEUSD",
		"solana": "SOLUSD", "solusd": "SOLUSD", "sol": "SOLUSD",
		"cardano": "ADAUSD", "adausd": "ADAUSD", "ada": "ADAUSD",
	}
}
