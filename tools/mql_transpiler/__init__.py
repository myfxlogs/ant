"""MQL4/5 → Python Strategy SDK transpiler (T2.1).

Deterministic, zero-hallucination, auditable.  Maps ~80% of MQL constructs
mechanically to the SDK; marks unmappable code with ``// TRANSPILER-GAP:``.

Language choice: Python (same ecosystem as SDK, no compilation step, easy to
embed in CI and to iterate on mapping rules).
"""
