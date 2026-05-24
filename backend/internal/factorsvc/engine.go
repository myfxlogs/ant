// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
package factorsvc

import (
	"context"
	"fmt"
	"sync"

	"anttrader/internal/factor/dsl"
)

// FactorDef defines a factor loaded from configuration.
type FactorDef struct {
	Name       string
	Expression string
	Symbols    []string
}

// Config holds the factor engine configuration.
type Config struct {
	Factors []FactorDef
}

// Engine manages factor computation.
type Engine struct {
	mu         sync.RWMutex
	factors    map[string]*compiledFactor
	compiler   *dsl.Compiler
	latestVals map[string]float64
	buffer     *WindowBuffer
}

type compiledFactor struct {
	def FactorDef
	op  dsl.Op
}

// NewEngine creates a new factor engine.
func NewEngine(cfg Config) *Engine {
	fields := dsl.FieldIndex{Fields: map[string]int{
		"open": 0, "high": 1, "low": 2, "close": 3, "volume": 4, "bid": 5, "ask": 6,
	}}
	e := &Engine{
		factors:  make(map[string]*compiledFactor),
		compiler: dsl.NewCompiler(fields, nil),
	}
	for _, f := range cfg.Factors {
		_ = e.Register(f) //nolint:errcheck
	}
	return e
}

// SetBuffer attaches a WindowBuffer for rolling window factor computation.
func (e *Engine) SetBuffer(buf *WindowBuffer) {
	e.mu.Lock()
	e.buffer = buf
	e.mu.Unlock()
}

// Register compiles and registers a factor definition.
func (e *Engine) Register(def FactorDef) error {
	op, err := e.compiler.Compile(def.Expression)
	if err != nil {
		return fmt.Errorf("register factor %q: %w", def.Name, err)
	}
	e.mu.Lock()
	e.factors[def.Name] = &compiledFactor{def: def, op: op}
	e.mu.Unlock()
	return nil
}

// Eval evaluates all registered factors for a given bar, returning name→value.
// Results are cached internally and available via LatestFactors().
// Uses WindowBuffer for rolling-window operators (SMA, EMA, RSI).
func (e *Engine) Eval(ctx context.Context, bar *Bar) map[string]float64 {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.buffer != nil {
		e.buffer.Push(bar.UserID, bar.Symbol, bar.Period, bar)
	}

	results := make(map[string]float64, len(e.factors))
	for name, cf := range e.factors {
		closeVals := e.windowedCloses(bar)
		var val float64
		if len(closeVals) > 0 {
			val = rollingEval(cf.op, closeVals)
		} else {
			v, _ := parseFloat(bar.Close)
			val = cf.op.Eval(v)
		}
		results[name] = val
	}
	e.latestVals = results
	return results
}

func (e *Engine) windowedCloses(bar *Bar) []float64 {
	if e.buffer == nil {
		return nil
	}
	records := e.buffer.Snapshot(bar.UserID, bar.Symbol, bar.Period, 200)
	if len(records) == 0 {
		return nil
	}
	closes := make([]float64, len(records))
	for i, r := range records {
		closes[i] = r.Close
	}
	return closes
}

func rollingEval(op dsl.Op, closes []float64) float64 {
	if len(closes) == 0 {
		return 0
	}
	var result float64
	for _, c := range closes {
		result = op.Eval(c)
	}
	return result
}

// LatestFactors returns factor values from the most recent Eval call.
func (e *Engine) LatestFactors() map[string]float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(e.latestVals) == 0 {
		return nil
	}
	out := make(map[string]float64, len(e.latestVals))
	for k, v := range e.latestVals {
		out[k] = v
	}
	return out
}

func parseFloat(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	var f float64
	n, err := fmt.Sscanf(s, "%f", &f)
	return f, n == 1 && err == nil
}
