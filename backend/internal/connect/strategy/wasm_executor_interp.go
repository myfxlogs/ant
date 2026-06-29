package strategy

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tetratelabs/wazero"
	"go.uber.org/zap"
)

// CompileInterpBacktest compiles the interp backtest harness to a WASM module.
// The harness is static (no per-strategy code) — compiled once and cached.
// The IR is passed at runtime via stdin.
func (w *WasmExecutor) CompileInterpBacktest(ctx context.Context) (wazero.CompiledModule, string, error) {
	if err := w.initRuntime(ctx); err != nil {
		return nil, "", err
	}

	w.interpBtOnce.Do(func() {
		wasmBytes, err := w.buildInterpWasm(ctx, generateInterpBacktestHarness())
		if err != nil {
			w.interpBtErr = fmt.Errorf("build interp backtest wasm: %w", err)
			return
		}

		compiled, err := w.runtime.CompileModule(ctx, wasmBytes)
		if err != nil {
			w.interpBtErr = fmt.Errorf("compile interp backtest wasm: %w", err)
			return
		}

		w.interpBtModule = compiled
		hash := wasmHash(wasmBytes)
		w.log.Info("WasmExecutor: interp backtest harness compiled",
			zap.String("hash", hash),
			zap.Int("wasm_bytes", len(wasmBytes)))
	})

	if w.interpBtErr != nil {
		return nil, "", w.interpBtErr
	}
	return w.interpBtModule, "", nil
}

// CompileInterpLive compiles the interp live harness to a WASM module.
// The harness is static — compiled once and cached. The IR is passed at runtime via stdin.
func (w *WasmExecutor) CompileInterpLive(ctx context.Context) (wazero.CompiledModule, string, error) {
	if err := w.initRuntime(ctx); err != nil {
		return nil, "", err
	}

	w.interpLiveOnce.Do(func() {
		wasmBytes, err := w.buildInterpWasm(ctx, generateInterpLiveHarness())
		if err != nil {
			w.interpLiveErr = fmt.Errorf("build interp live wasm: %w", err)
			return
		}

		compiled, err := w.runtime.CompileModule(ctx, wasmBytes)
		if err != nil {
			w.interpLiveErr = fmt.Errorf("compile interp live wasm: %w", err)
			return
		}

		w.interpLiveModule = compiled
		hash := wasmHash(wasmBytes)
		w.log.Info("WasmExecutor: interp live harness compiled",
			zap.String("hash", hash),
			zap.Int("wasm_bytes", len(wasmBytes)))
	})

	if w.interpLiveErr != nil {
		return nil, "", w.interpLiveErr
	}
	return w.interpLiveModule, "", nil
}

// RunInterpBacktest runs the interp backtest harness with the given IR and backtest request.
// stdin = u32 LE IR length + IR bytes + raw proto backtest request.
// stdout = raw proto backtest response.
func (w *WasmExecutor) RunInterpBacktest(ctx context.Context, compiled wazero.CompiledModule, irBytes []byte, reqBytes []byte) ([]byte, error) {
	if err := w.initRuntime(ctx); err != nil {
		return nil, err
	}

	stdinData := append(irLengthPrefix(irBytes), reqBytes...)
	stdin := bytes.NewReader(stdinData)
	var stdout, stderr bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdin(stdin).
		WithStdout(&stdout).
		WithStderr(&stderr).
		WithSysNanosleep().
		WithSysNanotime().
		WithName("interp-backtest")

	mod, err := w.runtime.InstantiateModule(ctx, compiled, config)
	if err != nil {
		w.log.Warn("wasm interp backtest failed",
			zap.Error(err),
			zap.String("stderr", stderr.String()))
		return nil, fmt.Errorf("wasm interp backtest: %w\n%s", err, stderr.String())
	}
	defer mod.Close(ctx)

	return stdout.Bytes(), nil
}

// RunInterpLive runs the interp live harness with the given IR and live request.
// stdin = u32 LE IR length + IR bytes + length-prefixed proto live request.
// stdout = length-prefixed proto live response.
func (w *WasmExecutor) RunInterpLive(ctx context.Context, compiled wazero.CompiledModule, irBytes []byte, reqBytes []byte) ([]byte, error) {
	if err := w.initRuntime(ctx); err != nil {
		return nil, err
	}

	liveReqPrefixed := make([]byte, 4+len(reqBytes))
	binary.BigEndian.PutUint32(liveReqPrefixed[:4], uint32(len(reqBytes)))
	copy(liveReqPrefixed[4:], reqBytes)

	stdinData := append(irLengthPrefix(irBytes), liveReqPrefixed...)
	stdin := bytes.NewReader(stdinData)
	var stdout, stderr bytes.Buffer

	config := wazero.NewModuleConfig().
		WithStdin(stdin).
		WithStdout(&stdout).
		WithStderr(&stderr).
		WithSysNanosleep().
		WithSysNanotime().
		WithName("interp-live")

	mod, err := w.runtime.InstantiateModule(ctx, compiled, config)
	if err != nil {
		w.log.Warn("wasm interp live failed",
			zap.Error(err),
			zap.String("stderr", stderr.String()))
		return nil, fmt.Errorf("wasm interp live: %w\n%s", err, stderr.String())
	}
	defer mod.Close(ctx)

	return stdout.Bytes(), nil
}

// buildInterpWasm compiles a single harness Go source file to a WASM binary.
// No strategy code needed — the IR is passed at runtime via stdin.
func (w *WasmExecutor) buildInterpWasm(ctx context.Context, harnessCode string) ([]byte, error) {
	runDir, err := os.MkdirTemp(w.tmpDir, "interp-wasm-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(runDir)

	harnessFile := filepath.Join(runDir, "harness.go")
	if err := os.WriteFile(harnessFile, []byte(harnessCode), 0600); err != nil {
		return nil, fmt.Errorf("write harness: %w", err)
	}

	outFile := filepath.Join(runDir, "strategy.wasm")

	cmd := exec.CommandContext(ctx, "go",
		"build",
		"-o", outFile,
		harnessFile,
	)
	cmd.Dir = w.goModDir
	cmd.Env = append(os.Environ(),
		"GOOS=wasip1",
		"GOARCH=wasm",
	)

	var buildStderr bytes.Buffer
	cmd.Stderr = &buildStderr

	if err := cmd.Run(); err != nil {
		w.log.Warn("go build interp wasm failed",
			zap.Error(err),
			zap.String("stderr", buildStderr.String()))
		return nil, fmt.Errorf("go build interp wasm: %w\n%s", err, buildStderr.String())
	}

	wasmBytes, err := os.ReadFile(outFile)
	if err != nil {
		return nil, fmt.Errorf("read wasm binary: %w", err)
	}

	return wasmBytes, nil
}
