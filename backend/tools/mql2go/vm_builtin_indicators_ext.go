package mql2go

import (
	"anttrader/tools/mql2go/interp"
)

// VM indicator builtins for the 24 shared/MQL5 indicators that have SDK methods
// but were not yet wired in the VM. These return real values from the SDK context.

// ── Shared MQL4/MQL5 indicators ──────────────────────────────────────

func builtinIAlligator(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	// MQL4: iAlligator(symbol, timeframe, jaw_period, jaw_shift, teeth_period, teeth_shift,
	//   lips_period, lips_shift, ma_method, applied_price, mode, shift)
	jawPeriod := int(argI(args, 2))
	jawShift := int(argI(args, 3))
	teethPeriod := int(argI(args, 4))
	teethShift := int(argI(args, 5))
	lipsPeriod := int(argI(args, 6))
	lipsShift := int(argI(args, 7))
	method := maMethodStr(argI(args, 8))
	appliedPrice := int(argI(args, 9))
	mode := argI(args, 10)
	shift := int(argI(args, 11))
	jaw, teeth, lips := vm.ctx.Indicators().Alligator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, appliedPrice, shift)
	switch mode {
	case 2:
		return interp.DecimalVal(teeth), nil
	case 3:
		return interp.DecimalVal(lips), nil
	default:
		return interp.DecimalVal(jaw), nil
	}
}

func builtinIIchimoku(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	// MQL4: iIchimoku(symbol, timeframe, tenkan_sen, kijun_sen, senkou_span_b, mode, shift)
	tenkan := int(argI(args, 2))
	kijun := int(argI(args, 3))
	senkou := int(argI(args, 4))
	mode := argI(args, 5)
	shift := int(argI(args, 6))
	t, k, sA, sB := vm.ctx.Indicators().Ichimoku(tenkan, kijun, senkou, shift)
	switch mode {
	case 2:
		return interp.DecimalVal(k), nil
	case 3:
		return interp.DecimalVal(sA), nil
	case 4:
		return interp.DecimalVal(sB), nil
	default:
		return interp.DecimalVal(t), nil
	}
}

func builtinIEnvelopes(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	// MQL4: iEnvelopes(symbol, timeframe, ma_period, ma_method, ma_shift,
	//   applied_price, deviation, mode, shift)
	period := int(argI(args, 2))
	method := maMethodStr(argI(args, 3))
	// ma_shift = args[4] — skip (visual offset, not bar shift)
	appliedPrice := int(argI(args, 5))
	deviation := argD(args, 6)
	mode := argI(args, 7)
	shift := int(argI(args, 8))
	upper, lower := vm.ctx.Indicators().Envelopes(period, deviation, method, appliedPrice, shift)
	if mode == 2 {
		return interp.DecimalVal(lower), nil
	}
	return interp.DecimalVal(upper), nil
}

func builtinIDeMarker(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	period := int(argI(args, 2))
	shift := int(argI(args, 3))
	return interp.DecimalVal(vm.ctx.Indicators().DeMarker(period, shift)), nil
}

func builtinIOsMA(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	fast := int(argI(args, 2))
	slow := int(argI(args, 3))
	signal := int(argI(args, 4))
	appliedPrice := int(argI(args, 5))
	shift := int(argI(args, 6))
	return interp.DecimalVal(vm.ctx.Indicators().OsMA(fast, slow, signal, appliedPrice, shift)), nil
}

func builtinIRVI(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	// MQL4: iRVI(symbol, timeframe, period, mode, shift)
	period := int(argI(args, 2))
	// mode = args[3] — skip (SDK returns single value)
	shift := int(argI(args, 4))
	return interp.DecimalVal(vm.ctx.Indicators().RVI(period, shift)), nil
}

func builtinIForce(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	period := int(argI(args, 2))
	method := maMethodStr(argI(args, 3))
	appliedPrice := int(argI(args, 4))
	shift := int(argI(args, 5))
	return interp.DecimalVal(vm.ctx.Indicators().Force(period, method, appliedPrice, shift)), nil
}

func builtinIFractals(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	// MQL4: iFractals(symbol, timeframe, mode, shift)
	mode := argI(args, 2)
	shift := int(argI(args, 3))
	upper, lower := vm.ctx.Indicators().Fractals(shift)
	if mode == 2 {
		return interp.DecimalVal(lower), nil
	}
	return interp.DecimalVal(upper), nil
}

func builtinIGator(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	// MQL4: iGator(symbol, timeframe, jaw_period, jaw_shift, teeth_period, teeth_shift,
	//   lips_period, lips_shift, ma_method, applied_price, mode, shift)
	jawPeriod := int(argI(args, 2))
	jawShift := int(argI(args, 3))
	teethPeriod := int(argI(args, 4))
	teethShift := int(argI(args, 5))
	lipsPeriod := int(argI(args, 6))
	lipsShift := int(argI(args, 7))
	method := maMethodStr(argI(args, 8))
	appliedPrice := int(argI(args, 9))
	mode := argI(args, 10)
	shift := int(argI(args, 11))
	upper, lower := vm.ctx.Indicators().Gator(jawPeriod, jawShift, teethPeriod, teethShift, lipsPeriod, lipsShift, method, appliedPrice, shift)
	if mode == 2 {
		return interp.DecimalVal(lower), nil
	}
	return interp.DecimalVal(upper), nil
}

func builtinIAC(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	shift := int(argI(args, 2))
	return interp.DecimalVal(vm.ctx.Indicators().AC(shift)), nil
}

func builtinIAD(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	shift := int(argI(args, 2))
	return interp.DecimalVal(vm.ctx.Indicators().AD(shift)), nil
}

func builtinIAO(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	shift := int(argI(args, 2))
	return interp.DecimalVal(vm.ctx.Indicators().AO(shift)), nil
}

func builtinIBearsPower(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	period := int(argI(args, 2))
	appliedPrice := int(argI(args, 3))
	shift := int(argI(args, 4))
	return interp.DecimalVal(vm.ctx.Indicators().BearsPower(period, appliedPrice, shift)), nil
}

func builtinIBullsPower(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	period := int(argI(args, 2))
	appliedPrice := int(argI(args, 3))
	shift := int(argI(args, 4))
	return interp.DecimalVal(vm.ctx.Indicators().BullsPower(period, appliedPrice, shift)), nil
}

func builtinIBWMFI(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	shift := int(argI(args, 2))
	return interp.DecimalVal(vm.ctx.Indicators().BWMFI(shift)), nil
}

// ── MQL5-only indicators ─────────────────────────────────────────────

func builtinIAMA(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	period := int(argI(args, 2))
	fastPeriod := int(argI(args, 3))
	slowPeriod := int(argI(args, 4))
	appliedPrice := int(argI(args, 6))
	shift := int(argI(args, 7))
	return interp.DecimalVal(vm.ctx.Indicators().AMA(period, fastPeriod, slowPeriod, appliedPrice, shift)), nil
}

func builtinIDEMA(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	period := int(argI(args, 2))
	appliedPrice := int(argI(args, 4))
	shift := int(argI(args, 5))
	return interp.DecimalVal(vm.ctx.Indicators().DEMA(period, appliedPrice, shift)), nil
}

func builtinITEMA(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	period := int(argI(args, 2))
	appliedPrice := int(argI(args, 4))
	shift := int(argI(args, 5))
	return interp.DecimalVal(vm.ctx.Indicators().TEMA(period, appliedPrice, shift)), nil
}

func builtinIFrAMA(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	period := int(argI(args, 2))
	appliedPrice := int(argI(args, 4))
	shift := int(argI(args, 5))
	return interp.DecimalVal(vm.ctx.Indicators().FrAMA(period, appliedPrice, shift)), nil
}

func builtinIVIDyA(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	cmoPeriod := int(argI(args, 2))
	cmoShift := int(argI(args, 3))
	maPeriod := int(argI(args, 4))
	maShift := int(argI(args, 5))
	appliedPrice := int(argI(args, 6))
	shift := int(argI(args, 7))
	return interp.DecimalVal(vm.ctx.Indicators().VIDyA(cmoPeriod, cmoShift, maPeriod, maShift, appliedPrice, shift)), nil
}

func builtinITriX(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	period := int(argI(args, 2))
	appliedPrice := int(argI(args, 4))
	shift := int(argI(args, 5))
	return interp.DecimalVal(vm.ctx.Indicators().TriX(period, appliedPrice, shift)), nil
}

func builtinIADXWilder(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	period := int(argI(args, 2))
	shift := int(argI(args, 3))
	return interp.DecimalVal(vm.ctx.Indicators().ADXWilder(period, shift)), nil
}

func builtinIChaikin(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	fastPeriod := int(argI(args, 2))
	slowPeriod := int(argI(args, 3))
	shift := int(argI(args, 4))
	return interp.DecimalVal(vm.ctx.Indicators().Chaikin(fastPeriod, slowPeriod, shift)), nil
}

func builtinIVolumes(vm *VM, args []interp.Value) (interp.Value, error) {
	if vm.ctx == nil {
		return interp.DecimalVal(decimalZero), nil
	}
	shift := int(argI(args, 2))
	return interp.DecimalVal(vm.ctx.Indicators().Volumes(shift)), nil
}

// maMethodStr converts an MQL moving average method constant to its string name.
func maMethodStr(method int32) string {
	switch method {
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
