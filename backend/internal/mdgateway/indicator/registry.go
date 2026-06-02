package indicator

import (
	"fmt"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// Registry maps indicator ID → {Def, ComputeFunc}.
var Registry = map[string]struct {
	Def     Def
	Compute ComputeFunc
}{
	"SMA":    {DefSMA, computeSMA},
	"EMA":    {DefEMA, computeEMA},
	"BOLL":   {DefBollinger, computeBollinger},
	"RSI":    {DefRSI, computeRSI},
	"MACD":   {DefMACD, computeMACD},
	"ATR":    {DefATR, computeATR},
	"CCI":    {DefCCI, computeCCI},
	"WILLR":  {DefWilliamsR, computeWilliamsR},
	"MFI":    {DefMFI, computeMFI},
	"ADX":    {DefADX, computeADX},
	"OBV":    {DefOBV, computeOBV},
	"ADOSC":  {DefADOSC, computeADOSC},
	"AD":     {DefAD, computeAD},
	"KDJ":    {DefKDJ, computeKDJ},
	"VOL":    {DefVolume, computeVOL},
}

// List returns all registered indicator definitions.
func List() []Def {
	out := make([]Def, 0, len(Registry))
	for _, v := range Registry {
		out = append(out, v.Def)
	}
	return out
}

// Compute calculates a single indicator from bar data.
func Compute(id string, bars []mdtick.Bar, params map[string]float64) (*Result, error) {
	entry, ok := Registry[id]
	if !ok {
		return nil, fmt.Errorf("indicator: unknown id %q", id)
	}
	// Merge user params with defaults
	merged := make(map[string]float64, len(entry.Def.Defaults)+len(params))
	for k, v := range entry.Def.Defaults {
		merged[k] = v
	}
	for k, v := range params {
		merged[k] = v
	}
	return entry.Compute(bars, merged)
}
