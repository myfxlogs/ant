#!/usr/bin/env python3
"""MQL → Python Strategy SDK transpiler — CLI.

Usage:
    python3 -m tools.mql_transpiler.main input.mq4 -o output.py
    python3 -m tools.mql_transpiler.main input.mq4           # prints to stdout
"""

from __future__ import annotations

import argparse
import sys

from tools.mql_transpiler.ast_transpiler import ASTTranspiler
from tools.mql_transpiler.quality_gate import QualityGate, QualityVerdict
from tools.mql_transpiler.gap_kb import record_gaps


def main() -> None:
    parser = argparse.ArgumentParser(
        description="MQL4/5 → Python Strategy SDK transpiler (tree-sitter, deterministic)",
    )
    parser.add_argument("input", help="MQL source file (.mq4/.mq5)")
    parser.add_argument("-o", "--output", help="Output Python file (default: stdout)")
    parser.add_argument(
        "-c", "--class-name", default="TranslatedStrategy",
        help="Python class name (default: TranslatedStrategy)",
    )
    parser.add_argument(
        "--stats", action="store_true",
        help="Print quality report to stderr",
    )
    args = parser.parse_args()

    with open(args.input) as f:
        source = f.read()

    tp = ASTTranspiler(class_name=args.class_name)
    result = tp.transpile(source)

    # Gate-based confidence (ADR-0020 D8, C2).
    gate = QualityGate.assess(result.output)
    confidence = "HIGH" if gate.verdict == QualityVerdict.HIGH else "LOW"

    if args.output:
        with open(args.output, "w") as f:
            f.write(result.output)
        print(f"Wrote {len(result.output.splitlines())} lines to {args.output}", file=sys.stderr)
    else:
        print(result.output)

    if args.stats or True:
        gaps = result.stats.get("gaps", 0)
        print(f"\n--- Transpiler Report ---", file=sys.stderr)
        print(f"  Confidence: {confidence}", file=sys.stderr)
        print(f"  Compiles:   {gate.compile_ok}", file=sys.stderr)
        print(f"  SDK imports:{gate.sdk_import_ok}", file=sys.stderr)
        print(f"  Lint clean: {gate.lint_ok}", file=sys.stderr)
        print(f"  GAPs:       {gaps}", file=sys.stderr)
        if gate.failures:
            print(f"  Gate failures:", file=sys.stderr)
            for f in gate.failures:
                print(f"    [{f.gate}] {f.message}", file=sys.stderr)
        if gate.warnings:
            for w in gate.warnings:
                print(f"  ⚠ {w}", file=sys.stderr)


if __name__ == "__main__":
    main()
