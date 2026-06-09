#!/usr/bin/env npx tsx
/**
 * e2e-i18n-check.ts — End-to-end i18n verification.
 *
 * Registers a test user via API, logs in, navigates every page, switches
 * through all 5 languages on each page, and checks for translation issues.
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

async function setLang(page: Page, code: string) {
  await page.evaluate((c: string) => localStorage.setItem('anttrader_lang', c), code);
  // Use domcontentloaded — SSE streams prevent networkidle from ever resolving
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(1500);
}

async function checkPage(page: Page, pageName: string, lang: string): Promise<Issue[]> {
  const issues: Issue[] = [];
  const text = await page.evaluate(() => document.body?.innerText || '');

  if (text.trim().length < 10) {
    issues.push({ page: pageName, lang, type: 'EMPTY', detail: `${text.length} chars` });
    return issues;
  }

  const keyLeak = text.match(/\b(errors|auth|common|strategy|trading|accounts|ai|notifications)\.[a-z_]+\.[a-z_]+/);
  if (keyLeak) {
    issues.push({ page: pageName, lang, type: 'KEY_LEAK', detail: keyLeak[0] });
  }

  if (lang !== 'en') {
    const suspicious = ['Sign in to continue', 'My Accounts', 'Strategy Workspace',
      'Dashboard', 'No data', 'Required', 'Loading...', 'Notifications', 'Settings', 'Profile'];
    const found = suspicious.filter(s => text.includes(s));
    if (found.length > 0) {
      issues.push({ page: pageName, lang, type: 'EN_FALLBACK', detail: found.join(', ') });
    }
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
    // Step 1: Register via API (POST)
    console.log('\n--- Register ---');
    const registerResp = await page.evaluate(async (data) => {
      const { email, password, base } = data;
      const resp = await fetch(base + '/ant.v1.AuthService/Register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });
      return { status: resp.status, body: await resp.json() };
    }, { email: TEST_EMAIL, password: TEST_PASS, base: BASE });

    console.log('Register status:', registerResp.status);
    if (registerResp.status !== 200) {
      console.log('Response:', JSON.stringify(registerResp.body).substring(0, 200));
    }

    // Step 2: Login via API
    console.log('\n--- Login ---');
    const loginResp = await page.evaluate(async (data) => {
      const { email, password, base } = data;
      const resp = await fetch(base + '/ant.v1.AuthService/Login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });
      return { status: resp.status, body: await resp.json() };
    }, { email: TEST_EMAIL, password: TEST_PASS, base: BASE });

    console.log('Login status:', loginResp.status);
    if (loginResp.status !== 200) {
      console.log('Response:', JSON.stringify(loginResp.body).substring(0, 300));
      allIssues.push({ page: 'SETUP', lang: '-', type: 'LOGIN_FAIL', detail: 'status=' + loginResp.status });
    } else {
      // Store token
      const token = loginResp.body?.token || loginResp.body?.accessToken || '';
      if (token) {
        await page.evaluate((t: string) => localStorage.setItem('auth_token', t), token);
        console.log('Token stored');
      }

      // Navigate to trigger app auth
      await page.goto(BASE + '/dashboard', { waitUntil: 'networkidle', timeout: 15000 });
      await page.waitForTimeout(2000);

      // Step 3: Verify each page in each language
      for (const lang of LANGS) {
        console.log(`\n=== ${lang.name} (${lang.code}) ===`);

        for (const { path, name } of PAGES) {
          checks++;
          try {
            await page.goto(BASE + path, { waitUntil: 'domcontentloaded', timeout: 15000 });
            await setLang(page, lang.code);

            const issues = await checkPage(page, name, lang.code);
            if (issues.length > 0) {
              allIssues.push(...issues);
              for (const i of issues) console.log(`  ${name}: ${i.type} — ${i.detail}`);
              await page.screenshot({ path: `/tmp/e2e-fail-${lang.code}-${name}.png` });
            } else {
              passed++;
              console.log(`  ${name}: OK`);
            }
            await page.screenshot({ path: `/tmp/e2e-${lang.code}-${name}.png` });
          } catch (e: unknown) {
            const msg = e instanceof Error ? e.message : String(e);
            allIssues.push({ page: name, lang: lang.code, type: 'ERROR', detail: msg });
            console.log(`  ${name}: ERROR — ${msg}`);
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

  console.log(`\n========================================`);
  console.log(`RESULTS: ${passed}/${checks} checks passed`);
  if (allIssues.length > 0) {
    console.log(`ISSUES (${allIssues.length}):`);
    for (const i of allIssues) console.log(`  [${i.type}] ${i.lang}/${i.page}: ${i.detail}`);
  } else {
    console.log(`ALL PASS`);
  }
  console.log(`========================================`);

  process.exit(allIssues.length > 0 ? 1 : 0);
}

main();
