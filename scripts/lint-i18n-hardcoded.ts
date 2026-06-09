#!/usr/bin/env npx tsx
/**
 * lint-i18n-hardcoded.ts — Scan for hardcoded English strings that bypass i18n.
 *
 * Detects:
 *  1. JSX text children that are literal English (not t()-wrapped)
 *  2. message.success/error/warning/info with literal English strings
 *  3. Props (title, placeholder, label, description, okText, cancelText,
 *     notFoundContent, emptyText) with literal English values
 *
 * Usage:
 *   npx tsx scripts/lint-i18n-hardcoded.ts            # report mode
 *   npx tsx scripts/lint-i18n-hardcoded.ts --strict   # CI mode (exit 1)
 *   npx tsx scripts/lint-i18n-hardcoded.ts --json     # JSON output (for tools)
 */

import * as fs from 'fs';
import * as path from 'path';

const strict = process.argv.includes('--strict');
const json = process.argv.includes('--json');

const SRC = path.join(import.meta.dirname || path.dirname(new URL(import.meta.url).pathname), '..', 'frontend', 'src');
const I18N_DIR = path.join(SRC, 'i18n', 'resources', 'en');

// Words that are NOT English user-facing text (skip these)
const IGNORE_PATTERNS = [
  /^[A-Z_]+$/,           // CONSTANTS
  /^[0-9.]+$/,           // numbers
  /^[^\w]*$/,            // symbols/emoji only
  /^(true|false|null|undefined)$/,
  /^https?:\/\//,         // URLs
  /^[./]/,               // paths
  /^#/,                  // colors
  /^(px|%|em|rem|vh|vw|ms|s)$/, // CSS units
];

function isEnglishUserText(s: string): boolean {
  const t = s.trim();
  if (t.length < 2) return false;
  if (t.match(/^[^a-zA-Z]*$/)) return false; // no letters
  for (const p of IGNORE_PATTERNS) {
    if (p.test(t)) return false;
  }
  // Must contain at least one English word (2+ consecutive letters)
  return /[a-zA-Z]{2,}/.test(t);
}

interface Issue {
  file: string; line: number; col: number;
  type: 'jsx-text' | 'message-call' | 'prop';
  value: string; suggestion: string;
}

function scanFile(filePath: string): Issue[] {
  const issues: Issue[] = [];
  const content = fs.readFileSync(filePath, 'utf8');
  const lines = content.split('\n');
  const shortPath = path.relative(SRC, filePath);

  // Helper: check if a line position is inside a JSX context
  // (simplified heuristic: we just look at patterns)

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const lineNum = i + 1;

    // Skip comments and imports
    if (line.trim().startsWith('//') || line.trim().startsWith('import ') || line.trim().startsWith('export ')) continue;
    if (line.trim().startsWith('*') || line.trim().startsWith('/*')) continue;

    // 1. JSX text children: >English Text<
    const jsxTextMatch = line.match(/>([A-Z][a-zA-Z].*?)</g);
    if (jsxTextMatch) {
      for (const m of jsxTextMatch) {
        const text = m.slice(1, -1).trim();
        if (isEnglishUserText(text) && !line.includes('t(') && !line.includes('{t(')) {
          issues.push({
            file: shortPath, line: lineNum, col: line.indexOf(m) + 1,
            type: 'jsx-text', value: text,
            suggestion: `{t('key.${text.toLowerCase().replace(/[^a-z0-9]+/g, '_')}')}`,
          });
        }
      }
    }

    // 2. message.success/error/warning/info('English string')
    const msgMatch = line.match(/message\.(success|error|warning|info)\s*\(\s*['"]([^'"]+)['"]/);
    if (msgMatch && isEnglishUserText(msgMatch[2]) && !line.includes('t(')) {
      issues.push({
        file: shortPath, line: lineNum, col: line.indexOf(msgMatch[0]) + 1,
        type: 'message-call', value: msgMatch[2],
        suggestion: `t('key.${msgMatch[2].toLowerCase().replace(/[^a-z0-9]+/g, '_')}')`,
      });
    }

    // 3. Props with literal English values
    const propNames = ['title', 'placeholder', 'label', 'description', 'okText', 'cancelText', 'notFoundContent', 'emptyText'];
    for (const prop of propNames) {
      // Match: prop="English" or prop='English'
      const propMatch = line.match(new RegExp(`${prop}\\s*=\\s*['"]([^'"]*[A-Z][^'"]*)['"]`));
      if (propMatch && isEnglishUserText(propMatch[1]) && !line.includes('t(') && !line.includes('{t(')) {
        // Check it's not already in a t() wrapper
        // Double check: the line should not have t() nearby
        if (!line.includes('t(' + prop) && !line.includes('t(`')) {
          issues.push({
            file: shortPath, line: lineNum, col: line.indexOf(propMatch[0]) + 1,
            type: 'prop', value: propMatch[1],
            suggestion: `${prop}={t('key.${propMatch[1].toLowerCase().replace(/[^a-z0-9]+/g, '_')}')}`,
          });
        }
      }
    }
  }

  return issues;
}

// Walk all TSX files
function walk(dir: string): string[] {
  const files: string[] = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === 'gen' || entry.name === 'node_modules' || entry.name.startsWith('.')) continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) files.push(...walk(full));
    else if (entry.name.endsWith('.tsx') || entry.name.endsWith('.ts')) {
      // Skip i18n resource files themselves
      if (!full.includes('/i18n/resources/')) files.push(full);
    }
  }
  return files;
}

const files = walk(SRC);
const allIssues: Issue[] = [];

for (const f of files) {
  const issues = scanFile(f);
  allIssues.push(...issues);
}

// Deduplicate by file+line+type
const seen = new Set<string>();
const unique = allIssues.filter(i => {
  const k = `${i.file}:${i.line}:${i.type}`;
  if (seen.has(k)) return false;
  seen.add(k);
  return true;
});

if (json) {
  console.log(JSON.stringify(unique, null, 2));
} else {
  if (unique.length === 0) {
    console.log('No hardcoded English strings found. All strings use t() or i18n properly.');
  } else {
    console.log(`Found ${unique.length} hardcoded English string(s):\n`);
    for (const issue of unique) {
      console.log(`  ${issue.file}:${issue.line} [${issue.type}] "${issue.value}"`);
    }
    console.log(`\nRun: npx tsx scripts/fix-i18n-hardcoded.ts --write  to auto-fix.`);
  }
}

process.exit(strict && unique.length > 0 ? 1 : 0);
