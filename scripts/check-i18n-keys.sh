#!/usr/bin/env bash
# check-i18n-keys.sh — Verify all non-en locale modules have every key from English canonical.
#
# English (en/) is the canonical source of truth. Every key present in English
# MUST also exist in every other locale. Missing keys mean the locale will
# silently fall back to English at runtime — which works but indicates the
# locale resource is stale.
#
# Usage:
#   ./scripts/check-i18n-keys.sh          # report mode (warnings only)
#   ./scripts/check-i18n-keys.sh --strict # CI mode (exit 1 on any missing key)
#
# Excludes: admin.* keys (managed separately)

set -euo pipefail

STRICT=false
if [[ "${1:-}" == "--strict" ]]; then
  STRICT=true
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
I18N_DIR="$PROJECT_ROOT/frontend/src/i18n/resources"

NODE_SCRIPT=$(cat << 'ENDSCRIPT'
const fs = require('fs');
const path = require('path');

const BASE = process.env.I18N_DIR;
const strict = process.env.STRICT === 'true';

const locales = fs.readdirSync(BASE).filter(d => {
  const stat = fs.statSync(path.join(BASE, d));
  return stat.isDirectory() && d !== 'en';
});

function getAllLeafPaths(obj, prefix = '') {
  const paths = new Map();
  for (const [k, v] of Object.entries(obj)) {
    const fullKey = prefix ? `${prefix}.${k}` : k;
    if (v !== null && typeof v === 'object' && !Array.isArray(v)) {
      for (const [p, val] of getAllLeafPaths(v, fullKey)) {
        paths.set(p, val);
      }
    } else {
      paths.set(fullKey, v);
    }
  }
  return paths;
}

function parseTS(content) {
  const match = content.match(/const\s+\w+\s*=\s*({[\s\S]*})\s*as\s+const/);
  if (!match) return null;
  try {
    return eval('(' + match[1] + ')');
  } catch (e) {
    console.error(`  Parse error: ${e.message}`);
    return null;
  }
}

const enDir = path.join(BASE, 'en');
const modules = fs.readdirSync(enDir)
  .filter(f => f.endsWith('.ts') && f !== 'index.ts')
  .map(f => f.replace('.ts', ''));

let totalMissing = 0;
let issues = [];

for (const mod of modules) {
  const enPath = path.join(enDir, `${mod}.ts`);
  const enObj = parseTS(fs.readFileSync(enPath, 'utf8'));
  if (!enObj) continue;
  const enPaths = getAllLeafPaths(enObj);

  for (const locale of locales) {
    const locPath = path.join(BASE, locale, `${mod}.ts`);
    if (!fs.existsSync(locPath)) {
      issues.push(`MISSING FILE: ${locale}/${mod}.ts`);
      totalMissing++;
      continue;
    }
    const locObj = parseTS(fs.readFileSync(locPath, 'utf8'));
    if (!locObj) {
      issues.push(`PARSE ERROR: ${locale}/${mod}.ts`);
      totalMissing++;
      continue;
    }
    const locPaths = getAllLeafPaths(locObj);

    const missing = [...enPaths.keys()].filter(k => !locPaths.has(k) && !k.startsWith('admin.'));
    for (const k of missing) {
      issues.push(`${locale}/${mod}.ts  missing: ${k}`);
      totalMissing++;
    }
  }
}

if (totalMissing > 0) {
  console.log(`Found ${totalMissing} missing i18n key(s) across non-en locales:`);
  for (const issue of issues) {
    console.log(`  ${issue}`);
  }
  console.log(`\nRun:  node scripts/fill-i18n-from-en.js  to auto-fill missing keys.`);
  process.exit(strict ? 1 : 0);
} else {
  console.log('All non-en locales have complete key coverage (matching English canonical).');
  process.exit(0);
}
ENDSCRIPT
)

export I18N_DIR STRICT
node -e "$NODE_SCRIPT"
