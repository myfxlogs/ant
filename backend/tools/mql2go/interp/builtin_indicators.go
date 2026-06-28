package interp

// callIndicator dispatches MQL indicator functions to the SDK IndicatorSet.
// Covers all 30+ MQL4/MQL5 indicators.

func (it *Interpreter) callIndicator(name string, args []Expr) (Value, bool) {
	if it.ctx == nil || it.ctx.Indicators() == nil {
		return NoneVal(), false
	}
	ind := it.ctx.Indicators()
	vals := make([]Value, len(args))
	for i := range args {
		vals[i] = it.evalExpr(&args[i])
	}

	switch name {
	// ── Fully implemented indicators ─────────────────────────────────
	case "iMA":
		if len(vals) >= 3 {
			period := int(vals[0].ToInt())
			shift := int(vals[1].ToInt())
			method := maMethodName(vals[2].ToInt())
			return DecimalVal(ind.MA(period, shift, method)), true
		}
	case "iRSI":
		if len(vals) >= 2 {
			return DecimalVal(ind.RSI(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iATR":
		if len(vals) >= 2 {
			return DecimalVal(ind.ATR(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iMACD":
		if len(vals) >= 4 {
			return DecimalVal(ind.MACD(int(vals[0].ToInt()), int(vals[1].ToInt()), int(vals[2].ToInt()), int(vals[3].ToInt()))), true
		}
	case "iBands", "iBollinger":
		if len(vals) >= 3 {
			period := int(vals[0].ToInt())
			dev := vals[1].ToDecimal()
			shift := int(vals[2].ToInt())
			upper, middle, lower := ind.Bollinger(period, dev, shift)
			// MQL iBands returns upper/lower/middle by mode; we return middle for now
			_ = upper
			_ = lower
			return DecimalVal(middle), true
		}
	case "iStochastic":
		if len(vals) >= 4 {
			k, _ := ind.Stochastic(int(vals[0].ToInt()), int(vals[1].ToInt()), int(vals[2].ToInt()), int(vals[3].ToInt()))
			return DecimalVal(k), true
		}
	case "iCCI":
		if len(vals) >= 2 {
			return DecimalVal(ind.CCI(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iADX":
		if len(vals) >= 2 {
			return DecimalVal(ind.ADX(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iMFI":
		if len(vals) >= 2 {
			return DecimalVal(ind.MFI(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iOBV":
		if len(vals) >= 1 {
			return DecimalVal(ind.OBV(int(vals[0].ToInt()))), true
		}
	case "iSAR":
		if len(vals) >= 3 {
			return DecimalVal(ind.SAR(vals[0].ToDecimal(), vals[1].ToDecimal(), int(vals[2].ToInt()))), true
		}
	case "iStdDev":
		if len(vals) >= 2 {
			return DecimalVal(ind.StdDev(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iWPR":
		if len(vals) >= 2 {
			return DecimalVal(ind.WPR(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iMomentum":
		if len(vals) >= 2 {
			return DecimalVal(ind.Momentum(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}

	// ── Shared MQL4/MQL5 indicators (SDK stubs) ──────────────────────
	case "iAlligator":
		if len(vals) >= 9 {
			jaw, _, _ := ind.Alligator(
				int(vals[0].ToInt()), int(vals[1].ToInt()),
				int(vals[2].ToInt()), int(vals[3].ToInt()),
				int(vals[4].ToInt()), int(vals[5].ToInt()),
				maMethodName(vals[6].ToInt()), int(vals[7].ToInt()), int(vals[8].ToInt()),
			)
			return DecimalVal(jaw), true
		}
	case "iIchimoku":
		if len(vals) >= 4 {
			tenkan, _, _, _ := ind.Ichimoku(int(vals[0].ToInt()), int(vals[1].ToInt()), int(vals[2].ToInt()), int(vals[3].ToInt()))
			return DecimalVal(tenkan), true
		}
	case "iEnvelopes":
		if len(vals) >= 5 {
			upper, _ := ind.Envelopes(int(vals[0].ToInt()), vals[1].ToDecimal(), maMethodName(vals[2].ToInt()), int(vals[3].ToInt()), int(vals[4].ToInt()))
			return DecimalVal(upper), true
		}
	case "iDeMarker":
		if len(vals) >= 2 {
			return DecimalVal(ind.DeMarker(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iOsMA":
		if len(vals) >= 5 {
			return DecimalVal(ind.OsMA(int(vals[0].ToInt()), int(vals[1].ToInt()), int(vals[2].ToInt()), int(vals[3].ToInt()), int(vals[4].ToInt()))), true
		}
	case "iRVI":
		if len(vals) >= 2 {
			return DecimalVal(ind.RVI(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iForce":
		if len(vals) >= 4 {
			return DecimalVal(ind.Force(int(vals[0].ToInt()), maMethodName(vals[1].ToInt()), int(vals[2].ToInt()), int(vals[3].ToInt()))), true
		}
	case "iFractals":
		if len(vals) >= 1 {
			upper, _ := ind.Fractals(int(vals[0].ToInt()))
			return DecimalVal(upper), true
		}
	case "iGator":
		if len(vals) >= 9 {
			upper, _ := ind.Gator(
				int(vals[0].ToInt()), int(vals[1].ToInt()),
				int(vals[2].ToInt()), int(vals[3].ToInt()),
				int(vals[4].ToInt()), int(vals[5].ToInt()),
				maMethodName(vals[6].ToInt()), int(vals[7].ToInt()), int(vals[8].ToInt()),
			)
			return DecimalVal(upper), true
		}
	case "iAC":
		if len(vals) >= 1 {
			return DecimalVal(ind.AC(int(vals[0].ToInt()))), true
		}
	case "iAD":
		if len(vals) >= 1 {
			return DecimalVal(ind.AD(int(vals[0].ToInt()))), true
		}
	case "iAO":
		if len(vals) >= 1 {
			return DecimalVal(ind.AO(int(vals[0].ToInt()))), true
		}
	case "iBearsPower":
		if len(vals) >= 3 {
			return DecimalVal(ind.BearsPower(int(vals[0].ToInt()), int(vals[1].ToInt()), int(vals[2].ToInt()))), true
		}
	case "iBullsPower":
		if len(vals) >= 3 {
			return DecimalVal(ind.BullsPower(int(vals[0].ToInt()), int(vals[1].ToInt()), int(vals[2].ToInt()))), true
		}
	case "iBWMFI":
		if len(vals) >= 1 {
			return DecimalVal(ind.BWMFI(int(vals[0].ToInt()))), true
		}

	// ── MQL5-only indicators ─────────────────────────────────────────
	case "iAMA":
		if len(vals) >= 4 {
			return DecimalVal(ind.AMA(int(vals[0].ToInt()), int(vals[1].ToInt()), int(vals[2].ToInt()), int(vals[3].ToInt()))), true
		}
	case "iDEMA":
		if len(vals) >= 2 {
			return DecimalVal(ind.DEMA(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iTEMA":
		if len(vals) >= 2 {
			return DecimalVal(ind.TEMA(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iFrAMA":
		if len(vals) >= 2 {
			return DecimalVal(ind.FrAMA(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iVIDyA":
		if len(vals) >= 5 {
			return DecimalVal(ind.VIDyA(int(vals[0].ToInt()), int(vals[1].ToInt()), int(vals[2].ToInt()), int(vals[3].ToInt()), int(vals[4].ToInt()))), true
		}
	case "iTriX":
		if len(vals) >= 2 {
			return DecimalVal(ind.TriX(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iADXWilder":
		if len(vals) >= 2 {
			return DecimalVal(ind.ADXWilder(int(vals[0].ToInt()), int(vals[1].ToInt()))), true
		}
	case "iChaikin":
		if len(vals) >= 3 {
			return DecimalVal(ind.Chaikin(int(vals[0].ToInt()), int(vals[1].ToInt()), int(vals[2].ToInt()))), true
		}
	case "iVolumes":
		if len(vals) >= 1 {
			return DecimalVal(ind.Volumes(int(vals[0].ToInt()))), true
		}
	}
	return NoneVal(), false
}

// maMethodName maps MQL MA method int to string.
// 0=sma, 1=ema, 2=smma, 3=lwma
func maMethodName(method int32) string {
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
