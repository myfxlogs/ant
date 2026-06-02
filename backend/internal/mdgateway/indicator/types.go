// Package indicator provides pure Go technical indicator calculation functions.
// All calculations use decimal.Decimal for price precision.
// This package has no dependencies on proto, SSE, or NATS — it is a pure math library.
package indicator

import (
	"github.com/shopspring/decimal"

	"anttrader/internal/mdgateway/adapter/mdtick"
)

// Kind classifies an indicator for frontend rendering.
type Kind string

const (
	KindOverlay Kind = "overlay" // rendered on main price chart (MA, Bollinger)
	KindSubPane Kind = "sub"     // rendered in separate sub-pane (RSI, MACD, VOL)
)

// Param describes a configurable indicator parameter.
type Param struct {
	Key      string
	Label    string
	Type     string  // "int", "float"
	Default  float64
	Min      float64
	Max      float64
	Step     float64
}

// Def describes an indicator's identity, parameters, and rendering hints.
type Def struct {
	ID       string
	Name     string
	Kind     Kind
	Params   []Param
	Defaults map[string]float64
}

// Result holds computed indicator values for a range of bars.
// Values aligns 1:1 with the input bars slice — index i corresponds to bars[i].
type Result struct {
	DefID string
	// Lines maps line name → values (one per input bar).
	// E.g. {"sma": [21.5, 21.6, ...]} or {"macd": [...], "signal": [...], "histogram": [...]}.
	Lines map[string][]decimal.Decimal
}

// ComputeFunc is the signature every indicator implementation must satisfy.
type ComputeFunc func(bars []mdtick.Bar, params map[string]float64) (*Result, error)
