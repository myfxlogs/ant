# M5-2: Python sandbox static scan — enhanced security for user-generated strategy code.
# Scans generated Python code for banned modules, dangerous builtins, and network I/O.
# Called by strategy-service before executing user code in the sandbox.

import ast
import sys
from dataclasses import dataclass, field
from typing import List

BANNED_MODULES = {
    "os", "subprocess", "shutil", "sys", "ctypes", "multiprocessing",
    "socket", "http", "urllib", "requests", "ftplib", "smtplib",
    "pickle", "marshal", "code", "codeop", "compileall",
    "importlib", "pkgutil", "runpy",
    "ptrace", "resource", "signal",
}

BANNED_BUILTINS = {
    "eval", "exec", "compile", "__import__", "open",
    "globals", "locals", "vars", "getattr", "setattr", "delattr",
    "breakpoint",
}

MAX_CODE_LENGTH = 10000  # characters


@dataclass
class ScanResult:
    passed: bool = True
    violations: List[str] = field(default_factory=list)
    warnings: List[str] = field(default_factory=list)


def scan_code(code: str) -> ScanResult:
    """Scan user-generated strategy code for security violations."""
    result = ScanResult()

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

    # Also do a simple string scan for dynamic imports
    _scan_strings(code, result)

    if result.violations:
        result.passed = False
    return result


class _SecurityVisitor(ast.NodeVisitor):
    def __init__(self, result: ScanResult):
        self.result = result

    def visit_Import(self, node):
        for alias in node.names:
            name = alias.name.split(".")[0]
            if name in BANNED_MODULES:
                self.result.violations.append(
                    f"banned import: {alias.name} (line {node.lineno})"
                )
            if name in ("os", "sys"):
                self.result.warnings.append(
                    f"potentially dangerous import: {alias.name} (line {node.lineno})"
                )
        self.generic_visit(node)

    def visit_ImportFrom(self, node):
        if node.module:
            name = node.module.split(".")[0]
            if name in BANNED_MODULES:
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


def _scan_strings(code: str, result: ScanResult):
    """Simple string-based scan for dynamic code execution patterns."""
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
