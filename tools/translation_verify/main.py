#!/usr/bin/env python3
"""Translation verification CLI (T2.3).

Usage:
    python3 -m tools.translation_verify.main input.mq4
    python3 -m tools.translation_verify.main input.mq4 --golden golden.json
    python3 -m tools.translation_verify.main input.mq4 --output report.json
"""

from __future__ import annotations

import argparse
import sys

from tools.translation_verify.verifier import TranslationVerifier


def main() -> None:
    parser = argparse.ArgumentParser(
        description="MQL→SDK translation verification (T2.3)",
    )
    parser.add_argument("input", help="MQL source file (.mq4/.mq5)")
    parser.add_argument("--golden", help="Golden reference JSON (optional)")
    parser.add_argument("-o", "--output", help="Output report JSON (default: stdout)")
    parser.add_argument("--format", choices=["json", "text"], default="json",
                        help="Output format (default: json)")
    args = parser.parse_args()

    verifier = TranslationVerifier()
    report = verifier.verify_file(args.input, args.golden)

    if args.format == "text":
        output = _format_text(report)
    else:
        output = report.to_json()

    if args.output:
        with open(args.output, "w") as f:
            f.write(output)
        print(f"Report written to {args.output}", file=sys.stderr)
    else:
        print(output)


def _format_text(report) -> str:
    lines = [
        f"Verification Report: {report.mql_file}",
        f"{'='*60}",
        f"",
        f"Transpile: {'PASS' if report.transpile_ok else 'FAIL'}",
        f"  Lines in:  {report.transpile_stats.get('lines_in', 0)}",
        f"  Lines out: {report.transpile_stats.get('lines_out', 0)}",
        f"  Patterns:  {report.transpile_stats.get('patterns_matched', 0)}",
        f"  Gaps:      {report.transpile_stats.get('gaps', 0)}",
        f"",
    ]
    if report.coverage_report:
        cov = report.coverage_report
        lines.append(f"Coverage: {cov.get('overall_pct', 0):.0f}%")
        lines.append(f"  Full:      {cov.get('supported', 0)}")
        lines.append(f"  Partial:   {cov.get('partial', 0)}")
        lines.append(f"  Unsupported: {cov.get('unsupported', 0)}")
        lines.append(f"")
        for cat in cov.get("categories", []):
            lines.append(f"  {cat['name']:<30} {cat['status']:<10} found={cat['found']} gaps={cat['gaps']}")

    if report.diff_deviations > 0:
        lines.append(f"")
        lines.append(f"Deviations: {report.diff_deviations}")
        for d in report.diff_details:
            lines.append(f"  seq={d['seq']} {d['field']}: {d['classification']} — {d['explanation']}")

    return "\n".join(lines)


if __name__ == "__main__":
    main()
