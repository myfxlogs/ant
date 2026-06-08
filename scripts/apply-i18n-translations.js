#!/usr/bin/env node
// apply-i18n-translations.js — Apply translated values to a locale's resource files.
//
// Usage:
//   node scripts/apply-i18n-translations.js <locale> <translations.json>
//
// The translations JSON maps merged key paths (e.g. "strategy.workspace.title")
// to translated values. Keys intentionally left in English (proper names, code,
// abbreviations) should be omitted from the JSON — they stay as English fallback.

const fs = require('fs');
const path = require('path');

const BASE = path.join(__dirname, '..', 'frontend', 'src', 'i18n', 'resources');
const locale = process.argv[2];
const transFile = process.argv[3];

if (!locale || !transFile) {
  console.error('Usage: node apply-i18n-translations.js <locale> <translations.json>');
  process.exit(1);
}

const translations = JSON.parse(fs.readFileSync(transFile, 'utf8'));

// Map merged key paths to their file module
const aiSubModules = {
  'ai.agentPrompts': 'ai', 'ai.consensus': 'ai', 'ai.conversation': 'ai',
  'ai.chatBox': 'ai', 'ai.reports': 'ai', 'ai.signalCard': 'ai',
  'ai.assistant': 'ai', 'ai.strategyCard': 'ai', 'ai.requireConfig': 'ai',
  'ai.riskEval': 'ai', 'ai.workflowRuns': 'ai', 'ai.backtestScoreCard': 'ai',
  'ai.systemAI': 'ai', 'ai.tabs': 'ai', 'ai.gate': 'ai', 'ai.client': 'ai',
  'ai.wizard': 'ai_wizard', 'ai.settings': 'ai_settings', 'ai.store': 'ai_store',
};

const topKeyToFile = {
  app: 'base', auth: 'base', common: 'base', language: 'base',
  menu: 'base', marketplace: 'base', market: 'base', topbar: 'base',
  profile: 'base', notifications: 'base', admin: 'base', errors: 'base',
  symbolDetection: 'base',
  accounts: 'accounts', analytics: 'analytics', dashboard: 'dashboard',
  trading: 'trading', strategy: 'strategy', indicatorCatalog: 'strategy',
  logs: 'logs', ai: 'ai',
};

function getFileForKey(key) {
  if (key.startsWith('ai.')) {
    let best = 'ai';
    for (const [prefix, file] of Object.entries(aiSubModules)) {
      if (key.startsWith(prefix + '.') || key === prefix) {
        if (prefix.length > (best === 'ai' ? 'ai'.length : 0)) best = file;
      }
    }
    // Check sub-keys of nested objects
    for (const [prefix, file] of Object.entries(aiSubModules)) {
      if (key.startsWith(prefix + '.') || key === prefix) {
        return file;
      }
    }
    return best;
  }
  const topKey = key.split('.')[0];
  return topKeyToFile[topKey] || topKey;
}

function setNestedValue(obj, keyPath, value) {
  const parts = keyPath.split('.');
  let current = obj;
  for (let i = 0; i < parts.length - 1; i++) {
    if (!(parts[i] in current)) current[parts[i]] = {};
    current = current[parts[i]];
  }
  current[parts[parts.length - 1]] = value;
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
    const escaped = obj.replace(/\\/g, '\\\\').replace(/`/g, '\\`').replace(/\${/g, '\\${');
    if (escaped.includes('\n') || escaped.includes("'")) return '`' + escaped + '`';
    return "'" + escaped + "'";
  }
  if (typeof obj === 'number' || typeof obj === 'boolean') return String(obj);
  if (Array.isArray(obj)) {
    if (obj.length === 0) return '[]';
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

// Group by file
const byFile = {};
for (const [key, value] of Object.entries(translations)) {
  const file = getFileForKey(key);
  if (!byFile[file]) byFile[file] = {};
  byFile[file][key] = value;
}

let totalApplied = 0;
let totalFiles = 0;

for (const [file, trans] of Object.entries(byFile)) {
  const locPath = path.join(BASE, locale, `${file}.ts`);
  if (!fs.existsSync(locPath)) {
    console.log(`SKIP ${file}.ts (not found)`);
    continue;
  }

  const content = fs.readFileSync(locPath, 'utf8');
  const obj = parseTS(content);
  if (!obj) { console.log(`SKIP ${file}.ts (parse fail)`); continue; }

  let changed = 0;
  for (const [key, value] of Object.entries(trans)) {
    try {
      setNestedValue(obj, key, value);
      changed++;
    } catch (e) {
      console.log(`  Error: ${key} — ${e.message}`);
    }
  }

  if (changed > 0) {
    const varMatch = content.match(/const\s+(\w+)\s*=/);
    const varName = varMatch ? varMatch[1] : file;
    const newContent = `const ${varName} = ${toTS(obj)} as const;\n\nexport default ${varName};\n`;
    fs.writeFileSync(locPath, newContent, 'utf8');
    console.log(`${locale}/${file}.ts: ${changed} keys translated`);
    totalApplied += changed;
    totalFiles++;
  }
}

console.log(`\nDone: ${totalApplied} translations applied across ${totalFiles} files for ${locale}`);
