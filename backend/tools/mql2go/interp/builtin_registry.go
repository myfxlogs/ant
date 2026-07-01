package interp

// Builtin name lists — the single source of truth for IsBuiltinImplemented /
// IsCTradeMethodImplemented / IsStubIndicator in analyze.go.
//
// When adding a new VM builtin, add the name here too.
// This file is the bridge between VM execution and static analysis (reporting) —
// both must reference the same names to guarantee "preview == execution".

// implementedMarketData — market data functions in the VM (vm_builtin_market.go).
var implementedMarketData = []string{
	"Ask", "ask", "Bid", "bid", "Point", "point", "_Point",
	"Symbol", "symbol", "_Symbol", "Digits", "digits",
	"Period", "M1", "M5", "M15", "M30", "H1", "H4", "D1", "W1", "MN1",
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
	"AccountMargin", "AccountLeverage",
	"AccountNumber", "AccountStopoutLevel", "AccountCurrency",
	"AccountName", "AccountCompany",
	// MQL4-only account functions (callTradeStubs)
	"AccountFreeMarginCheck", "AccountFreeMarginMode", "AccountServer",
	"AccountStopoutMode", "AccountCredit", "AccountProfit",
	// MQL5 AccountInfo* functions (callTradeStubs)
	"AccountInfoDouble", "AccountInfoInteger", "AccountInfoString",
}

// implementedCTradeMethods — case labels in builtin_ctrade.go execCTrade switch.
var implementedCTradeMethods = []string{
	"Buy", "Sell", "BuyLimit", "SellLimit", "BuyStop", "SellStop",
	"PositionClose", "PositionClosePartial", "PositionCloseBy",
	"PositionModify", "OrderDelete",
	"SetExpertMagicNumber", "SetDeviationInPoints",
}

// stubIndicators — indicators with SDK dispatch but stub implementations (return 0).
// All indicators now have real implementations; this map is empty.
var stubIndicators = map[string]bool{}

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
	"StringConcatenate", "StringFind", "StringSubstr", "StringLen",
	"StringReplace", "StringSplit", "StringTrimLeft", "StringTrimRight",
	"StringAdd", "StringCompare", "StringGetCharacter", "StringSetCharacter",
	"StringToLower", "StringToUpper", "StringBufferLen", "StringInit", "StringFill",
	"DoubleToString", "DoubleToStr", "IntegerToString",
	"StringToDouble", "StringToInteger", "NormalizeDouble",
	"CharToString", "CharArrayToString", "ShortToString", "ShortArrayToString",
	"StringToColor", "StringToCharArray", "StringToShortArray", "EnumToString",
	"TimeToString", "TimeCurrent", "TimeLocal", "TimeToStr", "StrToTime",
	"TimeGMT", "TimeGMTOffset", "TimeDaylightSavings", "TimeTradeServer",
	"TimeToStruct", "StructToTime", "PeriodSeconds",
	"EventKillTimer", "EventSetMillisecondTimer", "EventSetTimer",
	"ExpertRemove", "GetLastError", "ResetLastError", "IsTesting", "IsOptimization",
	"IsVisualMode", "RefreshRates",
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
	"GetTickCount", "GetTickCount64", "GetMicrosecondCount",
	"SetUserError", "SetReturnError", "CurTime",
	// MQL5 market info additions
	"SymbolInfoTick", "SymbolName", "SymbolSelect", "SymbolsTotal",
	"SymbolInfoMarginRate", "SymbolInfoSessionQuote", "SymbolInfoSessionTrade",
	"SymbolIsSynchronized",
	// MQL5 timeseries access
	"Bars", "iBarShift", "iHighest", "iLowest",
	"iTickVolume", "iRealVolume", "iVolume", "iSpread",
	"CopyRates", "CopyClose", "CopyHigh", "CopyLow", "CopyOpen",
	"CopyTime", "CopyBuffer", "CopyTickVolume", "CopyRealVolume", "CopySpread",
	"CopyTicks",
	"BarsCalculated",
	"SeriesInfoInteger",
	// MQL5 trade helpers
	"OrderCalcMargin", "OrderCalcProfit", "OrderCheck",
	"PositionSelect",
	// MQL5 order history
	"HistorySelect", "HistorySelectByPosition",
	"HistoryDealsTotal", "HistoryDealSelect", "HistoryDealGetTicket",
	"HistoryDealGetDouble", "HistoryDealGetInteger", "HistoryDealGetString",
	"HistoryOrdersTotal", "HistoryOrderSelect", "HistoryOrderGetTicket",
	"HistoryOrderGetDouble", "HistoryOrderGetInteger", "HistoryOrderGetString",
	// MQL5 order functions
	"OrderGetTicket", "OrderGetDouble", "OrderGetInteger", "OrderGetString",
	// MQL4-only check functions (not in MQL5)
	"IsConnected", "IsDemo", "IsDllsAllowed", "IsExpertEnabled",
	"IsLibrariesAllowed", "IsTradeAllowed", "IsTradeContextBusy",
	// MQL4 deprecated aliases
	"CurTime",
	// Common additions
	"GetTickCount64", "GetMicrosecondCount", "SetUserError", "SetReturnError",
	// Global Variables
	"GlobalVariableSet", "GlobalVariableGet", "GlobalVariableDel",
	"GlobalVariableCheck", "GlobalVariableTemp", "GlobalVariableFlush",
	"GlobalVariablesDeleteAll", "GlobalVariablesTotal", "GlobalVariableName",
}
