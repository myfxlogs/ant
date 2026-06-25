"""MQL standard library mapping coverage tracker.

Scans transpiler output for untranslated MQL artifacts (bare function names,
unconverted operators, unmapped constants) and records them to a knowledge
base for prioritised gap-closing.

Wire into the transpile pipeline after codegen::

    from tools.mql_transpiler.gap_kb import scan_unmapped, record_gaps
    unmapped = scan_unmapped(transpiled_output)
    record_gaps(unmapped, source_mql=original_mql)

KB file: ``tools/mql_transpiler/kb/unmapped_patterns.json``
"""

from __future__ import annotations

import json
import os.path
import re
from dataclasses import dataclass, asdict
from datetime import datetime, timezone
from typing import Dict, List, Set

KB_PATH = os.path.join(os.path.dirname(__file__), "kb", "unmapped_patterns.json")

# ── MQL functions that SHOULD be mapped to Python equivalents ──────────

_MQL_TIME_FUNCTIONS = {
    "Day", "Month", "Year", "Hour", "Minute", "Seconds",
    "DayOfWeek", "DayOfYear", "TimeDay", "TimeCurrent", "TimeLocal",
}

_MQL_MATH_FUNCTIONS = {
    "NormalizeDouble", "MathPow", "MathLog", "MathSqrt", "MathExp",
    "MathSin", "MathCos", "MathTan", "MathArcsin", "MathArccos",
    "MathArctan", "MathMod", "MathRand", "MathSrand", "MathAbs",
    "MathMax", "MathMin", "MathRound", "MathFloor", "MathCeil",
}

_MQL_STRING_FUNCTIONS = {
    "StringConcatenate", "StringFind", "StringLen", "StringSubstr",
    "StringReplace", "StringToDouble", "DoubleToString",
    "IntegerToString", "StringToInteger",
}

_MQL_TRADE_FUNCTIONS = {
    "OrdersTotal", "PositionsTotal", "OrderSelect", "PositionSelect",
    "OrderSend", "OrderClose", "OrderModify", "OrderDelete",
    "OrderTicket", "OrderLots", "OrderMagicNumber", "OrderSymbol",
    "OrderType", "OrderOpenPrice", "OrderClosePrice", "OrderStopLoss",
    "OrderTakeProfit", "OrderComment", "OrderCommission", "OrderSwap",
    "OrderProfit", "OrderOpenTime", "OrderCloseTime",
}

_MQL_ACCOUNT_FUNCTIONS = {
    "AccountBalance", "AccountEquity", "AccountFreeMargin",
    "AccountMargin", "AccountLeverage", "AccountNumber",
}

_MQL_MARKET_FUNCTIONS = {
    "MarketInfo", "SymbolInfoDouble", "SymbolInfoInteger",
}

_MQL_ARRAY_FUNCTIONS = {
    "ArrayInitialize", "ArrayResize", "ArrayCopy", "ArraySize",
    "ArrayGetAsSeries", "ArraySetAsSeries",
}

_MQL_OPERATORS = {
    "!": "not ",    # Only in expressions (not !=)
    "!=": "!=",     # Keep as-is
}

_ALL_KNOWN_MQL = (
    _MQL_TIME_FUNCTIONS | _MQL_MATH_FUNCTIONS | _MQL_STRING_FUNCTIONS |
    _MQL_TRADE_FUNCTIONS | _MQL_ACCOUNT_FUNCTIONS | _MQL_MARKET_FUNCTIONS |
    _MQL_ARRAY_FUNCTIONS
)


@dataclass
class UnmappedEntry:
    """A single unmapped MQL artifact found in transpiler output."""
    symbol: str           # e.g. "NormalizeDouble", "!", "Day"
    category: str         # "math", "time", "string", "trade", "operator", "array"
    context: str          # The Python line containing it
    count: int = 1
    first_seen: str = ""
    last_seen: str = ""


def scan_unmapped(python_output: str) -> List[UnmappedEntry]:
    """Scan Python output for MQL artifacts that weren't translated.

    Returns a list of ``UnmappedEntry`` — one per unique unmapped symbol.
    """
    found: Dict[str, UnmappedEntry] = {}

    for line in python_output.split("\n"):
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue

        # Detect MQL function calls that still appear as Python identifiers.
        for name in _ALL_KNOWN_MQL:
            # Look for bare name usage: fn(, fn (, .fn(, self.fn(
            if re.search(rf"\b{re.escape(name)}\s*\(", stripped):
                # Exclude already-mapped patterns.
                if name in ("OrderSend", "OrderClose", "OrderModify", "OrderDelete",
                             "OrderSelect", "PositionSelect", "MarketInfo",
                             "OrderTicket", "OrderLots", "OrderMagicNumber",
                             "OrderSymbol", "OrderType", "OrderOpenPrice",
                             "OrderClosePrice"):
                    continue  # These are handled by codegen, may appear in comments

                cat = _categorize(name)
                if name not in found:
                    found[name] = UnmappedEntry(
                        symbol=name, category=cat,
                        context=stripped[:120],
                    )
                else:
                    found[name].count += 1
                    found[name].context = stripped[:120]

        # Detect MQL ! operator (not inside !=).
        if "!self." in stripped or "!Use" in stripped or "!Open" in stripped:
            if "!=" not in stripped:
                name = "! (MQL not)"
                if name not in found:
                    found[name] = UnmappedEntry(
                        symbol="!", category="operator",
                        context=stripped[:120],
                    )
                else:
                    found[name].count += 1

    return list(found.values())


def _categorize(name: str) -> str:
    if name in _MQL_TIME_FUNCTIONS: return "time"
    if name in _MQL_MATH_FUNCTIONS: return "math"
    if name in _MQL_STRING_FUNCTIONS: return "string"
    if name in _MQL_TRADE_FUNCTIONS: return "trade"
    if name in _MQL_ACCOUNT_FUNCTIONS: return "account"
    if name in _MQL_MARKET_FUNCTIONS: return "market"
    if name in _MQL_ARRAY_FUNCTIONS: return "array"
    return "other"


def record_gaps(entries: List[UnmappedEntry], source_mql: str = "") -> Dict[str, str]:
    """Record unmapped symbols to the knowledge base.

    Returns a summary dict of {symbol: status} for display.
    """
    if not entries:
        return {}

    os.makedirs(os.path.dirname(KB_PATH), exist_ok=True)
    kb = _load_kb()
    now = datetime.now(timezone.utc).isoformat()
    summary = {}

    for entry in entries:
        key = entry.symbol
        if key in kb:
            kb[key]["count"] += entry.count
            kb[key]["last_seen"] = now
            summary[key] = f"updated (total={kb[key]['count']})"
        else:
            kb[key] = asdict(entry)
            kb[key]["first_seen"] = now
            kb[key]["last_seen"] = now
            summary[key] = "new"

    _save_kb(kb)
    return summary


def _load_kb() -> dict:
    if not os.path.exists(KB_PATH):
        return {}
    try:
        with open(KB_PATH) as f:
            return json.load(f)
    except Exception:
        return {}


def _save_kb(kb: dict) -> None:
    with open(KB_PATH, "w") as f:
        json.dump(kb, f, indent=2, ensure_ascii=False)


def coverage_report() -> dict:
    """Return a coverage report: how many MQL functions are mapped vs unmapped."""
    kb = _load_kb()
    by_category = {}
    for entry in kb.values():
        cat = entry.get("category", "other")
        by_category.setdefault(cat, []).append(entry["symbol"])

    return {
        "total_unmapped_symbols": len(kb),
        "by_category": {k: len(v) for k, v in by_category.items()},
        "unmapped": {k: sorted(v) for k, v in by_category.items()},
        "total_mql_functions_known": len(_ALL_KNOWN_MQL),
        "mapped_pct": round(
            (1 - len(kb) / max(len(_ALL_KNOWN_MQL), 1)) * 100, 1
        ),
    }
