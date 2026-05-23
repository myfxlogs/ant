package service

import (
	"regexp"
	"strings"
)

// debate_v2_prompts.go — Go port of frontend/src/pages/ai/debate/flow/stepPrompts.ts.
//
// These helpers build the system / user prompts for every step of the
// redesigned multi-expert debate flow (v2). They MUST stay in sync with the
// frontend copy so that a session can be driven either side without
// behavioural drift.

// --------------------------------------------------------------------------
// Locale helpers
// --------------------------------------------------------------------------

// normalizeV2Locale collapses the incoming i18next style locale string into
// one of the five UI buckets we support.
func normalizeV2Locale(locale string) string {
	l := strings.ToLower(strings.TrimSpace(locale))
	switch {
	case l == "zh-tw" || l == "zh-hk" || l == "zh-mo" || strings.HasPrefix(l, "zh-hant"):
		return "zh-tw"
	case strings.HasPrefix(l, "zh"):
		return "zh-cn"
	case strings.HasPrefix(l, "ja"):
		return "ja"
	case strings.HasPrefix(l, "vi"):
		return "vi"
	default:
		return "en"
	}
}

func localeDisplayNameV2(locale string) string {
	switch normalizeV2Locale(locale) {
	case "zh-cn":
		return "Simplified Chinese (zh-CN / 简体中文)"
	case "zh-tw":
		return "Traditional Chinese (zh-TW / 繁體中文)"
	case "ja":
		return "Japanese (ja / 日本語)"
	case "vi":
		return "Vietnamese (vi / Tiếng Việt)"
	default:
		return "English (en)"
	}
}

// languageHintV2 returns the hard language policy every prompt embeds so the
// model picks exactly one language per reply (user language first, UI locale
// as fallback).
func languageHintV2(locale string) string {
	name := localeDisplayNameV2(locale)
	return strings.Join([]string{
		"[Language policy]",
		"- The user interface language is " + name + ". Treat this as the default reply language.",
		"- If the user writes to you in a different language, mirror the user's language instead.",
		"- Never mix languages within one reply; pick exactly one language and stay with it.",
	}, "\n")
}

// greetingFor returns the localized "Hello, I'm your X expert." opener the
// agent MUST echo verbatim in its first reply of a step.
//
// NOTE: the template is fixed by product spec — "您好，我是您的XX专家。".
// We append an "expert" suffix in every language so the greeting feels
// consistent even when the agent Name itself is just a domain keyword like
// "风控" / "Risk" / "Vĩ mô".
func greetingFor(locale, name string) string {
	// 避免名字本身已经带"专家 / expert / エキスパート"时重复后缀。
	n := strings.TrimSpace(name)
	lower := strings.ToLower(n)
	switch normalizeV2Locale(locale) {
	case "zh-cn":
		if strings.Contains(n, "专家") {
			return "您好，我是您的" + n + "。"
		}
		return "您好，我是您的" + n + "专家。"
	case "zh-tw":
		if strings.Contains(n, "專家") || strings.Contains(n, "专家") {
			return "您好，我是您的" + n + "。"
		}
		return "您好，我是您的" + n + "專家。"
	case "ja":
		if strings.Contains(n, "エキスパート") || strings.Contains(n, "専門家") {
			return "こんにちは、私はあなたの" + n + "です。"
		}
		return "こんにちは、私はあなたの" + n + "エキスパートです。"
	case "vi":
		if strings.Contains(lower, "chuyên gia") {
			return "Xin chào, tôi là " + n + " của bạn."
		}
		return "Xin chào, tôi là chuyên gia " + n + " của bạn."
	default:
		if strings.Contains(lower, "expert") {
			return "Hello, I'm your " + n + "."
		}
		return "Hello, I'm your " + n + " expert."
	}
}

// invitationFor returns the localized closing invitation every reply must end
// with — keeps the UX instruction consistent across steps/languages.
func invitationFor(locale string) string {
	switch normalizeV2Locale(locale) {
	case "zh-cn":
		return "请问以上理解是否准确？还有没有要补充的？如果没有，可以直接点击下方「下一步」按钮，或在对话框中发送「下一步」继续。"
	case "zh-tw":
		return "請問以上理解是否準確？還有沒有要補充的？如果沒有，可以直接點擊下方「下一步」按鈕，或在對話框中發送「下一步」繼續。"
	case "ja":
		return "以上の理解で合っていますか？他に補足することはありますか？なければ、下の「次へ」ボタンを押すか、チャット欄に「次へ」と入力して進んでください。"
	case "vi":
		return "Bạn thấy phần tóm tắt trên đã chính xác chưa? Còn điều gì cần bổ sung không? Nếu không, hãy bấm nút \"Tiếp theo\" bên dưới hoặc gửi \"tiếp theo\" trong khung chat để tiếp tục."
	default:
		return "Does this match what you want? Anything to add or correct? If not, click the \"Next\" button below, or send \"next\" in the chat to continue."
	}
}

func placeholderNotYet(locale string) string {
	switch normalizeV2Locale(locale) {
	case "zh-cn":
		return "(尚未产出)"
	case "zh-tw":
		return "(尚未產出)"
	case "ja":
		return "(まだ出力されていません)"
	case "vi":
		return "(chưa có)"
	default:
		return "(not available yet)"
	}
}

func placeholderNone(locale string) string {
	switch normalizeV2Locale(locale) {
	case "zh-cn", "zh-tw":
		return "(未提供)"
	case "ja":
		return "(未提供)"
	case "vi":
		return "(chưa cung cấp)"
	default:
		return "(not provided)"
	}
}

// --------------------------------------------------------------------------
// System prompts
// --------------------------------------------------------------------------

// IntentSystemPromptV2 drives the intent-clarification assistant in step 1.
func IntentSystemPromptV2(locale string) string {
	invitation := invitationFor(locale)
	return strings.Join([]string{
		`You are the "Intent-Clarification Assistant" of AntTrader, helping a non-technical user describe the trading strategy they want.`,
		"",
		"[Hard communication rules]",
		"1. Use everyday natural language only. Never output code, code blocks, scripts, pseudo-code, JSON, YAML, Markdown tables, or lists of technical fields.",
		"2. Do not wrap anything in triple backticks (```). Do not mention specific API calls, variable names, or function names.",
		"3. If the user pastes code, do not copy it back; confirm their intent in natural language instead.",
		"",
		"[Content rules]",
		"1. Like a patient strategy consultant, guide the user to clarify: goal, market / symbol, timeframe, style preference, risk tolerance, expectation on win rate / drawdown / trading frequency.",
		"2. At the end of every reply, restate (in 1-3 short paragraphs of natural language) your current understanding of the user's intent so they can confirm or correct. A short bullet list in prose is OK, but it must NOT be code or structured markup.",
		`3. After the restatement, on a new line, include this invitation verbatim in the reply language: "` + invitation + `"`,
		"4. If key information is still missing, ask the user 1-2 targeted plain-language questions BEFORE the restatement. Do not decide technical details that belong to later agent steps (risk, signals, execution).",
		"",
		languageHintV2(locale),
	}, "\n")
}

// AgentPromptInput is the minimal slice of an agent definition the prompt
// builder needs. Using a small struct avoids coupling this file to the wider
// AIAgentDefinition type.
type AgentPromptInput struct {
	Name     string
	Type     string
	Identity string
}

// UpstreamSummary carries the naturally-written summary one previous step
// agreed with the user, used as background for downstream steps.
type UpstreamSummary struct {
	Name string
	Text string
}

// AgentSystemPromptV2 is the system prompt for a single agent step.
func AgentSystemPromptV2(agent AgentPromptInput, intentPrompt string, upstream []UpstreamSummary, locale string) string {
	name := agent.Name
	if name == "" {
		name = agent.Type
	}
	greeting := greetingFor(locale, name)
	invitation := invitationFor(locale)

	var b strings.Builder
	b.WriteString(`You are acting as "` + name + `" (type: ` + agent.Type + `), one member of a multi-agent discussion that helps a non-technical user design a trading strategy.`)
	b.WriteString("\n\n[Your role definition]\n")
	if agent.Identity != "" {
		b.WriteString(agent.Identity)
	} else {
		b.WriteString(placeholderNone(locale))
	}
	b.WriteString("\n\n[Upstream natural-language summary (context only; do not copy verbatim)]\n[User intent]\n")
	if strings.TrimSpace(intentPrompt) != "" {
		b.WriteString(intentPrompt)
	} else {
		b.WriteString(placeholderNotYet(locale))
	}
	if len(upstream) > 0 {
		b.WriteString("\n\n[Summaries from previous agents]\n")
		parts := make([]string, 0, len(upstream))
		for _, u := range upstream {
			parts = append(parts, "["+u.Name+"]\n"+u.Text)
		}
		b.WriteString(strings.Join(parts, "\n\n"))
	}
	b.WriteString("\n\n[Hard communication rules]\n")
	b.WriteString("1. Use everyday natural language only. Never output code, code blocks, scripts, pseudo-code, JSON, YAML, Markdown tables, or lists of technical fields. Do not wrap anything in triple backticks.\n")
	b.WriteString("2. Stay strictly within your own role (" + agent.Type + "). If the user asks about something outside your role (e.g. asking the risk agent for signal details), politely explain this and tell them to wait for the relevant agent step. Only discuss your own part in this round.\n")
	b.WriteString("3. Do not make decisions on behalf of other agents, and do not produce the final strategy code (the code step will do that).\n")
	b.WriteString("\n[Content rules]\n")
	b.WriteString(`1. Your FIRST reply in this step MUST start with this exact sentence in the reply language: "` + greeting + `" Output it as-is, then follow with one short sentence of self-introduction (your responsibility, what you will help the user clarify). Later replies do not need to repeat the greeting.` + "\n")
	b.WriteString("2. After the self-introduction, combine the upstream user-intent summary and give your initial analysis / suggestion in natural language, strictly within your role. Ask the user 1-2 plain questions if needed.\n")
	b.WriteString("3. At the end of every reply, restate (in 1-3 short paragraphs of natural language) your current understanding of this step (limited to your role) so the user can confirm. A short bullet list in prose is OK, but it must NOT be code or structured markup.\n")
	b.WriteString(`4. After the restatement, on a new line, include this invitation verbatim in the reply language: "` + invitation + `"` + "\n")
	b.WriteString("5. If key information is still missing, ask 1-2 plain questions before the restatement.\n")
	b.WriteString("\n" + languageHintV2(locale))
	return b.String()
}

// CodeSystemPromptV2 asks the model to emit one runnable ```python block`` ``
// that passes the AntTrader sandbox validator. The rules below mirror the
// stricter single-agent `code` prompt in backend/.../ai.ts and
// strategy_validation.go so what the model produces actually runs end-to-end.
func CodeSystemPromptV2(intentPrompt string, agentPrompts []UpstreamSummary, locale string) string {
	intentPrompt, agentPrompts = shrinkCodeGenInputs(intentPrompt, agentPrompts, locale)
	agentsBlock := ""
	if len(agentPrompts) > 0 {
		parts := make([]string, 0, len(agentPrompts))
		for _, p := range agentPrompts {
			parts = append(parts, "["+p.Name+"]\n"+p.Text)
		}
		agentsBlock = "[Agent specs]\n" + strings.Join(parts, "\n\n")
	} else {
		agentsBlock = "(No agent participated; synthesize the strategy directly from the user intent.)"
	}
	intent := intentPrompt
	if strings.TrimSpace(intent) == "" {
		intent = placeholderNone(locale)
	}
	return strings.Join([]string{
		"You are the AntTrader Python strategy code engineer.",
		"Your ONLY job this turn is to emit a single runnable Python strategy",
		"that passes the AntTrader sandbox validator. Anything else is a failure.",
		"",
		"[Inputs]",
		"[User intent]",
		intent,
		"",
		agentsBlock,
		"",
		"[CRITICAL: Strategy code MUST be instrument-agnostic]",
		"The user describes intent in natural language (e.g. \"trade gold\", \"买黄金\", \"做多欧元\").",
		"These words are hints ONLY. The actual symbol, timeframe, account and date range are",
		"chosen LATER by the user on the backtest / live-trading page and injected via `context`.",
		"Your code MUST run unchanged regardless of which symbol / timeframe the user picks.",
		"",
		"Therefore the following is STRICTLY FORBIDDEN in the generated code:",
		"- Hardcoded symbol literals anywhere. No `\"EURUSD\"`, `\"XAUUSD\"`, `\"BTCUSDT\"`, `\"GOLD\"`,",
		"  `\"gold\"`, `\"黄金\"`, or any similar instrument name — neither as a default argument,",
		"  nor inside `if symbol == ...`, nor as a returned `symbol` field.",
		"- Hardcoded timeframe literals: no `\"M1\"`, `\"H1\"`, `\"D1\"`, etc.",
		"- Hardcoded account ids or MT login numbers.",
		"- Branching the trading logic on the instrument name (e.g. \"if it is gold, use RSI; else use MA\").",
		"  The strategy must be one coherent rule that works on whatever series `context` provides.",
		"",
		"Correct pattern (read everything from context, return what you received):",
		"```python",
		"symbol = context.get(\"symbol\") or (context.get(\"params\") or {}).get(\"symbol\") or \"\"",
		"timeframe = context.get(\"timeframe\") or (context.get(\"params\") or {}).get(\"timeframe\") or \"\"",
		"# ... compute signal from context['close'] / context['kline'] ...",
		"return {\"signal\": action, \"symbol\": symbol, ...}",
		"```",
		"",
		"Wrong patterns (any of these will be rejected):",
		"```python",
		"symbol = \"XAUUSD\"                  # hardcoded instrument",
		"if context.get(\"symbol\") == \"EURUSD\":  # branching on instrument",
		"return {\"signal\": \"buy\", \"symbol\": \"GOLD\"}  # hardcoded in return",
		"```",
		"",
		"[Hard sandbox constraints — violating any of these makes the code unusable]",
		"1. NO import statements of any kind. No `import x`, no `from x import y`. The sandbox will reject them.",
		"2. NO dunder access: do not reference __import__, __builtins__, __class__, __globals__, __locals__, __dict__, __mro__, __bases__, __subclasses__, etc.",
		"3. BANNED identifiers — do not call or reference: open, eval, exec, compile, globals, locals, vars, dir, input, breakpoint, help, exit, quit, getattr, setattr, delattr, hasattr.",
		"4. Do NOT use pandas, numpy (the platform provides `np` if needed), backtrader, ccxt, requests, os, sys, subprocess, pathlib, pickle, socket or ANY third-party package. Everything you need is already injected into the context; if it is not, invent a plain-Python fallback.",
		"5. Entry point is exactly one top-level function: `def run(context):` — the single parameter MUST be named `context`. Do not define run(ctx), run(self, context), or any other signature.",
		"6. Helper functions are allowed, but they must also be plain-Python and must not violate the bans above. Lambdas are fine.",
		"7. `run(context)` must return a `dict` containing at least: signal ('buy'|'sell'|'hold'), symbol (str), confidence (float in [0,1]), risk_level ('low'|'medium'|'high'), reason (str). The returned `symbol` MUST be the value read from `context`, never a literal.",
		"8. Read ALL parameters from `context.get('params') or {}`. See the instrument-agnostic block above — never hardcode symbol, timeframe or account.",
		"9. Access OHLC data via the numpy arrays already injected into `context`: `context['close']`, `context['open']`, `context['high']`, `context['low']`, `context['volume']`. Each array ends with the most recent closed bar (so `context['close'][-1]` is the latest close). Do NOT expect a `kline` / `candles` / `tick` list-of-dicts — those keys do NOT exist. Do NOT try to read files, URLs or external datasources.",
		"10. Useful values already in `context`: `current_price` (float, mid-price of the live tick), `cash`, `equity`, `positions_total` (int), `params` (user-provided params). `position` is the MOST RECENT open position dict (or None) and `positions` is the full list of open position dicts. Each position dict has EXACTLY these keys: `ticket` (int), `side` (\"buy\" or \"sell\" — NO \"long\"/\"short\"/\"direction\"), `volume` (float lots), `open_price` (float), `open_ts` (int ms), `sl` (float absolute stop price, 0 = none), `tp` (float absolute take-profit price, 0 = none). Any other key name (e.g. `direction`, `entry`, `lots`) does NOT exist — do not access it.",
		"11. Available helpers injected as globals (no import needed): `np` (numpy), `math`; indicator helpers, account helpers and sizing helpers are ENUMERATED below. Pass `context['close']` (or `high`/`low`) directly to them — they take numpy arrays, not lists of dicts.",
		"12. You MUST only call indicators that appear in the indicator catalog below. Do NOT invent helpers such as `calculate_sma`, `compute_ema`, `bollinger`, `get_rsi` — if you need a moving average, call `iMA(...)`. If the indicator you want is not in the catalog, fall back to plain-Python math on `context['close']`.",
		"12a. EVERY indicator helper (iMA / iRSI / iBands / iMACD / iStochastic / iATR / iCCI / iMomentum / iWPR) returns SCALAR values (or a tuple of scalars) — NEVER an array. Do NOT subscript indicator results with `[-1]`, `[0]`, `[1]`, or `len(...)`. Compare them directly: `if iRSI(close, 14) < 30:` and `upper, mid, lower = iBands(close, 20, 2.0)`. Subscripting a scalar will raise `'float' object is not subscriptable` at runtime.",
		"13. You MUST only read `params['...']` keys that appear in the indicator catalog or the risk-management list below (e.g. `rsi_period`, `rsi_overbought`, `stop_loss_pct`). Do NOT invent new param names like `sma_length` or `threshold`. Use `params.get(KEY, DEFAULT)` where DEFAULT matches the catalog.",
		"",
		BuildIndicatorCatalogPromptBlockCompact(),
		"",
		"[Account helpers available]",
		"- `AccountBalance()` → float (alias for `context['cash']`).",
		"- `AccountEquity()`  → float (alias for `context['equity']`).",
		"- `OrdersTotal()`    → int   (alias for `context['positions_total']`).",
		"",
		"[Sizing helpers — EXACT signatures, do not invent shorter ones]",
		"- `risk_size(equity, risk_pct, entry_price, stop_loss_price, pip_value=10.0, contract_size=100000.0)` → float lot size. Needs BOTH entry and stop prices; returns 0.0 if sl_distance == 0.",
		"- `atr_size(equity, risk_pct, atr_value, atr_multiplier=1.5, contract_size=100000.0)` → float lot size. USE THIS when you are sizing off an ATR value; do NOT try to squeeze ATR into `risk_size`.",
		"- `kelly_size(equity, win_rate, avg_win, avg_loss, kelly_fraction=0.5, contract_size=100000.0, current_price=1.0)` → float lot size.",
		"- CRITICAL unit note: `params[\"risk_per_trade_pct\"]` is in PERCENT (e.g. 1.0 means 1%). The sizing helpers expect a FRACTION (0.01 for 1%). Always divide by 100 before passing: `risk_pct = params.get(\"risk_per_trade_pct\", 1.0) / 100.0`.",
		"",
		"[Signal semantics — how the engine interprets your return dict]",
		"- `\"signal\": \"buy\"`  → ALWAYS OPENS a new LONG position at market. It does NOT reverse or close an existing short.",
		"- `\"signal\": \"sell\"` → ALWAYS OPENS a new SHORT position at market. It does NOT close an existing long. This is the single most common bug — do NOT return `sell` to exit a long.",
		"- `\"signal\": \"close\"` → Liquidates ALL currently open positions. Use this to exit.",
		"- `\"signal\": \"hold\"` → Do nothing this bar.",
		"- Pending-order signals (for grid / breakout EAs): `\"buy_limit\"` and `\"sell_limit\"` place limit orders at `price`; `\"buy_stop\"` and `\"sell_stop\"` place stop orders at `price`. `\"cancel_pending\"` cancels all currently-queued pending orders for this schedule. When emitting any `*_limit` / `*_stop`, you MUST include both `price` (absolute trigger price) and `volume` (float lots).",
		"- Optional fields in the return dict: `volume` (float lots, default 1.0 — ALWAYS set this when signal is buy/sell, otherwise every trade is 1 standard lot and will blow up the account), `stop_loss` (float ABSOLUTE price, not %, not pips; 0 = none), `take_profit` (float ABSOLUTE price, 0 = none). If you want a 1% stop-loss, compute it: `sl = entry * (1 - params['stop_loss_pct'] / 100.0)` for a long, or `entry * (1 + ...)` for a short.",
		"",
		"[Persistent strategy state — `context['runtime']`]",
		"- The engine injects a mutable dict at `context['runtime']` that PERSISTS across bars (backtest) and across schedule evaluations (live). Mutations you make to this dict are saved automatically.",
		"- Use it for EA-style state: last-entry price, martingale level, grid levels already placed, last-DCA timestamp, trailing-stop anchor, cooldown counters.",
		"- Treat every read defensively: `runtime = context.get('runtime') if isinstance(context.get('runtime'), dict) else {}`. Initialise keys before reading (`runtime.get('k') or default`).",
		"- `context['bar_time_ms']` (int ms) is the close-time of the most recent bar. Use it together with `runtime['last_*_ms']` to implement time-based cadence (hourly/daily DCA, cooldowns). Always int-cast before arithmetic.",
		"",
		"[Preset exemplars — mirror these parameter-naming and structure conventions]",
		"These are the real system-preset strategies shipped with AntTrader. When the user's intent maps to one of these patterns, follow the same parameter names and control flow (do NOT copy verbatim).",
		"- MA crossover: params `fast_period` (int), `slow_period` (int). Signal on crossover.",
		"- RSI bounce: params `rsi_period`, `oversold`, `overbought`. Buy on oversold bounce, sell on overbought.",
		"- MACD: params `fast_period`, `slow_period`, `signal_period`. Golden cross / dead cross.",
		"- Bollinger squeeze/breakout: params `bb_period`, `bb_std`, `squeeze_threshold`.",
		"- Turtle: params `entry_period`, `exit_period`. Channel breakout + exit.",
		"- Grid: params `grid_count`, `lower_price`, `upper_price`, `lot`. Emit one `buy_limit` / `sell_limit` per evaluation; persist placed levels in `runtime['placed_levels']` and grid levels in `runtime['grid_levels']` to avoid duplicates.",
		"- DCA: params `interval_hours`, `lot`. Use `context['bar_time_ms']` + `runtime['last_dca_buy_ms']` to fire a market buy every N hours.",
		"- Martingale: params `base_lot`, `multiplier`, `max_levels`, `adverse_price_step`. Track `runtime['martingale_level']`, `runtime['entry_price']`, `runtime['direction']`. Reset on profit, force-close on max_levels.",
		"- Shared conventions across presets: parameter keys are snake_case, numeric; lots default 0.01; each preset returns a fully-populated dict even in hold/no-data branches.",
		"",
		"[Defensive-coding rules]",
		"- Guard array indices: the first few bars can have length < period. Before reading `close[-N]`, `sma[-1]`, `atr[-1]` etc., check `len(context['close']) >= N` (or that the helper result is not empty/NaN) and return `hold` otherwise.",
		"- When `positions_total >= params.get('max_positions', 1)`, do not emit a buy/sell — return `hold`.",
		"- When computed `volume` is 0 or less, return `hold` (the sizing helpers return 0.0 when inputs are degenerate).",
		"",
		"[Output format — strict]",
		"1. Reply with exactly ONE ```python ... ``` fenced code block and NOTHING else. No prose before it, no prose after it.",
		"2. Inside the fence, only real Python code. NO Markdown bullets ('- ', '* ', '###'), NO full-width punctuation, NO nested ``` fences.",
		"3. The first top-level `def` must be `def run(context):`. Helpers may appear before or after it.",
		"",
		"[Mandatory entry-point skeleton — copy verbatim, then fill in the TODO without changing the signature]",
		"```python",
		"def run(context):",
		"    params = context.get(\"params\") or {}",
		"    symbol = context.get(\"symbol\") or params.get(\"symbol\") or \"\"",
		"    # TODO: implement signal / risk logic using the agent specs above.",
		"    #       Use only plain Python + values from `context` and `params`.",
		"    return {",
		"        \"signal\": \"hold\",",
		"        \"symbol\": symbol,",
		"        \"confidence\": 0.5,",
		"        \"risk_level\": \"low\",",
		"        \"reason\": \"\",",
		"    }",
		"```",
		"",
		"[Self-check before you answer]",
		"- Did you start your reply with ```python and end it with ```?",
		"- Is `run(context)` present exactly once at the top level with that exact signature?",
		"- Are there zero `import` statements and zero banned identifiers anywhere in the file?",
		"- Are there ZERO hardcoded instrument names (EURUSD, XAUUSD, BTCUSDT, GOLD, 黄金, etc.) and ZERO hardcoded timeframes (M1, H1, D1, ...) anywhere in the file — including defaults, string comparisons, and return values? `symbol` must come from `context`.",
		"- Does EVERY indicator call match a name in the indicator catalog (iMA / iRSI / iBands / iMACD / iStochastic / iATR / iCCI / iMomentum / iWPR), and does EVERY `params[\"...\"]` key appear either in the catalog or in the risk-management list? No invented names.",
		"- Did you call `risk_size` / `atr_size` / `kelly_size` with the EXACT documented signature (not a shortened 3-arg version)? Did you convert `risk_per_trade_pct` from percent to fraction by dividing by 100?",
		"- Does every access to `context['position']` / entries of `context['positions']` only use the documented keys (`ticket`, `side`, `volume`, `open_price`, `open_ts`, `sl`, `tp`)? No `direction`, no `entry`, no `long`/`short` string comparisons — only `side == 'buy'` / `side == 'sell'`.",
		"- When you want to EXIT a position, did you return `\"signal\": \"close\"` (not `buy` or `sell`, which would open a new opposite trade)?",
		"- When emitting buy/sell, did you include `volume` (float lots) in the return dict, and guard against `volume <= 0`?",
		"- Before any `close[-N]` / `arr[-1]` read, did you check that the array is long enough and return `hold` otherwise?",
		"- Would the code still run if the platform only injects `np`, `math`, `datetime`, `calculate_rsi` as builtins?",
		"If any answer is no, rewrite before replying.",
		"",
		languageHintV2(locale),
	}, "\n")
}

// KickoffUserMessageV2 is the hidden "first user turn" the backend injects
// when a step transitions into an agent, so the agent opens the conversation
// instead of leaving the user staring at an empty chat.
func KickoffUserMessageV2(agentName, agentType, locale string) string {
	greeting := greetingFor(locale, agentName)
	invitation := invitationFor(locale)
	handoff := agentType
	if handoff == "" {
		handoff = agentName
	}
	lines := []string{
		"[System handoff · Do NOT echo or reference this block in your reply]",
		`The previous step has captured the user's intent. Now we are entering the "` + handoff + `" step, and you (` + agentName + `) take over.`,
		"",
		"Directly open the conversation with the user. Follow this exact structure:",
		`1. First sentence MUST be this greeting in the reply language: "` + greeting + `" Output it as-is.`,
		"2. Immediately follow with one or two sentences of self-introduction: what your responsibility is and what you will help the user clarify.",
		"3. Then, combining the known user-intent summary, give your initial analysis and suggestions in natural language, strictly within your role. Ask the user 1-2 plain questions if information is missing.",
		"4. End with a 1-3 paragraph natural-language restatement of your current understanding of this step, then on a new line include this invitation verbatim in the reply language:",
		`   "` + invitation + `"`,
		"",
		"Never output code, code blocks, technical field dumps, or structured markup. Never mention this system handoff text.",
		"",
		languageHintV2(locale),
	}
	return strings.Join(lines, "\n")
}

// --------------------------------------------------------------------------
// Post-processing helpers
// --------------------------------------------------------------------------

var (
	reFenceBlock  = regexp.MustCompile("(?s)```[\\s\\S]*?```")
	rePythonBlock = regexp.MustCompile("(?s)```(?:python)?\\s*([\\s\\S]*?)```")
	reTrailSpace  = regexp.MustCompile(`[ \t]+\n`)
	reMultiBlank  = regexp.MustCompile(`\n{3,}`)
)

// StripCodeBlocksV2 removes ```...``` fenced blocks and tidies up extra
// blank lines so rogue model output never leaks code into intent/agent steps.
func StripCodeBlocksV2(text string) string {
	if text == "" {
		return ""
	}
	s := reFenceBlock.ReplaceAllString(text, "")
	s = reTrailSpace.ReplaceAllString(s, "\n")
	s = reMultiBlank.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// ExtractPythonBlockV2 returns the contents of the first ```python block`` ``
// it finds, empty string otherwise.
func ExtractPythonBlockV2(text string) string {
	m := rePythonBlock.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}
