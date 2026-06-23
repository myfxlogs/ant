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

    if args.output:
        with open(args.output, "w") as f:
            f.write(result.output)
        print(f"Wrote {result.stats.lines_out} lines to {args.output}", file=sys.stderr)
    else:
        print(result.output)

    if args.stats:
        print(f"\n--- Transpiler Stats ---", file=sys.stderr)
        print(f"  Lines in:  {result.stats.lines_in}", file=sys.stderr)
        print(f"  Lines out: {result.stats.lines_out}", file=sys.stderr)
        print(f"  Patterns:  {result.stats.patterns_matched}", file=sys.stderr)
        print(f"  Gaps:      {result.stats.gaps}", file=sys.stderr)
        if result.stats.gap_reasons:
            print(f"  Gap reasons:", file=sys.stderr)
            for reason, count in sorted(result.stats.gap_reasons.items(), key=lambda x: -x[1]):
                print(f"    {count:3d}  {reason}", file=sys.stderr)


if __name__ == "__main__":
    main()
