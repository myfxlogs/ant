#!/usr/bin/env tsx
/**
 * Scans frontend .tsx files for t('key', { defaultValue: '...' }) calls
 * and generates missing textproto entries + map.json entries.
 *
 * Usage: npx tsx scripts/i18n-gen-missing.ts
 */
import * as fs from 'fs';
import * as path from 'path';

const FRONTEND_SRC = '/opt/ant/frontend/src';
const PROTO_DIR = '/opt/ant/proto/ant/v1/i18n';
const MAP_FILE = path.join(PROTO_DIR, 'base_map.json');

interface MissingKey {
  key: string;
  defaultValue: string;
  file: string;
}

function findTsxFiles(dir: string): string[] {
  const results: string[] = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      results.push(...findTsxFiles(full));
    } else if (entry.name.endsWith('.tsx') || entry.name.endsWith('.ts')) {
      results.push(full);
    }
  }
  return results;
}

function extractKeys(files: string[]): MissingKey[] {
  const keys: Map<string, MissingKey> = new Map();

  // Match: t('some.key', { defaultValue: 'Some Text' })
  // Also match: t('some.key', 'Default Text')
  // Also match: t('some.key', { defaultValue: "Some Text" })
  const re1 = /t\(\s*['"]([^'"]+)['"]\s*,\s*\{\s*defaultValue:\s*['"]([^'"]*)['"]\s*\}/g;
  const re2 = /t\(\s*['"]([^'"]+)['"]\s*,\s*['"]([^'"]*)['"]\s*\)/g;

  for (const file of files) {
    const content = fs.readFileSync(file, 'utf-8');
    let m: RegExpExecArray | null;

    while ((m = re1.exec(content)) !== null) {
      const key = m[1];
      const defaultValue = m[2];
      if (!keys.has(key)) {
        keys.set(key, { key, defaultValue, file });
      }
    }

    while ((m = re2.exec(content)) !== null) {
      const key = m[1];
      const defaultValue = m[2];
      if (!keys.has(key)) {
        keys.set(key, { key, defaultValue, file });
      }
    }
  }

  return Array.from(keys.values());
}

function loadMapPaths(): Set<string> {
  const content = fs.readFileSync(MAP_FILE, 'utf-8');
  const map = JSON.parse(content);
  return new Set(Object.values(map.fields));
}

function loadTextprotoKeys(locale: string): Set<string> {
  const file = path.join(PROTO_DIR, `base_${locale}.textproto`);
  const content = fs.readFileSync(file, 'utf-8');
  const keys = new Set<string>();
  const re = /^(\w+):/gm;
  let m: RegExpExecArray | null;
  while ((m = re.exec(content)) !== null) {
    keys.add(m[1]);
  }
  return keys;
}

function keyToProtoName(key: string): string {
  // Convert dotted path to snake_case proto field name
  // e.g. "admin.billing.title" -> "admin_billing_title"
  // e.g. "strategy.live.activeTab" -> "strategy_live_active_tab"
  // Handle camelCase: "activeTab" -> "active_tab"
  return key
    .replace(/\./g, '_')
    .replace(/([A-Z])/g, '_$1')
    .toLowerCase()
    .replace(/^_+/, '');
}

function keyToMapPath(key: string): string {
  // Keep dotted path but convert camelCase segments
  return key
    .split('.')
    .map(segment => segment.replace(/([A-Z])/g, (match, p1, offset) => {
      // Convert first char to lowercase, keep rest
      if (offset === 0) return segment.toLowerCase();
      return segment[0].toLowerCase() + segment.slice(1);
    }))
    .join('.');
}

function main() {
  const allKeys = extractKeys(findTsxFiles(FRONTEND_SRC));
  const mapPaths = loadMapPaths();
  const textprotoKeys = loadTextprotoKeys('en');

  // Find keys missing from map.json
  const missingFromMap = allKeys.filter(k => !mapPaths.has(k.key));
  // Find keys that are in map.json but missing from textprotos
  const missingFromTextproto = allKeys.filter(k => {
    const protoName = keyToProtoName(k.key);
    return !textprotoKeys.has(protoName);
  });

  console.log(`Total t() keys found: ${allKeys.length}`);
  console.log(`Missing from map.json: ${missingFromMap.length}`);
  console.log(`Missing from textproto (en): ${missingFromTextproto.length}`);

  // Generate map.json entries
  if (missingFromMap.length > 0) {
    const map = JSON.parse(fs.readFileSync(MAP_FILE, 'utf-8'));
    for (const k of missingFromMap) {
      const protoName = keyToProtoName(k.key);
      const mapPath = k.key; // Keep the dotted path as-is
      map.fields[protoName] = mapPath;
    }
    fs.writeFileSync(MAP_FILE, JSON.stringify(map, null, 2) + '\n');
    console.log(`\nUpdated base_map.json with ${missingFromMap.length} new entries`);

    // Also update other section map files if the key belongs to them
    // Check strategy_workspace_map.json and strategy_chart_tools_map.json
    for (const k of missingFromMap) {
      if (k.key.startsWith('strategy.workspace.')) {
        const swMap = JSON.parse(fs.readFileSync(path.join(PROTO_DIR, 'strategy_workspace_map.json'), 'utf-8'));
        const protoName = k.key.replace('strategy.workspace.', '').replace(/\./g, '_').replace(/([A-Z])/g, '_$1').toLowerCase();
        const relPath = k.key.replace('strategy.workspace.', '');
        if (!(protoName in swMap.fields)) {
          swMap.fields[protoName] = relPath;
          fs.writeFileSync(path.join(PROTO_DIR, 'strategy_workspace_map.json'), JSON.stringify(swMap, null, 2) + '\n');
        }
      }
      if (k.key.startsWith('strategy.chartTools.')) {
        const ctMap = JSON.parse(fs.readFileSync(path.join(PROTO_DIR, 'strategy_chart_tools_map.json'), 'utf-8'));
        const protoName = k.key.replace('strategy.chartTools.', '').replace(/\./g, '_').replace(/([A-Z])/g, '_$1').toLowerCase();
        const relPath = k.key.replace('strategy.chartTools.', '');
        if (!(protoName in ctMap.fields)) {
          ctMap.fields[protoName] = relPath;
          fs.writeFileSync(path.join(PROTO_DIR, 'strategy_chart_tools_map.json'), JSON.stringify(ctMap, null, 2) + '\n');
        }
      }
    }
  }

  // Generate textproto entries for en (using defaultValue as the translation)
  const newEntries: string[] = [];
  for (const k of missingFromTextproto) {
    const protoName = keyToProtoName(k.key);
    const value = k.defaultValue.replace(/'/g, "\\'");
    newEntries.push(`${protoName}: '${value}'`);
  }

  if (newEntries.length > 0) {
    // Append to en textproto (alphabetically at the end is fine, the parser doesn't care about order)
    const enFile = path.join(PROTO_DIR, 'base_en.textproto');
    const content = fs.readFileSync(enFile, 'utf-8');
    const newContent = content.trimEnd() + '\n' + newEntries.join('\n') + '\n';
    fs.writeFileSync(enFile, newContent);
    console.log(`\nAdded ${newEntries.length} entries to base_en.textproto`);

    // For other locales, add the same entries with English as placeholder
    // (translations can be done later)
    for (const locale of ['zh-cn', 'zh-tw', 'ja', 'vi']) {
      const file = path.join(PROTO_DIR, `base_${locale}.textproto`);
      const content = fs.readFileSync(file, 'utf-8');
      const newContent = content.trimEnd() + '\n' + newEntries.join('\n') + '\n';
      fs.writeFileSync(file, newContent);
      console.log(`Added ${newEntries.length} entries to base_${locale}.textproto (English placeholder)`);
    }
  }

  // Print summary of missing keys grouped by prefix
  const byPrefix: Record<string, number> = {};
  for (const k of missingFromTextproto) {
    const prefix = k.key.split('.').slice(0, 2).join('.');
    byPrefix[prefix] = (byPrefix[prefix] || 0) + 1;
  }
  console.log('\nMissing keys by prefix:');
  for (const [prefix, count] of Object.entries(byPrefix).sort((a, b) => b[1] - a[1])) {
    console.log(`  ${prefix}: ${count}`);
  }
}

main();
