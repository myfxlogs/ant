"""Legacy StrategyRunner — deprecated in-process execution.

**Do not use in new code.**  This predates the SDK-only migration (ADR-0020).
It uses ``threading.Timer`` + ``PyThreadState_SetAsyncExc`` which cannot
interrupt C extensions or blocking syscalls.

The current backtest path uses SDK-native ``SimBroker`` + ``StrategyRuntime``
directly (see :py:mod:`app.engine.runner`).  This class remains only for
backward compatibility with the old ``/Execute`` ConnectRPC endpoint.
"""

from __future__ import annotations

import ctypes
import threading
from typing import Any, Callable, Dict, Optional

from app.engine.compilation import _compile_source, build_sandbox_globals, code_sha256
from app.engine.sandbox_base import BaseSandbox
from app.engine.types import StrategyCompileError, StrategyRuntimeError
from app.engine.validation import validate_strategy_code

_production_mode: bool = False


def set_production_mode(enabled: bool) -> None:
    global _production_mode
    _production_mode = enabled


def is_production_mode() -> bool:
    return _production_mode


class SandboxBlockedError(Exception):
    """Raised when a strategy is executed in production mode."""


class SandboxTimeoutError(StrategyRuntimeError):
    """Raised when strategy execution exceeds the configured timeout."""


def _raise_async(thread_id: int, exc_class: type) -> int:
    return ctypes.pythonapi.PyThreadState_SetAsyncExc(
        ctypes.c_long(thread_id), ctypes.py_object(exc_class)
    )


class StrategyRunner(BaseSandbox):
    """In-process sandbox — compiles once, executes per bar.

    .. warning::
       **Legacy.**  This runner predates the SDK-only migration (ADR-0020).
       It uses ``threading.Timer`` + ``PyThreadState_SetAsyncExc`` which
       cannot interrupt C extensions or blocking syscalls.

       The current backtest execution path in ``runner.py`` uses SDK-native
       ``SimBroker`` + ``StrategyRuntime`` directly.

       For process-level isolation use:
       - :class:`app.engine.backtest_sandbox.BacktestSandbox`
       - :class:`app.engine.live_sandbox.LiveWorker`
    """

    def __init__(self, source: str, timeout_ms: int = 30_000) -> None:
        if _production_mode:
            raise SandboxBlockedError(
                "sandbox blocked: production mode active. "
                "Python strategy execution is research-only. "
                "Use DSL expressions or ONNX models for live trading."
            )
        validation = validate_strategy_code(source)
        if not validation.valid:
            raise StrategyCompileError("; ".join(validation.errors))
        self._source = source
        self._timeout_ms = timeout_ms
        try:
            self._bytecode = _compile_source(source)
        except SyntaxError as e:
            raise StrategyCompileError(f"Python 编译失败: {e}") from e

    @property
    def source_sha256(self) -> str:
        return code_sha256(self._source)

    def shutdown(self) -> None:
        pass

    def call(self, ctx: dict) -> Optional[dict]:
        tid = threading.get_ident()
        timer: Optional[threading.Timer] = None

        if self._timeout_ms > 0:
            timer = threading.Timer(
                self._timeout_ms / 1000.0,
                _raise_async, args=(tid, SandboxTimeoutError),
            )
            timer.start()

        try:
            return self._call_impl(ctx)
        except SandboxTimeoutError:
            raise SandboxTimeoutError(f"策略执行超时 ({self._timeout_ms}ms)")
        finally:
            if timer is not None:
                timer.cancel()

    def _call_impl(self, ctx: dict) -> Optional[dict]:
        globals_dict = self._build_globals()
        exec_scope = dict(globals_dict)
        try:
            exec(self._bytecode, exec_scope, exec_scope)
        except SandboxTimeoutError:
            raise
        except Exception as e:
            raise StrategyRuntimeError(f"策略代码执行错误: {e}") from e

        run_fn: Optional[Callable[[dict], Any]] = exec_scope.get("run")
        if callable(run_fn):
            try:
                result = run_fn(dict(ctx))
            except SandboxTimeoutError:
                raise
            except Exception as e:
                raise StrategyRuntimeError(f"run() 抛出异常: {e}") from e
            return self._coerce_signal(result)

        if "signal" in exec_scope:
            return self._coerce_signal(exec_scope["signal"])

        raise StrategyRuntimeError("策略代码必须定义 signal 变量或 run(context) 函数")

    def _build_globals(self) -> dict:
        return build_sandbox_globals()

    @staticmethod
    def _coerce_signal(value: Any) -> Optional[dict]:
        if value is None:
            return None
        if isinstance(value, dict):
            return value
        if isinstance(value, str):
            return {"signal": value}
        raise StrategyRuntimeError(
            f"策略返回值必须是 dict 或 None，收到 {type(value).__name__}"
        )
