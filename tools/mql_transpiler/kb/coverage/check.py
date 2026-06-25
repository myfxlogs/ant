#!/usr/bin/env python3
"""MQL feature coverage checker.

Reads kb/README.md, parses the feature coverage matrix, and generates
a summary report. Run after updating the catalog.
"""

from __future__ import annotations

import re
import sys
from collections import Counter
from pathlib import Path

KB_DIR = Path(__file__).resolve().parent.parent
CATALOG = KB_DIR / "README.md"

STATUS_RE = re.compile(r"\| .+ \| .+ \| .+ \| (🟢|🟡|🔴|⚫) (\w+) \|")


def parse_catalog(path: Path) -> Counter:
    """Parse the coverage catalog and return status counts."""
    counts = Counter()
    with open(path) as f:
        for line in f:
            m = STATUS_RE.search(line)
            if m:
                status = m.group(2)
                counts[status] += 1
    return counts


def main() -> None:
    if not CATALOG.exists():
        print(f"ERROR: catalog not found at {CATALOG}", file=sys.stderr)
        sys.exit(1)

    counts = parse_catalog(CATALOG)
    total = sum(counts.values())
    covered = counts.get("FULL", 0) + counts.get("PARTIAL", 0) * 0.5
    pct = (covered / total * 100) if total else 0

    print(f"MQL Feature Coverage: {pct:.0f}%")
    print(f"  🟢 FULL:      {counts.get('FULL', 0)}")
    print(f"  🟡 PARTIAL:   {counts.get('PARTIAL', 0)}")
    print(f"  🔴 GAP:       {counts.get('GAP', 0)}")
    print(f"  ⚫ UNSUPPORTED: {counts.get('UNSUPPORTED', 0)}")
    print(f"  Total:        {total}")


if __name__ == "__main__":
    main()
