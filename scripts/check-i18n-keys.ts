#!/usr/bin/env npx tsx
/**
 * check-i18n-keys.ts - Verify all non-en locale modules have every key from English canonical.
 *
 * Usage:
 *   npx tsx scripts/check-i18n-keys.ts           # report mode
 *   npx tsx scripts/check-i18n-keys.ts --strict  # CI mode (exit 1 on any missing key)
 *
 * English (en/) is the canonical source of truth. Every key present in English
 * MUST also exist in every other locale.
 * Excludes: admin.* keys (managed separately).
 */

import * as fs from 'fs';
import * as path from 'path';

const strict = process.argv.includes('--strict');
const BASE = path.join(import.meta.dirname, '..', 'frontend', 'src', 'i18n', 'resources');

const locales = fs.readdirSync(BASE).filter(d => {
  const stat = fs.statSync(path.join(BASE, d));
  return stat.isDirectory() && d !== 'en';
});

const modules = fs.readdirSync(path.join(BASE, 'en'))
  .filter(f => f.endsWith('.ts') && f !== 'index.ts')
  .map(f => f.replace('.ts', ''));

function leafPaths(obj: Record<string, unknown>, prefix = ''): Map<string, unknown> {
  const out = new Map<string, unknown>();
  for (const [k, v] of Object.entries(obj)) {
    const fullKey = prefix ? prefix + '.' + k : k;
    if (v !== null && typeof v === 'object' && !Array.isArray(v)) {
      for (const [p, val] of leafPaths(v as Record<string, unknown>, fullKey)) {
        out.set(p, val);
      }
    } else {
      out.set(fullKey, v);
    }
  }
  return out;
}

async function main() {
let totalMissing = 0;
const issues: string[] = [];

for (const mod of modules) {
  const enPath = path.join(BASE, 'en', mod + '.ts');
  const enMod = await import(enPath);
  const enObj = enMod.default || enMod;
  const enKeys = leafPaths(enObj);

  for (const locale of locales) {
    const locPath = path.join(BASE, locale, mod + '.ts');
    if (!fs.existsSync(locPath)) {
      issues.push('MISSING FILE: ' + locale + '/' + mod + '.ts');
      totalMissing++;
      continue;
    }
    try {
      const locMod = await import(locPath);
      const locObj = locMod.default || locMod;
      const locKeys = leafPaths(locObj);

      for (const [key] of enKeys) {
        if (key.startsWith('admin.')) continue;
        if (!locKeys.has(key)) {
          issues.push(locale + '/' + mod + '.ts  missing: ' + key);
          totalMissing++;
        }
      }
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : String(e);
      issues.push('PARSE ERROR: ' + locale + '/' + mod + '.ts - ' + msg);
      totalMissing++;
    }
  }
}

if (totalMissing > 0) {
  console.log('Found ' + totalMissing + ' missing i18n key(s) across non-en locales:');
  for (const issue of issues) {
    console.log('  ' + issue);
  }
  console.log('\nRun:  npx tsx scripts/fill-i18n-from-en.ts --write  to auto-fill missing keys.');
  process.exit(strict ? 1 : 0);
} else {
  console.log('All non-en locales have complete key coverage (matching English canonical).');
  process.exit(0);
}
}

main();
