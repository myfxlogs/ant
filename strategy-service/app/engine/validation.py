"""Strategy code validation — security scan + SDK structure check.

Single source of truth for all strategy code validation.  This is the
canonical validator called by every code path (validate, backtest, live).

Security scanning is AST-based (not regex) — accurately distinguishes
code from comments/strings.  The Go side must NOT duplicate these checks.
"""

from __future__ import annotations

import ast
from dataclasses import dataclass, field
from typing import Any, List

from app.engine.code_quality import analyze_code_quality


# ── Security scan constants ──────────────────────────────────────────────

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
    "breakpoint", "__builtins__",
}

MAX_CODE_LENGTH = 65536


# ── Security scan result ─────────────────────────────────────────────────


@dataclass
class SecurityScanResult:
    passed: bool = True
    violations: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)


# ── AST visitor ──────────────────────────────────────────────────────────


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


# ── Valid SDK exports (must match app/sdk/__init__.py __all__) ───────────

_VALID_SDK_EXPORTS: set[str] = {
    "AccountInfo", "AccountMode", "Broker", "Context", "DealType",
    "Indicators", "OrderRequest", "OrderResult", "OrderType",
    "PendingOrder", "Position", "PositionSide", "Retcode",
    "RuntimeContext", "Series", "StrategyBase", "StrategyRuntime",
    "SymbolInfo", "TypeFilling",
}

# ── Strategy validation (SDK-only) ───────────────────────────────────────


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

    # Validate SDK imports — only allow documented SDK exports.
    for node in ast.walk(tree):
        if isinstance(node, ast.ImportFrom):
            if node.module == "app.sdk" or (node.module or "").startswith("app.sdk."):
                for alias in node.names:
                    if alias.name not in _VALID_SDK_EXPORTS:
                        errors.append(f"`{alias.name}` 不是有效的 SDK 导出")
        # Detect hardcoded timeframe in bars() calls.
        if isinstance(node, ast.Call):
            for kw in node.keywords:
                if kw.arg == "timeframe" and isinstance(kw.value, ast.Constant):
                    if isinstance(kw.value.value, str) and kw.value.value.strip():
                        errors.append(
                            f"禁止硬编码周期 `timeframe='{kw.value.value}'`。"
                            f"请使用 `timeframe=None`（跟随回测配置）"
                            f"或 `self.ctx.param('timeframe', '1h')`（用户可选）。"
                        )

        # Detect hardcoded magic numbers in OrderRequest/order_send.
        if isinstance(node, ast.Call):
            fn = node.func
            fn_name = ""
            if isinstance(fn, ast.Name): fn_name = fn.id
            elif isinstance(fn, ast.Attribute): fn_name = fn.attr
            if fn_name in ("order_send", "OrderRequest"):
                for kw in node.keywords:
                    if kw.arg == "magic" and isinstance(kw.value, ast.Constant):
                        if isinstance(kw.value.value, int) and kw.value.value != 0:
                            warnings.append(
                                f"建议将 magic={kw.value.value} 改为 param 读取，避免多策略实例冲突。"
                            )
        # Detect float() for prices (should use Decimal).
        if isinstance(node, ast.Call) and isinstance(node.func, ast.Name) and node.func.id == "float":
            warnings.append("价格/手数计算中使用了 float()，建议改用 Decimal(str(x)) 避免精度丢失。")
        # Detect underscore-prefixed method names (against SDK convention).
        if isinstance(node, ast.FunctionDef) and node.name.startswith("_") and not node.name.startswith("__"):
            warnings.append(f"方法 `{node.name}` 使用了 _ 前缀，建议改为无前缀命名。")
        # Detect unnecessary imports of pre-injected modules.
        if isinstance(node, ast.Import):
            for alias in node.names:
                if alias.name in ("math", "numpy") and alias.asname is None:
                    warnings.append(f"`import {alias.name}` 多余——{alias.name} 已被沙箱预注入，可直接使用。")

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

    Security scanning (banned imports/dangerous builtins) is included —
    this is the single source of truth for all strategy validation.

    Structural checks (SDK imports, lifecycle hooks) run even when
    the code has syntax errors — they use string matching on the
    original source, not the AST.
    """
    errors: List[str] = []
    warnings: List[str] = []

    tree = None
    try:
        tree = ast.parse(code)
    except SyntaxError as e:
        errors.append(f"语法错误: {e}")
        # Syntax is broken, but we can still run string-based checks.
        # Structural errors (invalid SDK imports, missing hooks) are
        # detectable without a valid AST.

    # String-based checks run even when AST parse failed.
    if tree is None:
        # Check SDK class presence via regex (no AST needed).
        import re
        if not re.search(r'class\s+\w+\s*\(.*StrategyBase', code):
            errors.append(
                "只支持 SDK 策略（继承 StrategyBase 的类定义）。"
                "请使用类定义 + 生命周期方法（on_init/on_bar/on_tick 等）来编写策略代码。"
            )
        # Check SDK import validity via regex.
        for m in re.finditer(r'from\s+app\.sdk\s+import\s+(.+?)(?:\n|$)', code):
            imports = [x.strip() for x in m.group(1).split(',')]
            for name in imports:
                if name and name not in _VALID_SDK_EXPORTS:
                    errors.append(f"`{name}` 不是有效的 SDK 导出")
        # Detect hardcoded timeframe.
        for m in re.finditer(r'timeframe\s*=\s*["\']([^"\']+)["\']', code):
            if m.group(1) != 'None':
                errors.append(
                    f"禁止硬编码周期 `timeframe='{m.group(1)}'`。"
                    f"请使用 `timeframe=None`（跟随回测配置）。"
                )
        # Check lifecycle hooks.
        hooks_found = any(
            re.search(rf'def\s+{h}\s*\(', code)
            for h in ("on_init", "on_tick", "on_bar", "on_timer", "on_trade", "on_deinit")
        )
        if not hooks_found:
            warnings.append("策略未实现任何 SDK 生命周期方法")
        # Security scan on original code (skip syntax errors — already reported).
        security = scan_security(code)
        for v in security.violations:
            if not v.startswith("syntax error"):
                errors.append(v)
        warnings.extend(security.warnings)
        # Deduplicate.
        seen = set()
        deduped = []
        for e in errors:
            if e not in seen:
                seen.add(e)
                deduped.append(e)
        return StrategyValidationResult(valid=False, errors=deduped, warnings=warnings)

    if not _is_sdk_strategy(tree):
        errors.append(
            "只支持 SDK 策略（继承 StrategyBase 的类定义）。"
            "请使用类定义 + 生命周期方法（on_init/on_bar/on_tick 等）来编写策略代码。"
        )
        return StrategyValidationResult(valid=False, errors=errors, warnings=warnings)

    # Security scan: banned imports, dangerous builtins, dynamic code patterns.
    security = scan_security(code)
    if security.violations:
        errors.extend(security.violations)
    warnings.extend(security.warnings)

    quality_hints = analyze_code_quality(code)
    return _validate_sdk_strategy(code, tree, errors, warnings, quality_hints)
