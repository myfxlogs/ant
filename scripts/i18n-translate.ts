#!/usr/bin/env tsx
/**
 * i18n-translate — shared translation engine for all locales.
 *
 * For every `<section>_<locale>.textproto` with a matching
 * `<section>_en.textproto`, translates English-placeholder entries (zh value
 * identical to en) using the per-locale DICT from scripts/i18n-dict/<locale>.ts.
 *
 * Usage: npx tsx scripts/i18n-translate.ts --locale <loc>
 *   (default locale: zh-cn — backward compatible)
 *
 * I18N-MIXED-2 S1: generalized from the zh-cn-only script.
 */
import * as fs from 'fs';
import * as path from 'path';

const PROTO_DIR = path.resolve(__dirname, '..', 'proto', 'ant', 'v1', 'i18n');

const LOCALE_IDX = process.argv.indexOf('--locale');
const LOCALE = LOCALE_IDX !== -1 ? process.argv[LOCALE_IDX + 1] : 'zh-cn';

interface TextprotoEntry {
  key: string;
  value: string;
}

function parseTextproto(content: string): TextprotoEntry[] {
  const entries: TextprotoEntry[] = [];
  for (const line of content.split('\n')) {
    const m = line.match(/^(\w+):\s*'(.*)'\s*$/);
    if (m) {
      entries.push({ key: m[1], value: m[2] });
    }
  }
  return entries;
}

async function main() {
  const { DICT } = await import(`./i18n-dict/${LOCALE}.ts`);

  const zhFiles = fs
    .readdirSync(PROTO_DIR)
    .filter(f => f.endsWith(`_${LOCALE}.textproto`))
    .sort();

  let total = 0;
  for (const zhFile of zhFiles) {
    const prefix = zhFile.replace(new RegExp(`_${LOCALE}\\.textproto$`), '');
    const enFile = path.join(PROTO_DIR, `${prefix}_en.textproto`);
    if (!fs.existsSync(enFile)) {
      console.log(`SKIP ${zhFile}: no matching ${prefix}_en.textproto`);
      continue;
    }
    const zhPath = path.join(PROTO_DIR, zhFile);
    const translated = processPair(enFile, zhPath, DICT);
    total += translated;
    console.log(`Translated ${translated} entries in ${zhFile}`);
  }
  console.log(`Total translated: ${total}`);
}

function processPair(
  enFile: string,
  zhFile: string,
  DICT: Record<string, string>
): number {
  const enEntries = parseTextproto(fs.readFileSync(enFile, 'utf-8'));
  const zhContent = fs.readFileSync(zhFile, 'utf-8');

  let translated = 0;
  const newLines: string[] = [];

  for (const line of zhContent.split('\n')) {
    const m = line.match(/^(\w+):\s*'(.*)'\s*$/);
    if (m) {
      const key = m[1];
      const zhValue = m[2];
      const enEntry = enEntries.find(e => e.key === key);

      if (enEntry && zhValue === enEntry.value) {
        const translatedValue = translate(zhValue, DICT);
        if (translatedValue) {
          newLines.push(`${key}: '${translatedValue.replace(/'/g, "\\'")}'`);
          translated++;
        } else {
          newLines.push(line);
        }
      } else {
        newLines.push(line);
      }
    } else {
      newLines.push(line);
    }
  }

  fs.writeFileSync(zhFile, newLines.join('\n'));
  return translated;
}

function translate(value: string, DICT: Record<string, string>): string | null {
  // Exact match first
  if (DICT[value]) return DICT[value];

  // Partial replacements for composite strings
  let result = value;
  let changed = false;
  for (const [en, zh] of Object.entries(DICT)) {
    if (en.length < 3) continue;
    const re = new RegExp('\\b' + en.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '\\b', 'g');
    if (re.test(result)) {
      result = result.replace(re, zh);
      changed = true;
    }
  }

  return changed ? result : null;
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
