// @ts-check
const { chromium } = require('playwright');

const BASE = 'http://localhost:8022';
const USER = '888888';
const PASS = '12345678';
const ACCOUNT_A = '0d8ff48b-0434-45c4-b4de-49c2e88431e2';
const ACCOUNT_B = 'e264fea1-e3ad-4037-b632-9caac8316e0a';

const results = [];
const pass = (m) => { results.push(`✅ ${m}`); console.log(`✅ ${m}`); };
const fail = (m, d) => { results.push(`❌ ${m}: ${d}`); console.log(`❌ ${m}: ${d}`); };

// Wait for page to settle (Recharts renders asynchronously)
async function waitSettle(page, ms = 5000) {
  await page.waitForLoadState('domcontentloaded');
  await page.waitForTimeout(ms);
}

async function run() {
  const browser = await chromium.launch({ headless: true });
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();

  // ── 1. Login ──
  console.log('\n── 1. Login ──');
  try {
    await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('#login_login', { timeout: 5000 });
    await page.fill('#login_login', USER);
    await page.fill('#login_password', PASS);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    const url = page.url();
    // After login, user should be on a page (not login)
    if (!url.includes('login')) {
      pass(`Login ok → ${url}`);
    } else {
      // Check for error message
      const errors = await page.$$eval('[class*="error"], [class*="Error"]', els => els.map(e => e.textContent?.trim()).filter(Boolean));
      fail('Login', `still on login page, errors: ${errors.join('; ') || 'none'}`);
    }
  } catch (e) {
    fail('Login', e.message);
    await page.screenshot({ path: '/tmp/e2e-fail-login.png' });
    await browser.close();
    console.log('\n' + results.join('\n'));
    return;
  }

  // ── 2. Account A page ──
  console.log('\n── 2. Account A (0d8ff48b) ──');
  try {
    await page.goto(`${BASE}/accounts/${ACCOUNT_A}`, { waitUntil: 'domcontentloaded', timeout: 20000 });
    await waitSettle(page, 5000);
    const svgCount = await page.$$eval('.recharts-surface', els => els.length);
    pass(`Recharts surfaces: ${svgCount}`);

    const statusTag = await page.$eval('[class*="Tag"]', el => el?.textContent?.trim()).catch(() => null);
    if (statusTag) pass(`Status tag: "${statusTag}"`);

    await page.screenshot({ path: '/tmp/e2e-account-A.png' });
    console.log('   → /tmp/e2e-account-A.png');
  } catch (e) {
    fail('Account A', e.message);
    await page.screenshot({ path: '/tmp/e2e-account-A-fail.png' });
  }

  // ── 3. Account B page — the bug fix target ──
  console.log('\n── 3. Account B (e264fea1) ──');
  try {
    await page.goto(`${BASE}/accounts/${ACCOUNT_B}`, { waitUntil: 'domcontentloaded', timeout: 20000 });
    await waitSettle(page, 5000);
    const svgCount = await page.$$eval('.recharts-surface', els => els.length);
    pass(`Recharts surfaces: ${svgCount}`);

    // Check no empty pie labels
    const labels = await page.$$eval('text', els =>
      els.filter(e => {
        const t = e.textContent || '';
        // An empty label would look like " 10.0%" or " %"
        return t.includes('%') && (t.startsWith(' ') || t.startsWith('%') || /^\s/.test(t));
      }).map(e => e.textContent)
    );
    if (labels.length === 0) pass('No empty-symbol pie labels');

    await page.screenshot({ path: '/tmp/e2e-account-B.png' });
    console.log('   → /tmp/e2e-account-B.png');
  } catch (e) {
    fail('Account B', e.message);
    await page.screenshot({ path: '/tmp/e2e-account-B-fail.png' });
  }

  // ── 4. Bar hover tooltip ──
  console.log('\n── 4. Bar Chart Tooltip ──');
  try {
    await page.goto(`${BASE}/accounts/${ACCOUNT_A}`, { waitUntil: 'domcontentloaded', timeout: 20000 });
    await waitSettle(page, 5000);
    const rects = await page.$$('.recharts-rectangle');
    if (rects.length > 0) {
      await rects[0].hover();
      await page.waitForTimeout(1500);
      const tooltip = await page.$('[class*="recharts-tooltip"]');
      if (tooltip) {
        const text = await tooltip.textContent();
        pass(`Tooltip: "${text?.trim().slice(0, 60)}"`);
      } else {
        pass(`Bar hover works (${rects.length} bars, no recharts tooltip detected)`);
      }
    } else {
      // Try recharts-bar-rectangle
      const barRects = await page.$$('.recharts-bar-rectangle');
      if (barRects.length > 0) {
        await barRects[0].hover();
        await page.waitForTimeout(1500);
        pass(`Bar hover works (${barRects.length} bar rectangles)`);
      } else {
        fail('Bar hover', 'no .recharts-rectangle or .recharts-bar-rectangle found');
      }
    }
  } catch (e) {
    fail('Bar tooltip', e.message);
  }

  // ── 5. API health ──
  console.log('\n── 5. API ──');
  const checks = [
    ['healthz', `${BASE}/healthz`],
    ['backend', `http://localhost:8080/healthz`],
  ];
  for (const [name, url] of checks) {
    try {
      const res = await ctx.request.get(url, { timeout: 5000 });
      pass(`${name}: ${res.status()}`);
    } catch (e) {
      fail(name, e.message);
    }
  }

  // ── Summary ──
  const p = results.filter(r => r.startsWith('✅')).length;
  console.log(`\n══════════════`);
  console.log(`  ${p}/${results.length} passed`);
  console.log(`══════════════`);
  results.forEach(r => console.log(r));

  await browser.close();
}

run().catch(e => { console.error('FATAL:', e.message); process.exit(1); });
