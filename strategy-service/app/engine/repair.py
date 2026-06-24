"""Programmatic code repair — deterministic, zero-cost fixes for mechanical issues.

Applied BEFORE LLM-based repair.  Handles issues that can be fixed with
AST transformation alone, saving tokens and latency.

Current fixes:
  - Rename ``_``-prefixed identifiers → ``user_`` prefix
"""

from __future__ import annotations

import ast

_SAFE_PREFIX = "user_"


# --- Underscore-name transform --------------------------------------------


def _transform_underscore_names(source: str) -> str:
    """Rename ``_``-prefixed identifiers (not dunder, not bare ``_``)."""

    class _UnderscoreRenamer(ast.NodeTransformer):
        @staticmethod
        def _safe(name: str) -> str:
            if name.startswith("_") and not name.startswith("__") and name != "_":
                return _SAFE_PREFIX + name[1:]
            return name

        def visit_FunctionDef(self, node):
            node.name = self._safe(node.name)
            return self.generic_visit(node)

        def visit_Name(self, node):
            node.id = self._safe(node.id)
            return node

        def visit_Attribute(self, node):
            node.attr = self._safe(node.attr)
            return self.generic_visit(node)

        def visit_arg(self, node):
            node.arg = self._safe(node.arg)
            return node

    tree = ast.parse(source)
    transformed = _UnderscoreRenamer().visit(tree)
    ast.fix_missing_locations(transformed)
    return ast.unparse(transformed)


# --- Public API -----------------------------------------------------------


def repair_code_programmatic(code: str) -> tuple[str, list[str]]:
    """Apply deterministic, zero-cost fixes to strategy code.

    Returns ``(fixed_code, fixes_applied)``.  Call BEFORE LLM repair to
    handle mechanical issues without spending tokens.
    """
    fixes: list[str] = []
    result = code

    # Fix 1: rename _-prefixed identifiers
    underscored = _collect_underscore_identifiers(result)
    if underscored:
        result = _transform_underscore_names(result)
        fixes.append(
            f"重命名 _-prefix 标识符: {', '.join(underscored)} → "
            f"{', '.join(_SAFE_PREFIX + n[1:] for n in underscored)}"
        )

    return result, fixes


def _collect_underscore_identifiers(source: str) -> list[str]:
    """Collect ``_``-prefixed identifiers (not dunder, not bare ``_``)."""
    try:
        tree = ast.parse(source)
    except SyntaxError:
        return []
    names: set[str] = set()
    for node in ast.walk(tree):
        if isinstance(node, ast.FunctionDef):
            if node.name.startswith("_") and not node.name.startswith("__") and node.name != "_":
                names.add(node.name)
        if isinstance(node, ast.Attribute):
            if node.attr.startswith("_") and not node.attr.startswith("__") and node.attr != "_":
                names.add(node.attr)
        if isinstance(node, ast.Name):
            if node.id.startswith("_") and not node.id.startswith("__") and node.id != "_":
                names.add(node.id)
        if isinstance(node, ast.arg):
            if node.arg.startswith("_") and not node.arg.startswith("__") and node.arg != "_":
                names.add(node.arg)
    return sorted(names)
