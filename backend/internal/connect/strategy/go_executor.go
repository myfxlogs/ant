package strategy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
)

// GoExecutor runs a generated Go strategy via go run.
// The generated file must include a main() that reads JSON from stdin
// and writes a signal to stdout.
type GoExecutor struct {
	goModDir string // directory containing go.mod for compilation
	tmpDir   string // temp directory for generated strategy files
	log      *zap.Logger
}

// NewGoExecutor creates a GoExecutor.
func NewGoExecutor(goModDir string, log *zap.Logger) *GoExecutor {
	tmpDir, _ := os.MkdirTemp("", "strategy-*")
	return &GoExecutor{
		goModDir: goModDir,
		tmpDir:   tmpDir,
		log:      log,
	}
}

// ExecuteRequest mirrors the proto ExecuteStrategyRequest for stdin.
type ExecuteRequest struct {
	Symbol    string            `json:"symbol"`
	Timeframe string            `json:"timeframe"`
	Params    map[string]string `json:"params"`
	Bars      []BarJSON         `json:"bars"`
}

// BarJSON is a single OHLCV bar for JSON serialization.
type BarJSON struct {
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
	Time   int64   `json:"time_ms"`
}

// ExecuteResponse is the JSON signal from the strategy.
type ExecuteResponse struct {
	Signal     string  `json:"signal"`     // "buy", "sell", "hold"
	Volume     float64 `json:"volume"`
	Price      float64 `json:"price"`
	StopLoss   float64 `json:"sl"`
	TakeProfit float64 `json:"tp"`
	Comment    string  `json:"comment"`
	Error      string  `json:"error"`
}

// Run compiles and executes a strategy Go file, returning the signal.
func (e *GoExecutor) Run(ctx context.Context, code string, req ExecuteRequest) (*ExecuteResponse, error) {
	// Write the strategy file to temp dir
	strategyFile := filepath.Join(e.tmpDir, "strategy.go")
	if err := os.WriteFile(strategyFile, []byte(code), 0600); err != nil {
		return nil, fmt.Errorf("write strategy: %w", err)
	}
	defer os.Remove(strategyFile)

	// Serialize input as JSON
	input, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}

	// Run: go run strategy.go
	// Path is created by os.WriteFile in our controlled temp directory.
	safePath := filepath.Clean(strategyFile)
	if !filepath.HasPrefix(safePath, e.tmpDir) {
		return nil, fmt.Errorf("strategy file outside temp dir: %s", safePath)
	}
	cmd := exec.CommandContext(ctx, "go", "run", safePath)
	cmd.Dir = e.goModDir
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		e.log.Warn("go run failed", zap.Error(err), zap.String("stderr", stderr.String()))
		return nil, fmt.Errorf("go run: %w\n%s", err, stderr.String())
	}

	var resp ExecuteResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal output: %w\nstdout: %s", err, stdout.String())
	}
	return &resp, nil
}

// Cleanup removes temporary files.
func (e *GoExecutor) Cleanup() {
	os.RemoveAll(e.tmpDir)
}
