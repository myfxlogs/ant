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

	antv1 "anttrader/gen/proto/ant/v1"
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
	strategyFile := filepath.Join(e.tmpDir, "strategy.go")
	if err := os.WriteFile(strategyFile, []byte(code), 0600); err != nil {
		return nil, fmt.Errorf("write strategy: %w", err)
	}
	defer os.Remove(strategyFile)

	// Serialize request as proto binary
	input, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// #nosec G204 — path validated below
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

	var resp antv1.ExecuteStrategyResponse
	if err := proto.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

func (e *GoExecutor) Cleanup() {
	os.RemoveAll(e.tmpDir)
}
