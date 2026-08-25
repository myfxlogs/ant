package interp

// Builtin name lists — the data source for api_registry.go, which is the
// single source of truth for implementation status (Implemented/Unsupported).
//
// When adding a new VM builtin, add the name here so api_registry.go picks it up.
// This file is the bridge between VM execution and static analysis (reporting) —
// both must reference the same names to guarantee "preview == execution".

// implementedMarketData — market data functions in the VM (vm_builtin_market.go).
var implementedMarketData = []string{
	"Ask", "ask", "Bid", "bid", "Point", "point", "_Point",
	"Symbol", "symbol", "_Symbol", "Digits", "digits", "_Digits",
	"Period", "_Period", "M1", "M5", "M15", "M30", "H1", "H4", "D1", "W1", "MN1",
	"SymbolInfoDouble", "SymbolInfoInteger", "SymbolInfoString",
	"MarketInfo",
	// Cross-timeframe market data
	"iHigh", "iLow", "iOpen", "iClose", "iTime",
}

// implementedIndicators — case labels in builtin_indicators.go callIndicator switch.
var implementedIndicators = []string{
	"iMA", "iRSI", "iATR", "iMACD", "iBands", "iBollinger",
	"iStochastic", "iCCI", "iADX", "iMFI", "iOBV", "iSAR",
	"iStdDev", "iWPR", "iMomentum",
	// shared MQL4/MQL5 indicators (now fully implemented)
	"iAlligator", "iIchimoku", "iEnvelopes", "iDeMarker", "iOsMA",
	"iRVI", "iForce", "iFractals", "iGator", "iAC", "iAD", "iAO",
	"iBearsPower", "iBullsPower", "iBWMFI",
	// MQL5-only indicators (now fully implemented)
	"iAMA", "iDEMA", "iTEMA", "iFrAMA", "iVIDyA", "iTriX",
	"iADXWilder", "iChaikin", "iVolumes",
	// *OnArray variants (compute on user-provided arrays)
	"iMAOnArray", "iRSIOnArray", "iATROnArray", "iBandsOnArray",
	"iStdDevOnArray", "iMomentumOnArray", "iCCIOnArray", "iMACDOnArray",
}

// implementedMQL4Trade — MQL4 trade functions in the VM (vm_builtin_trade.go).
var implementedMQL4Trade = []string{
	"OrderSend", "OrderClose", "OrderCloseBy", "OrderModify", "OrderDelete",
	"OrdersTotal", "OrdersHistoryTotal", "OrderSelect", "OrderTicket", "OrderSymbol",
	"OrderType", "OrderLots", "OrderOpenPrice", "OrderStopLoss",
	"OrderTakeProfit", "OrderProfit", "OrderCommission", "OrderSwap",
	"OrderMagicNumber", "OrderComment", "OrderOpenTime", "OrderCloseTime",
	"OrderClosePrice",
}

// implementedMQL5Position — MQL5 position functions in the VM (vm_builtin_trade.go).
var implementedMQL5Position = []string{
	"PositionsTotal", "PositionGetTicket", "PositionSelectByTicket",
	"PositionGetSymbol", "PositionGetDouble", "PositionGetInteger",
	"PositionGetString",
}

// implementedAccount — account functions in the VM (vm_builtin_account.go + vm_builtin_trade.go).
var implementedAccount = []string{
	// MQL4 Account* functions (callTrade)
	"AccountBalance", "AccountEquity", "AccountFreeMargin",
	"AccountMargin", "AccountLeverage", "AccountNumber", "AccountCurrency",
	"AccountCompany", // VM-API-TRUTH-2: now reads Account().Company
}

// implementedCTradeMethods — case labels in builtin_ctrade.go execCTrade switch.
var implementedCTradeMethods = []string{
	"Buy", "Sell", "BuyLimit", "SellLimit", "BuyStop", "SellStop",
	"PositionClose", "PositionClosePartial", "PositionCloseBy",
	"PositionModify", "OrderDelete",
	"SetExpertMagicNumber", "SetDeviationInPoints",
}

// implementedPlatform — platform utility functions in the VM (vm_builtin_impls.go + vm_builtin_*.go).
var implementedPlatform = []string{
	"MathAbs", "MathMax", "MathMin", "MathSqrt", "MathPow",
	"MathLog", "MathRound", "MathFloor", "MathCeil", "MathExp",
	"MathCos", "MathSin", "MathTan", "MathArccos", "MathArcsin", "MathArctan",
	"MathLog10", "MathMod", "MathRand", "MathSrand", "MathIsValidNumber",
	"Print", "Alert", "Comment", "Sleep",
	"ArrayResize", "ArraySize", "ArrayCopy", "ArraySetAsSeries",
	"ArrayMaximum", "ArrayMinimum", "ArraySort", "ArrayInitialize",
	"ArrayFill", "ArrayFree", "ArrayRange", "ArrayIsSeries",
	"ArrayBsearch", "ArrayCompare", "ArrayInsert", "ArrayRemove",
	"ArrayReverse", "ArraySwap", "ArrayPrint", "ArrayGetAsSeries", "ArrayIsDynamic",
	"StringFind", "StringSubstr", "StringLen", "StringSplit",
	"StringTrimLeft", "StringTrimRight", "StringCompare", "StringGetCharacter",
	"StringToLower", "StringToUpper", "StringBufferLen",
	"DoubleToString", "DoubleToStr", "IntegerToString",
	"StringToDouble", "StringToInteger", "NormalizeDouble",
	"CharToString", "CharArrayToString", "ShortToString", "ShortArrayToString",
	"StringToColor", "StringToCharArray", "StringToShortArray", "EnumToString",
	"TimeToString", "TimeCurrent", "TimeLocal", "TimeToStr", "StrToTime",
	"TimeGMT", "TimeGMTOffset", "TimeDaylightSavings", "TimeTradeServer",
	"TimeToStruct", "StructToTime", "PeriodSeconds",
	"EventKillTimer", "EventSetMillisecondTimer", "EventSetTimer",
	"IsTesting", "IsOptimization", "IsVisualMode", "RefreshRates",
	"Day", "DayOfWeek", "Hour", "Minute", "Year", "Month",
	"StrToInteger",
	// MQL4 lowercase math aliases
	"ceil", "floor", "cos", "sin", "tan", "exp", "fabs",
	"fmax", "fmin", "fmod", "log", "log10", "pow", "round", "rand", "srand", "sqrt",
	// Checkup functions
	"IsConnected", "IsDemo", "IsDllsAllowed", "IsExpertEnabled",
	"IsLibrariesAllowed", "IsTradeAllowed", "IsTradeContextBusy",
	"IsStopped", "UninitializeReason",
	"MQLInfoInteger", "MQLInfoString",
	"TerminalInfoDouble", "TerminalInfoInteger", "TerminalInfoString",
	"GetTickCount",
	"CurTime",
	// MQL5 timeseries access that is faithfully available in the VM
	"Bars", "iBarShift", "iHighest", "iLowest", "iTickVolume", "iVolume",
	"CopyClose", "CopyHigh", "CopyLow", "CopyOpen", "CopyTime", "CopyTickVolume",
	// MQL5 position selection
	"PositionSelect",

	// Global Variables
	"GlobalVariableSet", "GlobalVariableGet", "GlobalVariableDel",
	"GlobalVariableCheck", "GlobalVariableTemp", "GlobalVariableFlush",
	"GlobalVariablesDeleteAll", "GlobalVariablesTotal", "GlobalVariableName",
	// Python-specific operators
	"operator_in",
	// Standalone position functions
	"PositionClose", "PositionModify",
	// Market data
	"Spread",
}

// Exported accessors for CI consistency tests (vm_builtins_test.go).

func ImplementedMarketData() []string    { return implementedMarketData }
func ImplementedIndicators() []string    { return implementedIndicators }
func ImplementedMQL4Trade() []string     { return implementedMQL4Trade }
func ImplementedMQL5Position() []string  { return implementedMQL5Position }
func ImplementedAccount() []string       { return implementedAccount }
func ImplementedPlatform() []string      { return implementedPlatform }
func ImplementedCTradeMethods() []string { return implementedCTradeMethods }
