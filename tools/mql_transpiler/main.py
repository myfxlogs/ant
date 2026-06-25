#!/usr/bin/env python3
"""MQL → Python Strategy SDK transpiler — CLI (T2.1).

Usage:
    python3 -m tools.mql_transpiler.main input.mq4 -o output.py
    python3 -m tools.mql_transpiler.main input.mq4           # prints to stdout
"""

from __future__ import annotations

import argparse
import sys

from tools.mql_transpiler.transpiler import MQLTranspiler
from tools.mql_transpiler.gap_kb import record_gaps

CONFIDENCE_HIGH = 0.95
CONFIDENCE_MEDIUM = 0.85


def confidence_label(score: float) -> str:
    if score >= CONFIDENCE_HIGH: return "HIGH"
    if score >= CONFIDENCE_MEDIUM: return "MEDIUM"
    return "LOW"


def main() -> None:
    parser = argparse.ArgumentParser(
        description="MQL4/5 → Python Strategy SDK transpiler (deterministic, auditable)",
    )
    parser.add_argument("input", help="MQL source file (.mq4/.mq5)")
    parser.add_argument("-o", "--output", help="Output Python file (default: stdout)")
    parser.add_argument(
        "-c", "--class-name", default="TranslatedStrategy",
        help="Python class name for the translated strategy (default: TranslatedStrategy)",
    )
    parser.add_argument(
        "--stats", action="store_true",
        help="Print transpilation statistics to stderr",
    )
    args = parser.parse_args()

    with open(args.input, "r") as f:
        source = f.read()

    transpiler = MQLTranspiler(class_name=args.class_name)
    result = transpiler.transpile(source, filename=args.input)

    confidence = transpiler.get_confidence()
    total_patterns = result.stats.patterns_matched + result.stats.gaps
    gap_samples = transpiler.get_gap_samples(10)

    # Record gaps to knowledge base for learning.
    kb_summary = record_gaps(result.stats.gap_reasons)

    if args.output:
        with open(args.output, "w") as f:
            f.write(result.output)
        print(f"Wrote {result.stats.lines_out} lines to {args.output}", file=sys.stderr)
    else:
        print(result.output)

    if args.stats or True:  # Always show confidence in stderr
        print(f"\n--- Transpiler Report ---", file=sys.stderr)
        print(f"  Confidence: {confidence:.0%} ({confidence_label(confidence)})", file=sys.stderr)
        print(f"  Patterns:   {result.stats.patterns_matched} matched + {result.stats.gaps} gaps = {total_patterns} total", file=sys.stderr)
        if gap_samples:
            print(f"  Gap samples:", file=sys.stderr)
            for reason in gap_samples:
                count = result.stats.gap_reasons[reason]
                status = kb_summary.get(reason, "")
                print(f"    {count}x  {reason}  [{status}]", file=sys.stderr)


if __name__ == "__main__":
    main()
