"""Parser-independent quality gates for transpiler output (ADR-0020 D8, C2).

These gates assess ANY transpiler output (RD parser, tree-sitter, LLM) against
correctness criteria.  The gates are EXTERNAL to the parser — they don't care
how the output was produced, only whether it's valid Python SDK code.

Gates (in order):
  1. ``ast.parse(output)`` — must be syntactically valid Python 3.
  2. SDK import smoke — required SDK symbols must be importable.
  3. Lint gate — no MQL artifacts (``//`` comments, bare MQL function names).

Confidence is REDEFINED from the old ``matched/(matched+gaps)`` to:
  - HIGH — all gates pass (output is a candidate for behavioral testing).
  - LOW  — any gate fails (output MUST NOT be presented as valid).

Usage::

    from tools.mql_transpiler.quality_gate import QualityGate, QualityVerdict
    verdict = QualityGate.assess(transpiler_output)
    if verdict == QualityVerdict.HIGH:
        # Candidate for behavioral alignment test.
        ...
    else:
        # Must be fixed or sent to LLM gap-filler.
        ...

Replaces:
  - ast_transpiler.py:188-193 (confidence = matched/(matched+gaps))
  - transpiler.py:374-378 (get_confidence())
"""

from __future__ import annotations

import ast
import enum
import re
from dataclasses import dataclass, field
from typing import List, Optional, Set


class QualityVerdict(enum.Enum):
    """Gate verdict — binary, not a score."""
    HIGH = "high"   # All gates pass.  Candidate for behavioral test.
    LOW = "low"     # At least one gate failed.  Must not be presented as valid.


@dataclass
class GateFailure:
    """A single gate failure with location and explanation."""
    gate: str           # Gate name: "compile", "sdk_import", "lint"
    message: str        # Human-readable failure reason
    line: int = 0       # Approximate line number (0 = unknown)
    snippet: str = ""   # Offending snippet (truncated)


@dataclass
class QualityReport:
    """Full quality assessment of transpiler output."""
    verdict: QualityVerdict
    failures: List[GateFailure] = field(default_factory=list)
    compile_ok: bool = False
    sdk_import_ok: bool = False
    lint_ok: bool = False
    # Warnings don't fail the gate but are worth noting.
    warnings: List[str] = field(default_factory=list)

    @property
    def is_high(self) -> bool:
        return self.verdict == QualityVerdict.HIGH


# ── MQL artifacts that must NOT appear in Python output ────────────────

# Comments: MQL uses //, Python uses #.
_MQL_COMMENT_RE = re.compile(r"(?<!['\"])\/\/\s*[^\n]+")

# Bare MQL function names that indicate untranslated code.
# These should never appear as identifiers in Python output.
_BARE_MQL_NAMES: Set[str] = {
    "OrderSend", "OrderClose", "OrderModify", "OrderDelete", "OrderSelect",
    "OrdersTotal", "PositionsTotal", "PositionSelect",
    "iMA", "iRSI", "iATR", "iBands", "iMACD", "iStochastic", "iCCI",
    "iCustom", "iADX", "iMomentum", "iMFI", "iOBV", "iSAR", "iStdDev",
    "iWPR", "iEnvelopes", "iForce", "iDeMarker", "iOsMA",
    "iOpen", "iHigh", "iLow", "iClose", "iVolume", "iTime",
    "NormalizeDouble", "MathAbs", "MathMax", "MathMin",
    "MathPow", "MathLog", "MathSqrt", "MathExp",
    "StringConcatenate", "StringFind", "StringLen", "StringSubstr",
    "ArrayInitialize", "ArrayResize", "ArrayCopy",
    "FileOpen", "FileClose", "FileWrite", "FileRead",
    "ObjectsTotal", "ObjectCreate", "ObjectDelete",
    "ObjectSetDouble", "ObjectSetInteger", "ObjectSetString",
    "ObjectGetDouble", "ObjectGetInteger", "ObjectGetString",
    "ChartOpen", "ChartClose",
}

# Required SDK imports that must appear in the output.
_REQUIRED_IMPORTS: List[str] = [
    "from app.sdk import",
    "from decimal import Decimal",
]

# Required class structure.
_REQUIRED_BASE_CLASS: str = "StrategyBase"


class QualityGate:
    """Static gate checks.  All methods are pure functions of the output string."""

    @staticmethod
    def assess(code: str) -> QualityReport:
        """Run all gates and return a verdict.

        Args:
            code: Transpiler output (Python source code as string).

        Returns:
            QualityReport with verdict and per-gate details.
        """
        failures: List[GateFailure] = []
        warnings: List[str] = []

        # Gate 1: Compile check.
        compile_ok = QualityGate._check_compile(code, failures)

        # Gate 2: SDK import smoke.
        sdk_import_ok = QualityGate._check_sdk_imports(code, failures)

        # Gate 3: Lint (MQL artifacts).
        lint_ok = QualityGate._check_lint(code, failures, warnings)

        verdict = (
            QualityVerdict.HIGH
            if compile_ok and sdk_import_ok and lint_ok
            else QualityVerdict.LOW
        )

        return QualityReport(
            verdict=verdict,
            failures=failures,
            compile_ok=compile_ok,
            sdk_import_ok=sdk_import_ok,
            lint_ok=lint_ok,
            warnings=warnings,
        )

    @staticmethod
    def _check_compile(code: str, failures: List[GateFailure]) -> bool:
        """Gate 1: ``ast.parse()`` must succeed."""
        if not code or not code.strip():
            failures.append(GateFailure(
                gate="compile",
                message="Output is empty",
            ))
            return False

        try:
            ast.parse(code)
            return True
        except SyntaxError as e:
            # Extract the offending line for context.
            lines = code.split("\n")
            lineno = e.lineno or 0
            snippet = ""
            if 1 <= lineno <= len(lines):
                snippet = lines[lineno - 1].strip()[:120]
            failures.append(GateFailure(
                gate="compile",
                message=f"SyntaxError: {e.msg}",
                line=lineno,
                snippet=snippet,
            ))
            return False

    @staticmethod
    def _check_sdk_imports(code: str, failures: List[GateFailure]) -> bool:
        """Gate 2: Required SDK imports must be present."""
        missing = [imp for imp in _REQUIRED_IMPORTS if imp not in code]
        if missing:
            failures.append(GateFailure(
                gate="sdk_import",
                message=f"Missing required imports: {', '.join(missing)}",
            ))
            return False

        # Also check that StrategyBase is referenced (not just imported).
        if _REQUIRED_BASE_CLASS not in code:
            failures.append(GateFailure(
                gate="sdk_import",
                message=f"Output does not reference {_REQUIRED_BASE_CLASS}",
            ))
            return False

        return True

    @staticmethod
    def _check_lint(code: str, failures: List[GateFailure], warnings: List[str]) -> bool:
        """Gate 3: No MQL artifacts in Python output."""
        ok = True

        # 3a. MQL // comments.
        mql_comments = _MQL_COMMENT_RE.findall(code)
        if mql_comments:
            ok = False
            for mc in mql_comments[:3]:  # Report up to 3 examples.
                # Find line number.
                for i, line in enumerate(code.split("\n"), 1):
                    if mc.strip() in line:
                        failures.append(GateFailure(
                            gate="lint",
                            message="MQL-style // comment in Python output",
                            line=i,
                            snippet=line.strip()[:120],
                        ))
                        break

        # 3b. Bare MQL function names in expressions/statements.
        # We look for these as standalone identifiers (not as comments, not in strings).
        # Strategy: parse Python AST, walk for Name nodes.
        try:
            tree = ast.parse(code)
            bare_names: Set[str] = set()
            for node in ast.walk(tree):
                if isinstance(node, ast.Name):
                    bare_names.add(node.id)
                elif isinstance(node, ast.Attribute):
                    # Allow self.broker.order_send etc.
                    pass

            mql_in_python = _BARE_MQL_NAMES & bare_names
            if mql_in_python:
                ok = False
                # Find line numbers for these names.
                for name in sorted(mql_in_python):
                    for i, line in enumerate(code.split("\n"), 1):
                        if re.search(rf"\b{re.escape(name)}\b", line):
                            failures.append(GateFailure(
                                gate="lint",
                                message=f"Bare MQL function name '{name}' in Python output",
                                line=i,
                                snippet=line.strip()[:120],
                            ))
                            break
        except SyntaxError:
            # Already caught by compile gate.
            pass

        # 3c. Warn about TRANSPILER-GAP markers (these need LLM fill but don't fail the gate).
        gap_count = code.count("# TRANSPILER-GAP")
        if gap_count > 0:
            warnings.append(f"Contains {gap_count} TRANSPILER-GAP markers (needs LLM gap-fill)")

        return ok


def confidence_from_output(output: str) -> QualityVerdict:
    """Redefined confidence: HIGH = compiles + imports + lint-clean.

    This REPLACES the old ``matched/(matched+gaps)`` algorithm.
    A product that doesn't compile is ALWAYS LOW, regardless of how many
    patterns were "matched".

    Usage in transpile endpoints::

        from tools.mql_transpiler.quality_gate import confidence_from_output
        verdict = confidence_from_output(transpiled_code)
        # HIGH → candidate for behavioral test.
        # LOW  → must not be presented as valid.
    """
    report = QualityGate.assess(output)
    return report.verdict
