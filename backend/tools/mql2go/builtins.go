package mql2go

import (
	"anttrader/tools/mql2go/interp"
)

// BuiltinFunc is a VM builtin function handler.
// It takes the VM and a slice of arguments, returns a result Value and error.
type BuiltinFunc func(vm *VM, args []interp.Value) (interp.Value, error)

// builtinEntry maps a builtin name to its handler function.
type builtinEntry struct {
	name string
	fn   BuiltinFunc
}

// builtinRegistry defines all available builtin functions.
// The order determines the BuiltinID (index in this slice).
var builtinRegistry = []builtinEntry{
	// Math functions
	{"MathAbs", nil},
	{"MathMax", nil},
	{"MathMin", nil},
	{"MathSqrt", nil},
	{"MathPow", nil},
	{"MathLog", nil},
	{"MathRound", nil},
	{"MathFloor", nil},
	{"MathCeil", nil},
	{"MathExp", nil},

	// Platform functions
	{"Print", nil},
	{"Alert", nil},
	{"Comment", nil},

	// Market data
	{"Close", nil},
	{"Open", nil},
	{"High", nil},
	{"Low", nil},
	{"Volume", nil},
	{"Time", nil},
	{"iClose", nil},
	{"iOpen", nil},
	{"iHigh", nil},
	{"iLow", nil},
	{"iVolume", nil},
	{"iTime", nil},

	// Price data
	{"Bid", nil},
	{"Ask", nil},
	{"Point", nil},
	{"Digits", nil},
	{"Symbol", nil},
	{"Period", nil},

	// Indicators (MQL4 + shared)
	{"iMA", nil},
	{"iRSI", nil},
	{"iATR", nil},
	{"iBands", nil},
	{"iBollinger", nil},
	{"iMACD", nil},
	{"iStochastic", nil},
	{"iCCI", nil},
	{"iADX", nil},
	{"iMomentum", nil},
	{"iWPR", nil},
	{"iMFI", nil},
	{"iOBV", nil},
	{"iSAR", nil},
	{"iStdDev", nil},
	{"iAlligator", nil},
	{"iIchimoku", nil},
	{"iEnvelopes", nil},
	{"iDeMarker", nil},
	{"iOsMA", nil},
	{"iRVI", nil},
	{"iForce", nil},
	{"iFractals", nil},
	{"iGator", nil},
	{"iAC", nil},
	{"iAD", nil},
	{"iAO", nil},
	{"iBearsPower", nil},
	{"iBullsPower", nil},
	{"iBWMFI", nil},

	// MQL5-only indicators
	{"iAMA", nil},
	{"iDEMA", nil},
	{"iTEMA", nil},
	{"iFrAMA", nil},
	{"iVIDyA", nil},
	{"iTriX", nil},
	{"iADXWilder", nil},
	{"iChaikin", nil},
	{"iVolumes", nil},

	// *OnArray variants (compute on user-provided arrays)
	{"iMAOnArray", nil},
	{"iRSIOnArray", nil},
	{"iATROnArray", nil},
	{"iBandsOnArray", nil},
	{"iStdDevOnArray", nil},
	{"iMomentumOnArray", nil},
	{"iCCIOnArray", nil},
	{"iMACDOnArray", nil},

	// Account functions
	{"AccountBalance", nil},
	{"AccountEquity", nil},
	{"AccountFreeMargin", nil},
	{"AccountMargin", nil},
	{"AccountLeverage", nil},
	{"AccountProfit", nil},
	{"AccountCurrency", nil},
	{"AccountCompany", nil},
	{"AccountNumber", nil},
	{"AccountStopoutLevel", nil},
	{"AccountName", nil},
	{"AccountServer", nil},
	{"AccountFreeMarginCheck", nil},
	{"AccountFreeMarginMode", nil},

	// Symbol info
	{"SymbolInfoDouble", nil},
	{"SymbolInfoInteger", nil},
	{"SymbolInfoString", nil},
	{"MarketInfo", nil},

	// String functions
	{"StringFormat", nil},
	{"StringFind", nil},
	{"StringSubstr", nil},
	{"StringLen", nil},
	{"StringReplace", nil},
	{"StringSplit", nil},
	{"StringTrimLeft", nil},
	{"StringTrimRight", nil},
	{"StringConcatenate", nil},
	{"StringToDouble", nil},

	// MQL4 trade functions
	{"OrderSend", nil},
	{"OrderClose", nil},
	{"OrderModify", nil},
	{"OrderDelete", nil},
	{"OrderCloseBy", nil},
	{"OrderSelect", nil},
	{"OrdersTotal", nil},
	{"OrderStopLoss", nil},
	{"OrderTakeProfit", nil},
	{"OrderTicket", nil},
	{"OrderType", nil},
	{"OrderLots", nil},
	{"OrderSymbol", nil},
	{"OrderOpenPrice", nil},
	{"OrderClosePrice", nil},
	{"OrderProfit", nil},
	{"OrderCommission", nil},
	{"OrderSwap", nil},
	{"OrderMagicNumber", nil},
	{"OrderComment", nil},
	{"OrderExpiration", nil},
	{"OrderOpenTime", nil},
	{"OrderCloseTime", nil},
	{"OrderPrint", nil},

	// MQL5 position functions
	{"PositionsTotal", nil},
	{"PositionGetTicket", nil},
	{"PositionGetDouble", nil},
	{"PositionGetInteger", nil},
	{"PositionGetString", nil},
	{"PositionGetSymbol", nil},
	{"PositionSelectByTicket", nil},

	// MQL5 CTrade methods (registered as method builtins)
	{"CTrade.Buy", nil},
	{"CTrade.Sell", nil},
	{"CTrade.BuyLimit", nil},
	{"CTrade.SellLimit", nil},
	{"CTrade.BuyStop", nil},
	{"CTrade.SellStop", nil},
	{"CTrade.PositionClose", nil},
	{"CTrade.PositionClosePartial", nil},
	{"CTrade.PositionCloseBy", nil},
	{"CTrade.PositionModify", nil},
	{"CTrade.OrderDelete", nil},
	{"CTrade.SetExpertMagicNumber", nil},
	{"CTrade.SetDeviationInPoints", nil},

	// Utility
	{"NormalizeDouble", nil},
	{"DoubleToString", nil},
	{"DoubleToStr", nil},
	{"IntegerToString", nil},
	{"StringToInteger", nil},
	{"StrToDouble", nil},
	{"StrToInteger", nil},
	{"TimeCurrent", nil},
	{"TimeLocal", nil},
	{"Sleep", nil},
	{"EventSetTimer", nil},
	{"EventKillTimer", nil},
	{"EventSetMillisecondTimer", nil},

	// Datetime functions
	{"Day", nil},
	{"DayOfWeek", nil},
	{"Hour", nil},
	{"Minute", nil},
	{"Year", nil},
	{"Month", nil},
	{"StrToTime", nil},
	{"TimeToStr", nil},

	// Platform functions
	{"RefreshRates", nil},
	{"GetLastError", nil},
	{"ResetLastError", nil},
	{"ExpertRemove", nil},
	{"IsTesting", nil},
	{"IsOptimization", nil},
	{"IsVisualMode", nil},

	// Array functions
	{"ArraySize", nil},
	{"ArrayResize", nil},
	{"ArrayCopy", nil},
	{"ArraySetAsSeries", nil},
	{"ArrayMaximum", nil},
	{"ArrayMinimum", nil},
	{"ArraySort", nil},
	{"ArrayFree", nil},
	{"ArrayInitialize", nil},
	{"ArrayFill", nil},
	{"ArrayRange", nil},
	{"ArrayIsSeries", nil},

	// Object/Chart (permanent blind spots — skip)
	{"ObjectCreate", nil},
	{"ObjectDelete", nil},
	{"ObjectSet", nil},
	{"ObjectGet", nil},
	{"ObjectSetText", nil},
	{"ObjectsTotal", nil},
	{"ObjectFind", nil},
	{"ObjectName", nil},
	{"ObjectGetType", nil},
	{"ChartApplyTemplate", nil},

	// File operations (permanent blind spots — skip)
	{"FileOpen", nil},
	{"FileClose", nil},
	{"FileWrite", nil},
	{"FileRead", nil},
	{"FileDelete", nil},
	{"FileIsEnding", nil},
	{"FileSeek", nil},
	{"FileTell", nil},
	{"FileFlush", nil},
	{"FileSize", nil},

	// ── Math functions (MQL4/MQL5 complete list) ──────────────────────
	{"MathCos", nil},
	{"MathSin", nil},
	{"MathTan", nil},
	{"MathArccos", nil},
	{"MathArcsin", nil},
	{"MathArctan", nil},
	{"MathLog10", nil},
	{"MathMod", nil},
	{"MathRand", nil},
	{"MathSrand", nil},
	{"MathIsValidNumber", nil},
	// MQL4 lowercase math aliases
	{"ceil", nil},
	{"floor", nil},
	{"cos", nil},
	{"sin", nil},
	{"tan", nil},
	{"exp", nil},
	{"fabs", nil},
	{"fmax", nil},
	{"fmin", nil},
	{"fmod", nil},
	{"log", nil},
	{"log10", nil},
	{"pow", nil},
	{"round", nil},
	{"rand", nil},
	{"srand", nil},
	{"sqrt", nil},

	// ── String functions (MQL4/MQL5 complete list) ────────────────────
	{"StringAdd", nil},
	{"StringCompare", nil},
	{"StringGetCharacter", nil},
	{"StringSetCharacter", nil},
	{"StringToLower", nil},
	{"StringToUpper", nil},
	{"StringBufferLen", nil},
	{"StringInit", nil},
	{"StringFill", nil},

	// ── Conversion functions (MQL4/MQL5 complete list) ────────────────
	{"CharToString", nil},
	{"CharArrayToString", nil},
	{"ShortToString", nil},
	{"ShortArrayToString", nil},
	{"StringToColor", nil},
	{"StringToCharArray", nil},
	{"StringToShortArray", nil},
	{"EnumToString", nil},
	{"TimeToString", nil},

	// ── Date/Time functions (MQL4/MQL5 complete list) ─────────────────
	{"TimeGMT", nil},
	{"TimeGMTOffset", nil},
	{"TimeDaylightSavings", nil},
	{"TimeTradeServer", nil},
	{"TimeToStruct", nil},
	{"StructToTime", nil},
	{"PeriodSeconds", nil},

	// ── Array functions (additions) ───────────────────────────────────
	{"ArrayBsearch", nil},
	{"ArrayCompare", nil},
	{"ArrayInsert", nil},
	{"ArrayRemove", nil},
	{"ArrayReverse", nil},
	{"ArraySwap", nil},
	{"ArrayPrint", nil},
	{"ArrayGetAsSeries", nil},
	{"ArrayIsDynamic", nil},

	// ── Checkup / Platform functions (MQL4/MQL5 complete list) ────────
	{"IsConnected", nil},
	{"IsDemo", nil},
	{"IsDllsAllowed", nil},
	{"IsExpertEnabled", nil},
	{"IsLibrariesAllowed", nil},
	{"IsTradeAllowed", nil},
	{"IsTradeContextBusy", nil},
	{"IsStopped", nil},
	{"UninitializeReason", nil},
	{"MQLInfoInteger", nil},
	{"MQLInfoString", nil},
	{"TerminalInfoDouble", nil},
	{"TerminalInfoInteger", nil},
	{"TerminalInfoString", nil},
	{"GetTickCount", nil},
	{"GetTickCount64", nil},
	{"GetMicrosecondCount", nil},
	{"SetUserError", nil},
	{"SetReturnError", nil},
	{"CurTime", nil},

	// ── MQL5 timeseries access ────────────────────────────────────────
	{"Bars", nil},
	{"iBarShift", nil},
	{"iHighest", nil},
	{"iLowest", nil},
	{"iTickVolume", nil},
	{"iRealVolume", nil},
	{"iSpread", nil},
	{"CopyRates", nil},
	{"CopyClose", nil},
	{"CopyHigh", nil},
	{"CopyLow", nil},
	{"CopyOpen", nil},
	{"CopyTime", nil},
	{"CopyBuffer", nil},
	{"CopyTickVolume", nil},
	{"CopyRealVolume", nil},
	{"CopySpread", nil},
	{"CopyTicks", nil},
	{"BarsCalculated", nil},
	{"SeriesInfoInteger", nil},

	// ── MQL5 market info additions ────────────────────────────────────
	{"SymbolInfoTick", nil},
	{"SymbolName", nil},
	{"SymbolSelect", nil},
	{"SymbolsTotal", nil},
	{"SymbolInfoMarginRate", nil},
	{"SymbolInfoSessionQuote", nil},
	{"SymbolInfoSessionTrade", nil},
	{"SymbolIsSynchronized", nil},

	// ── MQL5 trade helpers ────────────────────────────────────────────
	{"OrderCalcMargin", nil},
	{"OrderCalcProfit", nil},
	{"OrderCheck", nil},
	{"PositionSelect", nil},

	// ── MQL5 order functions (pending orders) ─────────────────────────
	{"OrderGetTicket", nil},
	{"OrderGetDouble", nil},
	{"OrderGetInteger", nil},
	{"OrderGetString", nil},
	{"OrdersTotalMQL5", nil},

	// ── MQL5 deal/history functions ───────────────────────────────────
	{"HistorySelect", nil},
	{"HistorySelectByPosition", nil},
	{"HistoryDealsTotal", nil},
	{"HistoryDealSelect", nil},
	{"HistoryDealGetTicket", nil},
	{"HistoryDealGetDouble", nil},
	{"HistoryDealGetInteger", nil},
	{"HistoryDealGetString", nil},
	{"HistoryOrdersTotal", nil},
	{"HistoryOrderSelect", nil},
	{"HistoryOrderGetTicket", nil},
	{"HistoryOrderGetDouble", nil},
	{"HistoryOrderGetInteger", nil},
	{"HistoryOrderGetString", nil},

	// ── Account info additions (MQL5) ─────────────────────────────────
	{"AccountInfoDouble", nil},
	{"AccountInfoInteger", nil},
	{"AccountInfoString", nil},
	{"AccountStopoutMode", nil},
	{"AccountCredit", nil},

	// ── Global Variables of the Terminal ──────────────────────────────
	{"GlobalVariableSet", nil},
	{"GlobalVariableGet", nil},
	{"GlobalVariableDel", nil},
	{"GlobalVariableCheck", nil},
	{"GlobalVariableTemp", nil},
	{"GlobalVariableFlush", nil},
	{"GlobalVariablesDeleteAll", nil},
	{"GlobalVariablesTotal", nil},
	{"GlobalVariableName", nil},
}

// registerBuiltins populates the Bytecode's builtin map.
func (c *astCompiler) registerBuiltins() {
	for i, entry := range builtinRegistry {
		c.bc.Builtins[entry.name] = BuiltinID(i)
	}
}

// registerMethodBuiltin registers a method call (obj.method) as a builtin
// and returns its ID. The method name is prefixed with the object type.
// Uses the bytecode's own builtin map for dynamic names; the global registry
// is pre-populated with all known method names so no append is needed.
func (c *astCompiler) registerMethodBuiltin(methodName string, _ int) BuiltinID {
	if bid, ok := c.bc.Builtins[methodName]; ok {
		return bid
	}
	// Unknown method — register as blind spot and return a dummy ID
	c.bc.Coverage.AddBlindSpot("unknown method: " + methodName)
	return 0
}
