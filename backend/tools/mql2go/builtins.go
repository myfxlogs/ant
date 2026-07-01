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

	// Account functions
	{"AccountBalance", nil},
	{"AccountEquity", nil},
	{"AccountFreeMargin", nil},
	{"AccountMargin", nil},
	{"AccountLeverage", nil},
	{"AccountProfit", nil},
	{"AccountCurrency", nil},
	{"AccountCompany", nil},

	// Symbol info
	{"SymbolInfoDouble", nil},
	{"SymbolInfoInteger", nil},
	{"SymbolInfoString", nil},
	{"MarketInfo", nil},

	// String functions
	{"StringFormat", nil},

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
	{"IntegerToString", nil},
	{"StringToInteger", nil},
	{"StrToDouble", nil},
	{"StrToInteger", nil},
	{"TimeCurrent", nil},
	{"TimeLocal", nil},
	{"Sleep", nil},
	{"EventSetTimer", nil},
	{"EventKillTimer", nil},

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
