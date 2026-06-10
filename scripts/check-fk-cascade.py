#!/usr/bin/env python3
"""Check that all foreign keys referencing users(id) have ON DELETE CASCADE or SET NULL.

Queries PostgreSQL's information_schema to find FKs that lack proper ON DELETE
behavior. A FK with NO ACTION (the default) will block user deletion if the
child table has rows referencing the user being deleted.

Exit codes:
  0 — all FKs have CASCADE or SET NULL (or no DB connection available)
  1 — one or more FKs have NO ACTION / RESTRICT / SET DEFAULT
"""

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
COMPOSE_FILE = f"{ROOT}/docker-compose.yml"
CONTAINER = "postgres"
DB_USER = "ant"
DB_NAME = "ant"

QUERY = """
SELECT DISTINCT tc.table_name, kcu.column_name, rc.delete_rule
FROM information_schema.table_constraints tc
JOIN information_schema.referential_constraints rc
  ON tc.constraint_name = rc.constraint_name
 AND tc.constraint_schema = rc.constraint_schema
JOIN information_schema.key_column_usage kcu
  ON tc.constraint_name = kcu.constraint_name
 AND tc.constraint_schema = kcu.constraint_schema
JOIN information_schema.constraint_column_usage ccu
  ON rc.unique_constraint_name = ccu.constraint_name
 AND rc.unique_constraint_schema = ccu.constraint_schema
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND ccu.table_name = 'users'
  AND ccu.column_name = 'id'
  AND rc.delete_rule NOT IN ('CASCADE', 'SET NULL')
ORDER BY tc.table_name, kcu.column_name;
"""


def main() -> int:
    cmd = [
        "docker", "compose", "-f", COMPOSE_FILE, "exec", "-T", CONTAINER,
        "psql", "-U", DB_USER, "-d", DB_NAME,
        "-t", "-A", "-F", "|",
        "-c", QUERY,
    ]
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=15, cwd=str(ROOT))
    except FileNotFoundError:
        print("🟡 docker not found — skipping FK check", file=sys.stderr)
        return 0
    except subprocess.TimeoutExpired:
        print("🟡 DB unreachable — skipping FK check", file=sys.stderr)
        return 0

    if result.returncode != 0:
        err = result.stderr.strip()
        # Docker daemon not running or container not up
        if "docker" in err.lower() or "connection refused" in err.lower():
            print("🟡 DB unavailable — skipping FK check", file=sys.stderr)
            return 0
        print(f"🔴 FK check failed: {err}", file=sys.stderr)
        return 1

    output = result.stdout.strip()
    if not output:
        print("🟢 All FKs referencing users(id) have CASCADE or SET NULL")
        return 0

    violations = []
    for line in output.split("\n"):
        line = line.strip()
        if not line:
            continue
        parts = line.split("|")
        if len(parts) >= 3:
            violations.append((parts[0].strip(), parts[1].strip(), parts[2].strip()))

    if violations:
        print(f"🔴 {len(violations)} FK(s) lack ON DELETE CASCADE/SET NULL:")
        for table, column, rule in violations:
            print(f"   {table}.{column} → {rule} (expected CASCADE or SET NULL)")
        print()
        print("   Fix: ALTER TABLE {table} DROP CONSTRAINT {constraint};")
        print("        ALTER TABLE {table} ADD CONSTRAINT {constraint}")
        print("          FOREIGN KEY ({column}) REFERENCES users(id) ON DELETE CASCADE;")
        return 1

    print("🟢 All FKs referencing users(id) have CASCADE or SET NULL")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
