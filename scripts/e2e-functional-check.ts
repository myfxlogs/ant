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
 * Falls back to defaults for local dev (NOT committed to repo).
 *
 * Usage:
 *   ANTTEST_EMAIL=admin@1.com ANTTEST_PASS=12345678 npx tsx scripts/e2e-functional-check.ts
 *   npx tsx scripts/e2e-functional-check.ts --base-url http://localhost:8022
 *   npx tsx scripts/e2e-functional-check.ts --screenshots /tmp/e2e-screens
 *   npx tsx scripts/e2e-functional-check.ts --slow-mo 300  (slower for visual debugging)
 */

import { chromium, type Page, type BrowserContext } from 'playwright';
import * as fs from 'fs';
import * as path from 'path';

// ── Config ──────────────────────────────────────────────────────────
const BASE = process.argv.includes('--base-url')
  ? process.argv[process.argv.indexOf('--base-url') + 1]
  : 'http://localhost:8022';

const SCREENSHOTS_DIR = process.argv.includes('--screenshots')
  ? process.argv[process.argv.indexOf('--screenshots') + 1]
  : '/tmp/e2e-screenshots';

const SLOW_MO = process.argv.includes('--slow-mo')
  ? parseInt(process.argv[process.argv.indexOf('--slow-mo') + 1], 10)
  : 100;

const EMAIL = process.env.ANTTEST_EMAIL || '';
const PASS = process.env.ANTTEST_PASS || '';

if (!EMAIL || !PASS) {
  console.error('ERROR: Set ANTTEST_EMAIL and ANTTEST_PASS environment variables.');
  console.error('  ANTTEST_EMAIL=admin@1.com ANTTEST_PASS=12345678 npx tsx scripts/e2e-functional-check.ts');
  process.exit(1);
}

// ── Page definitions ────────────────────────────────────────────────

interface PageDef {
  path: string;
  name: string;
  /** CSS selectors that should be present if the page loaded correctly */
  semanticSelectors: string[];
  /** Optional: click this selector after load to verify interactivity */
  interactWith?: string;
  /** Pages that need an account selected first */
  needsAccount?: boolean;
}

const MAIN_PAGES: PageDef[] = [
  {
    path: '/',
    name: 'Dashboard',
    semanticSelectors: ['h1, h2, h3, h4, h5, [class*="title"]', '.ant-card', '[class*="statistic"]'],
  },
  {
    path: '/accounts/bind',
    name: 'BindAccount',
    semanticSelectors: ['.ant-card', 'input, button, [class*="form"]'],
  },
  {
    path: '/profile',
    name: 'Profile',
    semanticSelectors: ['.ant-card', '.ant-descriptions, .ant-form'],
  },
  {
    path: '/strategy/templates',
    name: 'StrategyTemplates',
    semanticSelectors: ['.ant-card', '.ant-table, .ant-list, [class*="template"]'],
  },
  {
    path: '/strategy/workspace',
    name: 'StrategyWorkspace',
    semanticSelectors: ['.ant-btn', 'input, textarea, [class*="editor"]', '[class*="toolbar"]'],
  },
  {
    path: '/strategy/assets',
    name: 'StrategyAssets',
    semanticSelectors: ['.ant-card', '.ant-table, .ant-list'],
  },
  {
    path: '/strategy/schedules',
    name: 'StrategySchedules',
    semanticSelectors: ['.ant-card', '.ant-table, .ant-list, .ant-empty'],
  },
  {
    path: '/strategy/indicator-catalog',
    name: 'IndicatorCatalog',
    semanticSelectors: ['.ant-card', '.ant-table, .ant-list, [class*="indicator"]'],
  },
  {
    path: '/strategy/analysis',
    name: 'AssetAnalysis',
    semanticSelectors: ['.ant-card', 'input', '.ant-btn'],
  },
  {
    path: '/marketplace',
    name: 'Marketplace',
    semanticSelectors: ['.ant-card', '.ant-list', '.ant-tabs', '.ant-empty', '[class*="card"]'],
  },
  {
    path: '/logs',
    name: 'Logs',
    semanticSelectors: ['.ant-table, .ant-list, .ant-card, [class*="log"]'],
  },
];

const ADMIN_PAGES: PageDef[] = [
  {
    path: '/admin',
    name: 'AdminDashboard',
    semanticSelectors: ['.ant-card', '.ant-statistic, [class*="stat"]'],
  },
  {
    path: '/admin/users',
    name: 'UserManagement',
    semanticSelectors: ['.ant-table, .ant-list, .ant-card'],
  },
  {
    path: '/admin/accounts',
    name: 'AccountManagement',
    semanticSelectors: ['.ant-table, .ant-list, .ant-card'],
  },
  {
    path: '/admin/trading',
    name: 'TradingMonitor',
    semanticSelectors: ['.ant-table, .ant-list, .ant-card'],
  },
  {
    path: '/admin/logs',
    name: 'OperationLogs',
    semanticSelectors: ['.ant-table, .ant-list, .ant-card'],
  },
  {
    path: '/admin/config',
    name: 'SystemConfig',
    semanticSelectors: ['.ant-card', '.ant-form, .ant-table, .ant-list'],
  },
  {
    path: '/admin/jurisdiction',
    name: 'JurisdictionGate',
    semanticSelectors: ['.ant-card', '.ant-table, .ant-list, .ant-form'],
  },
  {
    path: '/admin/sre/killswitch',
    name: 'SREKillSwitch',
    semanticSelectors: ['.ant-card', '.ant-switch, .ant-btn, .ant-table'],
  },
  {
    path: '/admin/sre/breakers',
    name: 'SREBreakers',
    semanticSelectors: ['.ant-card', '.ant-switch, .ant-btn, .ant-table'],
  },
  {
    path: '/admin/sre/canary',
    name: 'SRECanary',
    semanticSelectors: ['.ant-card', '.ant-form, .ant-btn'],
  },
];

// ── Helpers ─────────────────────────────────────────────────────────

interface Issue {
  page: string;
  type: 'BLANK' | 'ERROR_BOUNDARY' | 'CONSOLE_ERROR' | 'NO_SEMANTIC' | 'NAV_FAIL' | 'TIMEOUT';
  detail: string;
}

interface PageResult {
  page: string;
  status: 'ok' | 'warn' | 'fail';
  issues: Issue[];
  screenshot?: string;
  url: string;
}

async function checkPage(
  page: Page,
  def: PageDef,
  screenshotsDir: string,
): Promise<PageResult> {
  const issues: Issue[] = [];
  const consoleErrors: string[] = [];
  const onConsole = (msg: any) => {
    if (msg.type() === 'error') consoleErrors.push(msg.text());
  };
  page.on('console', onConsole);

  try {
    // Navigate
    await page.goto(BASE + def.path, { waitUntil: 'domcontentloaded', timeout: 15000 });
    // Wait for React to render
    await page.waitForTimeout(3000);

    const url = page.url();

    // If redirected to login, the page is auth-gated (expected for main pages if unauthorized)
    if (url.includes('/login')) {
      issues.push({ page: def.name, type: 'NAV_FAIL', detail: `Redirected to login — auth required or session expired. URL: ${url}` });
      page.off('console', onConsole);
      return { page: def.name, status: 'fail', issues, url };
    }

    // Check 1: body text length — blank page detection
    const bodyText = await page.evaluate(() => document.body?.innerText?.trim() || '');
    if (bodyText.length < 20) {
      // Allow pages that are legitimately sparse (e.g., empty state)
      const hasEmpty = await page.evaluate(() => {
        return !!document.querySelector('.ant-empty') ||
          !!document.querySelector('[class*="empty"]') ||
          !!document.querySelector('[class*="no-data"]');
      });
      if (!hasEmpty) {
        issues.push({ page: def.name, type: 'BLANK', detail: `Body text: ${bodyText.length} chars — "${bodyText.substring(0, 80)}"` });
      }
    }

    // Check 2: Error boundary triggered
    const errorText = await page.evaluate(() => {
      const el = document.querySelector('.ant-result-error, [class*="error-boundary"], [class*="ErrorBoundary"]');
      return el?.textContent || '';
    });
    if (errorText && (errorText.includes('Unexpected Error') || errorText.includes('Page Error'))) {
      issues.push({ page: def.name, type: 'ERROR_BOUNDARY', detail: errorText.substring(0, 120) });
    }

    // Check 3: Console errors
    // Filter out known benign errors
    const realErrors = consoleErrors.filter(e =>
      !e.includes('ResolveSession failed') &&
      !e.includes('Failed to fetch') &&
      !e.includes('favicon') &&
      !e.includes('Third-party cookie') &&
      !e.includes('[PaperAccountPanel]')
    );
    for (const err of realErrors.slice(0, 3)) {
      issues.push({ page: def.name, type: 'CONSOLE_ERROR', detail: err.substring(0, 150) });
    }

    // Check 4: Semantic elements present
    let hasSemantic = false;
    for (const sel of def.semanticSelectors) {
      try {
        const count = await page.locator(sel).count();
        if (count > 0) { hasSemantic = true; break; }
      } catch { /* selector parse error — skip */ }
    }
    if (!hasSemantic && bodyText.length >= 20) {
      issues.push({ page: def.name, type: 'NO_SEMANTIC', detail: `No semantic selectors matched: ${def.semanticSelectors.join(', ')}` });
    }

    // Screenshot
    const ssPath = path.join(screenshotsDir, `${def.name}.png`);
    await page.screenshot({ path: ssPath, fullPage: false });
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    issues.push({ page: def.name, type: 'NAV_FAIL', detail: msg.substring(0, 200) });
  }

  page.off('console', onConsole);

  const status = issues.length === 0 ? 'ok' :
    issues.some(i => ['BLANK', 'ERROR_BOUNDARY', 'NAV_FAIL'].includes(i.type)) ? 'fail' : 'warn';

  return { page: def.name, status, issues, url: page.url() };
}

// ── Main ────────────────────────────────────────────────────────────

async function main() {
  console.log('═══════════════════════════════════════');
  console.log('  Ant E2E Functional Check');
  console.log('═══════════════════════════════════════');
  console.log(`Base:    ${BASE}`);
  console.log(`User:    ${EMAIL}`);
  console.log(`Shots:   ${SCREENSHOTS_DIR}`);
  console.log(`SlowMo:  ${SLOW_MO}ms`);
  console.log('');

  // Ensure screenshots dir
  fs.mkdirSync(SCREENSHOTS_DIR, { recursive: true });

  const browser = await chromium.launch({
    headless: true,
    executablePath: '/usr/bin/chromium-browser',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
    slowMo: SLOW_MO,
  });

  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    locale: 'en-US',
  });
  const page = await context.newPage();

  const results: PageResult[] = [];
  const startTime = Date.now();

  try {
    // ── Login ──────────────────────────────────────────────────────
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
      console.error('  Login form not found! Taking debug screenshot...');
      await page.screenshot({ path: path.join(SCREENSHOTS_DIR, 'DEBUG-login-form.png') });
      const bodyText = await page.evaluate(() => document.body?.innerText || '');
      console.error('  Body:', bodyText.substring(0, 300));
    }

    const loginUrl = page.url();
    console.log(`  Login result: ${loginUrl}`);
    if (loginUrl.includes('/login')) {
      console.error('  LOGIN FAILED — aborting.');
      const bodyText = await page.evaluate(() => document.body?.innerText || '');
      console.error('  Page text:', bodyText.substring(0, 300));
      await page.screenshot({ path: path.join(SCREENSHOTS_DIR, 'DEBUG-login-fail.png') });
      results.push({ page: 'LOGIN', status: 'fail', issues: [{ page: 'LOGIN', type: 'NAV_FAIL', detail: `Still on login page: ${bodyText.substring(0, 150)}` }], url: loginUrl });
    } else {
      results.push({ page: 'LOGIN', status: 'ok', issues: [], url: loginUrl });
      console.log('  ✅ Logged in successfully');

      // ── Main pages ────────────────────────────────────────────────
      console.log(`\n[2/3] Checking ${MAIN_PAGES.length} main pages...`);
      for (const def of MAIN_PAGES) {
        process.stdout.write(`  ${def.name}... `);
        const r = await checkPage(page, def, SCREENSHOTS_DIR);
        results.push(r);
        const icon = r.status === 'ok' ? '✅' : r.status === 'warn' ? '⚠️' : '❌';
        console.log(`${icon} (${r.issues.length} issues)`);
        for (const i of r.issues) {
          console.log(`      [${i.type}] ${i.detail}`);
        }
      }

      // ── Admin pages ────────────────────────────────────────────────
      console.log(`\n[3/3] Checking ${ADMIN_PAGES.length} admin pages...`);
      for (const def of ADMIN_PAGES) {
        process.stdout.write(`  ${def.name}... `);
        const r = await checkPage(page, def, SCREENSHOTS_DIR);
        results.push(r);
        const icon = r.status === 'ok' ? '✅' : r.status === 'warn' ? '⚠️' : '❌';
        console.log(`${icon} (${r.issues.length} issues)`);
        for (const i of r.issues) {
          console.log(`      [${i.type}] ${i.detail}`);
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

  // ── Report ──────────────────────────────────────────────────────
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
  console.log(`  ✅ OK:    ${ok}`);
  console.log(`  ⚠️  Warn:  ${warn}`);
  console.log(`  ❌ Fail:  ${fail}`);
  console.log(`  Issues:   ${totalIssues}`);

  if (fail > 0) {
    console.log('');
    console.log('  FAILURES:');
    for (const r of results.filter(r => r.status === 'fail')) {
      console.log(`    ${r.page}: ${r.issues.map(i => i.detail).join(' | ')}`);
    }
  }

  console.log(`\n  Screenshots: ${SCREENSHOTS_DIR}/`);
  console.log('═══════════════════════════════════════');

  process.exit(fail > 0 ? 1 : 0);
}

main();
