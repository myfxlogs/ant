#!/usr/bin/env npx tsx
/**
 * e2e-functional-check.ts — Full page functionality verification.
 *
 * Logs in with real credentials, visits every registered page, and checks:
 *  1. Page loads without crash (no blank body, no "Unexpected Error" fallback)
 *  2. No React error boundary triggered
 *  3. No console.error during navigation
 *  4. Key semantic elements present (headings, tables, cards, forms)
 *  5. Screenshot captured for manual visual review
 *
 * Credentials: read from env vars ANTTEST_EMAIL / ANTTEST_PASS.
 *
 * Usage:
 *   ANTTEST_EMAIL=admin@1.com ANTTEST_PASS=12345678 npx tsx scripts/e2e-functional-check.ts
 *   npx tsx scripts/e2e-functional-check.ts --base-url http://localhost:8022
 *   npx tsx scripts/e2e-functional-check.ts --screenshots /tmp/e2e-screens
 *   npx tsx scripts/e2e-functional-check.ts --slow-mo 300
 */

import { chromium } from 'playwright';
import * as fs from 'fs';
import * as path from 'path';
import { MAIN_PAGES, ADMIN_PAGES, checkPage, type PageResult } from './lib/e2e-page-check';

// ── Config ──────────────────────────────────────────────────────────
const BASE = process.argv.includes('--base-url')
  ? process.argv[process.argv.indexOf('--base-url') + 1] : 'http://localhost:8022';
const SCREENSHOTS_DIR = process.argv.includes('--screenshots')
  ? process.argv[process.argv.indexOf('--screenshots') + 1] : '/tmp/e2e-screenshots';
const SLOW_MO = process.argv.includes('--slow-mo')
  ? parseInt(process.argv[process.argv.indexOf('--slow-mo') + 1], 10) : 100;
const EMAIL = process.env.ANTTEST_EMAIL || '';
const PASS = process.env.ANTTEST_PASS || '';

if (!EMAIL || !PASS) {
  console.error('ERROR: Set ANTTEST_EMAIL and ANTTEST_PASS environment variables.');
  console.error('  ANTTEST_EMAIL=admin@1.com ANTTEST_PASS=12345678 npx tsx scripts/e2e-functional-check.ts');
  process.exit(1);
}

async function main() {
  console.log('═══════════════════════════════════════');
  console.log('  Ant E2E Functional Check');
  console.log('═══════════════════════════════════════');
  console.log(`Base:    ${BASE}`);
  console.log(`User:    ${EMAIL}`);
  console.log(`Shots:   ${SCREENSHOTS_DIR}`);
  console.log(`SlowMo:  ${SLOW_MO}ms\n`);

  fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true });

  const browser = await chromium.launch({
    headless: true,
    executablePath: '/usr/bin/chromium-browser',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
    slowMo: SLOW_MO,
  });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'en-US' });
  const page = await context.newPage();
  const results: PageResult[] = [];
  const startTime = Date.now();

  try {
    // ── Login ──
    console.log('[1/3] Logging in...');
    await page.goto(BASE + '/login', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1500);

    const loginInputs = page.locator('input:not([type="checkbox"])');
    const inputCount = await loginInputs.count();
    console.log(`  Found ${inputCount} inputs on login form`);

    if (inputCount >= 2) {
      await loginInputs.nth(0).fill(EMAIL);
      await loginInputs.nth(1).fill(PASS);
      await page.locator('button[type="submit"]').click();
      await page.waitForTimeout(4000);
    } else {
      console.error('  Login form not found!');
      await page.screenshot({ path: path.join(SCREENSHOTS_DIR, 'DEBUG-login-form.png') });
      const bodyText = await page.evaluate(() => document.body?.innerText || '');
      console.error('  Body:', bodyText.substring(0, 300));
    }

    const loginUrl = page.url();
    console.log(`  Login result: ${loginUrl}`);
    if (loginUrl.includes('/login')) {
      console.error('  LOGIN FAILED — aborting.');
      const t = await page.evaluate(() => document.body?.innerText || '');
      await page.screenshot({ path: path.join(SCREENSHOTS_DIR, 'DEBUG-login-fail.png') });
      results.push({ page: 'LOGIN', status: 'fail', issues: [{ page: 'LOGIN', type: 'NAV_FAIL', detail: `Still on login page: ${t.substring(0, 150)}` }], url: loginUrl });
    } else {
      results.push({ page: 'LOGIN', status: 'ok', issues: [], url: loginUrl });
      console.log('  ✅ Logged in successfully');

      for (const pages of [MAIN_PAGES, ADMIN_PAGES]) {
        const label = pages === MAIN_PAGES ? '[2/3]' : '[3/3]';
        console.log(`\n${label} Checking ${pages.length} ${pages === MAIN_PAGES ? 'main' : 'admin'} pages...`);
        for (const def of pages) {
          process.stdout.write(`  ${def.name}... `);
          const r = await checkPage(page, BASE, def, SCREENSHOTS_DIR);
          results.push(r);
          const icon = r.status === 'ok' ? '✅' : r.status === 'warn' ? '⚠️' : '❌';
          console.log(`${icon} (${r.issues.length} issues)`);
          for (const i of r.issues) console.log(`      [${i.type}] ${i.detail}`);
        }
      }
    }
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    console.error('FATAL:', msg);
    results.push({ page: 'FATAL', status: 'fail', issues: [{ page: 'FATAL', type: 'NAV_FAIL', detail: msg }], url: '' });
    await page.screenshot({ path: path.join(SCREENSHOTS_DIR, 'DEBUG-fatal.png') });
  } finally {
    await browser.close();
  }

  // ── Report ──
  const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
  const ok = results.filter(r => r.status === 'ok').length;
  const warn = results.filter(r => r.status === 'warn').length;
  const fail = results.filter(r => r.status === 'fail').length;
  const totalIssues = results.reduce((acc, r) => acc + r.issues.length, 0);

  console.log('');
  console.log('═══════════════════════════════════════');
  console.log('  Summary');
  console.log('═══════════════════════════════════════');
  console.log(`  Total pages: ${results.length}  Time: ${elapsed}s`);
  console.log(`  ✅ OK:    ${ok}\n  ⚠️  Warn:  ${warn}\n  ❌ Fail:  ${fail}`);
  console.log(`  Issues:   ${totalIssues}`);
  if (fail > 0) {
    console.log('\n  FAILURES:');
    for (const r of results.filter(r => r.status === 'fail')) {
      console.log(`    ${r.page}: ${r.issues.map(i => i.detail).join(' | ')}`);
    }
  }
  console.log(`\n  Screenshots: ${SCREENSHOTS_DIR}/`);
  console.log('═══════════════════════════════════════');
  process.exit(fail > 0 ? 1 : 0);
}

main();
