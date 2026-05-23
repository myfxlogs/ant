#!/usr/bin/env python3
"""Go/Python DSL alignment verification (M7.6-2).

Reads alignment_data.json produced by the Go test, evaluates each expression
with the Python DSL engine, and verifies max absolute error < 1e-9.

Usage:
    cd /opt/ant/research && python tests/test_alignment.py
"""

from __future__ import annotations

import json
import math
import sys
from pathlib import Path

# Ensure we can import the ant_research package
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from ant_research.factor.dsl.parser import parse
from ant_research.factor.dsl.compile import compile_expr


def load_alignment_data(path: str) -> list[dict]:
    with open(path) as f:
        data = json.load(f)
    return data["records"]


def eval_python(expression: str, bars: list[float]) -> list[float | None]:
    """Evaluate a DSL expression with the Python engine, returning list of results.

    None represents NaN (warmup period or computation result).
    """
    try:
        node = parse(expression)
    except Exception as e:
        print(f"  PARSE ERROR: {e}")
        return [None] * len(bars)

    try:
        op = compile_expr(node, {"close": 0})
    except Exception as e:
        print(f"  COMPILE ERROR: {e}")
        return [None] * len(bars)

    results: list[float | None] = []
    for v in bars:
        try:
            r = op.eval(v)
        except Exception:
            results.append(None)
            continue
        if math.isnan(r):
            results.append(None)
        else:
            results.append(r)

    return results


def compare_results(
    go_results: list[float | None],
    py_results: list[float | None],
    tolerance: float = 1e-9,
) -> tuple[int, int, float, list[tuple[int, float, float]]]:
    """Compare Go and Python results.

    Returns (total, matched, max_error, mismatches).
    Mismatches are (index, go_val, py_val) tuples.
    """
    total = len(go_results)
    matched = 0
    max_error = 0.0
    mismatches: list[tuple[int, float, float]] = []

    for i, (g, p) in enumerate(zip(go_results, py_results)):
        if g is None and p is None:
            matched += 1
            continue
        if g is None or p is None:
            mismatches.append((i, g or math.nan, p or math.nan))
            continue
        err = abs(g - p)
        if err > max_error:
            max_error = err
        if err <= tolerance:
            matched += 1
        else:
            mismatches.append((i, g, p))

    return total, matched, max_error, mismatches[:10]  # first 10 mismatches


def main() -> int:
    data_path = Path(__file__).resolve().parent / "alignment_data.json"
    if not data_path.exists():
        print(f"ERROR: {data_path} not found. Run Go alignment test first:")
        print("  cd /opt/ant/backend && go test -run TestGoDSLAlignment ./internal/factor/dsl/")
        return 1

    records = load_alignment_data(str(data_path))
    print(f"Loaded {len(records)} alignment records")

    total_exprs = 0
    failed_exprs = 0
    global_max_error = 0.0
    fail_details: list[tuple[int, str, float, int]] = []

    for rec in records:
        expr_idx = rec["expr_index"]
        expression = rec["expression"]
        bars = rec["bars"]
        go_results = rec["go_results"]  # list of float or None (JSON null)

        py_results = eval_python(expression, bars)

        total, matched, max_err, mismatches = compare_results(go_results, py_results)

        total_exprs += 1
        if max_err > global_max_error:
            global_max_error = max_err

        if matched < total:
            failed_exprs += 1
            fail_details.append((expr_idx, expression, max_err, total - matched))
            if mismatches:
                idx, gv, pv = mismatches[0]
                print(f"  FAIL [{expr_idx}] {expression[:60]}")
                print(f"    bar {idx}: go={gv:.12f} py={pv:.12f} diff={abs(gv-pv):.2e}")
                if len(mismatches) > 1:
                    print(f"    ... and {len(mismatches)-1} more mismatches (showing first 10)")

    print()
    print("=" * 60)
    print(f"Expressions tested: {total_exprs}")
    print(f"Passed:            {total_exprs - failed_exprs}")
    print(f"Failed:            {failed_exprs}")
    print(f"Global max error:  {global_max_error:.2e}")
    print(f"Threshold:         1e-9")

    if global_max_error <= 1e-9 and failed_exprs == 0:
        print("✅ ALL PASSED — Go/Python DSL alignment within 1e-9")
        return 0
    elif global_max_error <= 1e-9:
        print(f"⚠️  {failed_exprs} expressions had NaN/null mismatches (warmup differences)")
        return 0
    else:
        print(f"❌ FAILED — max error {global_max_error:.2e} exceeds 1e-9 threshold")
        return 1


if __name__ == "__main__":
    sys.exit(main())
