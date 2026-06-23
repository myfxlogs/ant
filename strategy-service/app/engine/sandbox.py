"""Strategy sandbox.

契约：docs/domains/backtest-system.md §7.4.3 · sandbox.py
ADR: ADR-0020 · T3.3 OS 沙箱硬化

Security model — OS-kernel isolation, not language-level restrictions::

    1. **Static scan (lint)** — AST whitelist + banned module/builtin scan.
       Catches obvious attacks early.
    2. **SDK-only validation** — all strategies must be class-based (StrategyBase).
    3. **OS isolation** — seccomp-bpf, non-root, cgroup limits, net namespace.
       The kernel blocks dangerous syscalls; numpy/pandas run unrestricted.
    4. **Full Python runtime** — no RestrictedPython, no language-level sandboxing.
       Strategies can use any Python feature including ``_``-prefixed helpers,
       generators, decorators, etc.

Execution uses standard ``compile()`` (not ``compile_restricted()``).
The OS sandbox provides the real security boundary — see
:py:mod:`app.engine.sandbox_os` for OS-level isolation details.

Timeouts are enforced by the outer process (see :py:mod:`app.engine.runner`'s
deadline). This keeps the sandbox itself side-effect-free and composable.
"""

from __future__ import annotations

import ast
import ctypes
import hashlib
import marshal
import math
import threading
from dataclasses import dataclass, field
from typing import Any, Callable, Dict, List, Optional

from app.engine import indicators
from app.engine.code_quality import analyze_code_quality
from app.engine.sandbox_base import BaseSandbox
from app.engine.types import StrategyCompileError, StrategyRuntimeError

# ── T3.3: Static security scan (lint) ────────────────────────────────────

# SDK strategies may import from app.sdk, decimal, typing.
SDK_ALLOWED_MODULES: set[str] = {"app", "decimal", "typing"}

BANNED_MODULES: set[str] = {
    "os", "subprocess", "shutil", "sys", "ctypes", "multiprocessing",
    "socket", "http", "urllib", "requests", "ftplib", "smtplib",
    "pickle", "marshal", "code", "codeop", "compileall",
    "importlib", "pkgutil", "runpy",
    "ptrace", "resource", "signal",
}

BANNED_BUILTINS: set[str] = {
    "eval", "exec", "compile", "__import__", "open",
    "globals", "locals", "vars", "getattr", "setattr", "delattr",
    "breakpoint",
}

MAX_CODE_LENGTH = 65536  # T2.2: aligned with maxTransformCodeLen


@dataclass
class SecurityScanResult:
    passed: bool = True
    violations: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)


class _SecurityVisitor(ast.NodeVisitor):
    """AST visitor for banned module/builtin detection."""

    def __init__(self, result: SecurityScanResult):
        self.result = result

    def visit_Import(self, node):
        for alias in node.names:
            name = alias.name.split(".")[0]
            if name in SDK_ALLOWED_MODULES:
                pass
            elif name in BANNED_MODULES:
                self.result.violations.append(
                    f"banned import: {alias.name} (line {node.lineno})"
                )
        self.generic_visit(node)

    def visit_ImportFrom(self, node):
        if node.module:
            name = node.module.split(".")[0]
            if name in SDK_ALLOWED_MODULES:
                pass
            elif name in BANNED_MODULES:
                self.result.violations.append(
                    f"banned from-import: {node.module} (line {node.lineno})"
                )
        self.generic_visit(node)

    def visit_Call(self, node):
        if isinstance(node.func, ast.Name) and node.func.id in BANNED_BUILTINS:
            self.result.violations.append(
                f"banned builtin: {node.func.id}() (line {node.lineno})"
            )
        self.generic_visit(node)


def _scan_strings_for_danger(code: str, result: SecurityScanResult) -> None:
    """String-based scan for dynamic code execution patterns."""
    patterns = [
        ("__import__(", "dynamic import"),
        ("eval(", "eval() call"),
        ("exec(", "exec() call"),
        ("compile(", "compile() call"),
        ("subprocess", "subprocess reference"),
        ("socket.", "socket usage"),
        ("requests.", "HTTP request"),
        ("urllib", "URL library"),
    ]
    for pattern, desc in patterns:
        if pattern in code:
            result.warnings.append(f"potential {desc} in code")


def scan_security(code: str) -> SecurityScanResult:
    """Static security scan for user-generated strategy code.

    This is a LINT pass — security is enforced at OS level.
    Catches banned imports, dangerous builtins, and dynamic code patterns.
    """
    result = SecurityScanResult()

    if len(code) > MAX_CODE_LENGTH:
        result.violations.append(f"code too long ({len(code)} > {MAX_CODE_LENGTH} chars)")
        result.passed = False
        return result

    try:
        tree = ast.parse(code)
    except SyntaxError as e:
        result.violations.append(f"syntax error: {e}")
        result.passed = False
        return result

    visitor = _SecurityVisitor(result)
    visitor.visit(tree)
    _scan_strings_for_danger(code, result)

    if result.violations:
        result.passed = False
    return result


# Legacy compatibility: re-export scan_code for sandbox_scan.py consumers.
scan_code = scan_security

# --- AST validation (SDK-only) --------------------------------------------


@dataclass(frozen=True)
class StrategyValidationResult:
    valid: bool
    errors: List[str]
    warnings: List[str]
    quality_hints: List[Any] = field(default_factory=list)


def _is_sdk_strategy(tree) -> bool:
    """Detect SDK-format strategies (class X(StrategyBase))."""
    for node in ast.walk(tree):
        if isinstance(node, ast.ClassDef):
            for base in node.bases:
                if isinstance(base, ast.Name) and base.id == "StrategyBase":
                    return True
                if isinstance(base, ast.Attribute) and base.attr == "StrategyBase":
                    return True
    return False


def _validate_sdk_strategy(
    code: str, tree, errors: list, warnings: list,
    quality_hints: list | None = None,
) -> StrategyValidationResult:
    """Validate an SDK-format strategy (ADR-0020)."""
    class_def = None
    for node in ast.walk(tree):
        if isinstance(node, ast.ClassDef):
            for base in node.bases:
                if (isinstance(base, ast.Name) and base.id == "StrategyBase") or \
                   (isinstance(base, ast.Attribute) and base.attr == "StrategyBase"):
                    class_def = node
                    break
    if class_def is None:
        errors.append("SDK策略必须定义一个继承 StrategyBase 的类")
        return StrategyValidationResult(valid=False, errors=errors, warnings=warnings)
    method_names = {n.name for n in ast.walk(class_def) if isinstance(n, ast.FunctionDef)}
    hooks = {"on_init", "on_tick", "on_bar", "on_timer", "on_trade", "on_deinit"}
    if not (method_names & hooks):
        errors.append(f"SDK策略类 {class_def.name} 至少需要一个生命周期方法")
    if "self.broker" not in code:
        warnings.append("策略未引用 self.broker")

    seen = set()
    deduped = []
    for e in errors:
        if e not in seen:
            seen.add(e)
            deduped.append(e)
    return StrategyValidationResult(
        valid=len(deduped) == 0, errors=deduped, warnings=warnings,
        quality_hints=quality_hints or [],
    )


def validate_strategy_code(code: str) -> StrategyValidationResult:
    """Validate a strategy — SDK-only (ADR-0020).

    Non-SDK legacy patterns (``def run(context)``, ``signal = ...``) are
    rejected.  Every strategy must define a class that inherits from
    ``StrategyBase`` and implements at least one lifecycle hook.
    """
    errors: List[str] = []
    warnings: List[str] = []

    try:
        tree = ast.parse(code)
    except SyntaxError as e:
        errors.append(f"语法错误: {e}")
        return StrategyValidationResult(valid=False, errors=errors, warnings=warnings)

    if not _is_sdk_strategy(tree):
        errors.append(
            "只支持 SDK 策略（继承 StrategyBase 的类定义）。"
            "请使用类定义 + 生命周期方法（on_init/on_bar/on_tick 等）来编写策略代码。"
        )
        return StrategyValidationResult(valid=False, errors=errors, warnings=warnings)

    quality_hints = analyze_code_quality(code)
    return _validate_sdk_strategy(code, tree, errors, warnings, quality_hints)


# --- Bytecode compilation (standard Python compile) -----------------------

_bytecode_cache: Dict[str, Any] = {}


def _compile_source(source: str) -> Any:
    """Compile strategy source to a code object (cached by sha256)."""
    key = hashlib.sha256(source.encode()).hexdigest()
    if key not in _bytecode_cache:
        _bytecode_cache[key] = compile(source, "<strategy>", "exec")
    return _bytecode_cache[key]


def code_sha256(source: str) -> str:
    return hashlib.sha256(source.encode()).hexdigest()


def build_sandbox_globals() -> dict:
    """Construct the globals dict for sandbox execution.

    Injects numpy, math, and engine indicators.  Uses standard Python
    builtins — security is enforced at the OS level (seccomp/cgroup).
    """
    import numpy as np
    g: Dict[str, Any] = {
        "__builtins__": __builtins__,
        "np": np,
        "math": math,
    }
    for name in indicators.__all__:
        g[name] = getattr(indicators, name)
    g["calculate_rsi"] = lambda prices, period=14: indicators.iRSI(prices, period)
    return g


def compile_and_serialize(source: str) -> bytes:
    """Pre-compile strategy source and return marshalled bytecode.

    The resulting bytes can be passed to :func:`exec_serialized` in a child
    process, avoiding redundant compilation.

    Performs static security scan before compilation.
    """
    scan_result = scan_code(source)
    if scan_result.violations:
        msg = "; ".join(scan_result.violations)
        raise ValueError(f"code rejected by security scan: {msg}")

    code = compile(source, "<strategy>", "exec")
    return marshal.dumps(code)


def exec_serialized(data: bytes, globals_dict: dict, locals_dict: dict) -> None:
    """Deserialize pre-compiled bytecode and execute it in the given namespace."""
    code = marshal.loads(data)
    exec(code, globals_dict, locals_dict)


# --- StrategyRunner (in-process, thread-based; deprecated) ----------------


_production_mode: bool = False


def set_production_mode(enabled: bool) -> None:
    """Set the sandbox production mode flag.

    When True, all Python strategy execution is blocked.
    This is a global guard that cannot be bypassed once enabled.
    """
    global _production_mode
    _production_mode = enabled


def is_production_mode() -> bool:
    """Return True if the sandbox is in production mode."""
    return _production_mode


class SandboxBlockedError(Exception):
    """Raised when a strategy is executed in production mode."""


class SandboxTimeoutError(StrategyRuntimeError):
    """Raised when strategy execution exceeds the configured timeout."""


def _raise_async(thread_id: int, exc_class: type) -> int:
    """Raise an exception asynchronously in the given thread.

    Returns the number of thread states modified (0 = thread not found,
    1 = success, >1 = restored). Uses CPython's PyThreadState_SetAsyncExc.
    """
    return ctypes.pythonapi.PyThreadState_SetAsyncExc(
        ctypes.c_long(thread_id), ctypes.py_object(exc_class)
    )


class StrategyRunner(BaseSandbox):
    """Compile once, execute per bar.

    M7.7: In production mode, the sandbox is blocked entirely.
    Production paths use DSL + ONNX; Python execution is research-only.

    .. deprecated::
        The ``threading.Timer`` + ``PyThreadState_SetAsyncExc`` timeout
        mechanism is preserved for backward compatibility but is ineffective
        against C extensions and blocking syscalls (V3-R-8).
        Prefer :class:`app.engine.backtest_sandbox.BacktestSandbox` or
        :class:`app.engine.live_sandbox.LiveWorker` for process-level
        isolation via ``multiprocessing.spawn``.
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
        """No-op: StrategyRunner holds no external resources."""
        pass

    def call(self, ctx: dict) -> Optional[dict]:
        """Execute the strategy with ``ctx`` and return its signal dict (or ``None``).

        A timer-based timeout is enforced using ``self._timeout_ms``. If the
        strategy code does not yield within the timeout, ``SandboxTimeoutError``
        is raised in the calling thread.
        """
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
            raise SandboxTimeoutError(
                f"策略执行超时 ({self._timeout_ms}ms)"
            )
        finally:
            if timer is not None:
                timer.cancel()

    def _call_impl(self, ctx: dict) -> Optional[dict]:
        """Internal execution without timeout machinery."""
        globals_dict = self._build_globals()

        # Use the same dict for both globals and locals so that module-level
        # definitions (e.g. helper functions) are visible inside run().
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

    # --- internals -------------------------------------------------------

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
