package ai

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSandboxScanLengthAligned(t *testing.T) {
	if maxTransformCodeLen < 65536 {
		t.Errorf("maxTransformCodeLen=%d too small; expected 65536", maxTransformCodeLen)
	}
	if maxCodeLen < maxTransformCodeLen {
		t.Errorf("maxCodeLen=%d must be >= maxTransformCodeLen=%d", maxCodeLen, maxTransformCodeLen)
	}
}

// TestTransformCodePromptRejectsPythonAPI ensures the actual TransformCode
// source does not reference deprecated Python SDK methods.
func TestTransformCodePromptRejectsPythonAPI(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("could not get test file path")
	}
	source, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "code_assist_transform.go"))
	if err != nil {
		t.Skip("could not read source file:", err)
	}

	banned := []string{
		"self.close_all(",
		"self.sma(",
		"def run(context):",
		"StrategyBase",
		"self.broker.order_send",
		"self.ctx.param",
	}

	for _, keyword := range banned {
		if strings.Contains(string(source), keyword) {
			t.Errorf("TransformCode source contains OLD Python API method: %q", keyword)
		}
	}
}
