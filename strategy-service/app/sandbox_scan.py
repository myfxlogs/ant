# T3.3: sandbox_scan.py merged into app/engine/sandbox.py.
# This file re-exports for backward compatibility.
# Security boundary is now at OS level (see app/engine/sandbox_os.py).

from app.engine.sandbox import (
    BANNED_BUILTINS,
    BANNED_MODULES,
    MAX_CODE_LENGTH,
    SecurityScanResult as ScanResult,
    scan_security as scan_code,
)

__all__ = [
    "BANNED_BUILTINS",
    "BANNED_MODULES",
    "MAX_CODE_LENGTH",
    "ScanResult",
    "scan_code",
]
