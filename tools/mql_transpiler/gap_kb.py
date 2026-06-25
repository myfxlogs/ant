"""MQL→Python gap knowledge base (T2.4).

Each time the transpiler encounters an unrecognized MQL pattern,
the original source line and gap reason are appended to a JSON file.
This becomes the learning corpus for improving the transpiler.

File: tools/mql_transpiler/gap_patterns.json
"""

import json
import os.path
from dataclasses import dataclass, asdict
from datetime import datetime, timezone
from typing import Dict, List, Optional

KB_PATH = os.path.join(os.path.dirname(__file__), "kb", "gap_patterns.json")


@dataclass
class GapEntry:
    reason: str
    original: str
    count: int
    first_seen: str
    last_seen: str


def load_kb() -> Dict[str, GapEntry]:
    """Load the gap knowledge base from disk."""
    if not os.path.exists(KB_PATH):
        return {}
    try:
        with open(KB_PATH) as f:
            data = json.load(f)
        return {k: GapEntry(**v) for k, v in data.items()}
    except Exception:
        return {}


def save_kb(kb: Dict[str, GapEntry]) -> None:
    """Persist the gap knowledge base."""
    with open(KB_PATH, "w") as f:
        json.dump({k: asdict(v) for k, v in kb.items()}, f, indent=2, ensure_ascii=False)


def record_gaps(gap_reasons: Dict[str, int], originals: List[str] = None) -> Dict[str, str]:
    """Record gap patterns from a translation run.

    Returns a summary dict of {reason: status} for display.
    """
    if not gap_reasons:
        return {}

    kb = load_kb()
    now = datetime.now(timezone.utc).isoformat()
    summary = {}

    for i, (reason, count) in enumerate(gap_reasons.items()):
        key = reason[:80]  # truncate for key
        original = (originals or [])[i] if originals and i < len(originals) else ""

        if key in kb:
            entry = kb[key]
            entry.count += count
            entry.last_seen = now
            if original and original not in entry.original:
                entry.original += f" | {original}"
            summary[reason] = f"updated (total={entry.count})"
        else:
            kb[key] = GapEntry(
                reason=reason,
                original=original,
                count=count,
                first_seen=now,
                last_seen=now,
            )
            summary[reason] = "new"

    save_kb(kb)
    return summary
