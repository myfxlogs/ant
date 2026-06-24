"""Strategy code pipeline — re-export shim.

The original ``sandbox.py`` has been split into focused modules:

* :py:mod:`app.engine.validation` — security scan + SDK structure check
* :py:mod:`app.engine.compilation` — bytecode compile, globals

This module re-exports public APIs for backward compatibility.
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
)

# Legacy compatibility alias.
scan_code = scan_security

__all__ = [
    "BANNED_BUILTINS", "BANNED_MODULES", "MAX_CODE_LENGTH", "SDK_ALLOWED_MODULES",
    "SecurityScanResult", "scan_code", "scan_security",
    "StrategyValidationResult", "validate_strategy_code",
    "_compile_source", "build_sandbox_globals", "code_sha256",
]
