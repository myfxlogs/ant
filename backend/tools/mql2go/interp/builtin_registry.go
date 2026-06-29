package interp

// Builtin name lists extracted from dispatch tables (builtins.go, builtin_trade.go,
// builtin_ctrade.go, builtin_indicators.go). These are the single source of truth
// for IsBuiltinImplemented / IsCTradeMethodImplemented / IsStubIndicator in analyze.go.
//
// When adding a new case to any switch in the dispatch functions, add the name here too.
// This file is the bridge between dispatch (execution) and analyze (reporting) —
// both must reference the same names to guarantee "preview == execution".

// implementedMarketData — case labels in builtins.go callMarketData switch.
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

// implementedMQL4Trade — MQL4 case labels in builtin_trade.go callTrade switch.
var implementedMQL4Trade = []string{
	"OrderSend", "OrderClose", "OrderCloseBy", "OrderModify", "OrderDelete",
	"OrdersTotal", "OrdersHistoryTotal", "OrderSelect", "OrderTicket", "OrderSymbol",
	"OrderType", "OrderLots", "OrderOpenPrice", "OrderStopLoss",
	"OrderTakeProfit", "OrderProfit", "OrderCommission", "OrderSwap",
	"OrderMagicNumber", "OrderComment", "OrderOpenTime", "OrderCloseTime",
	"OrderClosePrice",
}

// implementedMQL5Position — MQL5 case labels in builtin_trade.go callTrade switch.
var implementedMQL5Position = []string{
	"PositionsTotal", "PositionGetTicket", "PositionSelectByTicket",
	"PositionGetSymbol", "PositionGetDouble", "PositionGetInteger",
	"PositionGetString",
}

// implementedAccount — account case labels in builtin_trade.go callTrade switch.
var implementedAccount = []string{
	"AccountBalance", "AccountEquity", "AccountFreeMargin",
	"AccountMargin", "AccountLeverage",
	"AccountNumber", "AccountStopoutLevel", "AccountCurrency",
	"AccountName", "AccountCompany",
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

// implementedPlatform — platform utility functions in builtins.go and builtin_tools.go.
var implementedPlatform = []string{
	"MathAbs", "MathMax", "MathMin", "MathSqrt", "MathPow",
	"MathLog", "MathRound",
	"Print", "Alert", "Comment", "Sleep",
	"ArrayResize", "ArraySize", "ArrayCopy", "ArraySetAsSeries",
	"ArrayMaximum", "ArrayMinimum", "ArraySort", "ArrayInitialize",
	"StringConcatenate", "StringFind", "StringSubstr", "StringLen",
	"StringReplace", "StringSplit", "StringTrimLeft", "StringTrimRight",
	"DoubleToString", "DoubleToStr", "IntegerToString",
	"StringToDouble", "StringToInteger", "NormalizeDouble",
	"TimeToString", "TimeCurrent", "TimeLocal", "TimeToStr", "StrToTime",
	"EventKillTimer", "EventSetMillisecondTimer", "EventSetTimer",
	"ExpertRemove", "GetLastError", "ResetLastError", "IsTesting", "IsOptimization",
	"IsVisualMode", "RefreshRates",
	"Day", "DayOfWeek", "Hour", "Minute", "Year", "Month",
	"StrToInteger",
	// Math functions (MQL5 official list — aliases and full names)
	"MathCeil", "MathFloor", "MathCos", "MathSin", "MathTan",
	"MathExp", "MathMod", "MathRand", "MathSrand",
	"MathArccos", "MathArcsin", "MathArctan", "MathLog10",
	"MathIsValidNumber",
	// MQL4 lowercase math aliases
	"ceil", "floor", "cos", "sin", "tan", "exp", "fabs",
	"fmax", "fmin", "fmod", "log", "log10", "pow", "round", "rand", "srand", "sqrt",
	// String functions
	"StringAdd", "StringCompare", "StringFormat",
	"StringGetCharacter", "StringSetCharacter",
	"StringToLower", "StringToUpper",
	"StringBufferLen", "StringInit", "StringFill",
	// Array functions
	"ArrayBsearch", "ArrayCompare", "ArrayFill", "ArrayFree",
	"ArrayGetAsSeries", "ArrayIsDynamic", "ArrayIsSeries",
	"ArrayRange", "ArrayPrint", "ArrayInsert", "ArrayRemove",
	"ArrayReverse", "ArraySwap",
	// Conversion functions
	"CharToString", "CharArrayToString", "ShortToString",
	"ShortArrayToString", "StringToColor", "StringToCharArray",
	"StringToShortArray", "EnumToString",
	// Date/Time functions
	"TimeGMT", "TimeGMTOffset", "TimeDaylightSavings",
	"TimeTradeServer", "TimeToStruct", "StructToTime",
	// Checkup functions
	"PeriodSeconds", "UninitializeReason", "IsStopped",
	"MQLInfoInteger", "MQLInfoString",
	"TerminalInfoDouble", "TerminalInfoInteger", "TerminalInfoString",
	// Common functions
	"GetTickCount",
	// MQL5 account info (aliases for MQL4 Account* functions)
	"AccountInfoDouble", "AccountInfoInteger", "AccountInfoString",
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
	"PositionSelect", "PositionSelectByTicket",
	// MQL5 order history
	"HistorySelect", "HistorySelectByPosition",
	"HistoryDealsTotal", "HistoryDealSelect", "HistoryDealGetTicket",
	"HistoryDealGetDouble", "HistoryDealGetInteger", "HistoryDealGetString",
	"HistoryOrdersTotal", "HistoryOrderSelect", "HistoryOrderGetTicket",
	"HistoryOrderGetDouble", "HistoryOrderGetInteger", "HistoryOrderGetString",
	// MQL5 order functions
	"OrderGetTicket", "OrderSelect", "OrderGetDouble", "OrderGetInteger", "OrderGetString",
}
