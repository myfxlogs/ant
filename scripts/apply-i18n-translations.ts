#!/usr/bin/env npx tsx
/**
 * apply-i18n-translations.ts — Apply translated values to locale resource files.
 *
 * Architecture (zero parser, zero eval):
 *  1. import() the locale module via tsx → live JS object
 *  2. Set translated values by walking the object with dot-path keys
 *  3. Serialize with toTS() — clean, consistent output
 *
 * Usage:
 *   npx tsx scripts/apply-i18n-translations.ts <locale> <translations.json>
 */

import * as fs from 'fs';
import * as path from 'path';

const locale = process.argv[2];
const transFile = process.argv[3];

if (!locale || !transFile) {
  console.error('Usage: npx tsx scripts/apply-i18n-translations.ts <locale> <translations.json>');
  process.exit(1);
}

const scriptDir = path.dirname(new URL(import.meta.url).pathname);
const BASE = path.join(scriptDir, '..', 'frontend', 'src', 'i18n', 'resources');
const translations: Record<string, string> = JSON.parse(fs.readFileSync(transFile, 'utf8'));

// Map key prefixes to module files
const aiSubModules: Record<string, string> = {
  'ai.agentPrompts': 'ai', 'ai.consensus': 'ai', 'ai.conversation': 'ai',
  'ai.chatBox': 'ai', 'ai.reports': 'ai', 'ai.signalCard': 'ai',
  'ai.assistant': 'ai', 'ai.strategyCard': 'ai', 'ai.requireConfig': 'ai',
  'ai.riskEval': 'ai', 'ai.workflowRuns': 'ai', 'ai.backtestScoreCard': 'ai',
  'ai.systemAI': 'ai', 'ai.tabs': 'ai', 'ai.gate': 'ai', 'ai.client': 'ai',
  'ai.wizard': 'ai_wizard', 'ai.settings': 'ai_settings', 'ai.store': 'ai_store',
};
const topKeyToFile: Record<string, string> = {
  app: 'base', auth: 'base', common: 'base', language: 'base',
  menu: 'base', marketplace: 'base', market: 'base', topbar: 'base',
  profile: 'base', notifications: 'base', admin: 'base', errors: 'base',
  symbolDetection: 'base',
  accounts: 'accounts', analytics: 'analytics', dashboard: 'dashboard',
  trading: 'trading', strategy: 'strategy', indicatorCatalog: 'strategy',
  logs: 'logs', ai: 'ai',
};

function getFileForKey(key: string): string {
  if (key.startsWith('ai.')) {
    let best = 'ai', bestLen = 0;
    for (const [prefix, file] of Object.entries(aiSubModules)) {
      if (key.startsWith(prefix) && prefix.length > bestLen) { best = file; bestLen = prefix.length; }
    }
    return best;
  }
  const topKey = key.split('.')[0];
  return topKeyToFile[topKey] || topKey;
}

// Set a deeply nested value in an object by dot-path
function setNested(obj: Record<string, unknown>, keyPath: string, value: unknown): void {
  const parts = keyPath.split('.');
  let cur = obj;
  for (let i = 0; i < parts.length - 1; i++) {
    if (!(parts[i] in cur) || typeof cur[parts[i]] !== 'object' || cur[parts[i]] === null) {
      cur[parts[i]] = {};
    }
    cur = cur[parts[i]] as Record<string, unknown>;
  }
  cur[parts[parts.length - 1]] = value;
}

// toTS: serialize a JS value to TS object literal notation
function toTS(obj: unknown, indent = 0): string {
  const pad = '  '.repeat(indent);
  const pad1 = '  '.repeat(indent + 1);
  if (obj === null || obj === undefined) return 'null';
  if (typeof obj === 'string') {
    const s = obj as string;
    const escaped = s.replace(/\\/g, '\\\\').replace(/`/g, '\\`').replace(/\${/g, '\\${');
    if (s.includes('\n') || s.includes("'")) return '`' + escaped + '`';
    return "'" + escaped + "'";
  }
  if (typeof obj === 'number' || typeof obj === 'boolean') return String(obj);
  if (Array.isArray(obj)) {
    if (obj.length === 0) return '[]';
    return '[\n' + obj.map(v => pad1 + toTS(v, indent + 1)).join(',\n') + '\n' + pad + ']';
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

// Group translations by file
const byFile: Record<string, Record<string, string>> = {};
for (const [key, value] of Object.entries(translations)) {
  const file = getFileForKey(key);
  if (!byFile[file]) byFile[file] = {};
  byFile[file][key] = value;
}

let totalApplied = 0;

async function main() {

for (const [file, trans] of Object.entries(byFile)) {
  const locPath = path.join(BASE, locale, file + '.ts');
  if (!fs.existsSync(locPath)) { console.log('SKIP (no file): ' + file); continue; }

  // Load locale module via tsx import
  const locMod = await import(locPath);
  const obj = (locMod.default || locMod) as Record<string, unknown>;

  let changed = 0;
  for (const [key, value] of Object.entries(trans)) {
    setNested(obj, key, value);
    changed++;
  }

  if (changed > 0) {
    const varMatch = fs.readFileSync(locPath, 'utf8').match(/const\s+(\w+)\s*=/);
    const varName = varMatch ? varMatch[1] : file;
    const newContent = 'const ' + varName + ' = ' + toTS(obj) + ' as const;\n\nexport default ' + varName + ';\n';
    fs.writeFileSync(locPath, newContent, 'utf8');
    console.log(locale + '/' + file + '.ts: ' + changed + ' keys translated');
    totalApplied += changed;
  }
}

console.log('\nDone: ' + totalApplied + ' translations applied for ' + locale);
}

main();
