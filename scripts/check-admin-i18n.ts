// check-admin-i18n.ts
// Extracts all t('key') calls from admin .tsx files and checks if keys resolve
// to string values in the en and zh-cn translation resource objects.

import * as fs from 'fs';
import * as path from 'path';

const scriptDir = path.dirname(new URL(import.meta.url).pathname);
const ROOT = path.join(scriptDir, '..');
const ADMIN_DIR = path.join(ROOT, 'frontend', 'src', 'pages', 'admin');
const EN_INDEX = path.join(ROOT, 'frontend', 'src', 'i18n', 'resources', 'en', 'index.ts');
const ZH_INDEX = path.join(ROOT, 'frontend', 'src', 'i18n', 'resources', 'zh-cn', 'index.ts');

// Recursively find all .tsx/.ts files in admin dir
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

// Extract t('key') and t("key") patterns from file content
function extractKeys(content: string): Set<string> {
  const keys = new Set<string>();
  const regex = /\bt\(\s*['"]([a-zA-Z0-9_.]+)['"]/g;
  let match;
  while ((match = regex.exec(content)) !== null) {
    keys.add(match[1]);
  }
  return keys;
}

// Check if a dot-notation key exists in a nested object (resolves to a string)
function keyExists(obj: any, key: string): boolean {
  const parts = key.split('.');
  let current = obj;
  for (const part of parts) {
    if (current === null || current === undefined || typeof current !== 'object') {
      return false;
    }
    if (!(part in current)) {
      return false;
    }
    current = current[part];
  }
  return typeof current === 'string';
}

async function main() {
  const files = findTsxFiles(ADMIN_DIR);
  const allKeys = new Set<string>();

  for (const file of files) {
    const content = fs.readFileSync(file, 'utf-8');
    const keys = extractKeys(content);
    for (const k of keys) allKeys.add(k);
  }

  console.log(`Found ${allKeys.size} unique i18n keys in admin pages\n`);

  const enMod = await import(EN_INDEX);
  const enObj = enMod.default || enMod;
  const zhMod = await import(ZH_INDEX);
  const zhObj = zhMod.default || zhMod;

  const missingEn: string[] = [];
  const missingZh: string[] = [];

  for (const key of [...allKeys].sort()) {
    if (!keyExists(enObj, key)) missingEn.push(key);
    if (!keyExists(zhObj, key)) missingZh.push(key);
  }

  console.log(`=== Summary ===`);
  console.log(`Keys missing from EN:   ${missingEn.length}/${allKeys.size}`);
  console.log(`Keys missing from ZH:   ${missingZh.length}/${allKeys.size}`);
  console.log('');

  if (missingEn.length > 0) {
    console.log(`=== Missing from EN (${missingEn.length}) ===`);
    for (const k of missingEn) console.log(`  ${k}`);
    console.log('');
  }

  if (missingZh.length > 0) {
    console.log(`=== Missing from ZH-CN (${missingZh.length}) ===`);
    for (const k of missingZh) console.log(`  ${k}`);
  }

  if (missingEn.length === 0 && missingZh.length === 0) {
    console.log('All admin i18n keys are present in both EN and ZH-CN.');
  }
}

main().catch(console.error);
