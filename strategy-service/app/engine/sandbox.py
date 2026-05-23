"""Strategy sandbox.

契约：docs/domains/backtest-system.md §7.4.3 · sandbox.py

Layered defence for running untrusted user code:

1. **AST whitelist** — forbids ``import``, dunder access, dangerous builtins,
   validates ``run(context)`` signature (see :py:func:`validate_strategy_code`).
2. **RestrictedPython compile** — blocks a second class of attacks at bytecode
   level (attribute escapes, private names, etc.).
3. **Curated globals** — only safe builtins, numpy, math and the engine's
   indicators/helpers are exposed.

Timeouts are enforced by the outer process (see :py:mod:`app.engine.runner`'s
deadline). This keeps the sandbox itself side-effect-free and composable.
"""

from __future__ import annotations

import ast
import hashlib
import math
from dataclasses import dataclass
from typing import Any, Callable, Dict, List, Optional

from app.engine import indicators
from app.engine.types import StrategyCompileError, StrategyRuntimeError

_FORBIDDEN_CALLS = frozenset(
    {"open", "eval", "exec", "compile", "__import__",
     "input", "globals", "locals", "vars", "dir"}
)


# --- AST validation ------------------------------------------------------


@dataclass(frozen=True)
class StrategyValidationResult:
    valid: bool
    errors: List[str]
    warnings: List[str]


def validate_strategy_code(code: str) -> StrategyValidationResult:
    """Static check that mirrors legacy ``executor.validate_strategy_code``."""
    errors: List[str] = []
    warnings: List[str] = []

    try:
        tree = ast.parse(code)
    except SyntaxError as e:
        errors.append(f"语法错误: {e}")
        return StrategyValidationResult(valid=False, errors=errors, warnings=warnings)

    # Indicators in ``app/engine/indicators.py`` that return a SCALAR float.
    # Subscripting their result (``iMA(...)[-1]``) is always a bug and would
    # raise ``'float' object is not subscriptable`` at runtime. Catching it
    # statically gives the user a clear error before the backtest starts.
    _SCALAR_INDICATORS = {
        "iMA", "iRSI", "iATR", "iCCI", "iMomentum", "iWPR",
        "AccountBalance", "AccountEquity", "OrdersTotal",
    }

    # Track simple ``x = iMA(...)`` style aliases so we can flag ``x[-1]`` or
    # ``len(x)`` later in the same pass even when the user binds the call
    # result to a variable first (the most common pattern in AI-generated
    # code). Keep it intentionally shallow — no flow analysis, just last
    # assignment wins.
    scalar_aliases: dict = {}
    for node in ast.walk(tree):
        if isinstance(node, ast.Assign) and len(node.targets) == 1 and isinstance(node.targets[0], ast.Name):
            value = node.value
            if isinstance(value, ast.Call) and isinstance(value.func, ast.Name) and value.func.id in _SCALAR_INDICATORS:
                scalar_aliases[node.targets[0].id] = value.func.id

    def _scalar_origin(expr: ast.AST) -> str:
        if isinstance(expr, ast.Call) and isinstance(expr.func, ast.Name) and expr.func.id in _SCALAR_INDICATORS:
            return expr.func.id
        if isinstance(expr, ast.Name) and expr.id in scalar_aliases:
            return scalar_aliases[expr.id]
        return ""

    run_defs: List[ast.FunctionDef] = []
    for node in ast.walk(tree):
        if isinstance(node, (ast.Import, ast.ImportFrom)):
            errors.append("禁止 import")
        if isinstance(node, (ast.Global, ast.Nonlocal)):
            errors.append("禁止 global/nonlocal")
        if isinstance(node, ast.Attribute) and node.attr.startswith("__"):
            errors.append("禁止访问 dunder 属性")
        if isinstance(node, ast.Name) and node.id.startswith("__"):
            errors.append("禁止使用 dunder 名称")
        if isinstance(node, ast.Call):
            fn = node.func
            if isinstance(fn, ast.Name) and fn.id in _FORBIDDEN_CALLS:
                errors.append(f"禁止调用: {fn.id}()")
        if isinstance(node, ast.Subscript):
            origin = _scalar_origin(node.value)
            if origin:
                errors.append(
                    f"{origin}() 返回标量(单个数值)，不能用下标访问。"
                    f"例如 x = {origin}(...) 后写 x[-1] 会在运行时抛 'float' object is not subscriptable。"
                    f"请直接把 {origin}(...) 当数值用。"
                )
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and node.func.id == "len":
            if node.args:
                origin = _scalar_origin(node.args[0])
                if origin:
                    errors.append(
                        f"{origin}() 返回标量，不能对它的结果调用 len()。"
                    )
        if isinstance(node, ast.FunctionDef) and node.name == "run":
            run_defs.append(node)

    if len(run_defs) > 1:
        errors.append("只允许定义一个 run(context) 函数")
    elif len(run_defs) == 1:
        run_def = run_defs[0]
        if run_def.args.vararg is not None or run_def.args.kwarg is not None:
            errors.append("run(context) 禁止使用 *args/**kwargs")
        if len(run_def.args.args) != 1:
            errors.append("run(context) 必须且只能接收一个参数: context")

    if not errors and "signal" not in code and "def run" not in code:
        errors.append("必须定义 signal 变量或 run(context) 函数")

    # Deduplicate while preserving order so the frontend message stays stable.
    seen = set()
    deduped: List[str] = []
    for e in errors:
        if e not in seen:
            seen.add(e)
            deduped.append(e)

    return StrategyValidationResult(
        valid=len(deduped) == 0, errors=deduped, warnings=warnings
    )


# --- RestrictedPython adapter -------------------------------------------


class _RestrictedEnv:
    """Lazy-imported RestrictedPython bindings shared across runners."""

    _instance: Optional["_RestrictedEnv"] = None

    def __init__(self) -> None:
        from RestrictedPython import compile_restricted
        from RestrictedPython.Eval import (
            default_guarded_getattr,
            default_guarded_getitem,
            default_guarded_getiter,
        )
        from RestrictedPython.Guards import full_write_guard, safe_builtins
        from RestrictedPython.PrintCollector import PrintCollector

        self.compile_restricted = compile_restricted
        self.safe_builtins = dict(safe_builtins)
        self.safe_builtins.update(
            {
                # numeric / basic
                "sum": sum, "round": round, "min": min, "max": max,
                "len": len, "abs": abs, "pow": pow, "divmod": divmod,
                # type ctors (needed by system presets & LLM-generated strategies)
                "int": int, "float": float, "bool": bool, "str": str,
                "dict": dict, "list": list, "tuple": tuple, "set": set,
                "frozenset": frozenset, "bytes": bytes,
                # introspection / iteration
                "isinstance": isinstance, "issubclass": issubclass,
                "range": range, "enumerate": enumerate, "zip": zip,
                "reversed": reversed, "sorted": sorted, "filter": filter,
                "map": map, "any": any, "all": all,
                # misc
                "print": print,
            }
        )
        # Whitelisted __import__: required because numpy's bound methods
        # (e.g. ndarray.mean, ndarray.std) lazily import submodules the first
        # time they are called. User-level `import` / `__import__` calls are
        # already forbidden by validate_strategy_code's AST check, so this
        # only serves C-extension / stdlib internal needs.
        _real_import = __import__
        _import_whitelist = (
            "numpy", "numpy.", "math", "math.",
            "_ast", "_collections_abc", "collections", "collections.",
            "itertools", "functools", "operator", "copyreg",
            "array", "_operator", "warnings",
        )

        def _guarded_import(name, globals=None, locals=None, fromlist=(), level=0):
            if not isinstance(name, str) or name == "":
                raise ImportError("import blocked by sandbox")
            allowed = any(
                name == w or (w.endswith(".") and name.startswith(w))
                for w in _import_whitelist
            )
            if not allowed:
                raise ImportError(f"import blocked by sandbox: {name!r}")
            return _real_import(name, globals, locals, fromlist, level)

        self.safe_builtins["__import__"] = _guarded_import
        self.guarded_getattr = default_guarded_getattr
        self.guarded_getitem = default_guarded_getitem
        self.guarded_getiter = default_guarded_getiter
        # Required by RestrictedPython-generated bytecode for guarded writes.
        self.guarded_write = full_write_guard
        self.print_collector = PrintCollector

    @classmethod
    def get(cls) -> "_RestrictedEnv":
        if cls._instance is None:
            cls._instance = _RestrictedEnv()
        return cls._instance


# --- Bytecode cache (process-level, keyed by sha256) --------------------


_bytecode_cache: Dict[str, Any] = {}


def _get_bytecode(env: _RestrictedEnv, source: str) -> Any:
    key = hashlib.sha256(source.encode()).hexdigest()
    if key not in _bytecode_cache:
        _bytecode_cache[key] = env.compile_restricted(source, "<strategy>", "exec")
    return _bytecode_cache[key]


def code_sha256(source: str) -> str:
    return hashlib.sha256(source.encode()).hexdigest()


# --- StrategyRunner ------------------------------------------------------


class StrategyRunner:
    """Compile once, execute per bar."""

    def __init__(self, source: str, timeout_ms: int = 30_000) -> None:
        validation = validate_strategy_code(source)
        if not validation.valid:
            raise StrategyCompileError("; ".join(validation.errors))
        self._source = source
        self._timeout_ms = timeout_ms
        self._env = _RestrictedEnv.get()
        try:
            self._bytecode = _get_bytecode(self._env, source)
        except Exception as e:  # pragma: no cover - RestrictedPython edge case
            raise StrategyCompileError(f"RestrictedPython 编译失败: {e}") from e

    @property
    def source_sha256(self) -> str:
        return code_sha256(self._source)

    def call(self, ctx: dict) -> Optional[dict]:
        """Execute the strategy with ``ctx`` and return its signal dict (or ``None``)."""
        globals_dict = self._build_globals()
        locals_dict = dict(ctx)
        try:
            exec(self._bytecode, globals_dict, locals_dict)
        except Exception as e:
            raise StrategyRuntimeError(f"策略代码执行错误: {e}") from e

        run_fn: Optional[Callable[[dict], Any]] = locals_dict.get("run")  # type: ignore[assignment]
        if callable(run_fn):
            try:
                result = run_fn(dict(ctx))
            except Exception as e:
                raise StrategyRuntimeError(f"run() 抛出异常: {e}") from e
            return self._coerce_signal(result)

        if "signal" in locals_dict:
            return self._coerce_signal(locals_dict["signal"])

        raise StrategyRuntimeError("策略代码必须定义 signal 变量或 run(context) 函数")

    # --- internals -------------------------------------------------------

    def _build_globals(self) -> dict:
        env = self._env
        import numpy as np  # local import keeps top-level import cheap
        g: Dict[str, Any] = {
            "__builtins__": env.safe_builtins,
            "_getattr_": env.guarded_getattr,
            "_getitem_": env.guarded_getitem,
            "_getiter_": env.guarded_getiter,
            "_write_": env.guarded_write,
            "_print_": env.print_collector,
            "np": np,
            "math": math,
        }
        # Inject indicator / sizing / query helpers.
        for name in indicators.__all__:
            g[name] = getattr(indicators, name)
        # Legacy alias preserved for older strategies.
        g["calculate_rsi"] = lambda prices, period=14: indicators.iRSI(prices, period)
        return g

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
