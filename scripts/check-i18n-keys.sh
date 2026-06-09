#!/usr/bin/env bash
# check-i18n-keys.sh — Verify all non-en locale modules have every key from English canonical.
#
# Usage:
#   ./scripts/check-i18n-keys.sh          # report mode (warnings only)
#   ./scripts/check-i18n-keys.sh --strict # CI mode (exit 1 on any missing key)

set -euo pipefail
cd "$(dirname "$0")/.."
exec npx tsx scripts/check-i18n-keys.ts "$@"
