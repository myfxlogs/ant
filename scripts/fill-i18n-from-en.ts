#!/usr/bin/env npx tsx
/**
 * fill-i18n-from-en.ts - Deep-merge missing English keys into all non-en locales.
 *
 * English (en/) is canonical. Loads each locale module via tsx import,
 * deep-merges missing keys from English, and serializes back with toTS().
 *
 * Usage:
 *   npx tsx scripts/fill-i18n-from-en.ts          # dry-run
 *   npx tsx scripts/fill-i18n-from-en.ts --write  # apply changes
 */

import * as fs from 'fs';
import * as path from 'path';

const write = process.argv.includes('--write');
const scriptDir = path.dirname(new URL(import.meta.url).pathname);
const BASE = path.join(scriptDir, '..', 'frontend', 'src', 'i18n', 'resources');

const locales = fs.readdirSync(BASE).filter(d => {
  const stat = fs.statSync(path.join(BASE, d));
  return stat.isDirectory() && d !== 'en';
});

const modules = fs.readdirSync(path.join(BASE, 'en'))
  .filter(f => f.endsWith('.ts') && f !== 'index.ts')
  .map(f => f.replace('.ts', ''));

function deepMerge(target: Record<string, unknown>, source: Record<string, unknown>): Record<string, unknown> {
  const out = { ...target };
  for (const [key, value] of Object.entries(source)) {
    if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
      if (typeof out[key] === 'object' && out[key] !== null && !Array.isArray(out[key])) {
        out[key] = deepMerge(out[key] as Record<string, unknown>, value as Record<string, unknown>);
      } else if (!(key in out)) {
        out[key] = deepMerge({}, value as Record<string, unknown>);
      }
    } else if (!(key in out)) {
      out[key] = value;
    }
  }
  return out;
}

function countLeafKeys(obj: Record<string, unknown>): number {
  let count = 0;
  for (const v of Object.values(obj)) {
    if (v !== null && typeof v === 'object' && !Array.isArray(v)) count += countLeafKeys(v as Record<string, unknown>);
    else count++;
  }
  return count;
}

// toTS: serialize a plain object back to TS object literal notation.
// CRITICAL: always uses backtick templates for strings containing newlines
// or single quotes + backticks, to produce valid JavaScript.
function toTS(obj: unknown, indent = 0): string {
  const pad = '  '.repeat(indent);
  const pad1 = '  '.repeat(indent + 1);
  if (obj === null || obj === undefined) return 'null';
  if (typeof obj === 'string') {
    const s = obj as string;
    // Escape backslash, backtick, template injection
    const escaped = s.replace(/\\/g, '\\\\').replace(/`/g, '\\`').replace(/\${/g, '\\${');
    // Multi-line or contains single-quote → use backtick
    if (s.includes('\n') || s.includes("'")) {
      return '`' + escaped + '`';
    }
    return "'" + escaped + "'";
  }
  if (typeof obj === 'number' || typeof obj === 'boolean') return String(obj);
  if (Array.isArray(obj)) {
    if (obj.length === 0) return '[]';
    const items = obj.map(v => pad1 + toTS(v, indent + 1)).join(',\n');
    return '[\n' + items + '\n' + pad + ']';
  }
  if (typeof obj === 'object') {
    const keys = Object.keys(obj as Record<string, unknown>);
    if (keys.length === 0) return '{}';
    const kv = keys.map(k => {
      const v = toTS((obj as Record<string, unknown>)[k], indent + 1);
      const key = /^[a-zA-Z_$][a-zA-Z0-9_$]*$/.test(k) ? k : "'" + k + "'";
      return pad1 + key + ': ' + v;
    }).join(',\n');
    return '{\n' + kv + '\n' + pad + '}';
  }
  return String(obj);
}

let totalAdded = 0;

async function main() {
  for (const mod of modules) {
    const enPath = path.join(BASE, 'en', mod + '.ts');
    const enMod = await import(enPath);
    const enObj = (enMod.default || enMod) as Record<string, unknown>;
    const enCount = countLeafKeys(enObj);

    for (const locale of locales) {
      const locPath = path.join(BASE, locale, mod + '.ts');
      if (!fs.existsSync(locPath)) {
        console.log('SKIP (no file): ' + locale + '/' + mod + '.ts');
        continue;
      }

      const locMod = await import(locPath);
      const locObj = (locMod.default || locMod) as Record<string, unknown>;
      const oldCount = countLeafKeys(locObj);

      const merged = deepMerge(locObj, enObj);
      const newCount = countLeafKeys(merged);
      const added = newCount - oldCount;

      if (added > 0) {
        if (write) {
          const varMatch = fs.readFileSync(locPath, 'utf8').match(/const\s+(\w+)\s*=/);
          const varName = varMatch ? varMatch[1] : mod;
          const newContent = 'const ' + varName + ' = ' + toTS(merged) + ' as const;\n\nexport default ' + varName + ';\n';
          fs.writeFileSync(locPath, newContent, 'utf8');
          console.log('FILLED: ' + locale + '/' + mod + '.ts ' + added + ' keys added');
        } else {
          console.log('DRY-RUN: ' + locale + '/' + mod + '.ts ' + added + ' keys would be added');
        }
        totalAdded += added;
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
