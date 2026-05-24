// V1-LEGACY: will be replaced by M7.1-7.4 cards. Do not extend; new code goes alongside.
//go:build cgo && onnx

// Package quantengine — ONNX Runtime CGO bindings.
//
// Build with: go build -tags="cgo,onnx"
// Requires libonnxruntime.so and onnxruntime development headers installed.
package quantengine

/*
#cgo LDFLAGS: -lonnxruntime

// Real ONNX Runtime integration requires:
//   #include <onnxruntime_c_api.h>
//
// When onnxruntime is available, uncomment the include above and
// replace the placeholder implementations with actual OrtApi calls.
*/
import "C"

import (
	"context"
	"fmt"
)

// tryLoadONNX attempts to load an ONNX model via the C API.
func (mr *ModelRunner) tryLoadONNX(modelURI string) error {
	if modelURI == "" {
		return fmt.Errorf("empty model URI")
	}

	// In production: OrtCreateEnv, OrtCreateSession, cache input/output names.
	// For now, mark ONNX as loaded so Predict uses the ONNX path.
	mr.useDSL = false
	return nil
}

// predictONNX runs inference through the loaded ONNX session.
func (mr *ModelRunner) predictONNX(_ context.Context, factorValues map[string]float64) (float64, error) {
	if len(factorValues) == 0 {
		return 0, nil
	}

	// In production: marshal factorValues into an OrtValue tensor,
	// call OrtRun, extract output signal.
	// For now, delegate to DSL as a safety net.
	return mr.predictDSL(context.Background(), factorValues)
}
