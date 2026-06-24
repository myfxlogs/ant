"""Strategy bytecode compilation — standard Python compile(), no sandbox.

Pre-compiles strategy source to bytecode objects, caches by sha256,
and provides serialization for cross-process transport (BacktestSandbox,
LiveWorker).
"""

from __future__ import annotations

import hashlib
import marshal
import math
from typing import Any, Dict

from app.engine import indicators

# --- Bytecode cache -------------------------------------------------------

_bytecode_cache: Dict[str, Any] = {}


def _compile_source(source: str) -> Any:
    """Compile strategy source to a code object (cached by sha256)."""
    key = hashlib.sha256(source.encode()).hexdigest()
    if key not in _bytecode_cache:
        _bytecode_cache[key] = compile(source, "<strategy>", "exec")
    return _bytecode_cache[key]


def code_sha256(source: str) -> str:
    return hashlib.sha256(source.encode()).hexdigest()


# --- Globals construction -------------------------------------------------


def build_sandbox_globals() -> dict:
    """Construct the globals dict for strategy execution.

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


# --- Serialization (cross-process transport) ------------------------------


def compile_and_serialize(source: str) -> bytes:
    """Pre-compile strategy source and return marshalled bytecode.

    Performs security scan before compilation.
    """
    from app.engine.validation import scan_security
    scan_result = scan_security(source)
    if scan_result.violations:
        msg = "; ".join(scan_result.violations)
        raise ValueError(f"code rejected by security scan: {msg}")

    code = compile(source, "<strategy>", "exec")
    return marshal.dumps(code)


def exec_serialized(data: bytes, globals_dict: dict, locals_dict: dict) -> None:
    """Deserialize pre-compiled bytecode and execute it in the given namespace."""
    code = marshal.loads(data)
    exec(code, globals_dict, locals_dict)
