"""Tree-sitter MQL parser — single .so loader + parse → raw Tree.

ADR-0020 D8: this is the ONE AND ONLY MQL parser.  The grammar is compiled
from ``grammar/mql/grammar.js`` (fork of tree-sitter-c) into ``mql.so``.

For the CST→internal-AST bridge that produces ``ast_nodes`` types, use
``ast_bridge.parse_mql()`` instead.
"""

from __future__ import annotations

import ctypes
from pathlib import Path

_TS_LANG = None
_TS_PARSER = None


def _init():
    """Lazy-init tree-sitter parser for MQL."""
    global _TS_LANG, _TS_PARSER
    if _TS_PARSER is not None:
        return

    try:
        import tree_sitter as ts
    except ImportError:
        return

    so_path = Path(__file__).parent / "grammar" / "mql" / "mql.so"
    if not so_path.exists():
        return

    lib = ctypes.CDLL(str(so_path))
    lang_fn = lib.tree_sitter_mql
    lang_fn.restype = ctypes.c_void_p
    ptr = lang_fn()

    import warnings
    with warnings.catch_warnings():
        warnings.simplefilter("ignore", DeprecationWarning)
        _TS_LANG = ts.Language(ptr)

    _TS_PARSER = ts.Parser(_TS_LANG)


def available() -> bool:
    """Check if the tree-sitter MQL grammar is loadable."""
    _init()
    return _TS_PARSER is not None


def parse(source: str):
    """Parse MQL source and return a raw ``tree_sitter.Tree``.

    Returns None if tree-sitter is not available.
    For internal AST nodes, use ``ast_bridge.parse_mql()``.
    """
    _init()
    if _TS_PARSER is None:
        return None
    return _TS_PARSER.parse(source.encode("utf-8"))
