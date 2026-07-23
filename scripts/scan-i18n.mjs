#!/usr/bin/env node
/**
 * i18n hardcoded string scanner.
 * Scans all .tsx/.ts files under src/ for hardcoded user-visible strings.
 *
 * Detection patterns:
 * 1. JSX text children: <Tag>Hardcoded</Tag>
 * 2. JSX attributes: placeholder="...", title="...", label="..."
 * 3. Ant Design message.* / notification.* calls with string literals
 * 4. Ant Design Modal.confirm/etc. with string literals
 * 5. defaultValue in t() calls (fallback strings that should be real keys)
 *
 * Exclusions:
 * - Files in gen/ (auto-generated)
 * - Files in i18n/ (i18n infrastructure)
 * - Comments
 * - Import paths, type annotations
 * - Empty strings, single chars, pure numbers
 * - Strings already wrapped in t()
 * - CSS class names, variable names
 * - console.log/debug statements
 */

import { readFileSync, readdirSync, statSync } from 'fs';
import { join, extname, relative } from 'path';

const ROOT = join(process.cwd(), 'frontend', 'src');
const EXCLUDE_DIRS = new Set([
  'gen', 'i18n', 'node_modules', '.git', 'dist',
]);
const EXCLUDE_FILES = new Set([
  'index.css', 'App.css',
]);

// Patterns that are definitely not user-visible text
const NON_TEXT_PATTERNS = [
  /^[a-z][a-zA-Z0-9_-]*$/,  // css class / identifier like "flex", "container"
  /^[.#]?[\w-]+$/,           // css selector
  /^[\d.]+$/,                // pure number
  /^.$/,                     // single char
  /^https?:\/\//,            // URL
  /^\/[\w/-]*$/,             // path
  /^var\(/,                  // CSS var
  /^#[0-9a-fA-F]{3,8}$/,     // color hex
  /^@[\w-]+$/,               // CSS at-rule
  /^data-/,                  // data attribute
  /^[a-z-]+$/,               // kebab-case identifier (likely CSS/class)
];

// JSX attributes that contain user-visible text
const TEXT_ATTRIBUTES = new Set([
  'placeholder', 'title', 'label', 'tooltip', 'description',
  'okText', 'cancelText', 'confirmText', 'emptyText',
  'notFoundContent', 'suffix', 'prefix', 'alt',
  'header', 'footer', 'extra',
]);

// Function calls that take user-visible strings
const TEXT_FUNCTIONS = [
  { pattern: /message\.(success|error|warning|info|loading)\s*\(/g, name: 'message' },
  { pattern: /notification\.(success|error|warning|info|open)\s*\(/g, name: 'notification' },
  { pattern: /Modal\.(confirm|info|success|error|warning)\s*\(/g, name: 'Modal' },
  { pattern: /\.throwError\s*\(/g, name: 'throwError' },
];

// Strings to skip even if they look like text
const SKIP_STRINGS = new Set([
  '', ' ', '-', '|', '/', '·', '...', '---',
]);

function shouldSkip(str) {
  const trimmed = str.trim();
  if (SKIP_STRINGS.has(trimmed)) return true;
  if (trimmed.length === 0) return true;
  if (trimmed.length === 1) return true;
  for (const pat of NON_TEXT_PATTERNS) {
    if (pat.test(trimmed)) return true;
  }
  // Skip if it looks like a CSS value (contains only css-like tokens)
  if (/^[\d.]+(px|rem|em|vh|vw|%|s|ms|deg)$/.test(trimmed)) return true;
  // Skip if it's purely technical (contains only underscores, dots, slashes)
  if (/^[\w./_-]+$/.test(trimmed) && !trimmed.includes(' ')) return true;
  return false;
}

function scanDir(dir) {
  const results = [];
  const entries = readdirSync(dir);
  for (const entry of entries) {
    const fullPath = join(dir, entry);
    const stat = statSync(fullPath);
    if (stat.isDirectory()) {
      if (!EXCLUDE_DIRS.has(entry)) {
        results.push(...scanDir(fullPath));
      }
    } else if (stat.isFile()) {
      const ext = extname(entry);
      if ((ext === '.tsx' || ext === '.ts') && !EXCLUDE_FILES.has(entry)) {
        results.push(...scanFile(fullPath));
      }
    }
  }
  return results;
}

function scanFile(filePath) {
  const content = readFileSync(filePath, 'utf-8');
  const lines = content.split('\n');
  const relPath = relative(join(process.cwd(), 'frontend', 'src'), filePath);
  const findings = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const lineNum = i + 1;

    // Skip comments
    if (line.trim().startsWith('//') || line.trim().startsWith('*') || line.trim().startsWith('/*')) continue;
    // Skip import lines
    if (line.trim().startsWith('import ')) continue;
    // Skip type definitions
    if (line.trim().startsWith('type ') || line.trim().startsWith('interface ')) continue;

    // Pattern 1: JSX text children - <Tag>text</Tag> or >text<
    // Match text between > and < that isn't a variable or expression
    const jsxTextMatches = [...line.matchAll(/>\s*([A-Z][\w\s,.!?'":;()&/+-]+)\s*</g)];
    for (const m of jsxTextMatches) {
      const text = m[1].trim();
      if (shouldSkip(text)) continue;
      // Skip if it looks like it's already in a t() call on the same line
      if (line.includes('t(') && line.includes(text)) continue;
      findings.push({
        file: relPath,
        line: lineNum,
        type: 'JSX children',
        text,
        context: line.trim().substring(0, 120),
      });
    }

    // Pattern 2: JSX attributes with string literals
    for (const attr of TEXT_ATTRIBUTES) {
      const attrPattern = new RegExp(`\\b${attr}\\s*=\\s*"([^"]+)"`, 'g');
      const attrMatches = [...line.matchAll(attrPattern)];
      for (const m of attrMatches) {
        const text = m[1].trim();
        if (shouldSkip(text)) continue;
        // Skip if it's a t() expression in a template literal
        if (text.includes('${')) continue;
        findings.push({
          file: relPath,
          line: lineNum,
          type: `JSX attr: ${attr}`,
          text,
          context: line.trim().substring(0, 120),
        });
      }
    }

    // Pattern 3: message.* / notification.* / Modal.* with string literals (not t())
    for (const fn of TEXT_FUNCTIONS) {
      fn.pattern.lastIndex = 0;
      const fnMatches = [...line.matchAll(fn.pattern)];
      for (const m of fnMatches) {
        // Look for string literal in the arguments (same line or next few lines)
        // Check if t() is used in this line
        if (line.includes('t(') || line.includes('t(`')) continue;
        // Find string literals after the function call
        const afterCall = line.substring(m.index + m[0].length);
        const strMatch = afterCall.match(/["'`]([^"'`]{3,})["'`]/);
        if (strMatch) {
          const text = strMatch[1].trim();
          if (shouldSkip(text)) continue;
          findings.push({
            file: relPath,
            line: lineNum,
            type: `${fn.name} call`,
            text,
            context: line.trim().substring(0, 120),
          });
        }
      }
    }

    // Pattern 4: defaultValue in t() calls - indicates missing i18n key
    const dvMatches = [...line.matchAll(/t\([^,]+,\s*\{\s*defaultValue:\s*['"`]([^'"`]+)['"`]/g)];
    for (const m of dvMatches) {
      const text = m[1].trim();
      if (shouldSkip(text)) continue;
      findings.push({
        file: relPath,
        line: lineNum,
        type: 't() defaultValue (should be real key)',
        text,
        context: line.trim().substring(0, 120),
      });
    }

    // Pattern 5: throw new Error('user visible message')
    const throwMatches = [...line.matchAll(/throw\s+new\s+Error\(\s*["'`]([^"'`]+)["'`]\s*\)/g)];
    for (const m of throwMatches) {
      const text = m[1].trim();
      if (shouldSkip(text)) continue;
      findings.push({
        file: relPath,
        line: lineNum,
        type: 'throw Error',
        text,
        context: line.trim().substring(0, 120),
      });
    }
  }

  return findings;
}

// Run scan
const allFindings = scanDir(ROOT);

// Group by file
const byFile = {};
for (const f of allFindings) {
  if (!byFile[f.file]) byFile[f.file] = [];
  byFile[f.file].push(f);
}

// Print report
const files = Object.keys(byFile).sort();
let total = 0;
console.log('=== i18n Hardcoded String Scan Report ===\n');
console.log(`Files scanned: all .tsx/.ts under frontend/src/ (excluding gen/, i18n/)\n`);
console.log(`Files with findings: ${files.length}\n`);

for (const file of files) {
  const items = byFile[file];
  console.log(`\n--- ${file} (${items.length} findings) ---`);
  for (const item of items) {
    console.log(`  L${item.line} [${item.type}] "${item.text}"`);
    console.log(`    ${item.context}`);
    total++;
  }
}

console.log(`\n=== Total: ${total} findings across ${files.length} files ===`);

// Also output as JSON for programmatic processing
import { writeFileSync } from 'fs';
const reportPath = join(process.cwd(), 'scripts', 'i18n-scan-report.json');
writeFileSync(reportPath, JSON.stringify(allFindings, null, 2));
console.log(`\nJSON report saved to: ${reportPath}`);
