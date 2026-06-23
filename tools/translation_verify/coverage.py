"""MQL feature coverage scanner (T2.3).

Scans MQL source and SDK output to determine which MQL features are
supported, partially supported, or unsupported.  Produces a coverage
report for translation quality assessment.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from typing import Dict, List, Set


# ── MQL feature categories ─────────────────────────────────────────────


@dataclass
class FeatureCategory:
    name: str
    description: str
    patterns: List[str]  # regex patterns to match in MQL source


FEATURE_CATEGORIES: List[FeatureCategory] = [
    FeatureCategory("Lifecycle", "Event entry points",
                    [r"\bOnInit\b", r"\bOnTick\b", r"\bOnTimer\b", r"\bOnTrade\b",
                     r"\bOnDeinit\b", r"\bOnCalculate\b", r"\bstart\b"]),
    FeatureCategory("Market Orders", "Immediate execution orders",
                    [r"\bOP_BUY\b(?!LIMIT)(?!STOP)", r"\bOP_SELL\b(?!LIMIT)(?!STOP)"]),
    FeatureCategory("Pending Orders", "Limit/stop/stop-limit orders",
                    [r"\bOP_BUYLIMIT\b", r"\bOP_SELLLIMIT\b", r"\bOP_BUYSTOP\b",
                     r"\bOP_SELLSTOP\b", r"\bOP_BUYSTOPLIMIT\b", r"\bOP_SELLSTOPLIMIT\b"]),
    FeatureCategory("Position Management", "Close, modify, select",
                    [r"\bOrderClose\b", r"\bOrderModify\b", r"\bOrderDelete\b",
                     r"\bPositionClose\b", r"\bPositionModify\b"]),
    FeatureCategory("Order/Position Iteration", "OrderSelect/PositionsTotal loops",
                    [r"\bOrderSelect\b", r"\bOrdersTotal\b", r"\bPositionsTotal\b"]),
    FeatureCategory("Technical Indicators", "Built-in i* functions",
                    [r"\biMA\b", r"\biRSI\b", r"\biBands\b", r"\biMACD\b",
                     r"\biATR\b", r"\biStochastic\b", r"\biCCI\b", r"\biMomentum\b",
                     r"\biWPR\b", r"\biADX\b", r"\biMFI\b", r"\biOBV\b"]),
    FeatureCategory("Custom Indicators", "iCustom function",
                    [r"\biCustom\b"]),
    FeatureCategory("Price Data Access", "OHLCV series access",
                    [r"\b(?:Close|Open|High|Low|Volume|Time)\s*\[", r"\biClose\b",
                     r"\biOpen\b", r"\biHigh\b", r"\biLow\b", r"\biVolume\b", r"\biTime\b"]),
    FeatureCategory("Account Functions", "Balance, equity, margin queries",
                    [r"\bAccountBalance\b", r"\bAccountEquity\b", r"\bAccountFreeMargin\b",
                     r"\bAccountMargin\b", r"\bAccountLeverage\b", r"\bAccountCurrency\b"]),
    FeatureCategory("Symbol Metadata", "Symbol info, MarketInfo",
                    [r"\bSymbol\b", r"\bDigits\b", r"\bPoint\b", r"\bMarketInfo\b",
                     r"\bSymbolInfoDouble\b", r"\bSymbolInfoInteger\b"]),
    FeatureCategory("Parameters", "extern/input declarations",
                    [r"\bextern\b", r"\binput\b"]),
    FeatureCategory("Timer", "EventSetTimer/EventKillTimer",
                    [r"\bEventSetTimer\b", r"\bEventKillTimer\b"]),
    FeatureCategory("Math Functions", "MathAbs, MathMax, etc.",
                    [r"\bMathAbs\b", r"\bMathMax\b", r"\bMathMin\b", r"\bMathRound\b",
                     r"\bMathSqrt\b", r"\bMathPow\b"]),
    FeatureCategory("String Functions", "String manipulation",
                    [r"\bStringConcatenate\b", r"\bStringFind\b", r"\bStringLen\b",
                     r"\bStringSubstr\b", r"\bStringReplace\b"]),
    FeatureCategory("File I/O", "File operations — UNSUPPORTED",
                    [r"\bFileOpen\b", r"\bFileClose\b", r"\bFileWrite\b", r"\bFileRead\b",
                     r"\bFileFlush\b", r"\bFileDelete\b"]),
    FeatureCategory("GUI Objects", "Chart objects — UNSUPPORTED",
                    [r"\bObjectCreate\b", r"\bObjectDelete\b", r"\bObjectSet\w+\b",
                     r"\bObjectGet\w+\b", r"\bChart\w+\b", r"\bObjectsTotal\b"]),
    FeatureCategory("Network", "WebRequest, FTP, email — UNSUPPORTED",
                    [r"\bWebRequest\b", r"\bSendFTP\b", r"\bSendMail\b",
                     r"\bSendNotification\b"]),
    FeatureCategory("DLL / External", "DLL calls — UNSUPPORTED",
                    [r"\b#import\b", r"\b#resource\b", r"\bDLL", r"\bIndicatorCreate\b",
                     r"\bIndicatorRelease\b"]),
    FeatureCategory("Arrays", "Array functions — partial support",
                    [r"\bArrayInitialize\b", r"\bArrayResize\b", r"\bArrayCopy\b",
                     r"\bArraySize\b", r"\bArrayGetAsSeries\b", r"\bArraySetAsSeries\b"]),
    FeatureCategory("Trading Context", "Execution mode, margin mode",
                    [r"\bMODE_MARGIN", r"\bSYMBOL_TRADE_MODE", r"\bOrderCalcMargin\b"]),
]


@dataclass
class CategoryCoverage:
    name: str
    status: str  # "FULL", "PARTIAL", "NONE"
    found_in_mql: int = 0
    gaps_in_output: int = 0
    notes: str = ""


@dataclass
class CoverageReport:
    mql_file: str
    total_categories: int
    supported: int
    partial: int
    unsupported: int
    categories: List[CategoryCoverage] = field(default_factory=list)
    overall_pct: float = 0.0

    def to_dict(self) -> dict:
        return {
            "mql_file": self.mql_file,
            "total_categories": self.total_categories,
            "supported": self.supported,
            "partial": self.partial,
            "unsupported": self.unsupported,
            "overall_pct": round(self.overall_pct, 1),
            "categories": [
                {"name": c.name, "status": c.status, "found": c.found_in_mql, "gaps": c.gaps_in_output}
                for c in self.categories
            ],
        }


def scan_mql_features(mql_source: str) -> Dict[str, int]:
    """Scan MQL source for feature occurrences."""
    found: Dict[str, int] = {}
    for cat in FEATURE_CATEGORIES:
        count = 0
        for pattern in cat.patterns:
            count += len(re.findall(pattern, mql_source, re.IGNORECASE))
        if count > 0:
            found[cat.name] = count
    return found


def scan_transpiler_gaps(sdk_output: str) -> Set[str]:
    """Extract TRANSPILER-GAP reasons from SDK output."""
    gaps = set()
    for match in re.finditer(r"TRANSPILER-GAP:\s*(.+?)(?:\n|$)", sdk_output):
        reason = match.group(1).strip().split(",")[0].strip()
        gaps.add(reason)
    return gaps


def generate_coverage_report(mql_source: str, sdk_output: str, mql_file: str = "unknown.mq4") -> CoverageReport:
    """Generate a coverage report for MQL→SDK translation."""
    mql_features = scan_mql_features(mql_source)
    sdk_gaps = scan_transpiler_gaps(sdk_output)

    categories: List[CategoryCoverage] = []
    supported = 0
    partial = 0
    unsupported = 0

    for cat in FEATURE_CATEGORIES:
        found = mql_features.get(cat.name, 0)
        if found == 0:
            continue  # not used in this MQL source

        # Determine status.
        is_unsupported = any(
            kw in cat.name.lower() or kw in cat.description.lower()
            for kw in ["unsupported", "dll", "gui", "network", "file"]
        )
        # Check if TRANSPILER-GAPs match this category.
        cat_gaps = len([g for g in sdk_gaps if cat.name.lower() in g.lower()
                        or any(p.lower() in g.lower() for p in cat.patterns)])

        if is_unsupported or "TRANSPILER-GAP: GUI" in sdk_output and cat.name == "GUI Objects":
            status = "NONE"
            unsupported += 1
        elif cat_gaps > 0:
            status = "PARTIAL"
            partial += 1
        else:
            status = "FULL"
            supported += 1

        categories.append(CategoryCoverage(
            name=cat.name,
            status=status,
            found_in_mql=found,
            gaps_in_output=cat_gaps,
            notes="",
        ))

    total = supported + partial + unsupported
    overall_pct = (supported + partial * 0.5) / max(total, 1) * 100.0

    return CoverageReport(
        mql_file=mql_file,
        total_categories=total,
        supported=supported,
        partial=partial,
        unsupported=unsupported,
        categories=categories,
        overall_pct=overall_pct,
    )


def format_coverage_report(report: CoverageReport) -> str:
    """Format a coverage report as human-readable text."""
    lines = [
        f"Coverage Report: {report.mql_file}",
        f"{'='*60}",
        f"Overall: {report.overall_pct:.0f}% ({report.supported} full, {report.partial} partial, {report.unsupported} unsupported)",
        f"",
        f"{'Category':<30} {'Status':<10} {'Found':>6} {'Gaps':>6}",
        f"{'-'*60}",
    ]
    for c in sorted(report.categories, key=lambda x: (x.status, x.name)):
        lines.append(f"{c.name:<30} {c.status:<10} {c.found_in_mql:>6} {c.gaps_in_output:>6}")

    lines.append(f"{'-'*60}")
    return "\n".join(lines)
