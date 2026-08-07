#!/bin/bash
# Generate .down.sql for migrations missing one.
# Patterns: CREATE TABLE → DROP TABLE, CREATE INDEX → DROP INDEX,
#           ALTER TABLE ADD COLUMN → DROP COLUMN, CREATE TRIGGER → DROP TRIGGER
set -uo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
generated=0

for f in "$DIR"/*.up.sql; do
  base=$(basename "$f" .up.sql)
  down="$DIR/${base}.down.sql"
  if [ -f "$down" ]; then
    continue
  fi

  # Collect lines, strip comments/blank for analysis
  content=$(cat "$f")

  # Extract CREATE TABLE names
  tables=$(echo "$content" | grep -oiE 'CREATE TABLE\s+(IF NOT EXISTS\s+)?[a-z_]+' | sed -E 's/CREATE TABLE\s+(IF NOT EXISTS\s+)?//' | sort -u || true)

  # Extract CREATE INDEX names
  indexes=$(echo "$content" | grep -oiE 'CREATE\s+(UNIQUE\s+)?INDEX\s+(IF NOT EXISTS\s+)?[a-z_]+' | sed -E 's/CREATE\s+(UNIQUE\s+)?INDEX\s+(IF NOT EXISTS\s+)?//' | sort -u || true)

  # Extract ALTER TABLE ... ADD COLUMN
  columns=$(echo "$content" | grep -oiE 'ALTER TABLE\s+[a-z_]+\s+ADD COLUMN\s+(IF NOT EXISTS\s+)?[a-z_]+' | sed -E 's/ALTER TABLE\s+//; s/\s+ADD COLUMN\s+(IF NOT EXISTS\s+)?/ /' | sort -u || true)

  # Extract CREATE TRIGGER names
  triggers=$(echo "$content" | grep -oiE 'CREATE\s+(OR REPLACE\s+)?TRIGGER\s+[a-z_]+' | sed -E 's/CREATE\s+(OR REPLACE\s+)?TRIGGER\s+//' | sort -u || true)

  # Extract ALTER TABLE ... ALTER COLUMN ... TYPE (can't reverse, note only)
  type_changes=$(echo "$content" | grep -oiE 'ALTER TABLE\s+[a-z_]+\s+ALTER COLUMN\s+[a-z_]+\s+TYPE' | sort -u || true)

  # Generate down script
  {
    echo "-- ${base}.down.sql"
    echo "-- Auto-generated rollback for ${base}"
    echo ""

    # Drop triggers first (depend on tables)
    if [ -n "$triggers" ]; then
      echo "-- Drop triggers"
      for t in $triggers; do
        echo "DROP TRIGGER IF EXISTS ${t} ON public.${t%%_*};" 2>/dev/null || true
      done
      # Simpler: just DROP TRIGGER IF EXISTS name (PG needs table, but we try generic)
      echo ""
    fi

    # Drop indexes
    if [ -n "$indexes" ]; then
      echo "-- Drop indexes"
      for idx in $indexes; do
        echo "DROP INDEX IF EXISTS ${idx};"
      done
      echo ""
    fi

    # Drop columns
    if [ -n "$columns" ]; then
      echo "-- Drop added columns"
      echo "$columns" | while read -r tbl col; do
        [ -n "$col" ] && echo "ALTER TABLE ${tbl} DROP COLUMN IF EXISTS ${col};"
      done
      echo ""
    fi

    # Drop tables (reverse order — later tables may depend on earlier)
    if [ -n "$tables" ]; then
      echo "-- Drop tables"
      echo "$tables" | tac | while read -r tbl; do
        echo "DROP TABLE IF EXISTS ${tbl} CASCADE;"
      done
      echo ""
    fi

    # Note type changes (not reversible without knowing original type)
    if [ -n "$type_changes" ]; then
      echo "-- NOTE: Type changes cannot be auto-reversed:"
      echo "$type_changes" | while read -r line; do
        echo "-- ${line}"
      done
    fi

  } > "$down"

  generated=$((generated + 1))
  echo "Generated: ${base}.down.sql"
done

echo ""
echo "Total generated: ${generated}"
