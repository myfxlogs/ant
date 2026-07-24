package strategy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// GoExecutor runs a generated Go strategy via go run.
// Uses protobuf binary encoding for IPC — no JSON.
type GoExecutor struct {
	goModDir string
	tmpDir   string
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

// Run compiles and executes a strategy Go file.
// Input/output use antv1 proto binary encoding.
func (e *GoExecutor) Run(ctx context.Context, code string, req *antv1.ExecuteStrategyRequest) (*antv1.ExecuteStrategyResponse, error) {
	runDir, err := os.MkdirTemp(e.tmpDir, "run-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(runDir) }()

	strategyFile := filepath.Join(runDir, "strategy.go")
	if err := os.WriteFile(strategyFile, []byte(code), 0600); err != nil {
		return nil, fmt.Errorf("write strategy: %w", err)
	}

	input, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	cmd := exec.CommandContext(ctx, "go")
	cmd.Args = []string{"go", "run", strategyFile}
	cmd.Dir = e.goModDir
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		e.log.Warn("go run failed", zap.Error(err), zap.String("stderr", stderr.String()))
		return nil, fmt.Errorf("go run: %w\n%s", err, stderr.String())
	}

	var resp antv1.ExecuteStrategyResponse
	if err := proto.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

// RunBacktest compiles strategy code + harness and runs a backtest.
// Input/output use antv1 proto binary encoding.
func (e *GoExecutor) RunBacktest(ctx context.Context, code string, req *antv1.ExecuteBacktestRequest) (*antv1.ExecuteBacktestResponse, error) {
	strategyType, err := findStrategyTypeName(code)
	if err != nil {
		return nil, fmt.Errorf("find strategy type: %w", err)
	}

	runDir, err := os.MkdirTemp(e.tmpDir, "bt-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(runDir) }()

	strategyFile := filepath.Join(runDir, "strategy.go")
	if err := os.WriteFile(strategyFile, []byte(code), 0600); err != nil {
		return nil, fmt.Errorf("write strategy: %w", err)
	}

	harnessFile := filepath.Join(runDir, "harness.go")
	if err := os.WriteFile(harnessFile, []byte(generateBacktestHarness(strategyType)), 0600); err != nil {
		return nil, fmt.Errorf("write harness: %w", err)
	}

	input, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	cmd := exec.CommandContext(ctx, "go")
	cmd.Args = []string{"go", "run", strategyFile, harnessFile}
	cmd.Dir = e.goModDir
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		e.log.Warn("go run backtest failed", zap.Error(err), zap.String("stderr", stderr.String()))
		return nil, fmt.Errorf("go run: %w\n%s", err, stderr.String())
	}

	var resp antv1.ExecuteBacktestResponse
	if err := proto.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

// CompileCheck writes strategy code to a temp file and runs `go vet` to verify
// it compiles. Returns (true, "") if compilation succeeds, (false, stderr) otherwise.
func (e *GoExecutor) CompileCheck(ctx context.Context, code string) (bool, string) {
	runDir, err := os.MkdirTemp(e.tmpDir, "check-*")
	if err != nil {
		return false, fmt.Sprintf("create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(runDir) }()

	strategyFile := filepath.Join(runDir, "strategy.go")
	if err := os.WriteFile(strategyFile, []byte(code), 0600); err != nil {
		return false, fmt.Sprintf("write strategy: %v", err)
	}

	cmd := exec.CommandContext(ctx, "go")
	cmd.Args = []string{"go", "vet", strategyFile}
	cmd.Dir = e.goModDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, stderr.String()
	}
	return true, ""
}

func (e *GoExecutor) Cleanup() {
	_ = os.RemoveAll(e.tmpDir)
}

// RunLive compiles strategy code + live harness and runs a single-bar live evaluation.
// Input/output use antv1 proto binary encoding.
func (e *GoExecutor) RunLive(ctx context.Context, code string, req *antv1.ExecuteLiveRequest) (*antv1.ExecuteLiveResponse, error) {
	strategyType, err := findStrategyTypeName(code)
	if err != nil {
		return nil, fmt.Errorf("find strategy type: %w", err)
	}

	runDir, err := os.MkdirTemp(e.tmpDir, "live-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(runDir) }()

	strategyFile := filepath.Join(runDir, "strategy.go")
	if err := os.WriteFile(strategyFile, []byte(code), 0600); err != nil {
		return nil, fmt.Errorf("write strategy: %w", err)
	}

	harnessFile := filepath.Join(runDir, "harness.go")
	if err := os.WriteFile(harnessFile, []byte(generateLiveHarness(strategyType)), 0600); err != nil {
		return nil, fmt.Errorf("write harness: %w", err)
	}

	input, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	cmd := exec.CommandContext(ctx, "go")
	cmd.Args = []string{"go", "run", strategyFile, harnessFile}
	cmd.Dir = e.goModDir
	cmd.Stdin = bytes.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		e.log.Warn("go run live failed", zap.Error(err), zap.String("stderr", stderr.String()))
		return nil, fmt.Errorf("go run: %w\n%s", err, stderr.String())
	}

	var resp antv1.ExecuteLiveResponse
	if err := proto.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}
