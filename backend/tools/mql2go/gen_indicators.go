package mql2go

// ── Parameter emission ───────────────────────────────────────────

func (g *generator) emitParam(p ParamSpec) {
	dflt := p.Default
	switch p.Type {
	case ParamInt:
		if dflt == "" {
			dflt = "0"
		}
		g.emitf("s.%s = int32(ctx.ParamInt(\"%s\", %s))",
			p.Name, p.Name, dflt)
	case ParamDouble:
		if dflt == "" {
			dflt = "0"
		}
		g.emitf("s.%s = ctx.ParamDecimal(\"%s\", decimal.NewFromFloat(%s))",
			p.Name, p.Name, dflt)
	case ParamString:
		if dflt == "" {
			dflt = `""`
		}
		g.emitf("s.%s = ctx.ParamString(\"%s\", \"%s\")",
			p.Name, p.Name, stripQuotes(dflt))
	case ParamBool:
		if dflt == "true" || dflt == "1" {
			dflt = "true"
		} else {
			dflt = "false"
		}
		g.emitf("s.%s = ctx.ParamBool(\"%s\", %s)", p.Name, p.Name, dflt)
	}
}

// ── Indicator emission ────────────────────────────────────────

func (g *generator) emitIndicator(spec IndicatorSpec) {
	if spec.ResultVar == "" {
		return
	}
	varName := mqlToGoExpr(spec.ResultVar)
	period := spec.Params["period"]
	if period == "" {
		period = "14"
	}
	shift := spec.Params["shift"]
	if shift == "" || shift == "0" {
		shift = "1" // default: most recent completed bar
	}
	period = mqlToGoExpr(period)
	if !isNumeric(period) {
		period = "int(" + prefixRef(period) + ")"
	}
	shift = mqlToGoExpr(shift)
	if !isNumeric(shift) {
		shift = "int(" + prefixRef(shift) + ")"
	}

	switch spec.SDKMethod {
	case "ma", "ema":
		g.emitf("%s := ctx.Indicators().EMA(%s, %s)", varName, period, shift)
	case "rsi":
		g.emitf("%s := ctx.Indicators().RSI(%s, %s)", varName, period, shift)
	case "atr":
		g.emitf("%s := ctx.Indicators().ATR(%s, %s)", varName, period, shift)
	case "macd":
		fast := mqlToGoExpr(spec.Params["fast"])
		slow := mqlToGoExpr(spec.Params["slow"])
		sig := mqlToGoExpr(spec.Params["signal"])
		if fast == "" {
			fast = "12"
		}
		if slow == "" {
			slow = "26"
		}
		if sig == "" {
			sig = "9"
		}
		g.emitf("%s := ctx.Indicators().MACD(%s, %s, %s, %s)", varName, intCast(fast), intCast(slow), intCast(sig), shift)
	case "bands":
		deviation := mqlToGoExpr(spec.Params["deviation"])
		if deviation == "" {
			deviation = "2.0"
		}
		g.emitf("%sUpper, %sMid, %sLower := ctx.Indicators().Bollinger(%s, decimal.NewFromFloat(%s), %s)",
			varName, varName, varName, period, deviation, shift)
		g.emitf("_ = %sUpper", varName)
		g.emitf("_ = %sMid", varName)
		g.emitf("_ = %sLower", varName)
	case "stochastic":
		kPeriod := mqlToGoExpr(spec.Params["kperiod"])
		dPeriod := mqlToGoExpr(spec.Params["dperiod"])
		slowing := mqlToGoExpr(spec.Params["slowing"])
		if kPeriod == "" {
			kPeriod = "5"
		}
		if dPeriod == "" {
			dPeriod = "3"
		}
		if slowing == "" {
			slowing = "3"
		}
		g.emitf("%sK, %sD := ctx.Indicators().Stochastic(%s, %s, %s, %s)",
			varName, varName, intCast(kPeriod), intCast(dPeriod), intCast(slowing), shift)
		g.emitf("_ = %sK", varName)
		g.emitf("_ = %sD", varName)
	case "cci":
		g.emitf("%s := ctx.Indicators().CCI(%s, %s)", varName, period, shift)
	case "adx":
		g.emitf("%s := ctx.Indicators().ADX(%s, %s)", varName, period, shift)
	case "momentum":
		g.emitf("%s := ctx.Indicators().Momentum(%s, %s)", varName, period, shift)
	case "wpr":
		g.emitf("%s := ctx.Indicators().WPR(%s, %s)", varName, period, shift)
	case "mfi":
		g.emitf("%s := ctx.Indicators().MFI(%s, %s)", varName, period, shift)
	case "obv":
		g.emitf("%s := ctx.Indicators().OBV(%s)", varName, shift)
	case "sar":
		step := mqlToGoExpr(spec.Params["step"])
		maximum := mqlToGoExpr(spec.Params["maximum"])
		if step == "" {
			step = "0.02"
		}
		if maximum == "" {
			maximum = "0.2"
		}
		g.emitf("%s := ctx.Indicators().SAR(decimal.NewFromFloat(%s), decimal.NewFromFloat(%s), %s)", varName, step, maximum, shift)
	case "stddev":
		g.emitf("%s := ctx.Indicators().StdDev(%s, %s)", varName, period, shift)
	case "alligator":
		jaw := mqlToGoExpr(spec.Params["jaw_period"])
		teeth := mqlToGoExpr(spec.Params["teeth_period"])
		lips := mqlToGoExpr(spec.Params["lips_period"])
		if jaw == "" {
			jaw = "13"
		}
		if teeth == "" {
			teeth = "8"
		}
		if lips == "" {
			lips = "5"
		}
		g.emitf("%sJaw, %sTeeth, %sLips := ctx.Indicators().Alligator(%s, 8, %s, 5, %s, 3, \"sma\", 0, %s)",
			varName, varName, varName, intCast(jaw), intCast(teeth), intCast(lips), shift)
		g.emitf("_ = %sJaw", varName)
		g.emitf("_ = %sTeeth", varName)
		g.emitf("_ = %sLips", varName)
	case "ichimoku":
		tenkan := mqlToGoExpr(spec.Params["tenkan"])
		kijun := mqlToGoExpr(spec.Params["kijun"])
		senkouB := mqlToGoExpr(spec.Params["senkou_b"])
		if tenkan == "" {
			tenkan = "9"
		}
		if kijun == "" {
			kijun = "26"
		}
		if senkouB == "" {
			senkouB = "52"
		}
		g.emitf("%sTenkan, %sKijun, %sSenkouA, %sSenkouB := ctx.Indicators().Ichimoku(%s, %s, %s, %s)",
			varName, varName, varName, varName, intCast(tenkan), intCast(kijun), intCast(senkouB), shift)
		g.emitf("_ = %sTenkan", varName)
		g.emitf("_ = %sKijun", varName)
		g.emitf("_ = %sSenkouA", varName)
		g.emitf("_ = %sSenkouB", varName)
	case "envelopes":
		deviation := mqlToGoExpr(spec.Params["deviation"])
		if deviation == "" {
			deviation = "0.1"
		}
		g.emitf("%sUpper, %sLower := ctx.Indicators().Envelopes(%s, decimal.NewFromFloat(%s), \"sma\", 0, %s)",
			varName, varName, period, deviation, shift)
		g.emitf("_ = %sUpper", varName)
		g.emitf("_ = %sLower", varName)
	case "demarker":
		g.emitf("%s := ctx.Indicators().DeMarker(%s, %s)", varName, period, shift)
	case "osma":
		fast := mqlToGoExpr(spec.Params["fast"])
		slow := mqlToGoExpr(spec.Params["slow"])
		sig := mqlToGoExpr(spec.Params["signal"])
		if fast == "" {
			fast = "12"
		}
		if slow == "" {
			slow = "26"
		}
		if sig == "" {
			sig = "9"
		}
		g.emitf("%s := ctx.Indicators().OsMA(%s, %s, %s, 0, %s)", varName, intCast(fast), intCast(slow), intCast(sig), shift)
	case "rvi":
		g.emitf("%s := ctx.Indicators().RVI(%s, %s)", varName, period, shift)
	case "force":
		g.emitf("%s := ctx.Indicators().Force(%s, \"sma\", 0, %s)", varName, period, shift)
	case "fractals":
		g.emitf("%sUpper, %sLower := ctx.Indicators().Fractals(%s)", varName, varName, shift)
		g.emitf("_ = %sUpper", varName)
		g.emitf("_ = %sLower", varName)
	case "gator":
		jaw := mqlToGoExpr(spec.Params["jaw_period"])
		teeth := mqlToGoExpr(spec.Params["teeth_period"])
		lips := mqlToGoExpr(spec.Params["lips_period"])
		if jaw == "" {
			jaw = "13"
		}
		if teeth == "" {
			teeth = "8"
		}
		if lips == "" {
			lips = "5"
		}
		g.emitf("%sUp, %sDown := ctx.Indicators().Gator(%s, 8, %s, 5, %s, 3, \"sma\", 0, %s)",
			varName, varName, intCast(jaw), intCast(teeth), intCast(lips), shift)
		g.emitf("_ = %sUp", varName)
		g.emitf("_ = %sDown", varName)
	case "ac":
		g.emitf("%s := ctx.Indicators().AC(%s)", varName, shift)
	case "ad":
		g.emitf("%s := ctx.Indicators().AD(%s)", varName, shift)
	case "ao":
		g.emitf("%s := ctx.Indicators().AO(%s)", varName, shift)
	case "bears_power":
		g.emitf("%s := ctx.Indicators().BearsPower(%s, 0, %s)", varName, period, shift)
	case "bulls_power":
		g.emitf("%s := ctx.Indicators().BullsPower(%s, 0, %s)", varName, period, shift)
	case "bwmfi":
		g.emitf("%s := ctx.Indicators().BWMFI(%s)", varName, shift)
	case "ama":
		fast := mqlToGoExpr(spec.Params["fast"])
		slow := mqlToGoExpr(spec.Params["slow"])
		if fast == "" {
			fast = "2"
		}
		if slow == "" {
			slow = "30"
		}
		g.emitf("%s := ctx.Indicators().AMA(%s, %s, %s, %s)", varName, period, intCast(fast), intCast(slow), shift)
	case "dema":
		g.emitf("%s := ctx.Indicators().DEMA(%s, %s)", varName, period, shift)
	case "tema":
		g.emitf("%s := ctx.Indicators().TEMA(%s, %s)", varName, period, shift)
	case "frama":
		g.emitf("%s := ctx.Indicators().FrAMA(%s, %s)", varName, period, shift)
	case "vidya":
		cmoPeriod := mqlToGoExpr(spec.Params["cmo_period"])
		if cmoPeriod == "" {
			cmoPeriod = "9"
		}
		g.emitf("%s := ctx.Indicators().VIDyA(%s, 0, %s, 0, %s)", varName, cmoPeriod, period, shift)
	case "trix":
		g.emitf("%s := ctx.Indicators().TriX(%s, %s)", varName, period, shift)
	case "adx_wilder":
		g.emitf("%s := ctx.Indicators().ADXWilder(%s, %s)", varName, period, shift)
	case "chaikin":
		fast := mqlToGoExpr(spec.Params["fast"])
		slow := mqlToGoExpr(spec.Params["slow"])
		if fast == "" {
			fast = "3"
		}
		if slow == "" {
			slow = "10"
		}
		g.emitf("%s := ctx.Indicators().Chaikin(%s, %s, %s)", varName, intCast(fast), intCast(slow), shift)
	case "volumes":
		g.emitf("%s := ctx.Indicators().Volumes(%s)", varName, shift)
	}
}
