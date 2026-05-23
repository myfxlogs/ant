#!/usr/bin/env python3
"""migrate_legacy_strategy.py — 老 Python 策略 → DSL 自动迁移工具。

M7.7-3: 把策略数据库中 strategies.code（Python）自动转换为 DSL 表达式。
可自动转换的策略转为 DSL 并标记 migrated=true；
不可转换的标记为 unconvertible 并给出原因。

Usage:
    python migrate_legacy_strategy.py [--dry-run] [--strategy-id UUID] [--batch-size N]
"""

from __future__ import annotations

import ast
import json
import sys
from dataclasses import dataclass, field
from typing import Optional

# ── DSL 函数名映射（Python indicator → DSL function）─────────────────────

_INDICATOR_MAP: dict[str, str] = {
    # Moving averages
    "iMA": "sma",  # Simple MA is the closest match
    "iEMA": "ema",
    "SMA": "sma",
    "EMA": "ema",
    "WMA": "wma",
    # Oscillators
    "iRSI": "rsi",
    "iMACD": "macd",
    "iATR": "atr",
    "RSI": "rsi",
    "MACD": "macd",
    "ATR": "atr",
    # Bands
    "iBands": "bb_upper",  # partial — upper band only
    "BB_Upper": "bb_upper",
    "BB_Lower": "bb_lower",
    # Statistics
    "iStdDev": "std",
    "STD": "std",
    "VAR": "var",
    # Price
    "iClose": "$close",
    "iOpen": "$open",
    "iHigh": "$high",
    "iLow": "$low",
    "iVolume": "$volume",
    "Close": "$close",
    "Open": "$open",
    "High": "$high",
    "Low": "$low",
    "Volume": "$volume",
    # Scalar
    "MathAbs": "abs",
    "MathLog": "log",
    "MathSqrt": "sqrt",
    # Cross
    "CrossUp": "cross_up",
    "CrossDown": "cross_down",
    # Momentum
    "Momentum": "delta",
    "PctChange": "pct_change",
    "ZScore": "zscore",
    "Rank": "rank",
    "Correlation": "corr",
    "Covariance": "cov",
}

# Patterns that indicate the strategy cannot be auto-converted
_UNCONVERTIBLE_PATTERNS: list[str] = [
    "for ", "while ", "def ", "class ",
    "import ", "from ", "open(", "exec(", "eval(",
    "print(", "input(", "global ", "nonlocal ",
    "lambda ", "yield", "raise ", "try:", "except",
    "with ", "as ", "assert ",
    "len(", "range(", "enumerate(", "zip(",
    "dict(", "list(", "set(", "tuple(",
]


@dataclass
class MigrationResult:
    strategy_id: str
    original_code: str
    migrated: bool = False
    dsl_expression: str = ""
    warnings: list[str] = field(default_factory=list)
    errors: list[str] = field(default_factory=list)


def is_convertible(code: str) -> tuple[bool, str]:
    """Check if a Python strategy can be auto-converted to DSL."""
    for pattern in _UNCONVERTIBLE_PATTERNS:
        if pattern in code:
            return False, f"contains {pattern!r} — DSL does not support imperative programming"
    return True, ""


def convert_to_dsl(code: str) -> tuple[str, list[str], list[str]]:
    """Best-effort conversion of simple Python strategies to DSL expressions.

    Returns (dsl_expression, warnings, errors).
    """
    warnings: list[str] = []
    errors: list[str] = []

    # Try to parse as Python
    try:
        tree = ast.parse(code)
    except SyntaxError as e:
        errors.append(f"syntax error: {e}")
        return "", warnings, errors

    # Find signal assignments / return values
    dsl_parts: list[str] = []

    for node in ast.walk(tree):
        if isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name) and target.id == "signal":
                    dsl_parts.append(_expr_to_dsl(node.value))

    if not dsl_parts:
        # Try extracting return value from run() function
        for node in ast.walk(tree):
            if isinstance(node, ast.Return) and node.value:
                dsl_parts.append(_expr_to_dsl(node.value))

    if not dsl_parts:
        errors.append("could not locate signal assignment or return value")
        return "", warnings, errors

    dsl_expr = " && ".join(dsl_parts)
    return dsl_expr, warnings, errors


def _expr_to_dsl(node: ast.AST, depth: int = 0) -> str:
    """Recursively convert a Python AST expression node to DSL syntax."""
    if depth > 10:
        return "0"  # safety limit

    if isinstance(node, ast.Constant):
        if isinstance(node.value, bool):
            return "true" if node.value else "false"
        if isinstance(node.value, (int, float)):
            return str(node.value)
        if isinstance(node.value, str):
            return f'"{node.value}"'
        return "0"

    if isinstance(node, ast.Name):
        name = node.id
        if name in _INDICATOR_MAP:
            return _INDICATOR_MAP[name]
        if name.startswith("$"):
            return name
        return name

    if isinstance(node, ast.Call):
        fn_name = ""
        if isinstance(node.func, ast.Name):
            fn_name = node.func.id
        elif isinstance(node.func, ast.Attribute):
            fn_name = node.func.attr

        if fn_name in _INDICATOR_MAP:
            fn_name = _INDICATOR_MAP[fn_name]

        args = [_expr_to_dsl(a, depth + 1) for a in node.args]
        return f"{fn_name}({', '.join(args)})"

    if isinstance(node, ast.BinOp):
        left = _expr_to_dsl(node.left, depth + 1)
        right = _expr_to_dsl(node.right, depth + 1)
        op_map = {
            ast.Add: "+", ast.Sub: "-", ast.Mult: "*", ast.Div: "/",
            ast.Mod: "%", ast.Pow: "pow",
        }
        op = op_map.get(type(node.op), "?")
        return f"({left} {op} {right})"

    if isinstance(node, ast.Compare):
        left = _expr_to_dsl(node.left, depth + 1)
        parts = []
        for op, comp in zip(node.ops, node.comparators):
            op_map = {
                ast.Gt: ">", ast.GtE: ">=", ast.Lt: "<", ast.LtE: "<=",
                ast.Eq: "==", ast.NotEq: "!=",
            }
            op_str = op_map.get(type(op), "?")
            right = _expr_to_dsl(comp, depth + 1)
            parts.append(f"({left} {op_str} {right})")
        return " && ".join(parts)

    if isinstance(node, ast.BoolOp):
        op = "&&" if isinstance(node.op, ast.And) else "||"
        parts = [_expr_to_dsl(v, depth + 1) for v in node.values]
        return f" {op} ".join(parts)

    if isinstance(node, ast.UnaryOp):
        operand = _expr_to_dsl(node.operand, depth + 1)
        if isinstance(node.op, ast.USub):
            return f"-{operand}"
        if isinstance(node.op, ast.Not):
            return f"!{operand}"

    if isinstance(node, ast.IfExp):
        cond = _expr_to_dsl(node.test, depth + 1)
        true_val = _expr_to_dsl(node.body, depth + 1)
        false_val = _expr_to_dsl(node.orelse, depth + 1)
        return f"{cond} ? {true_val} : {false_val}"

    if isinstance(node, ast.Attribute):
        return _expr_to_dsl(node.value, depth + 1)

    return "0"


def migrate_strategy(strategy_id: str, code: str) -> MigrationResult:
    """Migrate a single strategy from Python to DSL."""
    result = MigrationResult(
        strategy_id=strategy_id,
        original_code=code,
    )

    ok, reason = is_convertible(code)
    if not ok:
        result.errors.append(f"unconvertible: {reason}")
        return result

    dsl_expr, warnings, errors = convert_to_dsl(code)
    if errors:
        result.errors = errors
        return result

    if not dsl_expr.strip():
        result.errors.append("conversion produced empty expression")
        return result

    result.migrated = True
    result.dsl_expression = dsl_expr
    result.warnings = warnings
    return result


# ── CLI ───────────────────────────────────────────────────────────────────


def main() -> None:
    import argparse

    parser = argparse.ArgumentParser(
        description="Migrate legacy Python strategies to DSL expressions (M7.7-3)"
    )
    parser.add_argument("--dry-run", action="store_true", help="Preview conversion without writing")
    parser.add_argument("--strategy-id", type=str, help="Migrate a single strategy by ID")
    parser.add_argument("--batch-size", type=int, default=100, help="Batch size for DB queries")
    parser.add_argument("--output", type=str, default="-", help="Output file (- for stdout)")

    args = parser.parse_args()

    # Demo: convert some sample strategies
    samples = [
        ("demo-001", "signal = iRSI(iClose(None, 0), 14) < 30"),
        ("demo-002", "signal = iClose(None, 0) > iMA(iClose(None, 0), 20)"),
        ("demo-003", "signal = iMACD(iClose(None, 0), 12, 26, 9) > 0"),
        ("demo-004", "signal = iBands(iClose(None, 0), 20, 2, 0, 0, 0, 1)"),
        ("demo-unconvertible", "for i in range(10):\n    signal = iRSI(iClose(None, 0), i)"),
    ]

    results: list[MigrationResult] = []
    for sid, code in samples:
        result = migrate_strategy(sid, code)
        results.append(result)

    if args.output == "-":
        out = sys.stdout
    else:
        out = open(args.output, "w")

    for r in results:
        status = "✅ MIGRATED" if r.migrated else "❌ UNCONVERTIBLE"
        print(f"\n[{status}] {r.strategy_id}", file=out)
        print(f"  Original: {r.original_code[:80]}...", file=out)
        if r.migrated:
            print(f"  DSL:      {r.dsl_expression}", file=out)
        if r.warnings:
            for w in r.warnings:
                print(f"  ⚠️  {w}", file=out)
        if r.errors:
            for e in r.errors:
                print(f"  ❌ {e}", file=out)

    migrated = sum(1 for r in results if r.migrated)
    print(f"\n--- Summary: {migrated}/{len(results)} strategies migrated ---", file=out)

    if args.output != "-":
        out.close()


if __name__ == "__main__":
    main()
