package agent

import (
	"fmt"
	"strings"

	antv1 "anttrader/gen/proto/ant/v1"
)

// pythonSubsetRules is the shared Python subset language rules block.
// Used by both bridgeSystemPrompt and generateSystemPrompt to avoid duplication.
// MUST stay in sync with:
//   - compile_py_mapping.go (Python SDK → VM builtin mapping)
//   - interp/builtin_registry.go (VM implementation source of truth)
const pythonSubsetRules = `## Python Subset Rules
- Class-based: class MyStrategy: with methods on_init, on_bar, on_tick, on_timer, on_deinit
- __init__ params become strategy parameters with type annotations
- All methods must have return type annotations (-> None, -> int, etc.)
- Allowed import: ONLY "from decimal import Decimal"
- NO: list comprehensions, lambda, try/except, with, yield, decorators, async/await
- NO: exec, eval, open, print, len, sorted, sum, enumerate, zip, range (outside for-loops)
- NO: f-strings, walrus operator (:=), global/nonlocal, del, assert, raise
- NO: slicing, tuple unpacking, *args, **kwargs

## SDK API Mapping
### Market Data
- Close[0] → ctx.bars().close(0)
- Open[0] → ctx.bars().open(0)
- High[0] → ctx.bars().high(0)
- Low[0] → ctx.bars().low(0)
- Volume[0] → ctx.bars().volume(0)
- Time[0] → ctx.bars().time(0)
- Bid → ctx.bid()
- Ask → ctx.ask()
- Point → ctx.point()
- Digits → ctx.digits()
- Spread → ctx.spread()
- Symbol() → ctx.symbol()

### Indicators (all map to ctx.indicators().<name>(ctx.symbol(), period, shift))
- iMA → ctx.indicators().ima(ctx.symbol(), period, shift)
- iRSI → ctx.indicators().irsi(ctx.symbol(), period, shift)
- iATR → ctx.indicators().iatr(ctx.symbol(), period, shift)
- iBands / iBollinger → ctx.indicators().ibands(ctx.symbol(), period, shift)
- iMACD → ctx.indicators().imacd(ctx.symbol(), fast, slow, signal, shift)
- iStochastic → ctx.indicators().istochastic(ctx.symbol(), kperiod, dperiod, shift)
- iCCI → ctx.indicators().icci(ctx.symbol(), period, shift)
- iADX → ctx.indicators().iadx(ctx.symbol(), period, shift)
- iMomentum → ctx.indicators().imomentum(ctx.symbol(), period, shift)
- iWPR → ctx.indicators().iwpr(ctx.symbol(), period, shift)
- iMFI → ctx.indicators().imfi(ctx.symbol(), period, shift)
- iOBV → ctx.indicators().iobv(ctx.symbol(), period, shift)
- iSAR → ctx.indicators().isar(ctx.symbol(), step, max, shift)
- iStdDev → ctx.indicators().istddev(ctx.symbol(), period, shift)
- iAlligator → ctx.indicators().ialligator(ctx.symbol(), jaw, teeth, lips, shift)
- iIchimoku → ctx.indicators().iichimoku(ctx.symbol(), tenkan, kijun, senkou, shift)
- iEnvelopes → ctx.indicators().ienvelopes(ctx.symbol(), period, deviation, shift)
- iDeMarker → ctx.indicators().idemarker(ctx.symbol(), period, shift)
- iOsMA → ctx.indicators().iosma(ctx.symbol(), fast, slow, signal, shift)
- iRVI → ctx.indicators().irvi(ctx.symbol(), period, shift)
- iForce → ctx.indicators().iforce(ctx.symbol(), period, shift)
- iFractals → ctx.indicators().ifractals(ctx.symbol(), shift)
- iGator → ctx.indicators().igator(ctx.symbol(), jaw, teeth, lips, shift)
- iAC → ctx.indicators().iac(shift)
- iAD → ctx.indicators().iad(shift)
- iAO → ctx.indicators().iao(shift)
- iBearsPower → ctx.indicators().ibearspower(ctx.symbol(), period, shift)
- iBullsPower → ctx.indicators().ibullspower(ctx.symbol(), period, shift)
- iBWMFI → ctx.indicators().ibwmfi(ctx.symbol(), period, shift)
### MQL5-only Indicators
- iAMA → ctx.indicators().iama(ctx.symbol(), period, fast, slow, shift)
- iDEMA → ctx.indicators().idema(ctx.symbol(), period, shift)
- iTEMA → ctx.indicators().itema(ctx.symbol(), period, shift)
- iFrAMA → ctx.indicators().iframa(ctx.symbol(), period, shift)
- iVIDyA → ctx.indicators().ividya(ctx.symbol(), period, cmo_period, shift)
- iTriX → ctx.indicators().itrix(ctx.symbol(), period, shift)
- iADXWilder → ctx.indicators().iadxwilder(ctx.symbol(), period, shift)
- iChaikin → ctx.indicators().ichaikin(ctx.symbol(), fast, slow, shift)
- iVolumes → ctx.indicators().ivolumes(ctx.symbol(), shift)

### Account
- AccountBalance() → ctx.account().balance()
- AccountEquity() → ctx.account().equity()
- AccountMargin() → ctx.account().margin()
- AccountFreeMargin() → ctx.account().free_margin()
- AccountProfit() → ctx.account().profit()
- AccountLeverage() → ctx.account().leverage()`

// writeProfileToPrompt writes a StrategyProfile as context to a prompt builder.
// Shared by buildGeneratePrompt, buildGenerateRetryPrompt, buildBridgeRetryPrompt,
// buildBridgeUserPrompt, and buildAnalysisUserPrompt.
func writeProfileToPrompt(sb *strings.Builder, profile *antv1.StrategyProfile, header string) {
	if profile == nil {
		return
	}
	sb.WriteString(header)
	sb.WriteString(fmt.Sprintf("Type: %s\n", profile.StrategyType))
	sb.WriteString(fmt.Sprintf("Description: %s\n", profile.Description))
	if len(profile.IndicatorsUsed) > 0 {
		sb.WriteString(fmt.Sprintf("Indicators: %s\n", strings.Join(profile.IndicatorsUsed, ", ")))
	}
	sb.WriteString(fmt.Sprintf("Entry: %s\n", profile.EntryLogic))
	sb.WriteString(fmt.Sprintf("Exit: %s\n", profile.ExitLogic))
	sb.WriteString(fmt.Sprintf("Risk: %s\n", profile.RiskManagement))
}

// writeRequestContext writes symbol, timeframe, and params from a generate request.
// Shared by buildGeneratePrompt, buildGenerateRetryPrompt, and buildProfileFromNLPrompt.
func writeRequestContext(sb *strings.Builder, msg *antv1.AgentGenerateStrategyRequest) {
	if msg.Symbol != "" {
		sb.WriteString(fmt.Sprintf("## Trading Symbol\n%s\n", msg.Symbol))
	}
	if msg.Timeframe != "" {
		sb.WriteString(fmt.Sprintf("## Timeframe\n%s\n", msg.Timeframe))
	}
	if len(msg.Params) > 0 {
		sb.WriteString("## Parameter Overrides\n")
		for k, v := range msg.Params {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
	}
}
