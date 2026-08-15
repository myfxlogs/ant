package mql2go

import (
	"github.com/shopspring/decimal"

	"alphaforge/tools/mql2go/interp"
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
	appliedPrice := int(argI(args, 5))
	val := vm.ctx.Indicators().MA(period, shift, method, appliedPrice)
	if shift == 0 {
		h := diagHash4(diagNameMA, int32(period), argI(args, 4), int32(appliedPrice))
		recordDiag(vm, h, diagKeyMA(period, method, appliedPrice), val)
	}
	return interp.DecimalVal(val), nil
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
	appliedPrice := int(argI(args, 3))
	shift := int(argI(args, 4))
	val := vm.ctx.Indicators().RSI(period, shift, appliedPrice)
	if shift == 0 {
		h := diagHash3(diagNameRSI, int32(period), int32(appliedPrice))
		recordDiag(vm, h, diagKeyRSI(period, appliedPrice), val)
	}
	return interp.DecimalVal(val), nil
}

func builtinIATR(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iATR(symbol, period, atr_period, shift)
	period := int(argI(args, 2))
	shift := int(argI(args, 3))
	val := vm.ctx.Indicators().ATR(period, shift)
	if shift == 0 {
		h := diagHash2(diagNameATR, int32(period))
		recordDiag(vm, h, diagKeyATR(period), val)
	}
	return interp.DecimalVal(val), nil
}

func builtinIBands(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iBands(symbol, period, bands_period, deviation, bands_shift, applied_price, mode, shift)
	period := int(argI(args, 2))
	deviation := argD(args, 3)
	appliedPrice := int(argI(args, 5))
	shift := int(argI(args, 7))
	upper, middle, lower := vm.ctx.Indicators().Bollinger(period, deviation, appliedPrice, shift)
	mode := argI(args, 6)
	if shift == 0 {
		devInt := int32(deviation.IntPart())
		h := diagHash5(diagNameBands, int32(period), devInt, int32(appliedPrice), mode)
		var sub string
		var val decimal.Decimal
		switch mode {
		case 1:
			sub = "upper"
			val = upper
		case 2:
			sub = "lower"
			val = lower
		default:
			sub = "middle"
			val = middle
		}
		recordDiag(vm, h, diagKeyBands(period, deviation, appliedPrice, sub), val)
	}
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
	appliedPrice := int(argI(args, 5))
	shift := int(argI(args, 7))
	mode := argI(args, 6)
	if mode == 1 {
		val := vm.ctx.Indicators().MACDSignal(fast, slow, signal, appliedPrice, shift)
		if shift == 0 {
			h := diagHash5(diagNameMACD, int32(fast), int32(slow), int32(signal), int32(appliedPrice))
			recordDiag(vm, h, diagKeyMACD(fast, slow, signal, appliedPrice, "signal"), val)
		}
		return interp.DecimalVal(val), nil
	}
	val := vm.ctx.Indicators().MACD(fast, slow, signal, appliedPrice, shift)
	if shift == 0 {
		h := diagHash5(diagNameMACD, int32(fast), int32(slow), int32(signal), int32(appliedPrice))
		// Use a different hash for main vs signal by including mode in hash
		h2 := h ^ uint64(0xDEAD)
		recordDiag(vm, h2, diagKeyMACD(fast, slow, signal, appliedPrice, "main"), val)
	}
	return interp.DecimalVal(val), nil
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
	if shift == 0 {
		h := diagHash4(diagNameStoch, int32(k), int32(d), int32(slowing))
		if mode == 1 {
			h2 := h ^ uint64(0xBEEF)
			recordDiag(vm, h2, diagKeyStoch(k, d, slowing, "d"), dVal)
		} else {
			recordDiag(vm, h, diagKeyStoch(k, d, slowing, "k"), kVal)
		}
	}
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
	appliedPrice := int(argI(args, 3))
	shift := int(argI(args, 4))
	return interp.DecimalVal(vm.ctx.Indicators().CCI(period, shift, appliedPrice)), nil
}

func builtinIADX(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iADX(symbol, period, adx_period, applied_price, mode, shift)
	period := int(argI(args, 2))
	mode := argI(args, 4)
	shift := int(argI(args, 5))
	switch mode {
	case 0: // MODE_MAIN (ADX line)
		val := vm.ctx.Indicators().ADX(period, shift)
		if shift == 0 {
			h := diagHash2(diagNameADX, int32(period))
			recordDiag(vm, h, diagKeyADX(period), val)
		}
		return interp.DecimalVal(val), nil
	case 1: // MODE_PLUSDI (+DI line)
		vm.recordBlindSpot("iADX:MODE_PLUSDI")
		return interp.DecimalVal(decimal.Zero), nil
	case 2: // MODE_MINUSDI (-DI line)
		vm.recordBlindSpot("iADX:MODE_MINUSDI")
		return interp.DecimalVal(decimal.Zero), nil
	default:
		val := vm.ctx.Indicators().ADX(period, shift)
		if shift == 0 {
			h := diagHash2(diagNameADX, int32(period))
			recordDiag(vm, h, diagKeyADX(period), val)
		}
		return interp.DecimalVal(val), nil
	}
}

func builtinIMomentum(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimal.Zero), nil
	}
	// iMomentum(symbol, period, mom_period, applied_price, shift)
	period := int(argI(args, 2))
	appliedPrice := int(argI(args, 3))
	shift := int(argI(args, 4))
	return interp.DecimalVal(vm.ctx.Indicators().Momentum(period, shift, appliedPrice)), nil
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
	appliedPrice := int(argI(args, 2))
	shift := int(argI(args, 3))
	return interp.DecimalVal(vm.ctx.Indicators().OBV(appliedPrice, shift)), nil
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
	method := maMethodName(argI(args, 4))
	appliedPrice := int(argI(args, 5))
	shift := int(argI(args, 6))
	return interp.DecimalVal(vm.ctx.Indicators().StdDev(period, shift, method, appliedPrice)), nil
}
