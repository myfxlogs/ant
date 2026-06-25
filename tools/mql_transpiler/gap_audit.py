"""LLM gap-fill audit infrastructure (T4).

When the deterministic transpiler cannot translate a construct, it marks it as
a hard GAP.  An LLM fills that GAP span, and the result is re-verified through
the quality gates (C2).  Every LLM-filled span is annotated with an audit marker
so the provenance of each line is traceable.

Audit marker format::

    # @audit:LLM span=<id> original="<MQL snippet>"

This makes it possible to:
  - Identify which lines were AI-generated (vs deterministic).
  - Re-verify that LLM output still passes gates.
  - Trace back to the original MQL for manual review.

Usage::

    from tools.mql_transpiler.gap_audit import GapSpan, GapAuditTrail
    trail = GapAuditTrail()
    trail.record_gap(GapSpan(start_line=44, end_line=46, reason="for loop", mql="for(...)"))
    # ... LLM fills ...
    filled = trail.apply_fill(gap_id, llm_output, source_lines)
    trail.verify_gates(filled)  # Must pass ast.parse + SDK import + lint
"""

from __future__ import annotations

import hashlib
import re
from dataclasses import dataclass, field
from typing import Dict, List, Optional

from tools.mql_transpiler.quality_gate import QualityGate, QualityVerdict

_AUDIT_RE = re.compile(r"# @audit:LLM\s+span=(\S+)\s+original=\"(.+?)\"")


@dataclass
class GapSpan:
    """A single untranslatable span in the transpiler output."""
    id: str = ""               # Unique span ID (hash)
    start_line: int = 0        # 1-based line number in transpiled output
    end_line: int = 0
    reason: str = ""           # Why it couldn't be translated
    mql_original: str = ""     # Original MQL source text
    filled: bool = False       # Has LLM filled this gap?
    filled_by: str = ""        # "llm" or "manual"


@dataclass
class GapAuditTrail:
    """Tracks all GAP spans and LLM fills in a translation session."""
    spans: List[GapSpan] = field(default_factory=list)
    _counter: int = 0

    def record_gap(self, span: GapSpan) -> str:
        """Record a gap and return its ID."""
        if not span.id:
            span.id = self._make_id(span)
        self.spans.append(span)
        return span.id

    def mark_llm_filled(self, gap_id: str) -> None:
        """Mark a gap as filled by LLM."""
        for s in self.spans:
            if s.id == gap_id:
                s.filled = True
                s.filled_by = "llm"
                return

    def get_unfilled(self) -> List[GapSpan]:
        """Return gaps that haven't been filled yet."""
        return [s for s in self.spans if not s.filled]

    @property
    def llm_filled_count(self) -> int:
        return sum(1 for s in self.spans if s.filled_by == "llm")

    @property
    def total_gaps(self) -> int:
        return len(self.spans)

    @staticmethod
    def _make_id(span: GapSpan) -> str:
        """Generate a stable ID from span content."""
        raw = f"{span.start_line}:{span.reason}:{span.mql_original[:40]}"
        return hashlib.sha256(raw.encode()).hexdigest()[:12]

    @staticmethod
    def annotate_fill(output_lines: List[str], gap_id: str,
                       original_mql: str, start_line: int) -> List[str]:
        """Insert audit markers around LLM-filled spans.

        Args:
            output_lines: Current transpiled output lines.
            gap_id: The gap span ID.
            original_mql: Original MQL text that was filled.
            start_line: 1-based line where the fill starts.

        Returns:
            Modified output lines with audit markers.
        """
        # Escape quotes in original MQL
        escaped = original_mql.replace('"', '\\"')[:120]
        marker = f'# @audit:LLM span={gap_id} original="{escaped}"'

        idx = start_line - 1
        if 0 <= idx < len(output_lines):
            # Insert marker before the filled line.
            output_lines.insert(idx, marker)
        return output_lines

    @staticmethod
    def extract_audit_spans(output: str) -> List[Dict[str, str]]:
        """Extract LLM audit markers from output for verification."""
        spans = []
        for line in output.split("\n"):
            m = _AUDIT_RE.search(line)
            if m:
                spans.append({
                    "span_id": m.group(1),
                    "original": m.group(2),
                })
        return spans

    @staticmethod
    def verify_gates(output: str) -> bool:
        """Re-verify that LLM-filled output still passes quality gates."""
        report = QualityGate.assess(output)
        return report.verdict == QualityVerdict.HIGH
