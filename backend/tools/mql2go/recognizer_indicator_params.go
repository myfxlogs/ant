package mql2go

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// ── Indicator parameter extraction ──────────────────────────────────

func extractIndicatorParams(name string, version string, named []*sitter.Node, params map[string]string) {
	switch name {
	case "iMA":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["shift"] = nodeText("", named[3])
		}
	case "iRSI":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["shift"] = nodeText("", named[4])
		}
	case "iATR":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["shift"] = nodeText("", named[3])
		}
	case "iMACD":
		if len(named) > 2 {
			params["fast"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["slow"] = nodeText("", named[3])
		}
		if len(named) > 4 {
			params["signal"] = nodeText("", named[4])
		}
		if len(named) > 5 {
			params["shift"] = nodeText("", named[5])
		}
	case "iBands":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["deviation"] = nodeText("", named[3])
		}
		if len(named) > 5 {
			params["shift"] = nodeText("", named[5])
		}
	case "iStochastic":
		if len(named) > 2 {
			params["kperiod"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["dperiod"] = nodeText("", named[3])
		}
		if len(named) > 4 {
			params["slowing"] = nodeText("", named[4])
		}
		if len(named) > 6 {
			params["shift"] = nodeText("", named[6])
		}
	case "iCCI":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["shift"] = nodeText("", named[4])
		}
	case "iADX":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["shift"] = nodeText("", named[3])
		}
	case "iMomentum":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["shift"] = nodeText("", named[4])
		}
	case "iWPR":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["shift"] = nodeText("", named[4])
		}
	case "iMFI":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["shift"] = nodeText("", named[4])
		}
	case "iOBV":
		if len(named) > 1 {
			params["shift"] = nodeText("", named[1])
		}
	case "iSAR":
		if len(named) > 2 {
			params["step"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["maximum"] = nodeText("", named[3])
		}
		if len(named) > 4 {
			params["shift"] = nodeText("", named[4])
		}
	case "iStdDev":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["shift"] = nodeText("", named[4])
		}
	case "iAlligator":
		if len(named) > 2 {
			params["jaw_period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["teeth_period"] = nodeText("", named[4])
		}
		if len(named) > 6 {
			params["lips_period"] = nodeText("", named[6])
		}
		if len(named) > 10 {
			params["shift"] = nodeText("", named[10])
		}
	case "iIchimoku":
		if len(named) > 2 {
			params["tenkan"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["kijun"] = nodeText("", named[3])
		}
		if len(named) > 4 {
			params["senkou_b"] = nodeText("", named[4])
		}
		if len(named) > 5 {
			params["shift"] = nodeText("", named[5])
		}
	case "iEnvelopes":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 5 {
			params["deviation"] = nodeText("", named[5])
		}
		if len(named) > 7 {
			params["shift"] = nodeText("", named[7])
		}
	case "iDeMarker":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["shift"] = nodeText("", named[3])
		}
	case "iOsMA":
		if len(named) > 2 {
			params["fast"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["slow"] = nodeText("", named[3])
		}
		if len(named) > 4 {
			params["signal"] = nodeText("", named[4])
		}
		if len(named) > 6 {
			params["shift"] = nodeText("", named[6])
		}
	case "iRVI":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["shift"] = nodeText("", named[4])
		}
	case "iForce":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 5 {
			params["shift"] = nodeText("", named[5])
		}
	case "iFractals":
		if len(named) > 3 {
			params["shift"] = nodeText("", named[3])
		}
	case "iGator":
		if len(named) > 2 {
			params["jaw_period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["teeth_period"] = nodeText("", named[4])
		}
		if len(named) > 6 {
			params["lips_period"] = nodeText("", named[6])
		}
		if len(named) > 10 {
			params["shift"] = nodeText("", named[10])
		}
	case "iAC":
		if len(named) > 2 {
			params["shift"] = nodeText("", named[2])
		}
	case "iAD":
		if len(named) > 2 {
			params["shift"] = nodeText("", named[2])
		}
	case "iAO":
		if len(named) > 2 {
			params["shift"] = nodeText("", named[2])
		}
	case "iBearsPower":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["shift"] = nodeText("", named[4])
		}
	case "iBullsPower":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["shift"] = nodeText("", named[4])
		}
	case "iBWMFI":
		if len(named) > 2 {
			params["shift"] = nodeText("", named[2])
		}
	}
	if version == "mql5" {
		extractMQL5IndicatorParams(name, named, params)
	}
}

func extractMQL5IndicatorParams(name string, named []*sitter.Node, params map[string]string) {
	switch name {
	case "iAMA":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["fast"] = nodeText("", named[3])
		}
		if len(named) > 4 {
			params["slow"] = nodeText("", named[4])
		}
		if len(named) > 7 {
			params["shift"] = nodeText("", named[7])
		}
	case "iDEMA":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 5 {
			params["shift"] = nodeText("", named[5])
		}
	case "iTEMA":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 5 {
			params["shift"] = nodeText("", named[5])
		}
	case "iFrAMA":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 5 {
			params["shift"] = nodeText("", named[5])
		}
	case "iVIDyA":
		if len(named) > 2 {
			params["cmo_period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["period"] = nodeText("", named[4])
		}
		if len(named) > 7 {
			params["shift"] = nodeText("", named[7])
		}
	case "iTriX":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 5 {
			params["shift"] = nodeText("", named[5])
		}
	case "iADXWilder":
		if len(named) > 2 {
			params["period"] = nodeText("", named[2])
		}
		if len(named) > 4 {
			params["shift"] = nodeText("", named[4])
		}
	case "iChaikin":
		if len(named) > 2 {
			params["fast"] = nodeText("", named[2])
		}
		if len(named) > 3 {
			params["slow"] = nodeText("", named[3])
		}
		if len(named) > 6 {
			params["shift"] = nodeText("", named[6])
		}
	case "iVolumes":
		if len(named) > 3 {
			params["shift"] = nodeText("", named[3])
		}
	}
}

func indicatorMethodCST(name string, version string) string {
	switch name {
	case "iMA":
		return "ema"
	case "iRSI":
		return "rsi"
	case "iATR":
		return "atr"
	case "iBands":
		return "bands"
	case "iMACD":
		return "macd"
	case "iStochastic":
		return "stochastic"
	case "iCCI":
		return "cci"
	case "iADX":
		return "adx"
	case "iMomentum":
		return "momentum"
	case "iWPR":
		return "wpr"
	case "iMFI":
		return "mfi"
	case "iOBV":
		return "obv"
	case "iSAR":
		return "sar"
	case "iStdDev":
		return "stddev"
	case "iCustom":
		return "i_custom"
	case "iAlligator":
		return "alligator"
	case "iIchimoku":
		return "ichimoku"
	case "iEnvelopes":
		return "envelopes"
	case "iDeMarker":
		return "demarker"
	case "iOsMA":
		return "osma"
	case "iRVI":
		return "rvi"
	case "iForce":
		return "force"
	case "iFractals":
		return "fractals"
	case "iGator":
		return "gator"
	case "iAC":
		return "ac"
	case "iAD":
		return "ad"
	case "iAO":
		return "ao"
	case "iBearsPower":
		return "bears_power"
	case "iBullsPower":
		return "bulls_power"
	case "iBWMFI":
		return "bwmfi"
	}
	if version != "mql5" {
		return ""
	}
	switch name {
	case "iAMA":
		return "ama"
	case "iDEMA":
		return "dema"
	case "iTEMA":
		return "tema"
	case "iFrAMA":
		return "frama"
	case "iVIDyA":
		return "vidya"
	case "iTriX":
		return "trix"
	case "iADXWilder":
		return "adx_wilder"
	case "iChaikin":
		return "chaikin"
	case "iVolumes":
		return "volumes"
	}
	return ""
}

// ── Utility functions ───────────────────────────────────────────────

func getNamedChildren(n *sitter.Node) []*sitter.Node {
	var out []*sitter.Node
	if n == nil {
		return out
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		out = append(out, n.NamedChild(i))
	}
	return out
}

func containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
