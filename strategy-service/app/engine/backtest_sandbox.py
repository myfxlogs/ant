"""Backtest sandbox: one process per backtest run (B-3.2).

Uses ``multiprocessing.get_context("spawn")`` for cross-platform safety and
hard-kill capability.  The parent sends pre-compiled bytecode + ctx via Pipe,
waits with a hard deadline, and terminates the child on timeout.  This fixes
V3-R-8: ``threading.Timer`` + ``PyThreadState_SetAsyncExc`` cannot interrupt C
extensions or blocking syscalls; ``Process.terminate()`` always works.

B-3.5: Bytecode is pre-compiled once in the parent via RestrictedPython and
serialized to the child, avoiding redundant compilation in every spawned
process.

Architecture::

    Parent                          Child (spawn)
    BacktestSandbox.__init__()
      └─ compile_and_serialize()    (parent caches bytecode)
    BacktestSandbox.call(ctx)
      │                             _backtest_worker(conn, bytecode)
      ├─ p.start() ──────────────►  exec_serialized(bytecode) → run(ctx)
      ├─ conn.send(bytecode, ctx) ─► conn.send(result | error)
      ├─ p.join(timeout)
      ├─ (timeout) p.terminate()
      └─ conn.recv() ◄───────────  conn.close()
"""

from __future__ import annotations

import multiprocessing as mp
from typing import Optional

from app.engine.sandbox_base import BaseSandbox

# Per Q5: use spawn (not fork) for cross-platform consistency.
_CTX = mp.get_context("spawn")


class SandboxTimeoutError(Exception):
    """Raised when strategy execution exceeds the configured timeout."""


class BacktestSandbox(BaseSandbox):
    """Spawns a fresh process per backtest run.

    Bytecode is compiled once in the parent and reused across runs.

    Usage::

        sandbox = BacktestSandbox(source, timeout_ms=120_000)
        result = sandbox.call(ctx)   # blocks until done or timeout
        sandbox.shutdown()
    """

    def __init__(self, source: str, timeout_ms: int = 120_000) -> None:
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
        # B-3.5: pre-compile once, reuse across every spawned child.
        self._serialized_code = compile_and_serialize(source)

    # -- BaseSandbox interface ------------------------------------------------

    def call(self, ctx: dict) -> Optional[dict]:
        parent_conn, child_conn = _CTX.Pipe()
        p = _CTX.Process(
            target=_backtest_worker,
            args=(child_conn, self._serialized_code),
            name=f"bt-sandbox-{self._sha256[:8]}",
        )
        p.start()
        child_conn.close()  # parent only writes to parent_conn

        try:
            parent_conn.send(ctx)
        except BrokenPipeError:
            p.join(5)
            raise SandboxTimeoutError("backtest: child process exited before receiving context")

        timeout_s = self._timeout_ms / 1000.0
        p.join(timeout_s)

        if p.is_alive():
            p.terminate()
            p.join(5)
            if p.is_alive():
                p.kill()
                p.join(2)
            raise SandboxTimeoutError(
                f"backtest exceeded timeout ({self._timeout_ms}ms)"
            )

        # Child exited — collect result.
        if parent_conn.poll():
            try:
                payload = parent_conn.recv()
            except EOFError:
                payload = None
        else:
            payload = None

        if isinstance(payload, dict) and "error" in payload:
            from app.engine.types import StrategyRuntimeError
            raise StrategyRuntimeError(payload["error"])
        if isinstance(payload, dict) and "compile_error" in payload:
            from app.engine.types import StrategyCompileError
            raise StrategyCompileError(payload["compile_error"])

        return payload  # may be None (hold signal) or dict

    def shutdown(self) -> None:
        pass  # this implementation is stateless between calls

    @property
    def source_sha256(self) -> str:
        return self._sha256


# -- Subprocess entry point --------------------------------------------------

def _backtest_worker(conn, serialized_code: bytes) -> None:
    """Execute the strategy in a spawn child process and send the result back.

    Bytecode is pre-compiled in the parent and deserialized here, avoiding
    redundant RestrictedPython compilation.
    """
    # Q7: spawn gives us a clean interpreter, but we defensively rebuild the
    # event loop in case code evolves to use asyncio inside the sandbox.
    try:
        import asyncio
        asyncio.set_event_loop(asyncio.new_event_loop())
    except Exception:
        pass

    result = None
    try:
        ctx = conn.recv()  # blocking wait for parent's context
        result = _exec_in_child(serialized_code, ctx)
    except Exception as exc:
        result = {"error": str(exc)}
    finally:
        try:
            conn.send(result)
        except (BrokenPipeError, OSError):
            pass
        conn.close()


def _exec_in_child(serialized_code: bytes, ctx: dict) -> Optional[dict]:
    """Deserialize pre-compiled bytecode and execute in the child process."""
    from app.engine.sandbox import (
        build_sandbox_globals,
        exec_serialized,
    )
    from app.engine.types import StrategyCompileError, StrategyRuntimeError

    globals_dict = build_sandbox_globals()
    locals_dict = dict(ctx)

    try:
        exec_serialized(serialized_code, globals_dict, locals_dict)
    except Exception as e:
        raise StrategyRuntimeError(f"策略代码执行错误: {e}") from e

    run_fn = locals_dict.get("run")
    if callable(run_fn):
        try:
            result = run_fn(dict(ctx))
        except Exception as e:
            raise StrategyRuntimeError(f"run() 抛出异常: {e}") from e
        return _coerce_signal(result)

    if "signal" in locals_dict:
        return _coerce_signal(locals_dict["signal"])

    raise StrategyRuntimeError("策略代码必须定义 signal 变量或 run(context) 函数")


def _coerce_signal(value) -> Optional[dict]:
    if value is None:
        return None
    if isinstance(value, dict):
        return value
    if isinstance(value, str):
        return {"signal": value}
    from app.engine.types import StrategyRuntimeError
    raise StrategyRuntimeError(
        f"策略返回值必须是 dict 或 None，收到 {type(value).__name__}"
    )
