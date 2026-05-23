// Package service — strategy parameter catalog.
//
// Single source of truth for which technical indicators and which risk-management
// parameters the AntTrader sandbox exposes to generated strategy code. Both the
// LLM prompt builder and (eventually) the frontend param-selection UI read from
// this file, so there is no place where a rogue `sma_period` vs `ma_period`
// naming drift can sneak in.
package service

import (
	"fmt"
	"sort"
	"strings"
)

// StrategyIndicator describes one indicator helper available in the sandbox
// globals. `CallSignature` is shown to the LLM verbatim so it knows exactly
// how to call it; `ParamKeys` enumerates the `context['params']` keys the
// generated code must read (and that the frontend form will render).
type StrategyIndicator struct {
	Name          string              // exact Python identifier, e.g. "iRSI"
	CallSignature string              // "iRSI(prices, period, shift=0)"
	Description   string              // one-liner shown to the LLM and user
	ParamKeys     []StrategyParamSpec // params this indicator expects in `context['params']`
}

// StrategyParamSpec pins down one parameter key: its machine name (used by
// both the generated Python code and the frontend form), its type, default,
// and optional bounds. Keep the set of `Type` values small and explicit so
// the frontend form renderer stays trivial.
type StrategyParamSpec struct {
	Key         string  // e.g. "rsi_period"
	Label       string  // human-readable, e.g. "RSI period"
	Type        string  // "int" | "float" | "percent"
	Default     float64 // always stored as float64 in JSON; int params are floored
	Min         float64
	Max         float64
	Description string
}

// StrategyIndicators is the authoritative indicator menu. Order matters only
// for presentation; the LLM is told it MUST pick exactly from this list.
var StrategyIndicators = []StrategyIndicator{
	{
		Name:          "iMA",
		CallSignature: "iMA(prices, period, shift=0, method='sma') -> float",
		Description:   "Moving average. Returns the LATEST scalar value (NOT an array). Do NOT subscript like `ma[-1]` — `iMA(...)` IS the value. `method` is one of 'sma' | 'ema' | 'smma' | 'lwma'.",
		ParamKeys: []StrategyParamSpec{
			{Key: "ma_period", Label: "MA period", Type: "int", Default: 20, Min: 2, Max: 500},
		},
	},
	{
		Name:          "iRSI",
		CallSignature: "iRSI(prices, period, shift=0) -> float",
		Description:   "Relative Strength Index in [0,100]. Returns a SCALAR (latest value). Do NOT subscript with `[-1]` — compare directly: `if iRSI(close, 14) < 30: ...`.",
		ParamKeys: []StrategyParamSpec{
			{Key: "rsi_period", Label: "RSI period", Type: "int", Default: 14, Min: 2, Max: 200},
			{Key: "rsi_overbought", Label: "RSI overbought", Type: "float", Default: 70, Min: 50, Max: 100},
			{Key: "rsi_oversold", Label: "RSI oversold", Type: "float", Default: 30, Min: 0, Max: 50},
		},
	},
	{
		Name:          "iBands",
		CallSignature: "iBands(prices, period, deviation) -> (upper, middle, lower)",
		Description:   "Bollinger Bands. Returns a 3-tuple of SCALARS (latest upper / middle / lower). Do NOT subscript with `[-1]`: use `upper, middle, lower = iBands(close, 20, 2.0)` directly.",
		ParamKeys: []StrategyParamSpec{
			{Key: "bb_period", Label: "Bollinger period", Type: "int", Default: 20, Min: 2, Max: 500},
			{Key: "bb_deviation", Label: "Bollinger std-dev", Type: "float", Default: 2.0, Min: 0.1, Max: 10.0},
		},
	},
	{
		Name:          "iMACD",
		CallSignature: "iMACD(prices, fast, slow, signal) -> (macd, signal, hist)",
		Description:   "MACD. Returns a 3-tuple of SCALARS (latest macd line / signal line / histogram). Do NOT subscript with `[-1]`: use `macd, sig, hist = iMACD(close, 12, 26, 9)` directly.",
		ParamKeys: []StrategyParamSpec{
			{Key: "macd_fast", Label: "MACD fast period", Type: "int", Default: 12, Min: 2, Max: 200},
			{Key: "macd_slow", Label: "MACD slow period", Type: "int", Default: 26, Min: 3, Max: 400},
			{Key: "macd_signal", Label: "MACD signal period", Type: "int", Default: 9, Min: 2, Max: 200},
		},
	},
	{
		Name:          "iStochastic",
		CallSignature: "iStochastic(high, low, close, kperiod, dperiod, slowing) -> (%K, %D)",
		Description:   "Stochastic oscillator. Returns a 2-tuple of SCALARS (latest %K, %D in [0,100]). Do NOT subscript with `[-1]`: use `k, d = iStochastic(...)` directly.",
		ParamKeys: []StrategyParamSpec{
			{Key: "stoch_k", Label: "%K period", Type: "int", Default: 14, Min: 2, Max: 200},
			{Key: "stoch_d", Label: "%D period", Type: "int", Default: 3, Min: 1, Max: 100},
			{Key: "stoch_slowing", Label: "Slowing", Type: "int", Default: 3, Min: 1, Max: 100},
			{Key: "stoch_overbought", Label: "Stoch overbought", Type: "float", Default: 80, Min: 50, Max: 100},
			{Key: "stoch_oversold", Label: "Stoch oversold", Type: "float", Default: 20, Min: 0, Max: 50},
		},
	},
	{
		Name:          "iATR",
		CallSignature: "iATR(high, low, close, period) -> float",
		Description:   "Average True Range. Returns a SCALAR (latest ATR). Do NOT subscript. Use it directly for position sizing / adaptive stops.",
		ParamKeys: []StrategyParamSpec{
			{Key: "atr_period", Label: "ATR period", Type: "int", Default: 14, Min: 2, Max: 200},
		},
	},
	{
		Name:          "iCCI",
		CallSignature: "iCCI(high, low, close, period) -> float",
		Description:   "Commodity Channel Index. Returns a SCALAR (latest value). Do NOT subscript with `[-1]`.",
		ParamKeys: []StrategyParamSpec{
			{Key: "cci_period", Label: "CCI period", Type: "int", Default: 20, Min: 2, Max: 200},
			{Key: "cci_overbought", Label: "CCI overbought", Type: "float", Default: 100, Min: 50, Max: 500},
			{Key: "cci_oversold", Label: "CCI oversold", Type: "float", Default: -100, Min: -500, Max: -50},
		},
	},
	{
		Name:          "iMomentum",
		CallSignature: "iMomentum(prices, period) -> float",
		Description:   "Momentum oscillator. Returns a SCALAR (latest close / close[-period] * 100). Do NOT subscript with `[-1]`.",
		ParamKeys: []StrategyParamSpec{
			{Key: "momentum_period", Label: "Momentum period", Type: "int", Default: 14, Min: 2, Max: 200},
		},
	},
	{
		Name:          "iWPR",
		CallSignature: "iWPR(high, low, close, period) -> float",
		Description:   "Williams %R in [-100, 0]. Returns a SCALAR (latest value). Do NOT subscript with `[-1]`.",
		ParamKeys: []StrategyParamSpec{
			{Key: "wpr_period", Label: "Williams %R period", Type: "int", Default: 14, Min: 2, Max: 200},
		},
	},
}

// StrategyRiskParams are always available and every strategy SHOULD respect
// them even if no specific indicator was picked. Keeping them universal lets
// the backtest modal always show "stop loss / take profit / position risk"
// fields regardless of which indicators the user chose.
var StrategyRiskParams = []StrategyParamSpec{
	{Key: "stop_loss_pct", Label: "Stop-loss %", Type: "percent", Default: 1.0, Min: 0.0, Max: 50.0, Description: "Close the trade when the adverse move from entry reaches this percent of price. 0 = disabled."},
	{Key: "take_profit_pct", Label: "Take-profit %", Type: "percent", Default: 2.0, Min: 0.0, Max: 100.0, Description: "Close the trade when the favourable move from entry reaches this percent of price. 0 = disabled."},
	{Key: "risk_per_trade_pct", Label: "Risk per trade %", Type: "percent", Default: 1.0, Min: 0.01, Max: 10.0, Description: "Fraction of account equity to risk on a single trade; feed into `risk_size(...)` / `atr_size(...)`."},
	{Key: "max_positions", Label: "Max concurrent positions", Type: "int", Default: 1, Min: 1, Max: 20, Description: "Strategy must skip new entries when `context['positions_total']` already reaches this."},
}

// IndicatorCatalog is the shape returned by the REST handler for the
// frontend. It is intentionally simple JSON: the frontend can render a
// checkbox list of indicators with nested param controls, and a separate
// always-on risk param section.
type IndicatorCatalog struct {
	Indicators []StrategyIndicator `json:"indicators"`
	RiskParams []StrategyParamSpec `json:"risk_params"`
}

// GetIndicatorCatalog returns a snapshot of the current catalog for use by
// REST handlers or other consumers. Callers MUST treat it as read-only.
func GetIndicatorCatalog() IndicatorCatalog {
	return IndicatorCatalog{
		Indicators: StrategyIndicators,
		RiskParams: StrategyRiskParams,
	}
}

// AllowedIndicatorNames returns the set of indicator identifiers the sandbox
// exposes, kept in presentation order. Useful for prompt builders and for
// server-side validation of Debate session payloads.
func AllowedIndicatorNames() []string {
	out := make([]string, 0, len(StrategyIndicators))
	for _, ind := range StrategyIndicators {
		out = append(out, ind.Name)
	}
	return out
}

// AllowedParamKeys returns every param key the sandbox understands: all
// indicator params plus every risk param. The return value is sorted so
// downstream callers (prompt diffing, tests) are stable.
func AllowedParamKeys() []string {
	seen := map[string]struct{}{}
	for _, ind := range StrategyIndicators {
		for _, p := range ind.ParamKeys {
			seen[p.Key] = struct{}{}
		}
	}
	for _, p := range StrategyRiskParams {
		seen[p.Key] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// BuildIndicatorCatalogPromptBlock renders the indicator menu as a human-
// readable block for the LLM system prompt. It is deterministic (no map
// iteration) so the prompt stays byte-stable between runs.
func BuildIndicatorCatalogPromptBlock() string {
	var b strings.Builder
	b.WriteString("[Indicator catalog — use ONLY helpers from this list, with EXACTLY these param keys]\n")
	for _, ind := range StrategyIndicators {
		fmt.Fprintf(&b, "- `%s` — %s\n", ind.CallSignature, ind.Description)
		for _, p := range ind.ParamKeys {
			fmt.Fprintf(&b, "    • `params[\"%s\"]` (%s, default=%s)%s\n",
				p.Key, p.Type, formatDefault(p),
				ifNonEmpty("  — "+p.Description, p.Description != ""))
		}
	}
	b.WriteString("\n[Risk-management params — ALWAYS available; respect them in every strategy]\n")
	for _, p := range StrategyRiskParams {
		fmt.Fprintf(&b, "- `params[\"%s\"]` (%s, default=%s) — %s\n",
			p.Key, p.Type, formatDefault(p), p.Description)
	}
	return b.String()
}

// BuildIndicatorCatalogPromptBlockCompact is a token-slim variant for the
// code-generation system prompt: same helpers and keys, without per-key type
// / default lines (the validator and catalog still enforce correctness).
func BuildIndicatorCatalogPromptBlockCompact() string {
	var b strings.Builder
	b.WriteString("[Indicator catalog — use ONLY these helpers; param keys MUST match exactly]\n")
	for _, ind := range StrategyIndicators {
		keys := make([]string, 0, len(ind.ParamKeys))
		for _, p := range ind.ParamKeys {
			keys = append(keys, p.Key)
		}
		fmt.Fprintf(&b, "- %s — %s | keys: %s\n", ind.CallSignature, ind.Description, strings.Join(keys, ", "))
	}
	b.WriteString("\n[Risk-management params — ALWAYS available]\n")
	for _, p := range StrategyRiskParams {
		fmt.Fprintf(&b, "- params[\"%s\"] — %s\n", p.Key, p.Description)
	}
	return b.String()
}

// formatDefault prints an int default without a trailing ".0".
func formatDefault(p StrategyParamSpec) string {
	if p.Type == "int" {
		return fmt.Sprintf("%d", int(p.Default))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", p.Default), "0"), ".")
}

func ifNonEmpty(s string, cond bool) string {
	if cond {
		return s
	}
	return ""
}
