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
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

// LiveSession manages a long-running WASM strategy instance that processes
// bars via piped proto streaming. The strategy is compiled to WASM once;
// OnInit is called once; OnBar is called for each streamed bar.
type LiveSession struct {
	wasm    *WasmExecutor
	code    string
	log     *zap.Logger

	// WASM runtime state for the active session.
	compiled  wazero.CompiledModule
	stdinW    *io.PipeWriter // host → WASM (requests)
	stdoutR   *io.PipeReader // WASM → host (responses)
	stderrBuf *lockedBuffer  // WASM stderr capture

	mod    api.Module
	cancel context.CancelFunc
	done   chan error

	started bool
}

// lockedBuffer is a thread-safe bytes.Buffer for stderr capture.
type lockedBuffer struct {
	buf []byte
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *lockedBuffer) String() string {
	return string(b.buf)
}

// NewLiveSession creates a LiveSession that executes strategy code via WASM.
func NewLiveSession(wasm *WasmExecutor, code string, log *zap.Logger) *LiveSession {
	return &LiveSession{
		wasm: wasm,
		code: code,
		log:  log,
	}
}

// Start compiles the strategy to WASM and runs the harness. The harness
// processes the initial request (OnInit + first OnBar), writes the response,
// then blocks on stdin waiting for the next bar.
func (s *LiveSession) Start(ctx context.Context, reqBytes []byte) ([]byte, error) {
	if s.started {
		return nil, fmt.Errorf("live session already started")
	}

	// Find strategy type name in the generated code.
	strategyType, err := findStrategyTypeName(s.code)
	if err != nil {
		return nil, fmt.Errorf("find strategy type: %w", err)
	}

	// Compile Go strategy code to WASM.
	compiled, hash, err := s.wasm.CompileStrategy(ctx, s.code, strategyType)
	if err != nil {
		return nil, fmt.Errorf("compile strategy: %w", err)
	}
	s.compiled = compiled

	// Create bidirectional pipes for stdio.
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	s.stdinW = stdinW
	s.stdoutR = stdoutR
	s.stderrBuf = &lockedBuffer{}

	// Prepend the initial request to stdin so the harness reads it first.
	// After the initial request is consumed, stdinR (pipe) takes over,
	// blocking the harness until the next bar request is written.
	stdin := io.MultiReader(
		&lengthPrefixedReader{data: reqBytes},
		stdinR,
	)

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
		s.mod = mod
		if mod != nil {
			// Module exited — close it.
			mod.Close(modCtx)
		}
		s.done <- err
	}()

	// Read the first response. The harness processes the initial request,
	// writes the response to stdoutW, then blocks on stdinR. The host is
	// unblocked when data arrives on stdoutR.
	resp, err := s.readPipeResponse()
	if err != nil {
		cancel()
		// Drain the done channel.
		<-s.done
		return nil, fmt.Errorf("read initial response: %w", err)
	}

	s.started = true

	s.log.Info("LiveSession: WASM session started",
		zap.String("strategy_type", strategyType),
		zap.String("hash", hash))

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

	// Write the bar request to stdin pipe — unblocks the harness from
	// its blocking readRequest(stdin).
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

// ── lengthPrefixedReader ──────────────────────────────────────────────

// lengthPrefixedReader wraps raw bytes as a length-prefixed stream,
// matching the format the harness expects: 4-byte BE length + data.
type lengthPrefixedReader struct {
	data   []byte
	offset int
	prefix bool // have we written the length prefix yet?
}

func (r *lengthPrefixedReader) Read(p []byte) (int, error) {
	if r.data == nil {
		return 0, io.EOF
	}
	if !r.prefix {
		// Write the 4-byte length prefix first.
		r.prefix = true
		var lenBuf [4]byte
		binary.BigEndian.PutUint32(lenBuf[:], uint32(len(r.data)))
		n := copy(p, lenBuf[:])
		if n < 4 {
			// Buffer too small — offset for next read.
			return n, nil
		}
		// Prefix fully written; write data if space remains.
		dn := copy(p[4:], r.data)
		r.offset = dn
		if dn == len(r.data) {
			r.data = nil // consumed
			return 4 + dn, io.EOF
		}
		return 4 + dn, nil
	}
	// Writing remaining data.
	n := copy(p, r.data[r.offset:])
	r.offset += n
	if r.offset >= len(r.data) {
		r.data = nil
		return n, io.EOF
	}
	return n, nil
}
