#!/usr/bin/env npx tsx
/**
 * apply-i18n-translations.ts - Apply translated values to locale resource files.
 *
 * Uses tsx import to load locale modules (zero eval) and precise string
 * replacement to modify only the changed values (zero format loss).
 *
 * Usage:
 *   npx tsx scripts/apply-i18n-translations.ts <locale> <translations.json>
 *
 * The translations JSON maps merged key paths to translated values.
 * Keys already matching the translation are skipped (idempotent).
 */

import * as fs from 'fs';
import * as path from 'path';
import { parseSource, replaceValue } from './lib/i18n-source-editor.js';

const locale = process.argv[2];
const transFile = process.argv[3];

if (!locale || !transFile) {
  console.error('Usage: npx tsx scripts/apply-i18n-translations.ts <locale> <translations.json>');
  process.exit(1);
}

const BASE = path.join(import.meta.dirname, '..', 'frontend', 'src', 'i18n', 'resources');
const translations: Record<string, string> = JSON.parse(fs.readFileSync(transFile, 'utf8'));

// Map merged key prefixes to file modules (matches the mergeResources structure)
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
    let best = 'ai';
    let bestLen = 0;
    for (const [prefix, file] of Object.entries(aiSubModules)) {
      if (key.startsWith(prefix) && prefix.length > bestLen) {
        best = file;
        bestLen = prefix.length;
      }
    }
    return best;
  }
  const topKey = key.split('.')[0];
  return topKeyToFile[topKey] || topKey;
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
  if (!fs.existsSync(locPath)) {
    console.log('SKIP (not found): ' + file + '.ts');
    continue;
  }

  // Load the locale module to check current values
  let locObj: Record<string, unknown>;
  try {
    const mod = await import(locPath);
    locObj = (mod.default || mod) as Record<string, unknown>;
  } catch (e: unknown) {
    console.log('SKIP (import fail): ' + file + '.ts - ' + (e instanceof Error ? e.message : String(e)));
    continue;
  }

  // Parse source for precise editing
  let source = fs.readFileSync(locPath, 'utf8');
  const locMap = parseSource(source);
  let changed = 0;

  for (const [key, newValue] of Object.entries(trans)) {
    const loc = locMap.get(key);
    if (!loc) {
      console.log('  WARN: key not found in source: ' + key);
      continue;
    }
    if (loc.stringValue === newValue) continue; // already correct
    if (loc.stringValue === null) {
      console.log('  WARN: non-string value at: ' + key);
      continue;
    }

    source = replaceValue(source, loc, newValue);
    changed++;
  }

  if (changed > 0) {
    fs.writeFileSync(locPath, source, 'utf8');
    console.log(locale + '/' + file + '.ts: ' + changed + ' keys translated');
    totalApplied += changed;
  }
}

console.log('\nDone: ' + totalApplied + ' translations applied for ' + locale);
}

main();
