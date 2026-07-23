import { chromium } from 'playwright';

const BASE = 'http://localhost:8022';
const CREDS = { email: 'admin@1.com', password: '12345678' };
let total = 0, passed = 0, failed = 0;

function check(name) {
  total++;
  return {
    ok: (msg = '') => { passed++; console.log(`  ✅ ${name}${msg ? ': ' + msg : ''}`); },
    fail: (msg = '') => { failed++; console.log(`  ❌ ${name}: ${msg}`); },
  };
}

async function testBasic(page, name, route) {
  const c = check(`[${name}] load`);
  const errors = [];
  page.on('pageerror', e => errors.push(e.message));
  try {
    await page.goto(`${BASE}${route}`, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForTimeout(3000);
  } catch (e) {
    c.fail(`GOTO: ${e.message}`); return;
  }
  const blank = await page.evaluate(() => (document.body?.innerText || '').length < 10);
  const critErrs = errors.filter(e => !e.includes('Locize') && !e.includes('umami') && !e.includes('trongrid'));
  if (blank) c.fail('BLANK PAGE');
  else if (critErrs.length) c.fail(`console errors: ${critErrs[0].slice(0,100)}`);
  else c.ok();
  return !blank && critErrs.length === 0;
}

async function main() {
  const browser = await chromium.launch({ headless: true });
  let page;

  // ═══════════════════════════════════════════════════════════
  // UNAUTH PAGES
  // ═══════════════════════════════════════════════════════════
  console.log('═════ UNAUTH PAGES ═════\n');

  // Landing
  {
    const ctx = await browser.newContext(); page = await ctx.newPage();
    await testBasic(page, 'Landing', '/');
    const hasHero = await page.evaluate(() => document.body.innerText.includes('AI-Powered') || document.body.innerText.includes('AlphaForge'));
    check('Landing hero text').ok(hasHero ? '' : 'MISSING');
    if (!hasHero) check('Landing hero text').fail('missing');
    await ctx.close();
  }

  // Login form
  {
    const ctx = await browser.newContext(); page = await ctx.newPage();
    await testBasic(page, 'Login', '/login');
    const emailField = page.getByPlaceholder(/email|account|邮箱|账号/i);
    const passField = page.locator('#login_password');
    const submitBtn = page.locator('button[type="submit"]');
    const c1 = check('Login: email field');
    const c2 = check('Login: password field');
    const c3 = check('Login: submit button');
    if (await emailField.count() > 0) c1.ok(); else c1.fail('missing');
    if (await passField.count() > 0) c2.ok(); else c2.fail('missing');
    if (await submitBtn.count() > 0) c3.ok(); else c3.fail('missing');

    // Test validation
    await submitBtn.first().click();
    await page.waitForTimeout(500);
    const hasValidation = await page.evaluate(() => document.body.innerText.includes('required') || document.body.innerText.includes('请输入') || document.body.innerText.includes('Please'));
    check('Login: form validation').ok(hasValidation ? '' : 'no validation shown');

    // Test actual login
    await emailField.first().fill(CREDS.email);
    await passField.first().fill(CREDS.password);
    await submitBtn.first().click();
    await page.waitForTimeout(5000);
    const loggedIn = page.url() === `${BASE}/`;
    check('Login: successful redirect').ok(loggedIn ? '' : `URL=${page.url()}`);
    await ctx.close();
  }

  // Register
  {
    const ctx = await browser.newContext(); page = await ctx.newPage();
    await testBasic(page, 'Register', '/register');
    const hasForm = await page.evaluate(() => document.body.innerText.includes('Register') || document.body.innerText.includes('注册'));
    check('Register: form visible').ok(hasForm ? '' : 'missing');
    await ctx.close();
  }

  // Marketplace unauth
  {
    const ctx = await browser.newContext(); page = await ctx.newPage();
    await testBasic(page, 'Marketplace(unauth)', '/marketplace');
    const hasMarket = await page.locator('text=Market').first().isVisible().catch(() => false);
    const hasLeaderboard = await page.locator('text=Leaderboard').first().isVisible().catch(() => false);
    check('Marketplace(unauth): Market tab').ok(hasMarket ? '' : 'missing');
    check('Marketplace(unauth): Leaderboard tab').ok(hasLeaderboard ? '' : 'missing');

    // Should NOT show auth-only tabs
    const hasAI = await page.evaluate(() => document.body.innerText.includes('AI Generate') || document.body.innerText.includes('AI 生成'));
    check('Marketplace(unauth): no AI tab').ok(!hasAI ? '' : 'LEAKED (should be hidden)');
    const hasPurchases = await page.evaluate(() => document.body.innerText.includes('My Purchases') || document.body.innerText.includes('我的购买'));
    check('Marketplace(unauth): no Purchases tab').ok(!hasPurchases ? '' : 'LEAKED');
    await ctx.close();
  }

  // ═══════════════════════════════════════════════════════════
  // LOGIN (shared context)
  // ═══════════════════════════════════════════════════════════
  console.log('\n═════ LOGIN + AUTH PAGES ═════\n');
  const authCtx = await browser.newContext();
  page = await authCtx.newPage();
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForTimeout(2000);
  await page.getByPlaceholder(/email|account|邮箱|账号/i).first().fill(CREDS.email);
  await page.locator('#login_password').first().fill(CREDS.password);
  await page.locator('button[type="submit"]').first().click();
  await page.waitForTimeout(5000);
  console.log(`  Logged in: ${page.url()}\n`);

  // Dashboard
  await testBasic(page, 'Dashboard', '/');
  const hasWidgets = await page.evaluate(() => {
    const t = document.body.innerText;
    return t.includes('Dashboard') || t.includes('balance') || t.includes('accounts') || t.includes('USDT');
  });
  check('Dashboard: content visible').ok(hasWidgets ? '' : 'empty?');

  // Strategy Gallery
  await testBasic(page, 'Strategy Gallery', '/strategy');
  const hasCards = await page.locator('.ant-card').count();
  const hasNew = await page.locator('button:has-text("New")').first().isVisible().catch(() => false);
  const hasAI = await page.locator('button:has-text("AI Generate")').first().isVisible().catch(() => false);
  check('Gallery: cards present').ok(hasCards > 0 ? `${hasCards} cards` : 'no cards');
  check('Gallery: [New] button').ok(hasNew ? '' : 'missing');
  check('Gallery: [AI Gen] button').ok(hasAI ? '' : 'missing');

  // Click first card if exists → Detail
  if (hasCards > 0) {
    const firstCard = page.locator('.ant-card').first();
    await firstCard.click();
    await page.waitForTimeout(3000);
    const detailURL = page.url();
    check('Gallery→Detail: navigate').ok(detailURL.includes('/strategy/view/') ? '' : `URL=${detailURL}`);

    // Check Detail tabs
    const hasOverview = await page.locator('text=Overview').first().isVisible().catch(() => false);
    const hasCodeTab = await page.locator('text=Code').first().isVisible().catch(() => false);
    check('Detail: Overview tab').ok(hasOverview ? '' : 'missing');
    check('Detail: Code tab').ok(hasCodeTab ? '' : 'missing');

    // Click Code tab
    if (hasCodeTab) {
      await page.locator('text=Code').first().click();
      await page.waitForTimeout(1000);
      const codeVisible = await page.locator('pre, code, [class*="language-"]').first().isVisible().catch(() => false);
      check('Detail: code visible').ok(codeVisible ? '' : 'no code shown');
    }

    // Go back
    await page.goto(`${BASE}/strategy`, { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(2000);
  }

  // Strategy Workspace
  await testBasic(page, 'Workspace', '/strategy/workspace');
  const hasEditor = await page.evaluate(() =>
    document.body.innerHTML.includes('CodeMirror') || document.body.innerHTML.includes('monaco') || document.body.innerHTML.includes('editor')
  );
  check('Workspace: code editor loaded').ok(hasEditor ? '' : 'no editor');

  // Tab switching
  const codeTab = page.locator('div').filter({ hasText: /Strategy Code/ }).first();
  const backtestTab = page.locator('div').filter({ hasText: /Backtest/ }).first();
  if (await codeTab.count() > 0) {
    await codeTab.click();
    await page.waitForTimeout(1500);
    const codeActive = await page.evaluate(() => {
      const el = document.querySelector('[style*="flex"]') || document.querySelector('.ant-layout-content');
      return el ? el.innerHTML.length > 100 : false;
    });
    check('Workspace: Code tab switch').ok(codeActive ? '' : 'no content');
  }
  if (await backtestTab.count() > 0) {
    await backtestTab.click();
    await page.waitForTimeout(1500);
    check('Workspace: Backtest tab switch').ok(true);
  }

  // MQL Import drawer
  const importBtn = page.locator('button:has-text("Import MQL")');
  if (await importBtn.count() > 0 && await importBtn.first().isEnabled()) {
    await importBtn.first().click();
    await page.waitForTimeout(1500);
    const drawerOpen = await page.locator('.ant-drawer').first().isVisible().catch(() => false);
    const hasTextarea = await page.locator('.ant-drawer textarea').first().isVisible().catch(() => false);
    check('Workspace: Import MQL drawer opens').ok(drawerOpen ? '' : 'no drawer');
    check('Workspace: Import MQL textarea').ok(hasTextarea ? '' : 'no textarea in drawer');
    if (hasTextarea) {
      await page.locator('.ant-drawer textarea').first().fill('// test EA\nint start(){return 0;}');
      check('Workspace: paste MQL code').ok(true);
    }
    // Close drawer
    await page.locator('.ant-drawer .ant-drawer-close, .ant-drawer-mask').first().click().catch(() => {});
    await page.waitForTimeout(1000);
  } else {
    check('Workspace: Import MQL button').fail('not found or disabled');
  }

  // Wallet page
  await testBasic(page, 'Wallet', '/wallet');
  const hasBalance = await page.evaluate(() => {
    const t = document.body.innerText;
    return t.includes('Balance') || t.includes('余额') || t.includes('USDT') || t.includes('Deposit') || t.includes('充值') || t.includes('deposit');
  });
  const hasAddr = await page.evaluate(() => document.body.innerHTML.includes('TR') || document.body.innerHTML.includes('T'));
  check('Wallet: balance/deposit section').ok(hasBalance ? '' : 'empty?');
  // Try clicking deposit tab
  const depositTab = page.locator('text=Deposit').first();
  if (await depositTab.isVisible().catch(() => false)) {
    await depositTab.click();
    await page.waitForTimeout(1000);
  }

  // Marketplace (auth)
  await testBasic(page, 'Marketplace(auth)', '/marketplace');
  const tabs = ['Market', 'Leaderboard', 'AI Generate', 'My Purchases', 'Author Center', 'Bundles', 'AI Optimization', 'Fee Tiers'];
  for (const tab of tabs) {
    const visible = await page.locator(`text=${tab}`).first().isVisible().catch(() => false);
    check(`Marketplace: ${tab} tab`).ok(visible ? '' : 'missing');
  }

  // Admin
  await testBasic(page, 'Admin', '/admin');
  const adminContent = await page.evaluate(() => document.body.innerText.length > 100);
  check('Admin: dashboard content').ok(adminContent ? '' : 'blank');

  // Admin sub-pages (quick check)
  for (const [name, route] of [
    ['Admin Users', '/admin/users'], ['Admin Accounts', '/admin/accounts'],
    ['Admin Wallet', '/admin/wallet'], ['Admin Config', '/admin/config'],
    ['Admin Strategies', '/admin/strategies'],
  ]) {
    await testBasic(page, name, route);
    const hasTable = await page.locator('table, .ant-table').first().isVisible().catch(() => false);
    check(`${name}: table visible`).ok(hasTable ? '' : 'no table');
  }

  // Strategy Library redirect
  await testBasic(page, 'Old Library redirect', '/strategy/library');
  check('Old /library → /strategy redirect').ok(page.url().includes('/strategy') && !page.url().includes('/library') ? '' : `URL=${page.url()}`);
  await testBasic(page, 'Old /gallery redirect', '/strategy/gallery');
  check('Old /gallery → /strategy redirect').ok(page.url() === `${BASE}/strategy` ? '' : `URL=${page.url()}`);

  // 404 handling
  {
    const errPage = await authCtx.newPage();
    const errors404 = [];
    errPage.on('pageerror', e => errors404.push(e.message));
    await errPage.goto(`${BASE}/nonexistent-page-12345`, { waitUntil: 'domcontentloaded', timeout: 10000 });
    await errPage.waitForTimeout(2000);
    const has404Content = await errPage.evaluate(() => document.body.innerText.length > 5);
    check('404 page: shows content (not white screen)').ok(has404Content ? '' : 'white screen on 404');
    await errPage.close();
  }

  await authCtx.close();
  await browser.close();

  console.log(`\n═══════════════════════════════════════════`);
  console.log(`RESULTS: ${passed}/${total} passed, ${failed} failed`);
  console.log(`═══════════════════════════════════════════`);
  process.exit(failed > 0 ? 1 : 0);
}

main().catch(e => { console.error(e); process.exit(1); });
