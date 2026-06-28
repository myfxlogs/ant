// live_session.go — LiveSession manages a long-running Go strategy execution
// via WASM (wazero). The strategy compiles once to wasip1/wasm; subsequent
// bars are streamed through in-memory pipes — no subprocess, no recompilation.
//
// Architecture:
//
//	Host goroutine          WASM goroutine (harness)
//	─────────────          ──────────────────────
//	InstantiateModule ──→  main():
//	                          readRequest(stdin) ─── blocks on pipe
//	write(initialReq) ──→      unblocks, processes OnInit + first OnBar
//	                          writeResponse(stdout)
//	readResponse() ←────      readRequest(stdin) ─── blocks on pipe
//	write(barReq) ─────→      unblocks, processes OnBar
//	                          writeResponse(stdout)
//	readResponse() ←────      readRequest(stdin) ─── blocks
//	... (loop per bar)

package strategy

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/tetratelabs/wazero"
	"go.uber.org/zap"
)

// LiveSession manages a long-running WASM strategy instance that processes
// bars via piped proto streaming. The strategy is compiled to WASM once;
// OnInit is called once; OnBar is called for each streamed bar.
type LiveSession struct {
	wasm    *WasmExecutor
	code    string
	irBytes []byte // if non-nil, use interpreter path instead of compiled strategy
	log     *zap.Logger

	// WASM runtime state for the active session.
	stdinW    *io.PipeWriter // host → WASM (requests)
	stdoutR   *io.PipeReader // WASM → host (responses)
	stderrBuf *lockedBuffer  // WASM stderr capture

	cancel context.CancelFunc
	done   chan error

	started bool
}

// lockedBuffer is a mutex-protected byte buffer for stderr capture.
// WASM goroutine writes, host goroutine reads.
type lockedBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

// NewLiveSession creates a LiveSession for a compiled Go strategy.
func NewLiveSession(wasm *WasmExecutor, code string, log *zap.Logger) *LiveSession {
	return &LiveSession{
		wasm: wasm,
		code: code,
		log:  log,
	}
}

// NewInterpLiveSession creates a LiveSession for an MQL interpreter strategy.
// irBytes is the serialized IR passed to the harness via stdin at startup.
func NewInterpLiveSession(wasm *WasmExecutor, irBytes []byte, log *zap.Logger) *LiveSession {
	return &LiveSession{
		wasm:    wasm,
		irBytes: irBytes,
		log:     log,
	}
}

// Start compiles the strategy to WASM and runs the harness. The harness
// processes the initial request (OnInit + first OnBar), writes the response,
// then blocks on stdin waiting for the next bar.
func (s *LiveSession) Start(ctx context.Context, reqBytes []byte) ([]byte, error) {
	if s.started {
		return nil, fmt.Errorf("live session already started")
	}

	var compiled wazero.CompiledModule
	var hash string

	if s.irBytes != nil {
		// Interpreter path: use cached interp harness, prepend IR to stdin.
		var err error
		compiled, hash, err = s.wasm.CompileInterpLive(ctx)
		if err != nil {
			return nil, fmt.Errorf("compile interp harness: %w", err)
		}
	} else {
		// Compiled path: find strategy type and compile to WASM.
		strategyType, err := findStrategyTypeName(s.code)
		if err != nil {
			return nil, fmt.Errorf("find strategy type: %w", err)
		}
		compiled, hash, err = s.wasm.CompileStrategy(ctx, s.code, strategyType)
		if err != nil {
			return nil, fmt.Errorf("compile strategy: %w", err)
		}
	}
	// Create bidirectional pipes for stdio.
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	s.stdinW = stdinW
	s.stdoutR = stdoutR
	s.stderrBuf = &lockedBuffer{}

	// Build stdin: interp path prepends IR (u32 LE length + IR bytes)
	// before the first length-prefixed request. After initial data is
	// consumed, stdinR (pipe) takes over for subsequent bar requests.
	var reqPrefix [4]byte
	binary.BigEndian.PutUint32(reqPrefix[:], uint32(len(reqBytes)))
	var stdin io.Reader
	if s.irBytes != nil {
		stdin = io.MultiReader(
			bytes.NewReader(irLengthPrefix(s.irBytes)),
			bytes.NewReader(reqPrefix[:]),
			bytes.NewReader(reqBytes),
			stdinR,
		)
	} else {
		stdin = io.MultiReader(
			bytes.NewReader(reqPrefix[:]),
			bytes.NewReader(reqBytes),
			stdinR,
		)
	}

	// Configure WASI with piped stdio.
	config := wazero.NewModuleConfig().
		WithStdin(stdin).
		WithStdout(stdoutW).
		WithStderr(s.stderrBuf).
		WithSysNanosleep().
		WithSysNanotime().
		WithName("strategy")

	// Launch WASM module in a goroutine — InstantiateModule blocks until
	// the module exits (harness main() returns).
	modCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan error, 1)

	go func() {
		mod, err := s.wasm.runtime.InstantiateModule(modCtx, compiled, config)
		if mod != nil {
			mod.Close(modCtx)
		}
		s.done <- err
	}()

	// Read the first response. The harness processes the initial request,
	// writes the response to stdoutW, then blocks on stdinR.
	resp, err := s.readPipeResponse()
	if err != nil {
		cancel()
		<-s.done
		return nil, fmt.Errorf("read initial response: %w", err)
	}

	s.started = true

	if s.irBytes != nil {
		s.log.Info("LiveSession: interp session started",
			zap.String("hash", hash))
	} else {
		s.log.Info("LiveSession: WASM session started",
			zap.String("hash", hash))
	}

	return resp, nil
}

// SendBar sends a single bar request to the WASM harness and reads the response.
// This is the per-bar hot path — no compilation, no IPC, just pipe I/O.
func (s *LiveSession) SendBar(reqBytes []byte) ([]byte, error) {
	if !s.started {
		return nil, fmt.Errorf("live session not started")
	}

	// Check if the WASM module has exited unexpectedly.
	select {
	case err := <-s.done:
		s.started = false
		if err != nil {
			return nil, fmt.Errorf("wasm session exited: %w", err)
		}
		return nil, fmt.Errorf("wasm session exited unexpectedly")
	default:
	}

	// Write length-prefixed bar request to stdin pipe — unblocks the
	// harness from its blocking readRequest(stdin).
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(reqBytes)))
	if _, err := s.stdinW.Write(lenBuf[:]); err != nil {
		return nil, fmt.Errorf("write bar length prefix: %w", err)
	}
	if _, err := s.stdinW.Write(reqBytes); err != nil {
		return nil, fmt.Errorf("write bar request: %w", err)
	}

	// Read the response from stdout pipe.
	resp, err := s.readPipeResponse()
	if err != nil {
		return nil, fmt.Errorf("read bar response: %w\nstderr: %s", err, s.stderrBuf.String())
	}

	return resp, nil
}

// Close terminates the WASM session. Closes stdin to signal EOF to the
// harness, which triggers OnDeinit and main() exit.
func (s *LiveSession) Close() error {
	if !s.started {
		return nil
	}
	s.started = false

	// Close stdin pipe — signals EOF to the harness, which calls OnDeinit
	// and returns from main().
	if s.stdinW != nil {
		s.stdinW.Close()
	}
	if s.cancel != nil {
		s.cancel()
	}

	// Wait for WASM module to exit.
	if s.done != nil {
		<-s.done
	}

	return nil
}

// readPipeResponse reads a length-prefixed protobuf message from the stdout pipe.
// Format: 4-byte big-endian length, then message bytes.
func (s *LiveSession) readPipeResponse() ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(s.stdoutR, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read length prefix: %w", err)
	}
	msgLen := binary.BigEndian.Uint32(lenBuf[:])
	if msgLen > 256*1024*1024 { // 256MB sanity cap
		return nil, fmt.Errorf("response too large: %d bytes", msgLen)
	}
	buf := make([]byte, msgLen)
	if _, err := io.ReadFull(s.stdoutR, buf); err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}
	return buf, nil
}
