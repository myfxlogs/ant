"""Tree-sitter MQL parser — dual grammar (C for MQL4, C++ for MQL5).

ADR-0020 D8: MQL4 ≈ C subset, MQL5 ≈ C++ subset.  This module auto-detects
the MQL dialect and selects the appropriate grammar.

Grammar files:
  grammar/mql/mql.so   — tree-sitter-c (MQL4, procedural)
  grammar/mql5/mql5.so — tree-sitter-cpp (MQL5, class/OOP)
"""

from __future__ import annotations

import ctypes
from pathlib import Path

_TS_LANG_C = None
_TS_LANG_CPP = None
_TS_PARSER_C = None
_TS_PARSER_CPP = None


def _init():
    """Lazy-init tree-sitter parsers for both MQL dialects."""
    global _TS_LANG_C, _TS_LANG_CPP, _TS_PARSER_C, _TS_PARSER_CPP
    if _TS_PARSER_C is not None:
        return

    try:
        import tree_sitter as ts
    except ImportError:
        return

    import warnings
    base = Path(__file__).parent / "grammar"

    # MQL4: tree-sitter-c
    so_c = base / "mql" / "mql.so"
    if so_c.exists():
        lib_c = ctypes.CDLL(str(so_c))
        fn_c = lib_c.tree_sitter_mql
        fn_c.restype = ctypes.c_void_p
        with warnings.catch_warnings():
            warnings.simplefilter("ignore", DeprecationWarning)
            _TS_LANG_C = ts.Language(fn_c())
        _TS_PARSER_C = ts.Parser(_TS_LANG_C)

    # MQL5: tree-sitter-cpp
    so_cpp = base / "mql5" / "mql5.so"
    if so_cpp.exists():
        lib_cpp = ctypes.CDLL(str(so_cpp))
        fn_cpp = lib_cpp.tree_sitter_cpp
        fn_cpp.restype = ctypes.c_void_p
        with warnings.catch_warnings():
            warnings.simplefilter("ignore", DeprecationWarning)
            _TS_LANG_CPP = ts.Language(fn_cpp())
        _TS_PARSER_CPP = ts.Parser(_TS_LANG_CPP)


def available() -> bool:
    """Check if at least one MQL grammar is loadable."""
    _init()
    return _TS_PARSER_C is not None or _TS_PARSER_CPP is not None


def available_mql4() -> bool:
    _init()
    return _TS_PARSER_C is not None


def available_mql5() -> bool:
    _init()
    return _TS_PARSER_CPP is not None


# MQL5-specific keywords that indicate class-based EA.
_MQL5_CLASS_KEYWORDS = {"class", "virtual", "override", "template", "namespace",
                         "public:", "private:", "protected:", "typename"}


def _is_mql5(source: str) -> bool:
    """Heuristic: does source contain MQL5 class/OOP syntax?"""
    # Quick scan for MQL5-specific keywords.
    tokens = set(source.split())
    return bool(tokens & _MQL5_CLASS_KEYWORDS)


def parse(source: str):
    """Parse MQL source with auto-detection of MQL4 vs MQL5.

    Returns a ``tree_sitter.Tree``, or None if no grammar is available.
    """
    _init()

    source_bytes = source.encode("utf-8")

    # Try MQL5 grammar first for class-based EAs.
    if _is_mql5(source) and _TS_PARSER_CPP is not None:
        tree = _TS_PARSER_CPP.parse(source_bytes)
        # Heuristic: if MQL5 grammar has minimal errors, use it.
        if not tree.root_node.has_error or _error_count(tree.root_node) <= 2:
            return tree

    # Default: MQL4 grammar (C-based).
    if _TS_PARSER_C is not None:
        return _TS_PARSER_C.parse(source_bytes)

    # Fallback: MQL5 grammar for any source.
    if _TS_PARSER_CPP is not None:
        return _TS_PARSER_CPP.parse(source_bytes)

    return None


def _error_count(node) -> int:
    n = 1 if node.type == "ERROR" else 0
    for child in node.children:
        n += _error_count(child)
    return n
