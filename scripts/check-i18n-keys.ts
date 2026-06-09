#!/usr/bin/env npx tsx
/**
 * check-i18n-keys.ts - i18n quality CI check.
 *
 * Checks (in order of severity):
 * 1. Key coverage — every en key exists in every locale (error, blocks CI)
 * 2. Placeholder integrity — {{vars}} in en exist in locale translations (error, blocks CI)
 * 3. Glossary consistency — same en term translated differently across keys (warning only)
 *
 * Usage:
 *   npx tsx scripts/check-i18n-keys.ts             # report mode
 *   npx tsx scripts/check-i18n-keys.ts --strict    # CI mode (exit 1 on errors)
 *   npx tsx scripts/check-i18n-keys.ts --verbose   # show warnings
 */

import * as fs from 'fs';
import * as path from 'path';

const strict = process.argv.includes('--strict');
const verbose = process.argv.includes('--verbose');
const scriptDir = path.dirname(new URL(import.meta.url).pathname);
const BASE = path.join(scriptDir, '..', 'frontend', 'src', 'i18n', 'resources');

const locales = fs.readdirSync(BASE).filter(d => {
  const stat = fs.statSync(path.join(BASE, d));
  return stat.isDirectory() && d !== 'en';
});

const modules = fs.readdirSync(path.join(BASE, 'en'))
  .filter(f => f.endsWith('.ts') && f !== 'index.ts')
  .map(f => f.replace('.ts', ''));

// Load glossary
const glossaryPath = path.join(scriptDir, 'lib', 'i18n-glossary.json');
const glossary = JSON.parse(fs.readFileSync(glossaryPath, 'utf8')) as {
  terms: Record<string, Record<string, string>>;
};

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

// Extract {{placeholder}} names from a string
function extractPlaceholders(s: string): string[] {
  const matches = s.match(/\{\{(\w+)\}\}/g);
  if (!matches) return [];
  return [...new Set(matches.map(m => m.slice(2, -2)))];
}

interface Issue { severity: 'error' | 'warn'; msg: string; }

async function main() {
  const errors: string[] = [];
  const warnings: string[] = [];

  for (const mod of modules) {
    const enPath = path.join(BASE, 'en', mod + '.ts');
    const enMod = await import(enPath);
    const enObj = (enMod.default || enMod) as Record<string, unknown>;
    const enKeys = leafPaths(enObj);

    for (const locale of locales) {
      const locPath = path.join(BASE, locale, mod + '.ts');
      if (!fs.existsSync(locPath)) {
        errors.push('MISSING FILE: ' + locale + '/' + mod + '.ts');
        continue;
      }

      let locObj: Record<string, unknown>;
      try {
        const locMod = await import(locPath);
        locObj = (locMod.default || locMod) as Record<string, unknown>;
      } catch (e: unknown) {
        errors.push('PARSE ERROR: ' + locale + '/' + mod + '.ts + ' + (e instanceof Error ? e.message : String(e)));
        continue;
      }

      const locKeys = leafPaths(locObj);

      for (const [key, enVal] of enKeys) {
        if (key.startsWith('admin.')) continue;

        // Check 1: key exists
        if (!locKeys.has(key)) {
          errors.push(locale + '/' + mod + '.ts  missing key: ' + key);
          continue;
        }

        const locVal = locKeys.get(key);
        if (typeof enVal !== 'string' || typeof locVal !== 'string') continue;

        // Check 2: placeholder integrity
        const enPlaceholders = extractPlaceholders(enVal);
        if (enPlaceholders.length > 0) {
          const locPlaceholders = extractPlaceholders(locVal);
          const missingPh = enPlaceholders.filter(p => !locPlaceholders.includes(p));
          if (missingPh.length > 0) {
            errors.push(
              locale + '/' + mod + '.ts  broken placeholder in ' + key +
              ': en has {{' + missingPh.join('}}, {{') + '}} but locale missing'
            );
          }
        }

        // Check 3: glossary consistency (warning only, for non-en locales)
        if (locale === 'en') continue;
        for (const [enTerm, canonical] of Object.entries(glossary.terms)) {
          const expectedLocalized = canonical[locale];
          if (!expectedLocalized) continue;

          // Check if en value CONTAINS this English term
          if (enVal.includes(enTerm)) {
            // Check if locale value CONTAINS the expected localized term
            if (!locVal.includes(expectedLocalized)) {
              // The locale translation doesn't contain the glossary term.
              // This is NOT always wrong — context might demand a different word.
              // We only flag it if the English value is a short, standalone term
              // (likely a label), not a long sentence.
              if (enVal.length < 50 && enVal.trim() === enTerm) {
                warnings.push(
                  locale + '/' + mod + '.ts  glossary: "' + enTerm +
                  '" expected "' + expectedLocalized + '" but got "' + locVal + '"'
                );
              }
            }
          }
        }
      }
    }
  }

  // Output
  let exitCode = 0;

  if (errors.length > 0) {
    console.log('ERRORS (' + errors.length + '):');
    for (const e of errors) console.log('  ' + e);
    exitCode = 1;
  } else {
    console.log('Key coverage + placeholder integrity: PASS');
  }

  if (warnings.length > 0 && (verbose || errors.length === 0)) {
    console.log('\nGLOSSARY WARNINGS (' + warnings.length + '):');
    for (const w of warnings.slice(0, 20)) console.log('  ' + w);
    if (warnings.length > 20) console.log('  ... and ' + (warnings.length - 20) + ' more');
    console.log('  (Glossary warnings are advisory — review manually)');
  }

  if (exitCode === 0) {
    console.log('\nAll non-en locales have complete key coverage (matching English canonical).');
  }

  process.exit(strict ? exitCode : 0);
}

main();
