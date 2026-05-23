#!/usr/bin/env python3
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
    parser.add_argument("--max-lines", type=int, default=800)
    parser.add_argument("--baseline", default="scripts/file-line-baseline.txt")
    args = parser.parse_args()

    baseline = load_baseline(ROOT / args.baseline)
    oversized = {}
    for path in iter_files():
        rel = path.relative_to(ROOT).as_posix()
        lines = line_count(path)
        if lines > args.max_lines:
            oversized[rel] = lines

    new_items = sorted(set(oversized) - baseline)
    fixed_items = sorted(baseline - set(oversized))

    if new_items:
        print(f"error: {len(new_items)} file(s) exceed {args.max_lines} lines and are not in baseline")
        for rel in new_items:
            print(f"{oversized[rel]:5d} {rel}")
    if fixed_items:
        print(f"note: {len(fixed_items)} baseline file(s) are now <= {args.max_lines} lines; remove them from baseline")
        for rel in fixed_items:
            print(rel)

    if new_items:
        return 1
    print(f"line check ok: {len(oversized)} oversized file(s) allowed by baseline")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
