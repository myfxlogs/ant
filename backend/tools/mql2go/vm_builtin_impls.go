package mql2go

import (
	"fmt"
	"math"
	"strings"

	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
)

// This file wires VM builtin IDs to actual implementations that call the SDK context.
// The builtinRegistry in builtins.go declares the names and order; this file sets the fn fields.
// Implementation functions are split across:
//   - vm_builtin_impls.go    — init(), helpers, math, platform, market data, utility
//   - vm_builtin_indicators.go — iMA, iRSI, iATR, etc.
//   - vm_builtin_trade.go    — OrderSend, OrderClose, CTrade, position functions
//   - vm_builtin_account.go  — Account*, SymbolInfo*, MarketInfo, StringFormat, Array*

func init() {
	registerMathBuiltins()
	registerPlatformBuiltins()
	registerIndicatorBuiltins()
	registerUtilityBuiltins()
	registerAccountBuiltins()
	registerStringBuiltins()
	registerTradeBuiltins()
	registerArrayBuiltins()
}

func registerMathBuiltins() {
	builtinRegistry[id("MathAbs")].fn = builtinMathAbs
	builtinRegistry[id("MathMax")].fn = builtinMathMax
	builtinRegistry[id("MathMin")].fn = builtinMathMin
	builtinRegistry[id("MathSqrt")].fn = builtinMathSqrt
	builtinRegistry[id("MathPow")].fn = builtinMathPow
	builtinRegistry[id("MathLog")].fn = builtinMathLog
	builtinRegistry[id("MathRound")].fn = builtinMathRound
	builtinRegistry[id("MathFloor")].fn = builtinMathFloor
	builtinRegistry[id("MathCeil")].fn = builtinMathCeil
	builtinRegistry[id("MathExp")].fn = builtinMathExp
}

func registerPlatformBuiltins() {
	builtinRegistry[id("Print")].fn = builtinPrint
	builtinRegistry[id("Alert")].fn = builtinPrint
	builtinRegistry[id("Comment")].fn = builtinPrint
	builtinRegistry[id("Close")].fn = builtinSeriesClose
	builtinRegistry[id("Open")].fn = builtinSeriesOpen
	builtinRegistry[id("High")].fn = builtinSeriesHigh
	builtinRegistry[id("Low")].fn = builtinSeriesLow
	builtinRegistry[id("Volume")].fn = builtinSeriesVolume
	builtinRegistry[id("Time")].fn = builtinSeriesTime
	builtinRegistry[id("Bid")].fn = builtinBid
	builtinRegistry[id("Ask")].fn = builtinAsk
	builtinRegistry[id("Point")].fn = builtinPoint
	builtinRegistry[id("Digits")].fn = builtinDigits
	builtinRegistry[id("Symbol")].fn = builtinSymbol
	builtinRegistry[id("Period")].fn = builtinPeriod
	builtinRegistry[id("RefreshRates")].fn = builtinNoopBool
	builtinRegistry[id("GetLastError")].fn = builtinNoopInt
	builtinRegistry[id("ResetLastError")].fn = builtinNoop
	builtinRegistry[id("ExpertRemove")].fn = builtinNoop
	builtinRegistry[id("IsTesting")].fn = builtinIsTesting
	builtinRegistry[id("IsOptimization")].fn = builtinNoopInt
	builtinRegistry[id("IsVisualMode")].fn = builtinNoopInt
}

func registerIndicatorBuiltins() {
	builtinRegistry[id("iMA")].fn = builtinIMA
	builtinRegistry[id("iRSI")].fn = builtinIRSI
	builtinRegistry[id("iATR")].fn = builtinIATR
	builtinRegistry[id("iBands")].fn = builtinIBands
	builtinRegistry[id("iBollinger")].fn = builtinIBands
	builtinRegistry[id("iMACD")].fn = builtinIMACD
	builtinRegistry[id("iStochastic")].fn = builtinIStochastic
	builtinRegistry[id("iCCI")].fn = builtinICCI
	builtinRegistry[id("iADX")].fn = builtinIADX
	builtinRegistry[id("iMomentum")].fn = builtinIMomentum
	builtinRegistry[id("iWPR")].fn = builtinIWPR
	builtinRegistry[id("iMFI")].fn = builtinIMFI
	builtinRegistry[id("iOBV")].fn = builtinIOBV
	builtinRegistry[id("iSAR")].fn = builtinISAR
	builtinRegistry[id("iStdDev")].fn = builtinIStdDev
	builtinRegistry[id("iMAOnArray")].fn = builtinIMAOnArray
	builtinRegistry[id("iRSIOnArray")].fn = builtinIRSIOnArray
	builtinRegistry[id("iATROnArray")].fn = builtinIATROnArray
	builtinRegistry[id("iBandsOnArray")].fn = builtinIBandsOnArray
	builtinRegistry[id("iStdDevOnArray")].fn = builtinIStdDevOnArray
	builtinRegistry[id("iMomentumOnArray")].fn = builtinIMomentumOnArray
	builtinRegistry[id("iCCIOnArray")].fn = builtinICCIOnArray
	builtinRegistry[id("iMACDOnArray")].fn = builtinIMACDOnArray
	builtinRegistry[id("iClose")].fn = builtinIClose
	builtinRegistry[id("iOpen")].fn = builtinIOpen
	builtinRegistry[id("iHigh")].fn = builtinIHigh
	builtinRegistry[id("iLow")].fn = builtinILow
	builtinRegistry[id("iTime")].fn = builtinITime
	builtinRegistry[id("iVolume")].fn = builtinIVolume
}

func registerUtilityBuiltins() {
	builtinRegistry[id("NormalizeDouble")].fn = builtinNormalizeDouble
	builtinRegistry[id("DoubleToString")].fn = builtinDoubleToString
	builtinRegistry[id("IntegerToString")].fn = builtinIntegerToString
	builtinRegistry[id("StrToDouble")].fn = builtinStringToDouble
	builtinRegistry[id("StrToInteger")].fn = builtinStringToInteger
	builtinRegistry[id("StringToInteger")].fn = builtinStringToInteger
	builtinRegistry[id("TimeCurrent")].fn = builtinTimeCurrent
	builtinRegistry[id("TimeLocal")].fn = builtinTimeCurrent
	builtinRegistry[id("Sleep")].fn = builtinNoop
	builtinRegistry[id("EventSetTimer")].fn = builtinEventSetTimer
	builtinRegistry[id("EventKillTimer")].fn = builtinEventKillTimer
	builtinRegistry[id("EventSetMillisecondTimer")].fn = builtinEventSetTimer
	builtinRegistry[id("Day")].fn = builtinDay
	builtinRegistry[id("DayOfWeek")].fn = builtinDayOfWeek
	builtinRegistry[id("Hour")].fn = builtinHour
	builtinRegistry[id("Minute")].fn = builtinMinute
	builtinRegistry[id("Year")].fn = builtinYear
	builtinRegistry[id("Month")].fn = builtinMonth
	builtinRegistry[id("StrToTime")].fn = builtinStrToTime
	builtinRegistry[id("TimeToStr")].fn = builtinTimeToStr
	builtinRegistry[id("DoubleToStr")].fn = builtinDoubleToString
}

func registerAccountBuiltins() {
	builtinRegistry[id("AccountBalance")].fn = builtinAccountBalance
	builtinRegistry[id("AccountEquity")].fn = builtinAccountEquity
	builtinRegistry[id("AccountFreeMargin")].fn = builtinAccountFreeMargin
	builtinRegistry[id("AccountMargin")].fn = builtinAccountMargin
	builtinRegistry[id("AccountLeverage")].fn = builtinAccountLeverage
	builtinRegistry[id("AccountProfit")].fn = builtinNoopDecimal
	builtinRegistry[id("AccountCurrency")].fn = builtinNoopString
	builtinRegistry[id("AccountCompany")].fn = builtinNoopString
	builtinRegistry[id("AccountNumber")].fn = builtinAccountNumber
	builtinRegistry[id("AccountStopoutLevel")].fn = builtinNoopInt
	builtinRegistry[id("AccountName")].fn = builtinNoopString
	builtinRegistry[id("AccountServer")].fn = builtinNoopString
	builtinRegistry[id("AccountFreeMarginCheck")].fn = builtinNoopDecimal
	builtinRegistry[id("AccountFreeMarginMode")].fn = builtinNoopInt
	builtinRegistry[id("SymbolInfoDouble")].fn = builtinSymbolInfoDouble
	builtinRegistry[id("SymbolInfoInteger")].fn = builtinSymbolInfoInteger
	builtinRegistry[id("SymbolInfoString")].fn = builtinSymbolInfoString
	builtinRegistry[id("MarketInfo")].fn = builtinMarketInfo
}

func registerStringBuiltins() {
	builtinRegistry[id("StringFormat")].fn = builtinStringFormat
	builtinRegistry[id("StringFind")].fn = builtinStringFind
	builtinRegistry[id("StringSubstr")].fn = builtinStringSubstr
	builtinRegistry[id("StringLen")].fn = builtinStringLen
	builtinRegistry[id("StringReplace")].fn = builtinStringReplace
	builtinRegistry[id("StringSplit")].fn = builtinStringSplit
	builtinRegistry[id("StringTrimLeft")].fn = builtinStringTrimLeft
	builtinRegistry[id("StringTrimRight")].fn = builtinStringTrimRight
	builtinRegistry[id("StringConcatenate")].fn = builtinStringConcatenate
	builtinRegistry[id("StringToDouble")].fn = builtinStringToDouble
}

func registerTradeBuiltins() {
	builtinRegistry[id("OrderSend")].fn = builtinOrderSend
	builtinRegistry[id("OrderClose")].fn = builtinOrderClose
	builtinRegistry[id("OrderModify")].fn = builtinOrderModify
	builtinRegistry[id("OrderDelete")].fn = builtinOrderDelete
	builtinRegistry[id("OrdersTotal")].fn = builtinOrdersTotal
	builtinRegistry[id("OrderSelect")].fn = builtinOrderSelect
	builtinRegistry[id("OrderStopLoss")].fn = builtinOrderStopLoss
	builtinRegistry[id("OrderTakeProfit")].fn = builtinOrderTakeProfit
	builtinRegistry[id("OrderTicket")].fn = builtinOrderTicket
	builtinRegistry[id("OrderType")].fn = builtinOrderType
	builtinRegistry[id("OrderLots")].fn = builtinOrderLots
	builtinRegistry[id("OrderSymbol")].fn = builtinOrderSymbol
	builtinRegistry[id("OrderOpenPrice")].fn = builtinOrderOpenPrice
	builtinRegistry[id("OrderClosePrice")].fn = builtinOrderClosePrice
	builtinRegistry[id("OrderProfit")].fn = builtinOrderProfit
	builtinRegistry[id("OrderMagicNumber")].fn = builtinOrderMagicNumber
	builtinRegistry[id("OrderComment")].fn = builtinOrderComment
	builtinRegistry[id("OrderOpenTime")].fn = builtinOrderOpenTime
	builtinRegistry[id("OrderCloseTime")].fn = builtinOrderCloseTime
	builtinRegistry[id("OrderCloseBy")].fn = builtinOrderCloseBy
	builtinRegistry[id("OrderExpiration")].fn = builtinNoopDecimal
	builtinRegistry[id("OrderPrint")].fn = builtinNoop
	builtinRegistry[id("OrderCommission")].fn = builtinNoopDecimal
	builtinRegistry[id("OrderSwap")].fn = builtinNoopDecimal
	builtinRegistry[id("PositionsTotal")].fn = builtinPositionsTotal
	builtinRegistry[id("PositionGetTicket")].fn = builtinPositionGetTicket
	builtinRegistry[id("PositionGetDouble")].fn = builtinPositionGetDouble
	builtinRegistry[id("PositionGetInteger")].fn = builtinPositionGetInteger
	builtinRegistry[id("PositionGetString")].fn = builtinPositionGetString
	builtinRegistry[id("PositionGetSymbol")].fn = builtinPositionGetSymbol
	builtinRegistry[id("PositionSelectByTicket")].fn = builtinPositionSelectByTicket
	builtinRegistry[id("CTrade.Buy")].fn = builtinCTradeBuy
	builtinRegistry[id("CTrade.Sell")].fn = builtinCTradeSell
	builtinRegistry[id("CTrade.BuyLimit")].fn = builtinCTradeBuyLimit
	builtinRegistry[id("CTrade.SellLimit")].fn = builtinCTradeSellLimit
	builtinRegistry[id("CTrade.BuyStop")].fn = builtinCTradeBuyStop
	builtinRegistry[id("CTrade.SellStop")].fn = builtinCTradeSellStop
	builtinRegistry[id("CTrade.PositionClose")].fn = builtinCTradePositionClose
	builtinRegistry[id("CTrade.PositionClosePartial")].fn = builtinCTradePositionClosePartial
	builtinRegistry[id("CTrade.PositionCloseBy")].fn = builtinCTradePositionCloseBy
	builtinRegistry[id("CTrade.PositionModify")].fn = builtinCTradePositionModify
	builtinRegistry[id("CTrade.OrderDelete")].fn = builtinCTradeOrderDelete
	builtinRegistry[id("CTrade.SetExpertMagicNumber")].fn = builtinNoop
	builtinRegistry[id("CTrade.SetDeviationInPoints")].fn = builtinNoop
}

func registerArrayBuiltins() {
	builtinRegistry[id("ArraySize")].fn = builtinArraySize
	builtinRegistry[id("ArrayResize")].fn = builtinArrayResize
	builtinRegistry[id("ArrayCopy")].fn = builtinArrayCopy
	builtinRegistry[id("ArraySetAsSeries")].fn = builtinNoopBool
	builtinRegistry[id("ArrayMaximum")].fn = builtinArrayMaximum
	builtinRegistry[id("ArrayMinimum")].fn = builtinArrayMinimum
	builtinRegistry[id("ArraySort")].fn = builtinArraySort
	builtinRegistry[id("ArrayFree")].fn = builtinNoop
	builtinRegistry[id("ArrayInitialize")].fn = builtinArrayInitialize
	builtinRegistry[id("ArrayFill")].fn = builtinArrayFill
	builtinRegistry[id("ArrayRange")].fn = builtinNoopInt
	builtinRegistry[id("ArrayIsSeries")].fn = builtinNoopBool

	// Python-specific operators
	builtinRegistry[id("operator_in")].fn = builtinOperatorIn

	// Standalone position functions (legacy snake_case mappings)
	builtinRegistry[id("PositionClose")].fn = builtinCTradePositionClose
	builtinRegistry[id("PositionModify")].fn = builtinCTradePositionModify

	// Market data — Spread
	builtinRegistry[id("Spread")].fn = builtinSpread
}

// id looks up a builtin name in the registry and returns its index.
// Panics if the name is not found — this is a compile-time configuration error.
func id(name string) int {
	for i, e := range builtinRegistry {
		if e.name == name {
			return i
		}
	}
	panic(fmt.Sprintf("builtin not found in registry: %s", name))
}

// ── Math builtins ────────────────────────────────────────────────────

func builtinMathAbs(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(argD(args, 0).Abs()), nil
}

func builtinMathMax(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimal.Max(argD(args, 0), argD(args, 1))), nil
}

func builtinMathMin(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimal.Min(argD(args, 0), argD(args, 1))), nil
}

func builtinMathSqrt(vm *VM, args []interp.Value) (interp.Value, error) {
	f := argD(args, 0).InexactFloat64()
	if f < 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(safeDecimalFromFloat(math.Sqrt(f))), nil
}

func builtinMathPow(vm *VM, args []interp.Value) (interp.Value, error) {
	base := argD(args, 0).InexactFloat64()
	exp := argD(args, 1).InexactFloat64()
	return interp.DecimalVal(safeDecimalFromFloat(math.Pow(base, exp))), nil
}

func builtinMathLog(vm *VM, args []interp.Value) (interp.Value, error) {
	f := argD(args, 0).InexactFloat64()
	if f <= 0 {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(safeDecimalFromFloat(math.Log(f))), nil
}

func builtinMathRound(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(argD(args, 0).Round(0)), nil
}

func builtinMathFloor(vm *VM, args []interp.Value) (interp.Value, error) {
	d := argD(args, 0)
	return interp.DecimalVal(d.Sub(d.Mod(decimal.NewFromInt(1)))), nil
}

func builtinMathCeil(vm *VM, args []interp.Value) (interp.Value, error) {
	d := argD(args, 0)
	mod := d.Mod(decimal.NewFromInt(1))
	if mod.IsZero() {
		return interp.DecimalVal(d), nil
	}
	return interp.DecimalVal(d.Sub(mod).Add(decimal.NewFromInt(1))), nil
}

func builtinMathExp(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(safeDecimalFromFloat(math.Exp(argD(args, 0).InexactFloat64()))), nil
}

// ── Platform builtins ────────────────────────────────────────────────

func builtinPrint(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx != nil {
		parts := make([]string, len(args))
		for i, a := range args {
			parts[i] = a.ToString()
		}
		vm.ctx.Log(strings.Join(parts, " "))
	}
	return interp.NoneVal(), nil
}

// ── Market data series builtins (Close(), Open(), etc. without subscript) ──

func builtinSeriesClose(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("Close", int(argI(args, 0))), nil
}

func builtinSeriesOpen(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("Open", int(argI(args, 0))), nil
}

func builtinSeriesHigh(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("High", int(argI(args, 0))), nil
}

func builtinSeriesLow(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("Low", int(argI(args, 0))), nil
}

func builtinSeriesVolume(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("Volume", int(argI(args, 0))), nil
}

func builtinSeriesTime(vm *VM, args []interp.Value) (interp.Value, error) {
	return vm.getSeriesHelper("Time", int(argI(args, 0))), nil
}

// ── Price data builtins ──────────────────────────────────────────────

func builtinBid(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(vm.ctx.Bid()), nil
}

func builtinAsk(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(vm.ctx.Ask()), nil
}

func builtinPoint(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(vm.ctx.Point()), nil
}

func builtinDigits(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(vm.ctx.Digits()), nil
}

func builtinSymbol(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.StringVal(""), nil
	}
	return interp.StringVal(vm.ctx.Symbol()), nil
}

func builtinPeriod(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	tf := vm.ctx.Timeframe()
	return interp.IntVal(tfToInt(tf)), nil
}

// builtinOperatorIn implements the Python `in` / `not in` operator.
// args[0] = left (needle), args[1] = right (haystack).
// Supports: string substring, array membership, scalar equality.
func builtinOperatorIn(vm *VM, args []interp.Value) (interp.Value, error) {
	if len(args) < 2 {
		return interp.BoolVal(false), nil
	}
	left := args[0]
	right := args[1]
	switch right.Kind {
	case interp.ValString:
		return interp.BoolVal(strings.Contains(right.Str, left.ToString())), nil
	case interp.ValArray:
		for _, v := range right.Array {
			if v.Equal(left) {
				return interp.BoolVal(true), nil
			}
		}
		return interp.BoolVal(false), nil
	default:
		return interp.BoolVal(right.Equal(left)), nil
	}
}

// builtinSpread returns the current spread in points (Ask - Bid) / Point.
func builtinSpread(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	ask := vm.ctx.Ask()
	bid := vm.ctx.Bid()
	point := vm.ctx.Point()
	if point.IsZero() {
		return interp.IntVal(0), nil
	}
	spread := ask.Sub(bid).Div(point)
	return interp.IntVal(int32(spread.IntPart())), nil
}

func tfToInt(tf string) int32 {
	switch tf {
	case "M1":
		return 1
	case "M5":
		return 5
	case "M15":
		return 15
	case "M30":
		return 30
	case "H1":
		return 60
	case "H4":
		return 240
	case "D1":
		return 1440
	case "W1":
		return 10080
	case "MN1":
		return 43200
	default:
		return 0
	}
}
