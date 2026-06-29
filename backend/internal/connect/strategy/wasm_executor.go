// wasm_executor.go — WASM-based strategy execution via wazero.
//
// Replaces the subprocess model (go run via exec.Cmd) with in-process WASM
// sandbox execution. Strategy Go code compiles to wasip1/wasm, loaded by
// wazero, and executed with WASI stdio piped through in-memory buffers.
//
// Benefits over subprocess model:
//   - No process spawn overhead (~5-10s go run → ~200ms wasm compile)
//   - wazero CompilationCache eliminates recompilation of unchanged code
//   - In-process execution — no pipe I/O, no kernel context switches
//   - WASM sandbox — memory isolation without process isolation
//   - Same harness code — no harness changes needed

package strategy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/tetratelabs/wazero"
	"go.uber.org/zap"
)

// WasmExecutor compiles and runs Go strategy code as WASM via wazero.
// Safe for concurrent use — the runtime is shared, compilation is synchronized.
type WasmExecutor struct {
	goModDir string
	tmpDir   string
	log      *zap.Logger

	// wazero runtime — created once, shared across all executions.
	runtime wazero.Runtime
	// Compilation cache — survives across sessions. wazero's built-in cache
	// stores compiled native code keyed by wasm module hash.
	cache wazero.CompilationCache

	initOnce sync.Once
	initErr  error

	// Interp harness caching — the harness code is static (no per-strategy
	// code), so the WASM binary is identical every call. Compile once, reuse.
	interpBtOnce   sync.Once
	interpBtModule wazero.CompiledModule
	interpBtErr    error

	interpLiveOnce   sync.Once
	interpLiveModule wazero.CompiledModule
	interpLiveErr    error
}

// NewWasmExecutor creates a WasmExecutor.
// goModDir is the directory containing go.mod for SDK dependencies.
func NewWasmExecutor(goModDir string, log *zap.Logger) *WasmExecutor {
	tmpDir, _ := os.MkdirTemp("", "wasm-strategy-*")
	return &WasmExecutor{
		goModDir: goModDir,
		tmpDir:   tmpDir,
		log:      log,
	}
}

// initRuntime lazily initializes the wazero runtime and compilation cache.
func (w *WasmExecutor) initRuntime(ctx context.Context) error {
	w.initOnce.Do(func() {
		// Create a compilation cache directory for persistence across restarts.
		cacheDir := filepath.Join(w.tmpDir, "wasm-cache")
		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			w.initErr = fmt.Errorf("wasm cache dir: %w", err)
			return
		}

		w.cache, w.initErr = wazero.NewCompilationCacheWithDir(cacheDir)
		if w.initErr != nil {
			return
		}

		w.runtime = wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().
			WithCompilationCache(w.cache))

		w.log.Info("WasmExecutor: runtime initialized",
			zap.String("cache_dir", cacheDir))
	})
	return w.initErr
}

// CompileStrategy compiles strategy Go source code to a WASM module.
// Returns the compiled module ready for instantiation.
// The compilation is cached by wazero keyed on the wasm binary content hash.
func (w *WasmExecutor) CompileStrategy(ctx context.Context, code string, strategyTypeName string) (wazero.CompiledModule, string, error) {
	if err := w.initRuntime(ctx); err != nil {
		return nil, "", err
	}

	// Compile Go source to WASM binary.
	wasmBytes, err := w.buildWasm(ctx, code, strategyTypeName)
	if err != nil {
		return nil, "", fmt.Errorf("build wasm: %w", err)
	}

	// Compile WASM module (wazero caches this internally).
	compiled, err := w.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, "", fmt.Errorf("compile wasm module: %w", err)
	}

	// Compute a stable hash for debugging (first 16 chars of SHA256 of wasm bytes).
	hash := wasmHash(wasmBytes)

	w.log.Info("WasmExecutor: strategy compiled",
		zap.String("hash", hash),
		zap.Int("wasm_bytes", len(wasmBytes)))

	return compiled, hash, nil
}

// buildWasm compiles strategy code + harness to a WASM binary.
func (w *WasmExecutor) buildWasm(ctx context.Context, code string, strategyTypeName string) ([]byte, error) {
	runDir, err := os.MkdirTemp(w.tmpDir, "wasm-build-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(runDir)

	strategyFile := filepath.Join(runDir, "strategy.go")
	if err := os.WriteFile(strategyFile, []byte(code), 0600); err != nil {
		return nil, fmt.Errorf("write strategy: %w", err)
	}

	harnessFile := filepath.Join(runDir, "harness.go")
	if err := os.WriteFile(harnessFile, []byte(generateLiveHarness(strategyTypeName)), 0600); err != nil {
		return nil, fmt.Errorf("write harness: %w", err)
	}

	outFile := filepath.Join(runDir, "strategy.wasm")

	// Build Go source to WASM targeting wasip1/WASM.
	// GOOS=wasip1 GOARCH=wasm go build -o strategy.wasm strategy.go harness.go
	cmd := exec.CommandContext(ctx, "go",
		"build",
		"-o", outFile,
		strategyFile, harnessFile,
	)
	cmd.Dir = w.goModDir
	cmd.Env = append(os.Environ(),
		"GOOS=wasip1",
		"GOARCH=wasm",
	)

	var buildStderr bytes.Buffer
	cmd.Stderr = &buildStderr

	if err := cmd.Run(); err != nil {
		w.log.Warn("go build wasm failed",
			zap.Error(err),
			zap.String("stderr", buildStderr.String()))
		return nil, fmt.Errorf("go build wasm: %w\n%s", err, buildStderr.String())
	}

	wasmBytes, err := os.ReadFile(outFile)
	if err != nil {
		return nil, fmt.Errorf("read wasm binary: %w", err)
	}

	return wasmBytes, nil
}


// Close releases the wazero runtime and compilation cache.
func (w *WasmExecutor) Close() error {
	w.initOnce.Do(func() {}) // ensure initOnce is consumed
	if w.runtime != nil {
		if err := w.runtime.Close(context.Background()); err != nil {
			return err
		}
	}
	if w.cache != nil {
		if err := w.cache.Close(context.Background()); err != nil {
			return err
		}
	}
	return os.RemoveAll(w.tmpDir)
}

// ── WASM binary hash ──────────────────────────────────────────────────

// wasmHash returns a short hex hash of the wasm binary for logging/debugging.
// Uses a simple FNV-1a hash for speed — not cryptographic.
func wasmHash(data []byte) string {
	if len(data) == 0 {
		return "empty"
	}
	// Simple FNV-1a 64-bit
	var h uint64 = 14695981039346656037
	for _, b := range data {
		h ^= uint64(b)
		h *= 1099511628211
	}
	return fmt.Sprintf("%016x", h)
}
