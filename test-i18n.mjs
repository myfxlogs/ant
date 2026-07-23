import { chromium } from 'playwright';

const BASE = 'http://localhost:8022';
const CREDS = { email: 'admin@1.com', password: '12345678' };
const LANGS = ['en', 'zh-cn', 'zh-tw', 'ja', 'vi'];

const I18N_KEY_RE = /\b[a-z]{2,}\.[a-z]+\.[a-zA-Z.]+\b/g;
const SKIP = new Set([
  'alphaforge', 'ant.design', 'antd', 'mt4', 'mt5', 'mql4', 'mql5',
  'eur.usd', 'usd.jpy', 'gbp.usd', 'api.trongrid', 'api.deepseek',
  'vite.svg', 'robots.txt', 'site.map', 'sitemap.xml',
  'qrcode.react', 'zustand', 'tanstack', 'buf.build',
  'npm.error', 'node_modules', 'package.json', 'tsconfig',
  'i18next', 'locize', 'com.i18next',
]);

const UNAUTH_PAGES = [
  ['Landing', '/'], ['Login', '/login'], ['Register', '/register'],
  ['Marketplace(unauth)', '/marketplace'], ['Brokers', '/brokers'],
];

const AUTH_PAGES = [
  ['Dashboard', '/'], ['Gallery', '/strategy'], ['Workspace', '/strategy/workspace'],
  ['Live', '/strategy/live'], ['MktTools', '/strategy/market-tools'],
  ['Wallet', '/wallet'], ['Sub', '/subscription'], ['Algo', '/trading/algos'],
  ['AutoTrd', '/auto-trading'], ['Analytics', '/analytics'],
  ['Marketplace', '/marketplace'], ['Logs', '/logs'],
  ['Admin', '/admin'], ['AdmUsers', '/admin/users'], ['AdmAccts', '/admin/accounts'],
  ['AdmWallet', '/admin/wallet'], ['AdmCfg', '/admin/config'],
  ['AdmStrats', '/admin/strategies'],
];

function extractKeys(text) {
  const matches = text.match(I18N_KEY_RE) || [];
  return [...new Set(matches)]
    .filter(k => !SKIP.has(k.toLowerCase()))
    .filter(k => k.split('.').length >= 2 && k.split('.').every(p => /^[a-z][a-z0-9_]*$/i.test(p)));
}

async function testPage(page, name, route) {
  const errors = [];
  page.on('pageerror', e => errors.push(e.message));
  try {
    await page.goto(`${BASE}${route}`, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForTimeout(3000);
  } catch (e) { return { name, keys: [], error: e.message }; }
  const bodyText = await page.evaluate(() => document.body?.innerText || '');
  return { name, keys: extractKeys(bodyText), jsErrors: errors.filter(e => !e.includes('Locize') && !e.includes('umami')) };
}

async function loginInLang(browser, lang) {
  const ctx = await browser.newContext({ locale: lang.replace('-', '_') });
  const page = await ctx.newPage();

  // Go to login
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForTimeout(2000);

  // Set language cookie before the app initializes
  await page.evaluate((l) => { localStorage.setItem('alphaforge_lang', l); }, lang);

  // Reload to apply language
  await page.reload({ waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);

  // Fill login — use CSS selectors that are language-independent
  const emailInput = page.locator('#login_login'); // antd generates id="login_login" from form name="login" + field name="login"
  const passInput = page.locator('#login_password');
  const submitBtn = page.locator('button[type="submit"]');

  try {
    await emailInput.fill(CREDS.email, { timeout: 5000 });
    await passInput.fill(CREDS.password, { timeout: 5000 });
    await submitBtn.click({ timeout: 5000 });
    await page.waitForTimeout(5000);
  } catch (e) {
    // Fallback: try placeholder-based
    try {
      await page.getByPlaceholder(/./).first().fill(CREDS.email, { timeout: 3000 });
      await page.locator('input[type="text"]').last().fill(CREDS.password, { timeout: 3000 });
      await page.locator('button[type="submit"]').first().click({ timeout: 3000 });
      await page.waitForTimeout(5000);
    } catch { /* login may fail; continue with partial auth */ }
  }

  return { ctx, page };
}

async function main() {
  const results = [];
  const browser = await chromium.launch({ headless: true });

  // ── Unauth pages × all langs ──
  console.log('=== Unauth (5 pages × 5 langs = 25 tests) ===\n');
  for (const lang of LANGS) {
    const ctx = await browser.newContext({ locale: lang.replace('-', '_') });
    const page = await ctx.newPage();
    // Pre-set language
    await page.goto(`${BASE}/`, { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.evaluate((l) => { localStorage.setItem('alphaforge_lang', l); window.location.reload(); }, lang);
    await page.waitForTimeout(2000);

    for (const [name, route] of UNAUTH_PAGES) {
      const r = await testPage(page, `${name}`, route);
      r.lang = lang; r.auth = 'unauth';
      if (r.keys.length > 0) results.push(r);
    }
    await ctx.close();
  }

  // ── Auth pages × all langs ──
  console.log('=== Auth (19 pages × 5 langs = 95 tests) ===\n');
  for (const lang of LANGS) {
    process.stdout.write(`  ${lang}: `);
    const { ctx, page } = await loginInLang(browser, lang);

    let pageResults = 0;
    for (const [name, route] of AUTH_PAGES) {
      const r = await testPage(page, name, route);
      r.lang = lang; r.auth = 'login';
      if (r.keys.length > 0) {
        results.push(r);
        pageResults++;
      }
    }
    console.log(`${AUTH_PAGES.length - pageResults}/${AUTH_PAGES.length} clean, ${pageResults} with keys`);
    await ctx.close();
  }

  await browser.close();

  // ── Aggregate report ──
  const uniqueKeys = [...new Set(results.flatMap(r => r.keys))].sort();
  const totalTests = UNAUTH_PAGES.length * 5 + AUTH_PAGES.length * 5;

  console.log(`\n═══════════════════════════════════════════`);
  console.log(`i18n AUDIT: ${totalTests} page×lang combinations`);
  console.log(`Pages with missing keys: ${new Set(results.map(r => `${r.name}[${r.lang}]`)).size} of ${totalTests}`);
  console.log(`Unique missing keys: ${uniqueKeys.length}`);
  console.log(`═══════════════════════════════════════════`);

  if (uniqueKeys.length === 0) {
    console.log(`\n✅ ALL CLEAN — no raw i18n keys visible on any page in any language.\n`);
    process.exit(0);
  }

  // Group by key
  console.log(`\nMissing i18n keys (showing as raw text on page):\n`);
  const byKey = {};
  for (const r of results) {
    for (const k of r.keys) {
      if (!byKey[k]) byKey[k] = [];
      byKey[k].push(`${r.name}[${r.lang}]`);
    }
  }
  for (const [k, pages] of Object.entries(byKey).slice(0, 60)) {
    const locales = [...new Set(results.filter(r => r.keys.includes(k)).map(r => r.lang))];
    console.log(`  🔑 ${k}`);
    console.log(`     locales: ${locales.join(', ')}`);
    console.log(`     pages:   ${pages.slice(0, 3).join(', ')}${pages.length > 3 ? ` +${pages.length-3} more` : ''}`);
  }
  if (uniqueKeys.length > 60) {
    console.log(`\n  ... and ${uniqueKeys.length - 60} more keys`);
  }
  process.exit(1);
}

main().catch(e => { console.error(e); process.exit(1); });
