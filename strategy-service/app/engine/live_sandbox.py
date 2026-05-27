"""Live sandbox: long-lived worker process with Pipe IPC (B-3.3).

Maintains a persistent worker process that receives pre-compiled bytecode and
reuses it across many ``call()`` invocations.  This avoids the per-call
spawn overhead for live trading where the same strategy processes every
incoming bar.

B-3.5: Bytecode is pre-compiled once in the parent and sent to the worker
on spawn, eliminating RestrictedPython compilation in the child.

Architecture::

    Parent (FastAPI)               Child (spawn)
    LiveWorker.call(ctx)
      │                             _live_worker_main(bytecode, send_pipe, recv_pipe)
      ├─ send_pipe.send(ctx) ────►  receive ctx
      │                              exec bytecode → result
      ├─ poll(self._timeout_ms)     send_pipe.send(result)
      ├─ (timeout) _restart()
      └─ recv() ◄────────────────  loop forever

Restart policy (Q8):
    - Every 500 calls → terminate + respawn
    - Worker RSS > 500 MB → terminate + respawn
"""

from __future__ import annotations

import multiprocessing as mp
from typing import Optional

from app.engine.sandbox_base import BaseSandbox

_CTX = mp.get_context("spawn")

# Q8 thresholds.
_MAX_CALLS_BEFORE_RESTART = 500
_MAX_RSS_MB = 500


class LiveWorker(BaseSandbox):
    """Long-lived sandbox worker for live signal computation.

    Usage::

        worker = LiveWorker(source, timeout_ms=5_000)
        signal = worker.call(ctx)   # blocks until done or timeout
        worker.shutdown()
    """

    def __init__(self, source: str, timeout_ms: int = 5_000) -> None:
        from app.engine.sandbox import (
            code_sha256,
            compile_and_serialize,
            validate_strategy_code,
        )
        from app.engine.types import StrategyCompileError

        validation = validate_strategy_code(source)
        if not validation.valid:
            raise StrategyCompileError("; ".join(validation.errors))

        self._source = source
        self._timeout_ms = timeout_ms
        self._sha256 = code_sha256(source)
        self._call_count = 0
        self._proc: Optional[mp.Process] = None
        self._send_pipe: Optional[mp.connection.Connection] = None
        self._recv_pipe: Optional[mp.connection.Connection] = None
        # B-3.5: pre-compile once, pass to worker on spawn.
        self._serialized_code = compile_and_serialize(source)

        self._spawn()

    # -- BaseSandbox interface ------------------------------------------------

    def call(self, ctx: dict) -> Optional[dict]:
        self._call_count += 1
        if self._call_count >= _MAX_CALLS_BEFORE_RESTART:
            self._restart()

        if self._send_pipe is None:
            self._spawn()

        try:
            self._send_pipe.send(ctx)  # type: ignore[union-attr]
        except (BrokenPipeError, OSError):
            self._restart()
            self._send_pipe.send(ctx)  # type: ignore[union-attr]

        timeout_s = self._timeout_ms / 1000.0
        if not self._recv_pipe.poll(timeout_s):  # type: ignore[union-attr]
            self._restart()
            from app.engine.backtest_sandbox import SandboxTimeoutError
            raise SandboxTimeoutError(
                f"live call exceeded {self._timeout_ms}ms"
            )

        try:
            payload = self._recv_pipe.recv()  # type: ignore[union-attr]
        except EOFError:
            self._restart()
            return None

        if isinstance(payload, dict) and "error" in payload:
            from app.engine.types import StrategyRuntimeError
            raise StrategyRuntimeError(payload["error"])

        return payload

    def shutdown(self) -> None:
        if self._proc is not None and self._proc.is_alive():
            self._proc.terminate()
            self._proc.join(5)
            if self._proc.is_alive():
                self._proc.kill()
                self._proc.join(2)
        self._proc = None
        self._send_pipe = None
        self._recv_pipe = None

    @property
    def source_sha256(self) -> str:
        return self._sha256

    # -- internals -----------------------------------------------------------

    def _spawn(self) -> None:
        parent_recv, child_send = _CTX.Pipe()
        child_recv, parent_send = _CTX.Pipe()

        p = _CTX.Process(
            target=_live_worker_main,
            args=(self._serialized_code, child_send, child_recv),
            name=f"live-sandbox-{self._sha256[:8]}",
        )
        p.start()

        # Parent keeps its end of the pipes; child ends are in the subprocess.
        child_send.close()
        child_recv.close()

        self._proc = p
        self._send_pipe = parent_send
        self._recv_pipe = parent_recv
        self._call_count = 0

    def _restart(self) -> None:
        self.shutdown()
        self._spawn()


# -- Subprocess entry point --------------------------------------------------

def _live_worker_main(serialized_code: bytes, send_pipe, recv_pipe) -> None:
    """Persistent worker: receive pre-compiled bytecode, serve many calls."""
    # Q7: defensive event loop rebuild.
    try:
        import asyncio
        asyncio.set_event_loop(asyncio.new_event_loop())
    except Exception:
        pass

    # Q8: soft memory limit.
    try:
        import resource
        limit = _MAX_RSS_MB * 1024 * 1024
        resource.setrlimit(resource.RLIMIT_AS, (limit, limit))
    except Exception:
        pass

    from app.engine.sandbox import (
        build_sandbox_globals,
        exec_serialized,
    )
    from app.engine.types import StrategyCompileError, StrategyRuntimeError

    # Build globals once — shared across all calls.
    sandbox_globals = build_sandbox_globals()

    try:
        while True:
            ctx = recv_pipe.recv()  # blocks until parent sends context
            try:
                locals_dict = dict(ctx)
                exec_serialized(serialized_code, sandbox_globals, locals_dict)

                run_fn = locals_dict.get("run")
                if callable(run_fn):
                    result = run_fn(dict(ctx))
                elif "signal" in locals_dict:
                    result = _coerce_signal_worker(locals_dict["signal"])
                else:
                    result = None
            except Exception as exc:
                result = {"error": str(exc)}
            send_pipe.send(result)
    except (EOFError, BrokenPipeError, OSError):
        pass
    finally:
        send_pipe.close()
        recv_pipe.close()


def _coerce_signal_worker(value) -> Optional[dict]:
    if value is None:
        return None
    if isinstance(value, dict):
        return value
    if isinstance(value, str):
        return {"signal": value}
    return None
