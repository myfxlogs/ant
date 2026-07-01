package mql2go

import (
	"fmt"
	"math"
	"strings"

	"github.com/shopspring/decimal"

	"anttrader/tools/mql2go/interp"
)

// This file wires VM builtin IDs to actual implementations that call the SDK context.
// The builtinRegistry in builtins.go declares the names and order; this file sets the fn fields.
// Implementation functions are split across:
//   - vm_builtin_impls.go    — init(), helpers, math, platform, market data, utility
//   - vm_builtin_indicators.go — iMA, iRSI, iATR, etc.
//   - vm_builtin_trade.go    — OrderSend, OrderClose, CTrade, position functions
//   - vm_builtin_account.go  — Account*, SymbolInfo*, MarketInfo, StringFormat, Array*

func init() {
	// Math functions
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

	// Platform functions
	builtinRegistry[id("Print")].fn = builtinPrint
	builtinRegistry[id("Alert")].fn = builtinPrint
	builtinRegistry[id("Comment")].fn = builtinPrint

	// Market data — series access via builtins (Close(), Open(), etc. without subscript)
	builtinRegistry[id("Close")].fn = builtinSeriesClose
	builtinRegistry[id("Open")].fn = builtinSeriesOpen
	builtinRegistry[id("High")].fn = builtinSeriesHigh
	builtinRegistry[id("Low")].fn = builtinSeriesLow
	builtinRegistry[id("Volume")].fn = builtinSeriesVolume
	builtinRegistry[id("Time")].fn = builtinSeriesTime

	// Price data
	builtinRegistry[id("Bid")].fn = builtinBid
	builtinRegistry[id("Ask")].fn = builtinAsk
	builtinRegistry[id("Point")].fn = builtinPoint
	builtinRegistry[id("Digits")].fn = builtinDigits
	builtinRegistry[id("Symbol")].fn = builtinSymbol
	builtinRegistry[id("Period")].fn = builtinPeriod

	// Indicators — fully implemented
	builtinRegistry[id("iMA")].fn = builtinIMA
	builtinRegistry[id("iRSI")].fn = builtinIRSI
	builtinRegistry[id("iATR")].fn = builtinIATR
	builtinRegistry[id("iBands")].fn = builtinIBands
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

	// Cross-timeframe market data
	builtinRegistry[id("iClose")].fn = builtinIClose
	builtinRegistry[id("iOpen")].fn = builtinIOpen
	builtinRegistry[id("iHigh")].fn = builtinIHigh
	builtinRegistry[id("iLow")].fn = builtinILow
	builtinRegistry[id("iTime")].fn = builtinITime
	builtinRegistry[id("iVolume")].fn = builtinIVolume

	// Utility
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

	// Account functions
	builtinRegistry[id("AccountBalance")].fn = builtinAccountBalance
	builtinRegistry[id("AccountEquity")].fn = builtinAccountEquity
	builtinRegistry[id("AccountFreeMargin")].fn = builtinAccountFreeMargin
	builtinRegistry[id("AccountMargin")].fn = builtinAccountMargin
	builtinRegistry[id("AccountLeverage")].fn = builtinAccountLeverage
	builtinRegistry[id("AccountProfit")].fn = builtinNoopDecimal
	builtinRegistry[id("AccountCurrency")].fn = builtinNoopString
	builtinRegistry[id("AccountCompany")].fn = builtinNoopString

	// Symbol info
	builtinRegistry[id("SymbolInfoDouble")].fn = builtinSymbolInfoDouble
	builtinRegistry[id("SymbolInfoInteger")].fn = builtinSymbolInfoInteger
	builtinRegistry[id("SymbolInfoString")].fn = builtinSymbolInfoString
	builtinRegistry[id("MarketInfo")].fn = builtinMarketInfo

	// String format
	builtinRegistry[id("StringFormat")].fn = builtinStringFormat

	// MQL4 trade functions
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

	// MQL5 position functions
	builtinRegistry[id("PositionsTotal")].fn = builtinPositionsTotal
	builtinRegistry[id("PositionGetTicket")].fn = builtinPositionGetTicket
	builtinRegistry[id("PositionGetDouble")].fn = builtinPositionGetDouble
	builtinRegistry[id("PositionGetInteger")].fn = builtinPositionGetInteger
	builtinRegistry[id("PositionGetString")].fn = builtinPositionGetString
	builtinRegistry[id("PositionGetSymbol")].fn = builtinPositionGetSymbol
	builtinRegistry[id("PositionSelectByTicket")].fn = builtinPositionSelectByTicket

	// MQL5 CTrade methods
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

	// Array functions — stubs
	builtinRegistry[id("ArraySize")].fn = builtinArraySize
	builtinRegistry[id("ArrayResize")].fn = builtinNoopInt
	builtinRegistry[id("ArrayCopy")].fn = builtinNoopInt
	builtinRegistry[id("ArraySetAsSeries")].fn = builtinNoopBool
	builtinRegistry[id("ArrayMaximum")].fn = builtinNoopInt
	builtinRegistry[id("ArrayMinimum")].fn = builtinNoopInt
	builtinRegistry[id("ArraySort")].fn = builtinNoopInt
	builtinRegistry[id("ArrayFree")].fn = builtinNoop
	builtinRegistry[id("ArrayInitialize")].fn = builtinNoopInt
	builtinRegistry[id("ArrayFill")].fn = builtinNoop
	builtinRegistry[id("ArrayRange")].fn = builtinNoopInt
	builtinRegistry[id("ArrayIsSeries")].fn = builtinNoopBool
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

// argD returns arg i as decimal, defaulting to zero if out of range.
func argD(args []interp.Value, i int) decimal.Decimal {
	if i >= len(args) {
		return decimal.Zero
	}
	return args[i].ToDecimal()
}

// argI returns arg i as int32, defaulting to 0 if out of range.
func argI(args []interp.Value, i int) int32 {
	if i >= len(args) {
		return 0
	}
	return args[i].ToInt()
}

// argS returns arg i as string, defaulting to "" if out of range.
func argS(args []interp.Value, i int) string {
	if i >= len(args) {
		return ""
	}
	return args[i].ToString()
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
	return interp.DecimalVal(decimal.NewFromFloat(math.Sqrt(argD(args, 0).InexactFloat64()))), nil
}

func builtinMathPow(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimal.NewFromFloat(math.Pow(argD(args, 0).InexactFloat64(), argD(args, 1).InexactFloat64()))), nil
}

func builtinMathLog(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimal.NewFromFloat(math.Log(argD(args, 0).InexactFloat64()))), nil
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
	return interp.DecimalVal(decimal.NewFromFloat(math.Exp(argD(args, 0).InexactFloat64()))), nil
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

// ── Utility builtins ─────────────────────────────────────────────────

func builtinNormalizeDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	value := argD(args, 0)
	digits := int(argI(args, 1))
	return interp.DecimalVal(value.Round(int32(digits))), nil
}

func builtinDoubleToString(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(argD(args, 0).String()), nil
}

func builtinIntegerToString(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(fmt.Sprintf("%d", argI(args, 0))), nil
}

func builtinStringToDouble(vm *VM, args []interp.Value) (interp.Value, error) {
	d, err := decimal.NewFromString(argS(args, 0))
	if err != nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	return interp.DecimalVal(d), nil
}

func builtinStringToInteger(vm *VM, args []interp.Value) (interp.Value, error) {
	var n int32
	fmt.Sscanf(argS(args, 0), "%d", &n)
	return interp.IntVal(n), nil
}

func builtinTimeCurrent(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.IntVal(0), nil
	}
	return interp.IntVal(int32(vm.ctx.ServerTime() / 1000)), nil
}

func builtinNoop(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.NoneVal(), nil
}

func builtinNoopBool(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.BoolVal(true), nil
}

func builtinNoopInt(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.IntVal(0), nil
}

func builtinNoopDecimal(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.DecimalVal(decimal.Zero), nil
}

func builtinNoopString(vm *VM, args []interp.Value) (interp.Value, error) {
	return interp.StringVal(""), nil
}

func builtinEventSetTimer(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx != nil {
		vm.ctx.SetTimer(int(argI(args, 0)))
	}
	return interp.NoneVal(), nil
}

func builtinEventKillTimer(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx != nil {
		vm.ctx.KillTimer()
	}
	return interp.NoneVal(), nil
}
