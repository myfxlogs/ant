#!/usr/bin/env node
// fill-i18n-from-en.js — Deep-merge missing English keys into all non-en locales.
//
// English (en/) is canonical. This script reads each non-en locale module,
// compares against the English version, and adds any missing keys with their
// English value as fallback. Existing locale translations are NEVER overwritten.
//
// Usage:
//   node scripts/fill-i18n-from-en.js          # dry-run (report only)
//   node scripts/fill-i18n-from-en.js --write  # actually write files
//
// Excludes: admin.* keys (managed separately)

const fs = require('fs');
const path = require('path');

const BASE = path.join(__dirname, '..', 'frontend', 'src', 'i18n', 'resources');
const write = process.argv.includes('--write');

function deepMerge(target, source) {
  const out = { ...target };
  for (const [key, value] of Object.entries(source)) {
    if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
      if (typeof out[key] === 'object' && out[key] !== null && !Array.isArray(out[key])) {
        out[key] = deepMerge(out[key], value);
      } else {
        out[key] = deepMerge({}, value);
      }
    } else if (!(key in out)) {
      out[key] = value;
    }
  }
  return out;
}

function parseTS(content) {
  const match = content.match(/const\s+\w+\s*=\s*({[\s\S]*})\s*as\s+const/);
  if (!match) return null;
  try { return eval('(' + match[1] + ')'); } catch (e) { return null; }
}

function toTS(obj, indent = 0) {
  const pad = '  '.repeat(indent);
  const pad1 = '  '.repeat(indent + 1);
  if (obj === null || obj === undefined) return 'null';
  if (typeof obj === 'string') {
    // Escape: backslash, backtick, dollar+brace (template literal injection)
    const escaped = obj.replace(/\\/g, '\\\\').replace(/`/g, '\\`').replace(/\${/g, '\\${');
    // Use backtick if string contains newlines OR single quotes, otherwise single quotes
    if (escaped.includes('\n') || escaped.includes("'")) {
      return '`' + escaped + '`';
    }
    return "'" + escaped + "'";
  }
  if (typeof obj === 'number' || typeof obj === 'boolean') return String(obj);
  if (Array.isArray(obj)) {
    const items = obj.map(v => `${pad1}${toTS(v, indent + 1)}`).join(',\n');
    return `[\n${items}\n${pad}]`;
  }
  if (typeof obj === 'object') {
    const keys = Object.keys(obj);
    if (keys.length === 0) return '{}';
    const kv = keys.map(k => {
      const v = toTS(obj[k], indent + 1);
      const key = /^[a-zA-Z_$][a-zA-Z0-9_$]*$/.test(k) ? k : `'${k}'`;
      return `${pad1}${key}: ${v}`;
    }).join(',\n');
    return `{\n${kv}\n${pad}}`;
  }
  return String(obj);
}

function countLeafKeys(obj) {
  let count = 0;
  for (const v of Object.values(obj)) {
    if (v !== null && typeof v === 'object' && !Array.isArray(v)) count += countLeafKeys(v);
    else count++;
  }
  return count;
}

const locales = fs.readdirSync(BASE).filter(d => {
  const stat = fs.statSync(path.join(BASE, d));
  return stat.isDirectory() && d !== 'en';
});

const enDir = path.join(BASE, 'en');
const modules = fs.readdirSync(enDir)
  .filter(f => f.endsWith('.ts') && f !== 'index.ts')
  .map(f => f.replace('.ts', ''));

let totalAdded = 0;

for (const mod of modules) {
  const enPath = path.join(enDir, `${mod}.ts`);
  const enObj = parseTS(fs.readFileSync(enPath, 'utf8'));
  if (!enObj) continue;

  for (const locale of locales) {
    const locPath = path.join(BASE, locale, `${mod}.ts`);
    if (!fs.existsSync(locPath)) {
      console.log(`SKIP (no file): ${locale}/${mod}.ts`);
      continue;
    }

    const locObj = parseTS(fs.readFileSync(locPath, 'utf8'));
    if (!locObj) { console.log(`SKIP (parse fail): ${locale}/${mod}.ts`); continue; }

    const oldCount = countLeafKeys(locObj);
    const merged = deepMerge(locObj, enObj);
    const newCount = countLeafKeys(merged);
    const added = newCount - oldCount;

    if (added > 0) {
      if (write) {
        const varMatch = fs.readFileSync(locPath, 'utf8').match(/const\s+(\w+)\s*=/);
        const varName = varMatch ? varMatch[1] : mod;
        const newContent = `const ${varName} = ${toTS(merged)} as const;\n\nexport default ${varName};\n`;
        fs.writeFileSync(locPath, newContent, 'utf8');
        console.log(`WROTE: ${locale}/${mod}.ts — ${added} keys added`);
      } else {
        console.log(`DRY-RUN: ${locale}/${mod}.ts — ${added} keys would be added`);
      }
      totalAdded += added;
    }
  }
}

if (totalAdded === 0) {
  console.log('\nAll non-en locales are up to date with English canonical.');
} else {
  console.log(`\nTotal keys to add: ${totalAdded}${write ? ' (written)' : ' (dry-run, use --write to apply)'}`);
}
