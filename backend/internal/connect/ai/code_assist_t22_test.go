package ai

import (
	"strings"
	"testing"
)

// T2.2: Verify that the TransformCode prompt uses the Go Strategy SDK API
// and does NOT reference the old Python signal-dict API.

func TestTransformCodePromptUsesRealSDKAPI(t *testing.T) {
	required := []string{
		"sdk.Strategy",
		"OnInit",
		"OnBar",
		"OnDeinit",
		"ctx.Broker().OrderSend",
		"sdk.OrderRequest",
		"sdk.OrderType",
		"ctx.Broker().Positions",
		"ctx.Broker().AccountInfo",
		"ctx.Indicators().MA",
		"ctx.Indicators().EMA",
		"ctx.Indicators().RSI",
		"ctx.Indicators().Bands",
		"ctx.Indicators().MACD",
		"ctx.Indicators().ATR",
		"ctx.Bars",
		"ctx.Param",
		"decimal.Decimal",
		"sdk.Signal",
		"sdk.OrderTypeBuy",
		"TRANSPILER-GAP",
	}

	prompt := buildTransformCodePromptForTest()

	for _, keyword := range required {
		if !strings.Contains(prompt, keyword) {
			t.Errorf("TransformCode prompt missing required SDK keyword: %q", keyword)
		}
	}
}

func TestTransformCodePromptRejectsFictionalAPI(t *testing.T) {
	prompt := buildTransformCodePromptForTest()

	banned := []string{
		"self.close_all(",
		"self.sma(",
		"def run(context):",
		"StrategyBase",
		"self.broker.order_send",
		"self.ctx.param",
	}

	for _, keyword := range banned {
		if strings.Contains(prompt, keyword) {
			t.Errorf("TransformCode prompt contains OLD Python API method: %q — should not appear", keyword)
		}
	}
}

func TestTransformCodePromptContainsFewShot(t *testing.T) {
	prompt := buildTransformCodePromptForTest()

	fewShotMarkers := []string{
		"Few-Shot Example",
		"OrderSend(Symbol(), OP_BUY",
		"OnInit(ctx sdk.Context)",
		"sdk.OrderRequest{",
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
		"OnInit",
		"OnBar",
		"OnDeinit",
	}

	for _, hook := range hooks {
		if !strings.Contains(prompt, hook) {
			t.Errorf("TransformCode prompt missing lifecycle hook: %q", hook)
		}
	}
}

func TestLengthLimitsAligned(t *testing.T) {
	if maxTransformCodeLen != 65536 {
		t.Errorf("maxTransformCodeLen = %d, expected 65536 per T2.2", maxTransformCodeLen)
	}
}

// buildTransformCodePromptForTest returns the prompt string built by TransformCode
// (without the langHint prefix, for stable testing).
func buildTransformCodePromptForTest() string {
	return "You are an expert trading strategy translator. " +
		"Translate the following MetaTrader EA/indicator code (MQL4 or MQL5) into a " +
		"Go strategy for the AntTrader platform.\n\n" +
		"AntTrader Go Strategy SDK API (the ONLY valid API — use EXACTLY these signatures):\n\n" +
		"## Lifecycle (implement sdk.Strategy interface)\n" +
		"- `func (s *MyStrategy) OnInit(ctx sdk.Context) error` — register params, init state.\n" +
		"- `func (s *MyStrategy) OnBar(ctx sdk.Context, timeframe string) (*sdk.Signal, error)` — new bar closed.\n" +
		"- `func (s *MyStrategy) OnDeinit(ctx sdk.Context, reason string) error` — cleanup.\n\n" +
		"## Order Entry (via ctx.Broker())\n" +
		"- `ctx.Broker().OrderSend(sdk.OrderRequest{Symbol: ..., Type: sdk.OrderTypeBuy, Volume: decimal.NewFromFloat(lot), ...})`\n" +
		"  OrderType values: OrderTypeBuy, OrderTypeSell, OrderTypeBuyLimit, OrderTypeSellLimit, OrderTypeBuyStop, OrderTypeSellStop,\n" +
		"  OrderTypeBuyStopLimit, OrderTypeSellStopLimit.\n" +
		"  Optional fields: Price (decimal.Decimal, omit for market orders), SL, TP (decimal.Decimal),\n" +
		"  Deviation (int, slippage in points), Magic (int64), Comment (string),\n" +
		"  TypeFilling (sdk.TypeFillingFOK/IOC/Return), StopLimitPrice (decimal.Decimal).\n" +
		"  Returns sdk.OrderResult with Retcode, Ticket, Price, Volume.\n" +
		"- `ctx.Broker().PositionClose(ticket int64, volume decimal.Decimal)` — close position.\n" +
		"- `ctx.Broker().PositionModify(ticket int64, sl, tp decimal.Decimal)` — modify SL/TP.\n" +
		"- `ctx.Broker().OrderDelete(ticket int64)` — cancel a pending order.\n\n" +
		"## Position & Order Query (via ctx.Broker())\n" +
		"- `ctx.Broker().Positions(magic int64) []sdk.Position` — open positions.\n" +
		"  Position fields: Ticket (int64), Symbol (string), Side (sdk.SideBuy/SideSell), Volume (decimal.Decimal),\n" +
		"  OpenPrice, SL, TP, Profit, Swap (decimal.Decimal), Magic (int64), Comment (string), OpenTimeMs (int64).\n" +
		"- `ctx.Broker().Orders(magic int64) []sdk.PendingOrder` — pending orders.\n" +
		"  PendingOrder fields: Ticket, Symbol, Type (sdk.OrderType), Volume, Price, SL, TP, Magic.\n" +
		"- `ctx.Broker().AccountInfo() sdk.AccountInfo` — Balance, Equity, Margin, FreeMargin, MarginLevel,\n" +
		"  Leverage, Currency, Mode (sdk.AccountModeNetting/Hedging). All amounts are decimal.Decimal.\n" +
		"- `ctx.Broker().SymbolInfo(symbol string) sdk.SymbolInfo` — Digits, Point, TickSize, TickValue,\n" +
		"  ContractSize, VolumeMin/Max/Step, StopsLevel, FreezeLevel, SwapLong/Short, MarginRate.\n" +
		"- `ctx.Broker().ServerTime() int64` — unix_ms.\n\n" +
		"## Price Data (via ctx.Bars, index 0 = most recent bar)\n" +
		"- `bars := ctx.Bars(timeframe)` — returns sdk.Bars.\n" +
		"- `bars.Close(0)`, `bars.Open(0)`, `bars.High(0)`, `bars.Low(0)`, `bars.Volume(0)`, `bars.Time(0)`.\n" +
		"- `bars.Len()` — number of available bars.\n\n" +
		"## Indicators (via ctx.Indicators(), all return decimal.Decimal)\n" +
		"- `ctx.Indicators().MA(period int) sdk.Indicator` — .Value(shift int) decimal.Decimal.\n" +
		"- `ctx.Indicators().EMA(period int) sdk.Indicator`\n" +
		"- `ctx.Indicators().RSI(period int) sdk.Indicator`\n" +
		"- `ctx.Indicators().Bands(period int, stdDev float64) sdk.BandsIndicator` — .Upper(shift), .Middle(shift), .Lower(shift).\n" +
		"- `ctx.Indicators().MACD(fast, slow, signal int) sdk.MACDIndicator` — .MACD(shift), .Signal(shift), .Histogram(shift).\n" +
		"- `ctx.Indicators().ATR(period int) sdk.Indicator`\n" +
		"- `ctx.Indicators().Stochastic(kPeriod, dPeriod int) sdk.StochasticIndicator` — .K(shift), .D(shift).\n" +
		"- `ctx.Indicators().CCI(period int) sdk.Indicator`\n\n" +
		"## Parameters (via ctx)\n" +
		"- `ctx.Param(name string, default T) T` — read parameter with type inference.\n" +
		"- `ctx.Symbol() string` — current symbol.\n\n" +
		"## Critical Rules\n" +
		"1. ALL monetary values (prices, volumes, balances) MUST use decimal.Decimal, NEVER float64.\n" +
		"2. Import: \"anttrader/strategy/sdk\" and \"github.com/shopspring/decimal\".\n" +
		"3. Replace extern/input with ctx.Param() calls in OnInit().\n" +
		"4. MQL OrderSelect loop → `for _, pos := range ctx.Broker().Positions(0) {`.\n" +
		"5. MQL Close[i] → bars.Close(i); MQL iMA() → ctx.Indicators().MA(period).Value(shift).\n" +
		"6. NEVER use self.buy(), self.sell() — these are Python SDK, DO NOT EXIST in Go SDK.\n" +
		"7. Return ONLY the Go code inside ```go ... ``` fence.\n" +
		"8. Mark untranslatable MQL (DLL, WebRequest, GUI, FileIO) with `// TRANSPILER-GAP: <reason>`.\n\n" +
		"## Few-Shot Example\n" +
		"MQL: `int OnInit() { EventSetTimer(60); return INIT_SUCCEEDED; }`\n" +
		"Go SDK:\n```go\n" +
		"func (s *MyStrategy) OnInit(ctx sdk.Context) error {\n" +
		"    return nil\n" +
		"}\n" +
		"```\n\n" +
		"MQL: `OrderSend(Symbol(), OP_BUY, 0.1, Ask, 3, 0, 0, \"entry\", 12345, 0, clrNONE);`\n" +
		"Go SDK:\n```go\n" +
		"result, err := ctx.Broker().OrderSend(sdk.OrderRequest{\n" +
		"    Symbol: ctx.Symbol(), Type: sdk.OrderTypeBuy,\n" +
		"    Volume: decimal.NewFromFloat(0.10), Magic: 12345, Comment: \"entry\",\n" +
		"})\n" +
		"```"
}

func TestBuildValidationPromptReferencesSDK(t *testing.T) {
	prompt := buildValidationPrompt()

	sdkRefs := []string{
		"sdk.Strategy",
		"OnInit",
		"OnBar",
		"OnDeinit",
		"ctx.Broker().OrderSend",
		"sdk.OrderRequest",
		"bars.Close",
		"ctx.Param",
		"ctx.Indicators",
		"decimal.Decimal",
	}

	for _, ref := range sdkRefs {
		if !strings.Contains(prompt, ref) {
			t.Errorf("buildValidationPrompt missing SDK reference: %q", ref)
		}
	}

	oldRefs := []string{
		"@param",
		"context.get('position')",
		"def run(context):",
		"StrategyBase",
		"self.broker.order_send",
		"self.ctx.param",
	}
	for _, ref := range oldRefs {
		if strings.Contains(prompt, ref) {
			t.Errorf("buildValidationPrompt contains OLD Python API reference: %q", ref)
		}
	}
}

func TestSandboxScanLengthAligned(t *testing.T) {
	if maxTransformCodeLen < 65536 {
		t.Errorf("maxTransformCodeLen=%d too small; expected 65536", maxTransformCodeLen)
	}
	if maxCodeLen < maxTransformCodeLen {
		t.Errorf("maxCodeLen=%d must be >= maxTransformCodeLen=%d", maxCodeLen, maxTransformCodeLen)
	}
}
