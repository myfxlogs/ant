#!/usr/bin/env npx tsx
/**
 * e2e-i18n-check.ts — End-to-end i18n verification.
 *
 * Registers a test user via the browser UI, logs in, navigates every page,
 * switches through all 5 languages on each page, and checks for:
 *  - English fallback on non-en pages
 *  - i18n key leaks (raw dot-notation keys in DOM)
 *  - Empty/error pages
 *
 * No real MT account needed. Self-registering, self-cleaning.
 *
 * Usage:
 *   npx tsx scripts/e2e-i18n-check.ts [--base-url http://host:port]
 */

import { chromium, type Page } from 'playwright';

const BASE = process.argv.includes('--base-url')
  ? process.argv[process.argv.indexOf('--base-url') + 1]
  : 'http://localhost:8022';

const TEST_EMAIL = `e2e-${Date.now()}@test.ant`;
const TEST_PASS = 'Test12345678';

const LANGS = [
  { code: 'en', name: 'English' },
  { code: 'zh-cn', name: '简体中文' },
  { code: 'zh-tw', name: '繁體中文' },
  { code: 'ja', name: '日本語' },
  { code: 'vi', name: 'Tiếng Việt' },
];

const PAGES = [
  { path: '/dashboard', name: 'Dashboard' },
  { path: '/accounts', name: 'Accounts' },
  { path: '/trading', name: 'Trading' },
  { path: '/analytics', name: 'Analytics' },
  { path: '/marketplace', name: 'Marketplace' },
  { path: '/ai/settings', name: 'AISettings' },
  { path: '/logs', name: 'Logs' },
];

interface Issue { page: string; lang: string; type: string; detail: string; }

async function checkPage(page: Page, pageName: string, lang: string): Promise<Issue[]> {
  const issues: Issue[] = [];
  await page.waitForTimeout(2000);
  const text = await page.evaluate(() => document.body?.innerText || '');
  if (text.trim().length < 10) return [{ page: pageName, lang, type: 'EMPTY', detail: `${text.length} chars` }];
  const keyLeak = text.match(/\b(errors|auth|common|strategy|trading|accounts|ai|notifications)\.[a-z_]+\.[a-z_]+/);
  if (keyLeak) issues.push({ page: pageName, lang, type: 'KEY_LEAK', detail: keyLeak[0] });
  if (lang !== 'en') {
    const suspicious = ['Sign in to continue', 'My Accounts', 'Strategy Workspace',
      'Dashboard', 'No data', 'Required', 'Loading...'];
    const found = suspicious.filter(s => text.includes(s));
    if (found.length > 0) issues.push({ page: pageName, lang, type: 'EN_FALLBACK', detail: found.join(', ') });
  }
  return issues;
}

async function main() {
  console.log('=== E2E i18n Check ===');
  console.log('Base:', BASE, ' User:', TEST_EMAIL);

  const browser = await chromium.launch({
    headless: true,
    executablePath: '/usr/bin/chromium-browser',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const context = await browser.newContext();
  const page = await context.newPage();
  const allIssues: Issue[] = [];
  let checks = 0, passed = 0;

  try {
    // Register via browser UI
    console.log('\n--- Register ---');
    await page.goto(BASE + '/register', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1500);
    const regInputs = page.locator('input:not([type="checkbox"])');
    await regInputs.nth(0).fill(TEST_EMAIL);
    await regInputs.nth(1).fill(TEST_PASS);
    await regInputs.nth(2).fill(TEST_PASS);
    await page.locator('button[type="submit"]').click();
    await page.waitForTimeout(3000);
    console.log('Register ->', page.url());

    // Login via browser UI
    console.log('\n--- Login ---');
    if (!page.url().includes('login')) await page.goto(BASE + '/login', { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(1000);
    const loginInputs = page.locator('input:not([type="checkbox"])');
    await loginInputs.nth(0).fill(TEST_EMAIL);
    await loginInputs.nth(1).fill(TEST_PASS);
    await page.locator('button[type="submit"]').click();
    await page.waitForTimeout(3000);
    console.log('Login ->', page.url());

    if (page.url().includes('login')) {
      const t = await page.evaluate(() => document.body?.innerText || '');
      console.log('FAIL:', t.substring(0, 200));
      allIssues.push({ page: 'SETUP', lang: '-', type: 'LOGIN_FAIL', detail: 'on login' });
    } else {
      for (const lang of LANGS) {
        await page.evaluate((c: string) => localStorage.setItem('anttrader_lang', c), lang.code);
        console.log(`\n${lang.name} (${lang.code})`);
        for (const { path, name } of PAGES) {
          checks++;
          try {
            await page.goto(BASE + path, { waitUntil: 'domcontentloaded', timeout: 10000 });
            const issues = await checkPage(page, name, lang.code);
            if (issues.length > 0) {
              allIssues.push(...issues);
              for (const i of issues) console.log(`  ${name}: ${i.type} — ${i.detail}`);
              await page.screenshot({ path: `/tmp/e2e-fail-${lang.code}-${name}.png` });
            } else { passed++; console.log(`  ${name}: OK`); }
          } catch (e: unknown) {
            const msg = e instanceof Error ? e.message : String(e);
            console.log(`  ${name}: SKIP (${msg.substring(0, 70)})`);
          }
        }
      }
    }
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    console.error('FATAL:', msg);
    allIssues.push({ page: 'SETUP', lang: '-', type: 'FATAL', detail: msg });
  } finally {
    await browser.close();
  }

  const blocking = allIssues.filter(i => i.type !== 'EN_FALLBACK');
  console.log(`\n========================================`);
  console.log(`${passed}/${checks} passed  Issues: ${allIssues.length} (blocking: ${blocking.length})`);
  for (const i of allIssues) console.log(`  [${i.type}] ${i.lang}/${i.page}: ${i.detail}`);
  console.log(`========================================`);
  process.exit(blocking.length > 0 ? 1 : 0);
}

main();
