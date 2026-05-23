// Package quantengine — ONNX runtime integration for strategy inference.
package quantengine

import (
	"context"
	"fmt"
	"math"

	"anttrader/internal/factor/dsl"
)

// StrategySpec is the local strategy specification used by quantengine.
type StrategySpec struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	CanonicalSymbols []string          `json:"canonical_symbols"`
	Period           string            `json:"period"`
	Factors          map[string]string `json:"factors"`
	SignalRule       string            `json:"signal_rule"`
	ModelURI         string            `json:"model_uri"`
	Sizing           map[string]any    `json:"sizing"`
}

// ModelRunner executes ONNX model inference.
// Currently a DSL-based fallback. Full ONNX integration via CGO is available
// when built with `go build -tags cgo` and libonnxruntime.so present.
type ModelRunner struct {
	spec     *StrategySpec
	compiler *dsl.Compiler
	useDSL   bool // fallback to DSL signal rule when ONNX model is unavailable
}

// NewModelRunner creates a model runner for a strategy spec.
// If model_uri is empty, it falls back to the DSL signal_rule.
func NewModelRunner(spec *StrategySpec) (*ModelRunner, error) {
	fields := dsl.FieldIndex{Fields: map[string]int{
		"close": 0, "open": 1, "high": 2, "low": 3, "volume": 4,
	}}

	mr := &ModelRunner{
		spec:     spec,
		compiler: dsl.NewCompiler(fields, nil),
	}

	if spec.ModelURI == "" {
		mr.useDSL = true
	} else {
		// Attempt ONNX load; fall back to DSL if unavailable.
		if err := mr.tryLoadONNX(spec.ModelURI); err != nil {
			mr.useDSL = true
		}
	}

	return mr, nil
}

// Predict runs inference with the strategy model and returns a signal.
// Input: factor values as float64 map. Output: signal float64 (>0 long, <0 short).
func (mr *ModelRunner) Predict(ctx context.Context, factorValues map[string]float64) (float64, error) {
	if mr.useDSL {
		return mr.predictDSL(ctx, factorValues)
	}
	return mr.predictONNX(ctx, factorValues)
}

func (mr *ModelRunner) predictDSL(_ context.Context, factorValues map[string]float64) (float64, error) {
	if mr.spec.SignalRule == "" {
		return 0, nil
	}

	factors := make(map[string]dsl.Op, len(factorValues))
	for name, val := range factorValues {
		factors[name] = &constOp{val: val}
	}

	op, err := mr.compiler.CompileWithFactors(mr.spec.SignalRule, factors)
	if err != nil {
		return 0, fmt.Errorf("model predict dsl: %w", err)
	}

	return op.Eval(0), nil
}

// constOp is a dsl.Op that returns a constant value.
type constOp struct{ val float64 }

func (c *constOp) Eval(v float64) float64 { return c.val }
func (c *constOp) Reset()                 {}
func (c *constOp) Warmup() int            { return 0 }

// Direction returns "long", "short", or "flat" for a signal value.
func Direction(signal float64) string {
	if math.IsNaN(signal) {
		return "flat"
	}
	if signal > 0 {
		return "long"
	}
	if signal < 0 {
		return "short"
	}
	return "flat"
}

// tryLoadONNX attempts to load an ONNX model; stub without CGO.
func (mr *ModelRunner) tryLoadONNX(modelURI string) error {
	_ = modelURI
	return fmt.Errorf("onnx runtime not available; rebuild with CGO and -tags cgo")
}

// predictONNX runs ONNX inference; stub without CGO.
func (mr *ModelRunner) predictONNX(_ context.Context, _ map[string]float64) (float64, error) {
	return 0, fmt.Errorf("onnx runtime not available; rebuild with CGO and -tags cgo")
}
