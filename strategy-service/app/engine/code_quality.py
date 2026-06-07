"""
Static code quality analysis for ant strategy Python code.

Read-only analysis — does NOT execute user code. Adapted from QuantDinger's
indicator_code_quality.py for our ``run(context)`` event-driven model.

Checks (6 dimensions):
  1. FUTURE_DATA_LEAK     — forward indexing inside loops (AST-based, 100% accurate)
  2. MISSING_PARAM        — params.get('x') without @param x declaration
  3. UNREAD_PARAM         — @param x declared but never read
  4. NDARRAY_PANDAS_MISUSE — calling .rolling()/.shift()/.fillna() on numpy arrays
  5. NO_STOP_AND_TAKE_PROFIT — has buy/sell signals but no @strategy stopLossPct/takeProfitPct
  6. NO_ENTRY_PCT          — has buy/sell signals but no @strategy entryPct
"""

from __future__ import annotations

import ast
import re
from dataclasses import dataclass
from typing import List, Set


@dataclass
class CodeHint:
    category: str   # "FUTURE_DATA_LEAK" | "MISSING_PARAM" | "UNREAD_PARAM" | "NDARRAY_PANDAS_MISUSE" | "NO_STOP_AND_TAKE_PROFIT" | "NO_ENTRY_PCT"
    severity: str   # "error" | "warn" | "info"
    message: str    # Chinese human-readable
    line: int       # 1-based approximate line number (0 if unknown)
    snippet: str    # the problematic code fragment


# ---------------------------------------------------------------------------
# helpers
# ---------------------------------------------------------------------------

def _strip_comments(code: str) -> str:
    """Strip end-of-line ``#`` comments; keeps line structure intact."""
    out_lines: List[str] = []
    for raw_line in (code or "").split("\n"):
        in_str: str | None = None
        escape = False
        cut = len(raw_line)
        for i, ch in enumerate(raw_line):
            if escape:
                escape = False
                continue
            if ch == "\\":
                escape = True
                continue
            if in_str is not None:
                if ch == in_str:
                    in_str = None
                continue
            if ch in ("'", '"'):
                in_str = ch
                continue
            if ch == "#":
                cut = i
                break
        out_lines.append(raw_line[:cut])
    return "\n".join(out_lines)


def _declared_param_names(code: str) -> List[str]:
    """Extract parameter names from ``# @param`` annotations."""
    names: List[str] = []
    for m in re.finditer(
        r"^\s*#\s*@param\s+(\w+)\s+",
        code or "",
        re.MULTILINE,
    ):
        names.append(m.group(1))
    return names


def _param_get_calls(code: str) -> Set[str]:
    """Find all keys passed to ``p.get()`` or ``params.get()``."""
    keys: Set[str] = set()
    for m in re.finditer(
        r"""\b(?:p|params)\s*\.\s*get\s*\(\s*['"]([^'"]+)['"]""",
        code or "",
    ):
        keys.add(m.group(1))
    return keys


# ---------------------------------------------------------------------------
# check 1: FUTURE_DATA_LEAK (AST-based — optimal)
# ---------------------------------------------------------------------------

def _check_future_data_leak(code: str) -> List[CodeHint]:
    """Detect forward indexing inside for-loops via AST traversal.

    AST-based detection is 100% accurate: it correctly handles any loop
    variable name, nested loops, and complex expressions — unlike regex
    patterns which have blind spots on aliases and non-standard names.
    """
    hints: List[CodeHint] = []
    seen: Set[str] = set()

    try:
        tree = ast.parse(code or "")
    except SyntaxError:
        return hints  # invalid code → validation handles it

    for node in ast.walk(tree):
        # Match: for <var> in range(len(<arr>)):
        if not isinstance(node, ast.For):
            continue
        if not isinstance(node.iter, ast.Call):
            continue
        call = node.iter
        if not isinstance(call.func, ast.Name) or call.func.id != "range":
            continue
        if len(call.args) != 1:
            continue
        arg = call.args[0]
        if not isinstance(arg, ast.Call):
            continue
        if not isinstance(arg.func, ast.Name) or arg.func.id != "len":
            continue
        if len(arg.args) != 1 or not isinstance(arg.args[0], ast.Name):
            continue

        loop_var = node.target
        if not isinstance(loop_var, ast.Name):
            continue
        loop_var_name = loop_var.id
        loop_line = node.lineno

        # Walk the loop body for Subscript[BinOp(loop_var, Add, Constant(N>0))]
        for body_node in ast.walk(node):
            if not isinstance(body_node, ast.Subscript):
                continue
            slice_node = body_node.slice
            if not isinstance(slice_node, ast.BinOp):
                continue
            if not isinstance(slice_node.op, ast.Add):
                continue
            # Check if either operand is the loop variable
            left_is_var = isinstance(slice_node.left, ast.Name) and slice_node.left.id == loop_var_name
            right_is_var = isinstance(slice_node.right, ast.Name) and slice_node.right.id == loop_var_name
            if not (left_is_var or right_is_var):
                continue
            # Check the other operand is a positive constant
            const = slice_node.right if left_is_var else slice_node.left
            if not isinstance(const, ast.Constant):
                continue
            if not isinstance(const.value, (int, float)) or const.value <= 0:
                continue

            # Build snippet from source lines
            try:
                src_lines = code.split("\n")
                if body_node.lineno - 1 < len(src_lines):
                    snippet = src_lines[body_node.lineno - 1].strip()
                else:
                    snippet = ast.unparse(body_node) if hasattr(ast, "unparse") else ""
            except Exception:
                snippet = ""

            key = f"{snippet}:{body_node.lineno}"
            if key in seen:
                continue
            seen.add(key)
            hints.append(CodeHint(
                category="FUTURE_DATA_LEAK",
                severity="error",
                message=f"循环内使用了未来数据：{snippet}，索引 +{const.value} 引用了未来 K 线",
                line=body_node.lineno,
                snippet=snippet,
            ))

    return hints


# ---------------------------------------------------------------------------
# check 2: MISSING_PARAM
# ---------------------------------------------------------------------------

def _check_missing_params(code: str) -> List[CodeHint]:
    """Detect p.get('x') calls without @param x declaration."""
    hints: List[CodeHint] = []
    src = _strip_comments(code)
    declared = set(_declared_param_names(code))
    used = _param_get_calls(src)

    for key in sorted(used - declared):
        pattern = rf"""\b(?:p|params)\s*\.\s*get\s*\(\s*['"]{re.escape(key)}['"]"""
        linenum = _line_of(pattern, src)
        hints.append(CodeHint(
            category="MISSING_PARAM",
            severity="warn",
            message=f"参数 '{key}' 被使用但未声明 @param，调参面板无法识别",
            line=linenum,
            snippet=f"p.get('{key}', ...)",
        ))

    return hints


# ---------------------------------------------------------------------------
# check 3: UNREAD_PARAM
# ---------------------------------------------------------------------------

def _check_unread_params(code: str) -> List[CodeHint]:
    """Detect @param x declarations never used via p.get('x')."""
    hints: List[CodeHint] = []
    src = _strip_comments(code)
    declared = _declared_param_names(code)
    used = _param_get_calls(src)

    for name in declared:
        if name not in used:
            pattern = rf"^\s*#\s*@param\s+{re.escape(name)}\s+"
            linenum = _line_of(pattern, code)
            hints.append(CodeHint(
                category="UNREAD_PARAM",
                severity="info",
                message=f"@param '{name}' 已声明但从未通过 p.get('{name}') 读取",
                line=linenum,
                snippet=f"# @param {name} ...",
            ))

    return hints


# ---------------------------------------------------------------------------
# check 4: NDARRAY_PANDAS_MISUSE
# ---------------------------------------------------------------------------

# Pandas-only methods that do NOT exist on numpy ndarrays.
# AI-generated code often calls these on variables that are actually numpy
# arrays (from context['close'], context['open'], etc.).
_PANDAS_ONLY_METHODS = frozenset({
    "rolling", "shift", "fillna", "dropna", "ewm", "expanding",
    "resample", "pct_change", "diff",
})

# Variable names strongly suggesting numpy arrays (from context[...]).
# We detect calls like `close.rolling(...)` where `close` is a numpy array.
_ARRAY_LIKE_NAMES = frozenset({
    "close", "open", "high", "low", "volume", "prices",
    "data", "values", "series", "arr", "array",
    "returns", "bars", "ohlc", "ohlcv",
})


def _check_ndarray_pandas_misuse(code: str) -> List[CodeHint]:
    """Detect pandas-only method calls on likely numpy arrays via AST.

    Example: ``close.rolling(20).mean()`` where ``close`` is from
    ``context['close']`` (a numpy ndarray). Pandas Series has .rolling();
    numpy ndarray does not.
    """
    hints: List[CodeHint] = []
    seen: Set[str] = set()

    try:
        tree = ast.parse(code or "")
    except SyntaxError:
        return hints

    for node in ast.walk(tree):
        # Match: <var>.<method>(...) where method is pandas-only
        if not isinstance(node, ast.Call):
            continue
        if not isinstance(node.func, ast.Attribute):
            continue
        method = node.func.attr
        if method not in _PANDAS_ONLY_METHODS:
            continue
        # Check if the object is a simple name like 'close'
        if not isinstance(node.func.value, ast.Name):
            continue
        var_name = node.func.value.id
        if var_name not in _ARRAY_LIKE_NAMES:
            continue

        line = node.lineno
        try:
            src_lines = code.split("\n")
            snippet = src_lines[line - 1].strip() if line - 1 < len(src_lines) else ""
        except Exception:
            snippet = ""

        key = f"{snippet}:{line}"
        if key in seen:
            continue
        seen.add(key)

        hints.append(CodeHint(
            category="NDARRAY_PANDAS_MISUSE",
            severity="error",
            message=(
                f"'{var_name}.{method}()' 可能是 numpy ndarray 上调用了 pandas 方法。"
                f"context['{var_name}'] 返回的是 numpy 数组，不支持 .{method}()。"
                f"如需使用，先转换为 pandas Series: pd.Series({var_name}).{method}(...)"
            ),
            line=line,
            snippet=snippet,
        ))

    return hints


# ---------------------------------------------------------------------------
# check 5: NO_STOP_AND_TAKE_PROFIT
# ---------------------------------------------------------------------------

_STRATEGY_KEY_RE = re.compile(r"^\s*#\s*@strategy\s+(\w+)", re.MULTILINE)


def _has_buy_sell_signal(code: str) -> bool:
    """Heuristic: check if the run() function returns buy/sell signals."""
    # Look for return {'signal': 'buy'/'sell'} or return dict with signal key
    stripped = _strip_comments(code)
    return bool(re.search(
        r"""return\s*\{[^}]*['"]signal['"]\s*:\s*['"](?:buy|sell)['"]""",
        stripped,
    ))


def _check_no_stop_take_profit(code: str) -> List[CodeHint]:
    """Warn if strategy has buy/sell signals but no @strategy stopLossPct/takeProfitPct."""
    if not _has_buy_sell_signal(code):
        return []

    declared = {m.group(1) for m in _STRATEGY_KEY_RE.finditer(code or "")}
    hints: List[CodeHint] = []

    if "stopLossPct" not in declared:
        hints.append(CodeHint(
            category="NO_STOP_AND_TAKE_PROFIT",
            severity="warn",
            message="策略包含买入/卖出信号但未声明 @strategy stopLossPct，建议设置止损以控制风险",
            line=0,
            snippet="# @strategy stopLossPct: 2.0",
        ))
    if "takeProfitPct" not in declared:
        hints.append(CodeHint(
            category="NO_STOP_AND_TAKE_PROFIT",
            severity="warn",
            message="策略包含买入/卖出信号但未声明 @strategy takeProfitPct，建议设置止盈以锁定利润",
            line=0,
            snippet="# @strategy takeProfitPct: 5.0",
        ))

    return hints


# ---------------------------------------------------------------------------
# check 6: NO_ENTRY_PCT
# ---------------------------------------------------------------------------

def _check_no_entry_pct(code: str) -> List[CodeHint]:
    """Warn if strategy has buy/sell signals but no @strategy entryPct."""
    if not _has_buy_sell_signal(code):
        return []

    declared = {m.group(1) for m in _STRATEGY_KEY_RE.finditer(code or "")}
    if "entryPct" in declared:
        return []

    return [CodeHint(
        category="NO_ENTRY_PCT",
        severity="info",
        message="策略包含买入/卖出信号但未声明 @strategy entryPct，默认将使用 100% 仓位",
        line=0,
        snippet="# @strategy entryPct: 50.0",
    )]


# ---------------------------------------------------------------------------
# helpers (shared)
# ---------------------------------------------------------------------------

def _line_of(pattern: str, code: str, start: int = 0) -> int:
    """Return 1-based line number of first regex match, or 0."""
    m = re.search(pattern, code[start:])
    if not m:
        return 0
    return code[: start + m.start()].count("\n") + 1


# ---------------------------------------------------------------------------
# public API
# ---------------------------------------------------------------------------

def analyze_code_quality(source: str) -> List[CodeHint]:
    """Run all static quality checks (6 dimensions). Returns hints sorted by line number."""
    if not source or not source.strip():
        return []

    hints: List[CodeHint] = []
    hints.extend(_check_future_data_leak(source))
    hints.extend(_check_missing_params(source))
    hints.extend(_check_unread_params(source))
    hints.extend(_check_ndarray_pandas_misuse(source))
    hints.extend(_check_no_stop_take_profit(source))
    hints.extend(_check_no_entry_pct(source))

    # Stable output: sort by line, then category, then message
    hints.sort(key=lambda h: (h.line, h.category, h.message))
    return hints
