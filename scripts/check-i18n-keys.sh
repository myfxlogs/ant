#!/usr/bin/env bash
# check-i18n-keys.sh — Verify all non-en locale modules have every key from English canonical.
#
# Usage:
#   ./scripts/check-i18n-keys.sh          # report mode (warnings only)
#   ./scripts/check-i18n-keys.sh --strict # CI mode (exit 1 on any missing key)

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR/.."
exec npx tsx "$SCRIPT_DIR/check-i18n-keys.ts" "$@"
