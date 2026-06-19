#!/usr/bin/env python3
"""File line count checker with per-language limits from CLAUDE.md.

Limits:
  Go:         300 lines (gen/ & test: 450 / 50% overage)
  TypeScript: 250 lines (gen/ & test: 375 / 50% overage)
  i18n:       500 lines (translation resource files — data, not code)
  Other:      800 lines (Python, proto, shell scripts)

Severity (non-exempt files only):
  🔴 ERROR   >50% over limit   — must split, CI fails
  🟡 WARNING 20-50% over limit — evaluate cohesion before splitting
  🟢 INFO    <20% over limit   — slight, splitting may harm readability
"""

from pathlib import Path
import argparse

ROOT = Path(__file__).resolve().parents[1]
SCOPES = [
    "backend/internal",
    "backend/cmd",
    "frontend/src",
    "proto",
    "strategy-service/app",
    "scripts",
    ".windsurf",
]
SKIP_DIRS = {".git", "node_modules", "dist", "build", ".cache", "__pycache__"}

# Base limits from CLAUDE.md
GO_BASE = 300
TS_BASE = 250
I18N_LIMIT = 500  # translation resource files are data, not code
OTHER_LIMIT = 800

# Overage multipliers
EXEMPT_MUL = 1.5   # gen/ and test files: 50% overage
ERROR_MUL = 1.5    # >50% over base = error
WARN_MUL = 1.2     # 20-50% over base = warning


def classify_file(rel: str) -> tuple[str, int]:
    """Return (category, effective_limit) for a file path."""
    r = rel.replace("\\", "/")
    ext = Path(rel).suffix.lower()

    # Scripts — linear procedural code, not subject to AI cognitive limits
    if r.startswith("scripts/"):
        return "scripts", OTHER_LIMIT

    # Generated protobuf code
    if "/gen/" in r:
        if ext == ".go":
            return "gen", int(GO_BASE * EXEMPT_MUL)
        if ext in (".ts", ".tsx"):
            return "gen", int(TS_BASE * EXEMPT_MUL)
        return "gen", OTHER_LIMIT

    # Test files
    fname = Path(rel).name
    if fname.endswith("_test.go") or ".test.ts" in fname or ".test.tsx" in fname:
        if ext == ".go":
            return "test", int(GO_BASE * EXEMPT_MUL)
        if ext in (".ts", ".tsx"):
            return "test", int(TS_BASE * EXEMPT_MUL)
        return "test", OTHER_LIMIT

    # textproto / proto files — pure data, exempt from line limits
    if ext in (".textproto", ".proto"):
        return "other", 9999

    # i18n JSON maps — generated field mappings, not hand-maintained
    if "/i18n/" in r and ext == ".json":
        return "other", 9999

    # i18n translation resource files — data, not logic
    if "/i18n/" in r:
        return "i18n", I18N_LIMIT

    # Normal source code
    if ext == ".go":
        return "go", GO_BASE
    if ext in (".ts", ".tsx"):
        return "ts", TS_BASE

    return "other", OTHER_LIMIT


def severity(lines: int, limit: int, category: str) -> str:
    """Return severity level for a file given its line count and category."""
    if category in ("gen", "i18n", "other", "scripts"):
        return "info"  # non-code files: informational only
    if category == "test":
        # Test files: warn if >50% over exempt limit, info otherwise
        return "warn" if lines > int(limit * 1.33) else "info"
    # Normal source: graduated
    if lines > int(limit * ERROR_MUL):
        return "error"
    if lines > int(limit * WARN_MUL):
        return "warn"
    return "info"


def load_baseline(path: Path) -> set[str]:
    if not path.exists():
        return set()
    return {
        line.strip()
        for line in path.read_text(encoding="utf-8").splitlines()
        if line.strip() and not line.lstrip().startswith("#")
    }


def is_text(path: Path) -> bool:
    try:
        return b"\0" not in path.read_bytes()[:4096]
    except OSError:
        return False


def line_count(path: Path) -> int:
    text = path.read_text(encoding="utf-8", errors="ignore")
    return text.count("\n") + int(bool(text) and not text.endswith("\n"))


def iter_files() -> list[Path]:
    out = []
    for scope in SCOPES:
        base = ROOT / scope
        if not base.exists():
            continue
        for path in base.rglob("*"):
            if not path.is_file() or path.suffix.lower() == ".md":
                continue
            if any(part in SKIP_DIRS for part in path.relative_to(ROOT).parts):
                continue
            if is_text(path):
                out.append(path)
    return out


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--baseline", default="scripts/file-line-baseline.txt")
    parser.add_argument(
        "--strict",
        action="store_true",
        help="Apply per-language limits with graduated severity (Go 300, TS 250)",
    )
    args = parser.parse_args()

    baseline = load_baseline(ROOT / args.baseline)
    flat_limit = 800

    oversized = {}  # rel -> lines (all files over their applicable limit)
    for path in iter_files():
        rel = path.relative_to(ROOT).as_posix()
        lines = line_count(path)
        cat, eff_limit = classify_file(rel)
        # Flat mode: only check business code (go/ts categories), skip exempt categories
        if not args.strict and cat in ("gen", "i18n", "scripts", "other"):
            continue
        threshold = flat_limit if not args.strict else eff_limit
        if lines > threshold:
            oversized[rel] = (lines, cat, eff_limit)

    new_items = sorted(set(oversized) - baseline)
    fixed_items = sorted(baseline - set(oversized))

    if not args.strict:
        if new_items:
            print(f"error: {len(new_items)} file(s) exceed {flat_limit} lines and are not in baseline")
            for rel in new_items:
                print(f"  {oversized[rel][0]:5d}  {rel}")
        if fixed_items:
            print(f"note: {len(fixed_items)} baseline file(s) are now <= {flat_limit} lines; remove them from baseline")
            for rel in fixed_items:
                print(f"  {rel}")
        if new_items:
            return 1
        print(f"line check ok: {len(oversized)} oversized file(s) allowed by baseline")
        return 0

    # Strict mode — graduated severity
    errors = []
    warnings = []
    infos = []

    for rel in new_items:
        lines, cat, eff_limit = oversized[rel]
        sev = severity(lines, eff_limit, cat)
        item = (lines, eff_limit, cat, rel)
        if sev == "error":
            errors.append(item)
        elif sev == "warn":
            warnings.append(item)
        else:
            infos.append(item)

    # Also check baseline items for severity — they may be allowed but worth noting
    base_errors = []
    base_warnings = []
    for rel in sorted(baseline & set(oversized)):
        lines, cat, eff_limit = oversized[rel]
        sev = severity(lines, eff_limit, cat)
        if sev == "error":
            base_errors.append((lines, eff_limit, cat, rel))
        elif sev == "warn":
            base_warnings.append((lines, eff_limit, cat, rel))

    # Print report
    icon = lambda s: {"error": "🔴", "warn": "🟡", "info": "🟢"}.get(s, "⚪")

    if errors:
        print(f"🔴 ERROR ({len(errors)} files >50% over limit — must split):")
        for lines, lim, cat, rel in errors:
            pct = int((lines - lim) / lim * 100)
            print(f"  {lines:5d}/{lim:<4d} (+{pct}%)  {rel}")
        if base_errors:
            print(f"  (+{len(base_errors)} in baseline — also need splitting)")
        print()

    if warnings:
        print(f"🟡 WARNING ({len(warnings)} files 20-50% over limit — evaluate cohesion):")
        for lines, lim, cat, rel in warnings:
            pct = int((lines - lim) / lim * 100)
            print(f"  {lines:5d}/{lim:<4d} (+{pct}%)  {rel}")
        if base_warnings:
            print(f"  (+{len(base_warnings)} in baseline)")
        print()

    if infos:
        print(f"🟢 INFO ({len(infos)} files <20% over — splitting not recommended):")
        for lines, lim, cat, rel in infos:
            print(f"  {lines:5d}/{lim:<4d}  {rel}")
        print()

    if fixed_items:
        print(f"💚 {len(fixed_items)} baseline file(s) now within limits — remove from baseline")
        for rel in fixed_items:
            print(f"  {rel}")
        print()

    # Exit code: fail on errors only
    if errors:
        print(f"Result: {len(errors)} ERROR(s) — must fix before commit")
        return 1
    if warnings:
        print(f"Result: 0 errors, {len(warnings)} warning(s) — review but not blocking")
    else:
        print("Result: all files within acceptable limits ✅")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
