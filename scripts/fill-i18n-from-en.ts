#!/usr/bin/env npx tsx
/**
 * fill-i18n-from-en.ts - Deep-merge missing English keys into all non-en locales.
 *
 * English (en/) is canonical. This script:
 * 1. Loads each locale module via tsx import (zero eval)
 * 2. Finds keys present in en but missing in the locale
 * 3. Inserts missing keys using precise string editing (preserves all formatting)
 *
 * Usage:
 *   npx tsx scripts/fill-i18n-from-en.ts          # dry-run
 *   npx tsx scripts/fill-i18n-from-en.ts --write  # apply changes
 *
 * Excludes: admin.* keys (managed separately)
 */

import * as fs from 'fs';
import * as path from 'path';
import { parseSource, insertKey } from './lib/i18n-source-editor.js';

const write = process.argv.includes('--write');
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

// Determine the top-level key in the merged object that corresponds to a module.
// e.g. ai_settings.ts exports { ai: { settings: {...} } }, so its top-level path is "ai.settings"
function getModulePrefix(mod: string): string {
  const mapping: Record<string, string> = {
    base: '',
    accounts: 'accounts',
    analytics: 'analytics',
    dashboard: 'dashboard',
    trading: 'trading',
    strategy: 'strategy',
    errors: 'errors',
    logs: 'logs',
    ai: 'ai',
    ai_settings: 'ai.settings',
    ai_store: 'ai.store',
    ai_wizard: 'ai.wizard',
  };
  return mapping[mod] ?? mod;
}

let totalAdded = 0;

async function main() {

for (const mod of modules) {
  const enPath = path.join(BASE, 'en', mod + '.ts');
  const enMod = await import(enPath);
  const enObj = (enMod.default || enMod) as Record<string, unknown>;
  const enKeys = leafPaths(enObj);
  const modPrefix = getModulePrefix(mod);

  for (const locale of locales) {
    const locPath = path.join(BASE, locale, mod + '.ts');
    if (!fs.existsSync(locPath)) {
      console.log('SKIP (no file): ' + locale + '/' + mod + '.ts');
      continue;
    }

    const locMod = await import(locPath);
    const locObj = (locMod.default || locMod) as Record<string, unknown>;
    const locKeys = leafPaths(locObj);

    // Find missing string keys (not admin)
    const missing: Array<{ key: string; value: string }> = [];
    for (const [key, value] of enKeys) {
      if (key.startsWith('admin.')) continue;
      if (!locKeys.has(key) && typeof value === 'string') {
        missing.push({ key, value });
      }
    }

    if (missing.length === 0) {
      // console.log('OK: ' + locale + '/' + mod + '.ts');  // too noisy
      continue;
    }

    if (write) {
      // Read source, parse structure, insert missing keys
      let source = fs.readFileSync(locPath, 'utf8');
      const locMap = parseSource(source);

      // Group missing keys by their parent path
      // e.g. "auth.forgotPassword.title" -> parent = "auth.forgotPassword"
      const byParent = new Map<string, Array<{ name: string; value: string }>>();
      for (const { key, value } of missing) {
        const lastDot = key.lastIndexOf('.');
        const parentPath = lastDot === -1 ? '' : key.slice(0, lastDot);
        const keyName = lastDot === -1 ? key : key.slice(lastDot + 1);
        if (!byParent.has(parentPath)) byParent.set(parentPath, []);
        byParent.get(parentPath)!.push({ name: keyName, value });
      }

      for (const [parentPath, entries] of byParent) {
        const parentLoc = parentPath ? locMap.get(parentPath) : null;
        // If parent doesn't exist in locale, try to find it by walking the tree
        if (!parentLoc && parentPath) {
          // The parent might not exist yet - we'd need to create intermediate objects.
          // For now, skip these (they're rare - usually whole branches are missing).
          console.log('  SKIP (parent not found): ' + parentPath + ' in ' + locale + '/' + mod + '.ts');
          continue;
        }

        for (const { name, value } of entries) {
          if (parentLoc) {
            source = insertKey(source, parentLoc, name, value);
          }
        }
      }

      fs.writeFileSync(locPath, source, 'utf8');
      console.log('FILLED: ' + locale + '/' + mod + '.ts + ' + missing.length + ' keys added');
      totalAdded += missing.length;
    } else {
      console.log('DRY-RUN: ' + locale + '/' + mod + '.ts + ' + missing.length + ' keys would be added');
      totalAdded += missing.length;
    }
  }
}

if (totalAdded === 0) {
  console.log('\nAll non-en locales are up to date with English canonical.');
} else {
  console.log('\nTotal keys ' + (write ? 'added' : 'to add') + ': ' + totalAdded);
}
}

main();
