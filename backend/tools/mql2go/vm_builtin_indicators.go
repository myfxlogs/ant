package mql2go

import (
	"github.com/shopspring/decimal"

	"anttrader/tools/mql2go/interp"
)

// ── Indicator builtins ───────────────────────────────────────────────

func builtinIMA(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iMA(symbol, period, ma_period, ma_shift, ma_method, applied_price, shift)
	shift := int(argI(args, 6))
	period := int(argI(args, 2))
	method := maMethodName(argI(args, 4))
	return interp.DecimalVal(vm.ctx.Indicators().MA(period, shift, method)), nil
}

func maMethodName(id int32) string {
	switch id {
	case 0:
		return "sma"
	case 1:
		return "ema"
	case 2:
		return "smma"
	case 3:
		return "lwma"
	default:
		return "sma"
	}
}

func builtinIRSI(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iRSI(symbol, period, rsi_period, applied_price, shift)
	period := int(argI(args, 2))
	shift := int(argI(args, 4))
	return interp.DecimalVal(vm.ctx.Indicators().RSI(period, shift)), nil
}

func builtinIATR(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iATR(symbol, period, atr_period, shift)
	period := int(argI(args, 2))
	shift := int(argI(args, 3))
	return interp.DecimalVal(vm.ctx.Indicators().ATR(period, shift)), nil
}

func builtinIBands(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iBands(symbol, period, bands_period, deviation, bands_shift, applied_price, mode, shift)
	period := int(argI(args, 2))
	deviation := argD(args, 3)
	shift := int(argI(args, 7))
	upper, middle, lower := vm.ctx.Indicators().Bollinger(period, deviation, shift)
	mode := argI(args, 6)
	switch mode {
	case 1:
		return interp.DecimalVal(upper), nil
	case 2:
		return interp.DecimalVal(lower), nil
	default:
		return interp.DecimalVal(middle), nil
	}
}

func builtinIMACD(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iMACD(symbol, period, fast_ema, slow_ema, signal, applied_price, mode, shift)
	fast := int(argI(args, 2))
	slow := int(argI(args, 3))
	signal := int(argI(args, 4))
	shift := int(argI(args, 7))
	mode := argI(args, 6)
	if mode == 1 {
		return interp.DecimalVal(vm.ctx.Indicators().MACDSignal(fast, slow, signal, shift)), nil
	}
	return interp.DecimalVal(vm.ctx.Indicators().MACD(fast, slow, signal, shift)), nil
}

func builtinIStochastic(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iStochastic(symbol, period, Kperiod, Dperiod, slowing, ma_method, price_field, mode, shift)
	k := int(argI(args, 2))
	d := int(argI(args, 3))
	slowing := int(argI(args, 4))
	shift := int(argI(args, 8))
	kVal, dVal := vm.ctx.Indicators().Stochastic(k, d, slowing, shift)
	mode := argI(args, 7)
	if mode == 1 {
		return interp.DecimalVal(dVal), nil
	}
	return interp.DecimalVal(kVal), nil
}

func builtinICCI(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iCCI(symbol, period, cci_period, applied_price, shift)
	period := int(argI(args, 2))
	shift := int(argI(args, 4))
	return interp.DecimalVal(vm.ctx.Indicators().CCI(period, shift)), nil
}

func builtinIADX(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iADX(symbol, period, adx_period, applied_price, mode, shift)
	period := int(argI(args, 2))
	shift := int(argI(args, 5))
	return interp.DecimalVal(vm.ctx.Indicators().ADX(period, shift)), nil
}

func builtinIMomentum(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iMomentum(symbol, period, mom_period, applied_price, shift)
	period := int(argI(args, 2))
	shift := int(argI(args, 4))
	return interp.DecimalVal(vm.ctx.Indicators().Momentum(period, shift)), nil
}

func builtinIWPR(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iWPR(symbol, period, calc_period, shift)
	period := int(argI(args, 2))
	shift := int(argI(args, 3))
	return interp.DecimalVal(vm.ctx.Indicators().WPR(period, shift)), nil
}

func builtinIMFI(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iMFI(symbol, period, mfi_period, shift)
	period := int(argI(args, 2))
	shift := int(argI(args, 3))
	return interp.DecimalVal(vm.ctx.Indicators().MFI(period, shift)), nil
}

func builtinIOBV(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iOBV(symbol, period, applied_price, shift)
	shift := int(argI(args, 3))
	return interp.DecimalVal(vm.ctx.Indicators().OBV(shift)), nil
}

func builtinISAR(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iSAR(symbol, period, step, maximum, shift)
	step := argD(args, 2)
	maximum := argD(args, 3)
	shift := int(argI(args, 4))
	return interp.DecimalVal(vm.ctx.Indicators().SAR(step, maximum, shift)), nil
}

func builtinIStdDev(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iStdDev(symbol, period, ma_period, ma_shift, ma_method, applied_price, shift)
	period := int(argI(args, 2))
	shift := int(argI(args, 6))
	return interp.DecimalVal(vm.ctx.Indicators().StdDev(period, shift)), nil
}
