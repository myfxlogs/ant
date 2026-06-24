"""Strategy code pipeline — re-export shim.

The original ``sandbox.py`` has been split into focused modules:

* :py:mod:`app.engine.validation` — security scan + SDK structure check
* :py:mod:`app.engine.compilation` — bytecode compile, serialize, globals

This module re-exports all public APIs for backward compatibility.
New code should import from the specific modules directly.
"""

from app.engine.validation import (
    BANNED_BUILTINS,
    BANNED_MODULES,
    MAX_CODE_LENGTH,
    SDK_ALLOWED_MODULES,
    SecurityScanResult,
    scan_security,
    StrategyValidationResult,
    validate_strategy_code,
)
from app.engine.compilation import (
    _compile_source,
    build_sandbox_globals,
    code_sha256,
    compile_and_serialize,
    exec_serialized,
)

# Legacy compatibility alias.
scan_code = scan_security

# Re-export types that used to be imported through this module.
from app.engine.types import StrategyRuntimeError  # noqa: E402

# Legacy execution — deprecated, will be removed when Execute endpoint migrates.
from app.engine.legacy_runner import (  # noqa: E402
    SandboxBlockedError,
    SandboxTimeoutError,
    StrategyRunner,
)

__all__ = [
    # validation
    "BANNED_BUILTINS", "BANNED_MODULES", "MAX_CODE_LENGTH", "SDK_ALLOWED_MODULES",
    "SecurityScanResult", "scan_code", "scan_security",
    "StrategyValidationResult", "validate_strategy_code",
    # compilation
    "_compile_source", "build_sandbox_globals", "code_sha256",
    "compile_and_serialize", "exec_serialized",
    # types (re-exported for backward compat)
    "StrategyRuntimeError",
    # legacy
    "SandboxBlockedError", "SandboxTimeoutError", "StrategyRunner",
]
