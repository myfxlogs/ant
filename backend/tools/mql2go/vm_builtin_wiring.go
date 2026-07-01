package mql2go

// This file wires the MQL4/MQL5 function implementations added in the
// function coverage audit. It is a separate init() from vm_builtin_impls.go
// to keep both files under the 450-line hard limit.

func init() {
	// ── Indicator builtins (shared MQL4/MQL5 — 15 indicators) ────────
	builtinRegistry[id("iAlligator")].fn = builtinIAlligator
	builtinRegistry[id("iIchimoku")].fn = builtinIIchimoku
	builtinRegistry[id("iEnvelopes")].fn = builtinIEnvelopes
	builtinRegistry[id("iDeMarker")].fn = builtinIDeMarker
	builtinRegistry[id("iOsMA")].fn = builtinIOsMA
	builtinRegistry[id("iRVI")].fn = builtinIRVI
	builtinRegistry[id("iForce")].fn = builtinIForce
	builtinRegistry[id("iFractals")].fn = builtinIFractals
	builtinRegistry[id("iGator")].fn = builtinIGator
	builtinRegistry[id("iAC")].fn = builtinIAC
	builtinRegistry[id("iAD")].fn = builtinIAD
	builtinRegistry[id("iAO")].fn = builtinIAO
	builtinRegistry[id("iBearsPower")].fn = builtinIBearsPower
	builtinRegistry[id("iBullsPower")].fn = builtinIBullsPower
	builtinRegistry[id("iBWMFI")].fn = builtinIBWMFI

	// ── Indicator builtins (MQL5-only — 9 indicators) ────────────────
	builtinRegistry[id("iAMA")].fn = builtinIAMA
	builtinRegistry[id("iDEMA")].fn = builtinIDEMA
	builtinRegistry[id("iTEMA")].fn = builtinITEMA
	builtinRegistry[id("iFrAMA")].fn = builtinIFrAMA
	builtinRegistry[id("iVIDyA")].fn = builtinIVIDyA
	builtinRegistry[id("iTriX")].fn = builtinITriX
	builtinRegistry[id("iADXWilder")].fn = builtinIADXWilder
	builtinRegistry[id("iChaikin")].fn = builtinIChaikin
	builtinRegistry[id("iVolumes")].fn = builtinIVolumes

	// ── Math functions (additions) ────────────────────────────────────
	builtinRegistry[id("MathCos")].fn = builtinMathCos
	builtinRegistry[id("MathSin")].fn = builtinMathSin
	builtinRegistry[id("MathTan")].fn = builtinMathTan
	builtinRegistry[id("MathArccos")].fn = builtinMathArccos
	builtinRegistry[id("MathArcsin")].fn = builtinMathArcsin
	builtinRegistry[id("MathArctan")].fn = builtinMathArctan
	builtinRegistry[id("MathLog10")].fn = builtinMathLog10
	builtinRegistry[id("MathMod")].fn = builtinMathMod
	builtinRegistry[id("MathRand")].fn = builtinMathRand
	builtinRegistry[id("MathSrand")].fn = builtinMathSrand
	builtinRegistry[id("MathIsValidNumber")].fn = builtinMathIsValidNumber
	// MQL4 lowercase math aliases
	builtinRegistry[id("ceil")].fn = builtinAliasCeil
	builtinRegistry[id("floor")].fn = builtinAliasFloor
	builtinRegistry[id("cos")].fn = builtinAliasCos
	builtinRegistry[id("sin")].fn = builtinAliasSin
	builtinRegistry[id("tan")].fn = builtinAliasTan
	builtinRegistry[id("exp")].fn = builtinAliasExp
	builtinRegistry[id("fabs")].fn = builtinAliasFabs
	builtinRegistry[id("fmax")].fn = builtinAliasFmax
	builtinRegistry[id("fmin")].fn = builtinAliasFmin
	builtinRegistry[id("fmod")].fn = builtinAliasFmod
	builtinRegistry[id("log")].fn = builtinAliasLog
	builtinRegistry[id("log10")].fn = builtinAliasLog10
	builtinRegistry[id("pow")].fn = builtinAliasPow
	builtinRegistry[id("round")].fn = builtinAliasRound
	builtinRegistry[id("rand")].fn = builtinAliasRand
	builtinRegistry[id("srand")].fn = builtinAliasSrand
	builtinRegistry[id("sqrt")].fn = builtinAliasSqrt

	// ── String functions (additions) ──────────────────────────────────
	builtinRegistry[id("StringAdd")].fn = builtinStringAdd
	builtinRegistry[id("StringCompare")].fn = builtinStringCompare
	builtinRegistry[id("StringGetCharacter")].fn = builtinStringGetCharacter
	builtinRegistry[id("StringSetCharacter")].fn = builtinStringSetCharacter
	builtinRegistry[id("StringToLower")].fn = builtinStringToLower
	builtinRegistry[id("StringToUpper")].fn = builtinStringToUpper
	builtinRegistry[id("StringBufferLen")].fn = builtinStringBufferLen
	builtinRegistry[id("StringInit")].fn = builtinStringInit
	builtinRegistry[id("StringFill")].fn = builtinStringFill

	// ── Conversion functions (additions) ──────────────────────────────
	builtinRegistry[id("CharToString")].fn = builtinCharToString
	builtinRegistry[id("CharArrayToString")].fn = builtinCharArrayToString
	builtinRegistry[id("ShortToString")].fn = builtinShortToString
	builtinRegistry[id("ShortArrayToString")].fn = builtinShortArrayToString
	builtinRegistry[id("StringToColor")].fn = builtinStringToColor
	builtinRegistry[id("StringToCharArray")].fn = builtinStringToCharArray
	builtinRegistry[id("StringToShortArray")].fn = builtinStringToShortArray
	builtinRegistry[id("EnumToString")].fn = builtinEnumToString
	builtinRegistry[id("TimeToString")].fn = builtinTimeToString

	// ── Date/Time functions (additions) ───────────────────────────────
	builtinRegistry[id("TimeGMT")].fn = builtinTimeGMT
	builtinRegistry[id("TimeGMTOffset")].fn = builtinTimeGMTOffset
	builtinRegistry[id("TimeDaylightSavings")].fn = builtinTimeDaylightSavings
	builtinRegistry[id("TimeTradeServer")].fn = builtinTimeTradeServer
	builtinRegistry[id("TimeToStruct")].fn = builtinTimeToStruct
	builtinRegistry[id("StructToTime")].fn = builtinStructToTime
	builtinRegistry[id("PeriodSeconds")].fn = builtinPeriodSeconds

	// ── Array functions (additions) ───────────────────────────────────
	builtinRegistry[id("ArrayBsearch")].fn = builtinArrayBsearch
	builtinRegistry[id("ArrayCompare")].fn = builtinArrayCompare
	builtinRegistry[id("ArrayInsert")].fn = builtinArrayInsert
	builtinRegistry[id("ArrayRemove")].fn = builtinArrayRemove
	builtinRegistry[id("ArrayReverse")].fn = builtinArrayReverse
	builtinRegistry[id("ArraySwap")].fn = builtinArraySwap
	builtinRegistry[id("ArrayPrint")].fn = builtinArrayPrint
	builtinRegistry[id("ArrayGetAsSeries")].fn = builtinArrayGetAsSeries
	builtinRegistry[id("ArrayIsDynamic")].fn = builtinArrayIsDynamic

	// ── Checkup / Platform functions (additions) ──────────────────────
	builtinRegistry[id("IsConnected")].fn = builtinIsConnected
	builtinRegistry[id("IsDemo")].fn = builtinIsDemo
	builtinRegistry[id("IsDllsAllowed")].fn = builtinIsDllsAllowed
	builtinRegistry[id("IsExpertEnabled")].fn = builtinIsExpertEnabled
	builtinRegistry[id("IsLibrariesAllowed")].fn = builtinIsLibrariesAllowed
	builtinRegistry[id("IsTradeAllowed")].fn = builtinIsTradeAllowed
	builtinRegistry[id("IsTradeContextBusy")].fn = builtinIsTradeContextBusy
	builtinRegistry[id("IsStopped")].fn = builtinIsStopped
	builtinRegistry[id("UninitializeReason")].fn = builtinUninitializeReason
	builtinRegistry[id("MQLInfoInteger")].fn = builtinMQLInfoInteger
	builtinRegistry[id("MQLInfoString")].fn = builtinMQLInfoString
	builtinRegistry[id("TerminalInfoDouble")].fn = builtinTerminalInfoDouble
	builtinRegistry[id("TerminalInfoInteger")].fn = builtinTerminalInfoInteger
	builtinRegistry[id("TerminalInfoString")].fn = builtinTerminalInfoString
	builtinRegistry[id("GetTickCount")].fn = builtinGetTickCount
	builtinRegistry[id("GetTickCount64")].fn = builtinGetTickCount64
	builtinRegistry[id("GetMicrosecondCount")].fn = builtinGetMicrosecondCount
	builtinRegistry[id("SetUserError")].fn = builtinSetUserError
	builtinRegistry[id("SetReturnError")].fn = builtinSetReturnError
	builtinRegistry[id("CurTime")].fn = builtinCurTime

	// ── MQL5 timeseries access ────────────────────────────────────────
	builtinRegistry[id("Bars")].fn = builtinBars
	builtinRegistry[id("iBarShift")].fn = builtinIBarShift
	builtinRegistry[id("iHighest")].fn = builtinIHighest
	builtinRegistry[id("iLowest")].fn = builtinILowest
	builtinRegistry[id("iTickVolume")].fn = builtinITickVolume
	builtinRegistry[id("iRealVolume")].fn = builtinIRealVolume
	builtinRegistry[id("iSpread")].fn = builtinISpread
	builtinRegistry[id("CopyRates")].fn = builtinCopyRates
	builtinRegistry[id("CopyClose")].fn = builtinCopyClose
	builtinRegistry[id("CopyHigh")].fn = builtinCopyHigh
	builtinRegistry[id("CopyLow")].fn = builtinCopyLow
	builtinRegistry[id("CopyOpen")].fn = builtinCopyOpen
	builtinRegistry[id("CopyTime")].fn = builtinCopyTime
	builtinRegistry[id("CopyBuffer")].fn = builtinCopyBuffer
	builtinRegistry[id("CopyTickVolume")].fn = builtinCopyTickVolume
	builtinRegistry[id("CopyRealVolume")].fn = builtinCopyRealVolume
	builtinRegistry[id("CopySpread")].fn = builtinCopySpread
	builtinRegistry[id("CopyTicks")].fn = builtinCopyTicks
	builtinRegistry[id("BarsCalculated")].fn = builtinBarsCalculated
	builtinRegistry[id("SeriesInfoInteger")].fn = builtinSeriesInfoInteger

	// ── MQL5 market info additions ────────────────────────────────────
	builtinRegistry[id("SymbolInfoTick")].fn = builtinSymbolInfoTick
	builtinRegistry[id("SymbolName")].fn = builtinSymbolName
	builtinRegistry[id("SymbolSelect")].fn = builtinSymbolSelect
	builtinRegistry[id("SymbolsTotal")].fn = builtinSymbolsTotal
	builtinRegistry[id("SymbolInfoMarginRate")].fn = builtinSymbolInfoMarginRate
	builtinRegistry[id("SymbolInfoSessionQuote")].fn = builtinSymbolInfoSessionQuote
	builtinRegistry[id("SymbolInfoSessionTrade")].fn = builtinSymbolInfoSessionTrade
	builtinRegistry[id("SymbolIsSynchronized")].fn = builtinSymbolIsSynchronized

	// ── MQL5 trade helpers ────────────────────────────────────────────
	builtinRegistry[id("OrderCalcMargin")].fn = builtinOrderCalcMargin
	builtinRegistry[id("OrderCalcProfit")].fn = builtinOrderCalcProfit
	builtinRegistry[id("OrderCheck")].fn = builtinOrderCheck
	builtinRegistry[id("PositionSelect")].fn = builtinPositionSelect

	// ── MQL5 order functions (pending orders) ─────────────────────────
	builtinRegistry[id("OrderGetTicket")].fn = builtinOrderGetTicket
	builtinRegistry[id("OrderGetDouble")].fn = builtinOrderGetDouble
	builtinRegistry[id("OrderGetInteger")].fn = builtinOrderGetInteger
	builtinRegistry[id("OrderGetString")].fn = builtinOrderGetString
	builtinRegistry[id("OrdersTotalMQL5")].fn = builtinOrdersTotalMQL5

	// ── MQL5 deal/history functions ───────────────────────────────────
	builtinRegistry[id("HistorySelect")].fn = builtinHistorySelect
	builtinRegistry[id("HistorySelectByPosition")].fn = builtinHistorySelectByPosition
	builtinRegistry[id("HistoryDealsTotal")].fn = builtinHistoryDealsTotal
	builtinRegistry[id("HistoryDealSelect")].fn = builtinHistoryDealSelect
	builtinRegistry[id("HistoryDealGetTicket")].fn = builtinHistoryDealGetTicket
	builtinRegistry[id("HistoryDealGetDouble")].fn = builtinHistoryDealGetDouble
	builtinRegistry[id("HistoryDealGetInteger")].fn = builtinHistoryDealGetInteger
	builtinRegistry[id("HistoryDealGetString")].fn = builtinHistoryDealGetString
	builtinRegistry[id("HistoryOrdersTotal")].fn = builtinHistoryOrdersTotal
	builtinRegistry[id("HistoryOrderSelect")].fn = builtinHistoryOrderSelect
	builtinRegistry[id("HistoryOrderGetTicket")].fn = builtinHistoryOrderGetTicket
	builtinRegistry[id("HistoryOrderGetDouble")].fn = builtinHistoryOrderGetDouble
	builtinRegistry[id("HistoryOrderGetInteger")].fn = builtinHistoryOrderGetInteger
	builtinRegistry[id("HistoryOrderGetString")].fn = builtinHistoryOrderGetString

	// ── Account info additions (MQL5) ─────────────────────────────────
	builtinRegistry[id("AccountInfoDouble")].fn = builtinAccountInfoDouble
	builtinRegistry[id("AccountInfoInteger")].fn = builtinAccountInfoInteger
	builtinRegistry[id("AccountInfoString")].fn = builtinAccountInfoString
	builtinRegistry[id("AccountStopoutMode")].fn = builtinAccountStopoutMode
	builtinRegistry[id("AccountCredit")].fn = builtinAccountCredit

	// ── Global Variables of the Terminal ──────────────────────────────
	builtinRegistry[id("GlobalVariableSet")].fn = builtinGlobalVariableSet
	builtinRegistry[id("GlobalVariableGet")].fn = builtinGlobalVariableGet
	builtinRegistry[id("GlobalVariableDel")].fn = builtinGlobalVariableDel
	builtinRegistry[id("GlobalVariableCheck")].fn = builtinGlobalVariableCheck
	builtinRegistry[id("GlobalVariableTemp")].fn = builtinGlobalVariableTemp
	builtinRegistry[id("GlobalVariableFlush")].fn = builtinGlobalVariableFlush
	builtinRegistry[id("GlobalVariablesDeleteAll")].fn = builtinGlobalVariablesDeleteAll
	builtinRegistry[id("GlobalVariablesTotal")].fn = builtinGlobalVariablesTotal
	builtinRegistry[id("GlobalVariableName")].fn = builtinGlobalVariableName
}
