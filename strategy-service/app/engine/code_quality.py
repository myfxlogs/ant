"""
Static code quality analysis for ant strategy Python code.

Read-only analysis — does NOT execute user code. Adapted from QuantDinger's
indicator_code_quality.py for our ``run(context)`` event-driven model.

Three checks:
  1. FUTURE_DATA_LEAK  — forward indexing inside loops (lookahead bias)
  2. MISSING_PARAM     — params.get('x') without @param x declaration
  3. UNREAD_PARAM      — @param x declared but never read
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from typing import List, Set


@dataclass
class CodeHint:
    category: str   # "FUTURE_DATA_LEAK" | "MISSING_PARAM" | "UNREAD_PARAM"
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
    # Match: p.get('key', ...) or params.get("key", ...)
    for m in re.finditer(
        r"""\b(?:p|params)\s*\.\s*get\s*\(\s*['"]([^'"]+)['"]""",
        code or "",
    ):
        keys.add(m.group(1))
    return keys


def _line_of(pattern: str, code: str, start: int = 0) -> int:
    """Return 1-based line number of first regex match, or 0."""
    m = re.search(pattern, code[start:])
    if not m:
        return 0
    return code[: start + m.start()].count("\n") + 1


# ---------------------------------------------------------------------------
# check 1: FUTURE_DATA_LEAK
# ---------------------------------------------------------------------------

# Forward indexing inside a loop: variable[i+N] where N > 0.
# We look for array-like names used with [<var>+<int>] inside for-loops.
_FOR_LOOP_RE = re.compile(
    r"^\s*for\s+(\w+)\s+in\s+range\s*\(\s*len\s*\(\s*(\w+)\s*\)",
    re.MULTILINE,
)
_FORWARD_IDX_RE = re.compile(
    r"(\w+)\s*\[\s*(\w+)\s*\+\s*(\d+)\s*\]"
)


def _check_future_data_leak(src: str) -> List[CodeHint]:
    """Detect forward indexing (i+N) inside for-loops — lookahead bias."""
    hints: List[CodeHint] = []
    seen: Set[tuple[str, str]] = set()

    # Find all for-loops and their loop vars
    for loop_m in _FOR_LOOP_RE.finditer(src):
        loop_var = loop_m.group(1)     # e.g. "i"
        arr_name = loop_m.group(2)     # e.g. "prices"
        loop_start = loop_m.start()
        loop_line = src[:loop_start].count("\n") + 1

        # Find the loop body end (next line at same or lesser indent)
        lines = src[loop_start:].split("\n")
        indent_match = re.match(r"^(\s*)", loop_m.group(0))
        loop_indent = len(indent_match.group(1)) if indent_match else 0
        body_end = loop_start + len(loop_m.group(0))
        for body_line in lines[1:]:  # skip the 'for' line itself
            body_end += len(body_line) + 1  # +1 for \n
            stripped = body_line.strip()
            if not stripped:
                continue
            body_indent = len(body_line) - len(body_line.lstrip())
            if body_indent <= loop_indent and not stripped.startswith("#"):
                break

        body = src[loop_start:body_end]

        # Find forward indexing using the loop variable
        for idx_m in _FORWARD_IDX_RE.finditer(body):
            target = idx_m.group(1)
            idx_var = idx_m.group(2)
            n_val = int(idx_m.group(3))
            if idx_var == loop_var and n_val > 0:
                snippet = idx_m.group(0)
                key = (snippet, str(loop_line))
                if key in seen:
                    continue
                seen.add(key)
                hints.append(CodeHint(
                    category="FUTURE_DATA_LEAK",
                    severity="error",
                    message=f"循环内使用了未来数据：{snippet}，索引 +{n_val} 引用了未来 K 线",
                    line=loop_line,
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
        # Find the line of the first p.get('key') usage
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
            # Find the line of the @param declaration
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
# public API
# ---------------------------------------------------------------------------

def analyze_code_quality(source: str) -> List[CodeHint]:
    """Run all static quality checks. Returns hints sorted by line number."""
    if not source or not source.strip():
        return []

    hints: List[CodeHint] = []
    hints.extend(_check_future_data_leak(source))
    hints.extend(_check_missing_params(source))
    hints.extend(_check_unread_params(source))

    # Stable output: sort by line, then category, then message
    hints.sort(key=lambda h: (h.line, h.category, h.message))
    return hints
