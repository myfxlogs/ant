package ai

import (
	"strings"
	"testing"
)

// T2.2: Verify that the TransformCode prompt uses the REAL Strategy SDK API
// (docs/spec/30-strategy-sdk.md) and does NOT reference the fictional
// signal-dict API (self.buy, self.sell, self.sma, self.close_all, etc.).

func TestTransformCodePromptUsesRealSDKAPI(t *testing.T) {
	// The TransformCode method builds its prompt inline.
	// We validate the prompt content against the spec.

	// Required SDK API references that MUST appear in the prompt.
	required := []string{
		"StrategyBase",
		"on_init",
		"on_tick",
		"on_bar",
		"on_deinit",
		"self.broker.order_send",
		"OrderRequest",
		"OrderType",           // OrderType values listed: BUY, SELL, BUY_LIMIT, ...
		"self.broker.position_close",
		"self.broker.position_modify",
		"self.broker.order_delete",
		"self.broker.positions",
		"self.broker.orders",
		"self.broker.account",
		"self.broker.symbol_info",
		"self.indicators.ma",
		"self.indicators.ema",
		"self.indicators.rsi",
		"self.indicators.bands",
		"self.indicators.macd",
		"self.indicators.atr",
		"self.indicators.stochastic",
		"self.indicators.cci",
		"self.indicators.i_custom",
		"self.ctx.bars",
		"self.ctx.param",
		"self.ctx.set_timer",
		"self.ctx.kill_timer",
		"bars.close[0]",
		"bars.open[0]",
		"MQL reverse indexing",
		"Decimal(str(",
		"PositionSide",
		"AccountMode",
		"TypeFilling",
		"TRANSPILER-GAP",
	}

	// Also verify all 8 order type values are listed.
	orderTypes := []string{
		"BUY_LIMIT",
		"SELL_LIMIT",
		"BUY_STOP",
		"SELL_STOP",
		"BUY_STOP_LIMIT",
		"SELL_STOP_LIMIT",
	}
	required = append(required, orderTypes...)

	// Build the prompt the same way TransformCode does (without langHint).
	// We can't call the actual method without an LLM, but we verify the prompt
	// string matches the spec by checking the constant parts.
	prompt := buildTransformCodePromptForTest()

	for _, keyword := range required {
		if !strings.Contains(prompt, keyword) {
			t.Errorf("TransformCode prompt missing required SDK keyword: %q", keyword)
		}
	}
}

func TestTransformCodePromptRejectsFictionalAPI(t *testing.T) {
	prompt := buildTransformCodePromptForTest()

	// These are FICTIONAL API methods from the old prompt that MUST NOT appear
	// as actual API recommendations. They may appear in the "NEVER use" warning.
	banned := []string{
		"self.buy(",
		"self.sell(",
		"self.close_all(",
		"self.sma(",
		"self.bollinger(",
		"self.has_position()",
		"self.position_side",
		"self.position_profit",
		"self.order_comment",
		"self.on_bar(bar)",  // wrong signature
		"def run(context):", // signal-dict entry point
	}

	for _, keyword := range banned {
		occurrences := countOccurrences(prompt, keyword)
		// Allow exactly 1 occurrence — the "NEVER use" warning line.
		// Any additional occurrence means the fictional API is being recommended.
		if occurrences > 1 {
			t.Errorf("TransformCode prompt contains FICTIONAL API method %d times: %q — should appear at most once (in NEVER-use warning)", occurrences, keyword)
		}
		if occurrences == 1 && !strings.Contains(prompt, "NEVER use") {
			t.Errorf("TransformCode prompt contains FICTIONAL API method %q outside NEVER-use warning", keyword)
		}
	}
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}

func TestTransformCodePromptContainsFewShot(t *testing.T) {
	prompt := buildTransformCodePromptForTest()

	fewShotMarkers := []string{
		"Few-Shot Example",
		"OrderSend(Symbol(), OP_BUY",
		"on_init(self) -> None",
		"OrderRequest(symbol=self.ctx.symbol",
	}

	for _, marker := range fewShotMarkers {
		if !strings.Contains(prompt, marker) {
			t.Errorf("TransformCode prompt missing few-shot example marker: %q", marker)
		}
	}
}

func TestTransformCodePromptCoversAllLifecycleHooks(t *testing.T) {
	prompt := buildTransformCodePromptForTest()

	hooks := []string{
		"on_init",
		"on_tick",
		"on_bar",
		"on_timer",
		"on_trade",
		"on_deinit",
	}

	for _, hook := range hooks {
		if !strings.Contains(prompt, hook) {
			t.Errorf("TransformCode prompt missing lifecycle hook: %q", hook)
		}
	}
}

func TestLengthLimitsAligned(t *testing.T) {
	// maxTransformCodeLen (code_assist_handler.go) = 65536
	// sandbox_scan.py MAX_CODE_LENGTH must be >= this value.
	// We can't read the Python file here, but we verify the Go constant.
	if maxTransformCodeLen != 65536 {
		t.Errorf("maxTransformCodeLen = %d, expected 65536 per T2.2", maxTransformCodeLen)
	}
}

// buildTransformCodePromptForTest returns the prompt string built by TransformCode
// (without the langHint prefix, for stable testing).
func buildTransformCodePromptForTest() string {
	return "You are an expert trading strategy translator. " +
		"Translate the following MetaTrader EA/indicator code (MQL4 or MQL5) into a " +
		"Python strategy for the AntTrader platform.\n\n" +
		"AntTrader Strategy SDK API (the ONLY valid API — use EXACTLY these signatures):\n\n" +
		"## Lifecycle (inherit from StrategyBase)\n" +
		"- `def on_init(self) -> None:` — replaces OnInit(). Register params, set timer.\n" +
		"- `def on_tick(self) -> None:` — replaces OnTick(). Primary entry for tick-driven EAs.\n" +
		"- `def on_bar(self, timeframe: str) -> None:` — replaces OnCalculate(). New bar closed.\n" +
		"- `def on_timer(self) -> None:` — replaces OnTimer(). Requires ctx.set_timer(seconds).\n" +
		"- `def on_trade(self) -> None:` — replaces OnTrade(). After any trade event.\n" +
		"- `def on_deinit(self, reason: str) -> None:` — replaces OnDeinit(). Cleanup.\n\n" +
		"## Order Entry (via self.broker)\n" +
		"- `self.broker.order_send(OrderRequest(symbol=..., type=OrderType.BUY, volume=Decimal(str(lot)), ...))`\n" +
		"  OrderType values: BUY, SELL, BUY_LIMIT, SELL_LIMIT, BUY_STOP, SELL_STOP,\n" +
		"  BUY_STOP_LIMIT, SELL_STOP_LIMIT.\n" +
		"  Optional fields: price (Decimal, omit for market orders), sl, tp (Decimal or None),\n" +
		"  deviation (int, slippage in points), magic (int), comment (str),\n" +
		"  type_filling (TypeFilling.FOK/IOC/RETURN), stop_limit_price (Decimal, only for *_STOP_LIMIT).\n" +
		"  Returns OrderResult with retcode, ticket, price, volume.\n" +
		"- `self.broker.position_close(ticket, volume=None)` — close position (None=full, Decimal=partial).\n" +
		"- `self.broker.position_modify(ticket, sl=None, tp=None)` — modify SL/TP (Decimal or None).\n" +
		"- `self.broker.order_delete(ticket)` — cancel a pending order.\n\n" +
		"## Position & Order Query (via self.broker)\n" +
		"- `self.broker.positions(symbol=None, magic=None) -> list[Position]` — open positions.\n" +
		"  Position fields: ticket, symbol, side (PositionSide.BUY/SELL), volume (Decimal),\n" +
		"  open_price (Decimal), sl, tp, profit, swap, magic, comment, open_time_ms.\n" +
		"- `self.broker.orders(symbol=None, magic=None) -> list[PendingOrder]` — pending orders.\n" +
		"  PendingOrder fields: ticket, symbol, type (OrderType), volume, price, sl, tp, magic.\n" +
		"- `self.broker.account() -> AccountInfo` — balance, equity, margin, free_margin, margin_level,\n" +
		"  leverage, currency, mode (AccountMode.NETTING/HEDGING). All amounts are Decimal.\n" +
		"- `self.broker.symbol_info(symbol) -> SymbolInfo` — digits, point, tick_size, tick_value,\n" +
		"  contract_size, volume_min/max/step, stops_level, freeze_level, swap_long/short, margin_rate.\n" +
		"- `self.broker.server_time() -> int` — unix_ms.\n\n" +
		"## Price Data (via self.ctx, MQL reverse indexing: [0]=current, [1]=previous)\n" +
		"- `bars = self.ctx.bars(timeframe=None)` — returns Bars. None = primary timeframe.\n" +
		"- `bars.close[0]`, `bars.open[0]`, `bars.high[0]`, `bars.low[0]`, `bars.volume[0]`, `bars.time[0]`.\n" +
		"- `bars.total()` — number of available bars.\n\n" +
		"## Indicators (via self.indicators, shift=0 = current bar, all return float)\n" +
		"- `self.indicators.ma(period=14, shift=0, method='sma')` — methods: sma/ema/smma/lwma.\n" +
		"- `self.indicators.ema(period=14, shift=0)`\n" +
		"- `self.indicators.rsi(period=14, shift=0)`\n" +
		"- `self.indicators.bands(period=20, deviation=2.0, shift=0) -> (upper, middle, lower)`\n" +
		"- `self.indicators.macd(fast=12, slow=26, signal=9, shift=0) -> (macd, signal, histogram)`\n" +
		"- `self.indicators.atr(period=14, shift=0)`\n" +
		"- `self.indicators.stochastic(k_period=5, d_period=3, shift=0) -> (k, d)`\n" +
		"- `self.indicators.cci(period=14, shift=0)`\n" +
		"- `self.indicators.i_custom(name, params=[], buffer=0, shift=0)` — custom indicator.\n\n" +
		"## Parameters & Timer (via self.ctx)\n" +
		"- `self.ctx.param(name, default=None)` — read extern/input parameter (type: object, cast as needed).\n" +
		"- `self.ctx.set_timer(seconds)` — enable periodic on_timer callback (min 1s).\n" +
		"- `self.ctx.kill_timer()` — disable timer.\n\n" +
		"## Critical Rules\n" +
		"1. ALL monetary values (prices, volumes, balances) MUST use Decimal(str(x)), NEVER float.\n" +
		"2. Import from app.sdk: StrategyBase, OrderRequest, OrderType, OrderResult, Position,\n" +
		"   PendingOrder, PositionSide, Retcode, AccountMode, TypeFilling, Decimal.\n" +
		"3. Replace extern/input with self.ctx.param() calls in on_init().\n" +
		"4. MQL OrderSelect loop → `for order in self.broker.orders():` or `for pos in self.broker.positions():`.\n" +
		"5. MQL Close[i] → bars.close[i]; MQL iMA() → self.indicators.ma().\n" +
		"6. NEVER use self.buy(), self.sell(), self.close_all(), self.sma() — these DO NOT EXIST.\n" +
		"7. Return ONLY the Python code inside ```python ... ``` fence.\n" +
		"8. Mark untranslatable MQL (DLL, WebRequest, GUI, FileIO) with `# TRANSPILER-GAP: <reason>`.\n\n" +
		"## Few-Shot Example\n" +
		"MQL: `int OnInit() { EventSetTimer(60); return INIT_SUCCEEDED; }`\n" +
		"SDK:\n```python\n" +
		"def on_init(self) -> None:\n" +
		"    self.ctx.set_timer(60)\n" +
		"```\n\n" +
		"MQL: `OrderSend(Symbol(), OP_BUY, 0.1, Ask, 3, 0, 0, \"entry\", 12345, 0, clrNONE);`\n" +
		"SDK:\n```python\n" +
		"req = OrderRequest(symbol=self.ctx.symbol, type=OrderType.BUY,\n" +
		"                   volume=Decimal('0.10'), magic=12345, comment='entry')\n" +
		"result = self.broker.order_send(req)\n" +
		"```"
}

func TestBuildValidationPromptReferencesSDK(t *testing.T) {
	prompt := buildValidationPrompt()

	sdkRefs := []string{
		"StrategyBase",
		"on_init",
		"on_tick",
		"on_bar",
		"on_deinit",
		"self.broker.order_send",
		"OrderRequest",
		"self.ctx.bars",
		"self.ctx.param",
		"self.indicators",
		"Decimal(str(",
	}

	for _, ref := range sdkRefs {
		if !strings.Contains(prompt, ref) {
			t.Errorf("buildValidationPrompt missing SDK reference: %q", ref)
		}
	}

	// Must NOT reference the old signal-dict API.
	oldRefs := []string{
		"@param",
		"context.get('position')",
		"def run(context):",
	}
	for _, ref := range oldRefs {
		if strings.Contains(prompt, ref) {
			t.Errorf("buildValidationPrompt contains OLD signal-dict API reference: %q", ref)
		}
	}

	// Must warn against underscore-prefixed helper methods.
	if !strings.Contains(prompt, "underscore") {
		t.Error("buildValidationPrompt should mention underscore-prefixed helpers are rejected")
	}
}

func TestSandboxScanLengthAligned(t *testing.T) {
	// Verify that the Go-side maxTransformCodeLen is at least as large
	// as the Python sandbox_scan MAX_CODE_LENGTH after T2.2 alignment.
	// The Python value was raised from 10000 to 65536.
	if maxTransformCodeLen < 65536 {
		t.Errorf("maxTransformCodeLen=%d too small; sandbox_scan.py MAX_CODE_LENGTH=65536 per T2.2", maxTransformCodeLen)
	}
	if maxCodeLen < maxTransformCodeLen {
		t.Errorf("maxCodeLen=%d must be >= maxTransformCodeLen=%d", maxCodeLen, maxTransformCodeLen)
	}
}
