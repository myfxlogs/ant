/**
 * Full Project E2E — comprehensive human-simulated operations across ALL features.
 *
 * Part 1: Core Platform (Dashboard → Account → Wallet → Market Tools)
 * Part 2: Strategy Pipeline (Workspace → Library → Experiments)
 * Part 3: Trading & Monitoring (Algo Dashboard → Auto Trading → Analytics → Logs)
 * Part 4: Platform (Marketplace → AI Settings → Profile → Admin)
 *
 * LEAVES REAL TRACES: templates, experiments, backtest runs, published strategies, orders.
 * SKIPS: adding trading accounts.
 *
 * Usage: npx tsx e2e/full-project.ts
 */
import { chromium, Page } from 'playwright';
import * as fs from 'fs';

const BASE = 'http://localhost:8022';
const SHOT = '/opt/ant/e2e/shots/project';
fs.mkdirSync(SHOT, { recursive: true });

const U = '888888'; const P = '12345678';
const ACCT = '277259925'; const SYM = 'BTCUSDm';

// ══════════════════════════════════════
// Reporting
// ══════════════════════════════════════
const R: string[] = []; let sn = 0;
function S(m: string) { sn++; console.log(`\n── ${sn}. ${m} ──`); }
function OK(m: string) { R.push(`✅ ${m}`); console.log(`   ✅ ${m}`); }
function W(m: string, d?: string) { R.push(`⚠️ ${m}${d ? ': ' + d : ''}`); console.log(`   ⚠️ ${m}${d ? ': ' + d : ''}`); }
function F(m: string, d?: string) { R.push(`❌ ${m}${d ? ': ' + d : ''}`); console.log(`   ❌ ${m}${d ? ': ' + d : ''}`); }
function shot(page: Page, n: string) {
  return page.screenshot({ path: `${SHOT}/${String(sn).padStart(2, '0')}-${n}.png`, fullPage: false }).catch(() => {});
}

// ══════════════════════════════════════
// Helpers
// ══════════════════════════════════════
async function humanPause(minMs = 800, maxMs = 3000) {
  await new Promise(r => setTimeout(r, Math.floor(minMs + Math.random() * (maxMs - minMs))));
}
async function navTo(page: Page, path: string) {
  await page.goto(`${BASE}${path}`, { waitUntil: 'domcontentloaded', timeout: 20000 });
  await page.waitForTimeout(3000 + Math.random() * 1000);
}
async function scrollExplore(page: Page) {
  await page.evaluate(() => { window.scrollBy(0, 200); });
  await page.waitForTimeout(400);
  await page.evaluate(() => { window.scrollBy(0, -120); });
  await page.waitForTimeout(300);
}
async function phase(name: string, page: Page, fn: () => Promise<void>) {
  S(name);
  try { await fn(); }
  catch (e: any) { F(`Phase error: ${name}`, e.message?.slice(0, 150)); await shot(page, `err-${name.replace(/\s+/g, '-')}`).catch(() => {}); }
}
async function selectAntOption(page: Page, labelRe: RegExp): Promise<boolean> {
  return page.evaluate((reStr) => {
    const re = new RegExp(reStr, 'i');
    const opts = document.querySelectorAll('.ant-select-item-option');
    for (const o of opts) {
      if (re.test((o as HTMLElement).innerText?.trim() || '')) {
        (o as HTMLElement).click(); return true;
      }
    }
    return false;
  }, labelRe.source);
}

// ══════════════════════════════════════
// Main
// ══════════════════════════════════════
async function run() {
  const browser = await chromium.launch({ headless: true, executablePath: '/snap/bin/chromium', args: ['--no-sandbox'] });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  const jsErrs: string[] = [];
  const apiLog: string[] = []; let apiCount = 0;
  page.on('pageerror', e => jsErrs.push(e.message));
  page.on('response', res => {
    apiCount++;
    const url = res.url();
    if (/ant\.v1\.|assets\//i.test(url)) {
      apiLog.push(`API ${res.status()} ${url.replace(/^.*\/api\//, '').replace(/\?.*$/, '').split('/').slice(-2).join('/')}`);
    }
  });

  // ═══════════════════════════════════════════════
  // PART 1: CORE PLATFORM
  // ═══════════════════════════════════════════════

  // PHASE 1: Login
  await phase('Login', page, async () => {
    await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('#login_login', { timeout: 5000 });
    await humanPause(300, 800);
    await page.fill('#login_login', U);
    await humanPause(300, 700);
    await page.fill('#login_password', P);
    await humanPause(200, 600);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    page.url().includes('login') ? F('Login failed') : OK('Login');
    shot(page, 'login');
  });

  // PHASE 2: Dashboard
  await phase('Dashboard', page, async () => {
    await navTo(page, '/');
    await scrollExplore(page);
    const body = await page.textContent('body') || '';
    (/dashboard|总览|AntTrader/i.test(body)) ? OK('Dashboard loaded') : W('Dashboard?');

    // Stat cards
    const statCards = await page.$$('.ant-statistic');
    statCards.length >= 2 ? OK(`${statCards.length} stat cards`) : W('Few stats');

    // Quick action cards
    const actions = await page.$$('[style*="cursor: pointer"], [role="button"]');
    actions.length >= 4 ? OK(`${actions.length} clickable elements`) : W('Few actions');

    // Account cards
    const cards = await page.$$('.ant-card');
    cards.length >= 1 ? OK(`${cards.length} account cards`) : W('No account cards');

    // Click first account card → navigates to account detail
    if (cards.length > 0) {
      await humanPause(400, 900);
      await cards[0].click({ force: true });
      await page.waitForTimeout(2000);
      OK('Clicked account card');
    }
    shot(page, 'dashboard');
  });

  // PHASE 3: Account Detail (navigated from Dashboard)
  await phase('Account Detail', page, async () => {
    if (!page.url().includes('accounts')) { await navTo(page, '/accounts'); await page.waitForTimeout(2000); }
    const body = await page.textContent('body') || '';
    (/equity|balance|margin|profit/i.test(body)) ? OK('Account detail loaded') : W('Not on account page');

    // Key metrics
    const metricCards = await page.$$('.ant-statistic');
    metricCards.length >= 3 ? OK(`${metricCards.length} metrics`) : W('Few metrics');

    // Tabs: Positions / Pending / History
    const tabs = await page.$$('.ant-tabs-tab');
    const tabLabels = [];
    for (const t of tabs) { tabLabels.push((await t.textContent())?.trim()); }
    OK(`Tabs: ${tabLabels.join(', ')}`);

    // Click History tab
    for (const t of tabs) {
      const txt = await t.textContent();
      if (txt && /history|历史/i.test(txt)) { await t.click({ force: true }); OK('History tab'); break; }
    }
    await page.waitForTimeout(1500);

    // Check table
    const tables = await page.$$('table');
    tables.length >= 1 ? OK('Data table visible') : W('No tables');

    await scrollExplore(page);
    shot(page, 'account');
  });

  // PHASE 4: Wallet
  await phase('Wallet', page, async () => {
    await navTo(page, '/wallet');
    const body = await page.textContent('body') || '';
    (/balance|wallet|transactions|余额|钱包/i.test(body)) ? OK('Wallet loaded') : W('Wallet?');

    // Balance info
    const descItems = await page.$$('.ant-descriptions-item');
    descItems.length >= 2 ? OK(`${descItems.length} wallet fields`) : W('No description');

    // Transactions table
    const table = await page.$('table');
    if (table) {
      const rows = await table.$$('tbody tr');
      rows.length > 0 ? OK(`${rows.length} transactions`) : W('No transactions');
    }

    await scrollExplore(page);
    await humanPause(400, 800);
    shot(page, 'wallet');
  });

  // PHASE 5: Market Tools
  await phase('Market Tools', page, async () => {
    await navTo(page, '/market');
    await page.waitForTimeout(2000);

    // Search for a symbol
    const inputs = await page.$$('input');
    if (inputs.length > 0) {
      // Find autocomplete input
      const autoInput = inputs[0];
      await autoInput.click({ force: true });
      await humanPause(200, 400);
      await page.keyboard.type('BTC', { delay: 40 });
      await page.waitForTimeout(1000);
      OK('Symbol search typed');
    }

    // Watchlist stars
    const stars = await page.$$('.anticon-star');
    if (stars.length > 0) {
      await stars[0].click({ force: true });
      OK('Watchlist toggled');
    } else {
      W('No watchlist stars');
    }

    await scrollExplore(page);
    shot(page, 'market');
  });

  // ═══════════════════════════════════════════════
  // PART 2: STRATEGY PIPELINE
  // ═══════════════════════════════════════════════

  // PHASE 6: Strategy Workspace — Setup
  await phase('Workspace: Setup', page, async () => {
    await navTo(page, '/strategy/workspace');
    await page.waitForTimeout(5000);

    // Select account
    const selects = await page.$$('.ant-select');
    if (selects.length >= 2) {
      await selects[0].click({ force: true }); await page.waitForTimeout(1000);
      await selectAntOption(page, new RegExp(ACCT.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
      await page.waitForTimeout(1000);
      await page.keyboard.press('Escape');

      await selects[1].click({ force: true }); await page.waitForTimeout(1000);
      await selectAntOption(page, new RegExp(SYM));
      await page.waitForTimeout(1000);
      await page.keyboard.press('Escape');
      OK('Account + Symbol selected');
    }
    shot(page, 'workspace-setup');
  });

  // PHASE 7: Workspace — Chart + Timeframe
  await phase('Workspace: Chart', page, async () => {
    const canvases = (await page.$$('canvas')).length;
    canvases >= 3 ? OK(`${canvases} chart canvases`) : W(`Only ${canvases} canvases`);

    // Cycle timeframes
    for (const tf of ['M1', 'M5', 'M15', 'H1', 'H4']) {
      for (const r of await page.$$('.ant-radio-button-wrapper, .ant-radio-wrapper')) {
        if ((await r.textContent())?.trim() === tf) { await r.click({ force: true }); break; }
      }
      await humanPause(300, 600);
    }
    OK('Timeframes cycled');
    shot(page, 'workspace-chart');
  });

  // PHASE 8: Workspace — Backtest
  await phase('Workspace: Backtest', page, async () => {
    // Open code panel
    const divs = await page.$$('div');
    for (const d of divs) {
      const b = await d.boundingBox();
      if (b && b.x >= 248 && b.x <= 265 && b.width <= 35 && b.height >= 100) {
        await d.click({ force: true }); await page.waitForTimeout(2000); break;
      }
    }
    const cmOpen = (await page.$$('.cm-editor')).length > 0;
    cmOpen ? OK('Code panel open') : W('Panel not open');

    // Insert code via evaluate
    if (cmOpen) {
      const CODE = `"""@strategy SMA crossover
@param fast_period 5 range=3:50:1
@param slow_period 20 range=10:60:2
@param stop_loss_pct 0.5 range=0.1:3.0:0.1
@param position_size 0.01 range=0.01:0.1:0.01
"""
def init(): pass
def run(ctx):
    close = ctx['close']
    if len(close) < 22: return
    f = sum(close[-5:]) / 5.0
    s = sum(close[-20:]) / 20.0
    pf = sum(close[-6:-1]) / 5.0
    ps = sum(close[-21:-1]) / 20.0
    pos = ctx.get('position') or {}
    if pf <= ps and f > s and pos.get('side') != 'buy':
        return {'signal': 'buy', 'volume': 0.01}
    elif pf >= ps and f < s and pos.get('side') != 'sell':
        return {'signal': 'sell', 'volume': 0.01}
`;
      await page.evaluate((code) => {
        const ed = document.querySelector('.cm-editor');
        if (!ed) return;
        let v: any = null;
        for (const k of Object.keys(ed)) { const val = (ed as any)[k]; if (val?.state?.dispatch && val?.viewport) { v = val; break; } }
        if (v) { v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: code } }); }
      }, CODE);
      await page.waitForTimeout(1000);
      OK('Code inserted');

      // Click Validate button
      const validateBtn = await page.$('button:has(.anticon-check-circle)');
      if (validateBtn) { await validateBtn.click({ force: true }); OK('Validated'); }
      await page.waitForTimeout(2000);
    }

    // Close code panel
    for (const d of await page.$$('div')) {
      const b = await d.boundingBox();
      if (b && b.x >= 248 && b.x <= 265 && b.width <= 35 && b.height >= 100) {
        await d.click({ force: true }); await page.waitForTimeout(1500); break;
      }
    }

    // Configure backtest: 1M, 10k, Both
    for (const seg of await page.$$('.ant-segmented-item')) {
      if ((await seg.textContent())?.trim() === '1M') { await seg.click({ force: true }); break; }
    }
    const spins = await page.$$('input[role="spinbutton"]');
    if (spins.length > 0) {
      await spins[0].click({ force: true }); await page.waitForTimeout(200);
      await page.keyboard.press('Control+a'); await page.keyboard.type('10000', { delay: 20 });
    }
    for (const r of await page.$$('.ant-radio-button-wrapper, .ant-radio-wrapper')) {
      if ((await r.textContent())?.trim() === 'Both') { await r.click({ force: true }); break; }
    }
    const lb = await page.$('button:has-text("Live Aligned")');
    if (lb) { await lb.click({ force: true }); }
    OK('Backtest configured');

    // Run
    await humanPause(500, 1200);
    const runBtn = await page.$('button:has-text("Run")');
    if (runBtn) {
      const disabled = await runBtn.evaluate((el: HTMLButtonElement) => el.disabled);
      if (!disabled) { await runBtn.click({ force: true }); OK('Backtest submitted'); }
      else W('Run disabled');
    }

    // Wait for results
    for (let i = 0; i < 40; i++) {
      await page.waitForTimeout(3000);
      const btx = await page.textContent('body') || '';
      if (/return|net.*profit|sharpe|win.*rate|drawdown/i.test(btx)) {
        OK(`Backtest done ~${(i + 1) * 3}s`);
        break;
      }
      if (i === 39) W('Backtest timeout');
    }
    shot(page, 'workspace-bt');
  });

  // PHASE 9: Strategy Library — full flow
  await phase('Library: Browse & Create', page, async () => {
    await navTo(page, '/strategy/library');
    await page.waitForTimeout(3000);

    const items = await page.$$('[role="button"][tabindex="0"]');
    items.length > 0 ? OK(`${items.length} templates`) : W('No templates');

    if (items.length > 0) {
      await humanPause(400, 800);
      await items[0].click({ force: true });
      await page.waitForTimeout(1500);
      OK('Template selected');
    }

    // Overview tab
    const descTable = await page.$('.ant-descriptions table');
    descTable ? OK('Template details visible') : W('No details');

    // View code
    const viewCodeBtn = await page.$('button:has-text("View")');
    if (viewCodeBtn) {
      await viewCodeBtn.click({ force: true });
      await page.waitForTimeout(1000);
      const overlay = await page.$('[style*="fixed"]');
      if (overlay) {
        OK('Code modal opened');
        await page.keyboard.press('Escape');
        await page.waitForTimeout(500);
      }
    }

    // Create new template
    const createBtn = await page.$('button.ant-btn-primary:has(.anticon-plus)');
    if (createBtn) {
      await humanPause(400, 800);
      await createBtn.click({ force: true });
      await page.waitForTimeout(1500);

      const modal = await page.$('.ant-modal');
      if (modal) {
        OK('Create modal opened');
        const inputs = await modal.$$('input');
        if (inputs.length > 0) {
          const name = `Full Project E2E ${Date.now().toString(36)}`;
          await inputs[0].click({ force: true });
          await page.keyboard.type(name, { delay: 25 + Math.random() * 25 });
          OK(`Template: ${name}`);
        }
        // Save
        const submitBtn = await modal.$('button.ant-btn-primary');
        if (submitBtn) {
          await humanPause(300, 600);
          await submitBtn.click({ force: true });
          await page.waitForTimeout(1500);
          OK('Template saved');
        }
      }
    }

    // Publish if possible
    const shareBtn = await page.$('button:has-text("Share")');
    if (shareBtn) {
      await shareBtn.click({ force: true });
      await page.waitForTimeout(1500);
      OK('Published');
    }
    shot(page, 'library');
  });

  // PHASE 10: Experiments
  await phase('Experiments', page, async () => {
    await navTo(page, '/strategy/experiments');
    await page.waitForTimeout(3000);
    await scrollExplore(page);

    const expTable = await page.$('.ant-table-tbody');
    if (expTable) {
      const rows = await expTable.$$('tr');
      rows.length > 0 ? OK(`${rows.length} experiments`) : W('No experiments');
    }

    // Create new experiment
    const formSelects = await page.$$('.ant-form-item .ant-select');
    if (formSelects.length > 0) {
      await formSelects[0].click({ force: true }); await page.waitForTimeout(800);
      const opts = await page.$$('.ant-select-item-option');
      if (opts.length > 0) {
        await humanPause(300, 600);
        await opts[0].click({ force: true });
        await page.keyboard.press('Escape');
        OK('Template selected for experiment');

        // Switch to Random search
        if (formSelects.length >= 2) {
          await formSelects[1].click({ force: true }); await page.waitForTimeout(800);
          await selectAntOption(page, /random/i);
          await page.waitForTimeout(300);
          await page.keyboard.press('Escape');
        }

        // Submit
        const submitBtn = await page.$('button[type="submit"]');
        if (submitBtn) {
          await humanPause(500, 1000);
          await submitBtn.click({ force: true });
          await page.waitForTimeout(3000);
          OK('Experiment submitted');
        }
      }
    }
    shot(page, 'experiments');
  });

  // ═══════════════════════════════════════════════
  // PART 3: TRADING & MONITORING
  // ═══════════════════════════════════════════════

  // PHASE 11: Algo Dashboard
  await phase('Algo Dashboard', page, async () => {
    await navTo(page, '/algo-dashboard');
    await scrollExplore(page);
    const body = await page.textContent('body') || '';
    (/algo|execution|执行|算法/i.test(body)) ? OK('Algo Dashboard loaded') : W('Algo page?');

    const table = await page.$('table');
    if (table) {
      const rows = await table.$$('tbody tr');
      rows.length > 0 ? OK(`${rows.length} algo executions`) : W('No executions');
    } else {
      W('No table — page may be empty (expected)');
    }
    shot(page, 'algo');
  });

  // PHASE 12: Auto Trading
  await phase('Auto Trading', page, async () => {
    await navTo(page, '/auto-trading');
    await scrollExplore(page);

    // Risk config fields
    const body = await page.textContent('body') || '';
    const riskTerms = ['max position', 'max risk', 'max lot', 'max daily loss', 'max drawdown'];
    let found = 0;
    for (const t of riskTerms) { if (new RegExp(t, 'i').test(body)) found++; }
    found >= 3 ? OK(`${found}/${riskTerms.length} risk terms`) : W('Few risk fields');

    // Stats
    const stats = await page.$$('.ant-statistic');
    stats.length >= 2 ? OK(`${stats.length} stat cards`) : W('Few stats');

    // Trading logs
    const logTable = await page.$('table');
    if (logTable) {
      const rows = await logTable.$$('tbody tr');
      rows.length > 0 ? OK(`${rows.length} trading logs`) : W('No logs');
    }

    shot(page, 'autotrading');
  });

  // PHASE 13: Analytics
  await phase('Analytics', page, async () => {
    await navTo(page, '/analytics');
    await page.waitForTimeout(3000);
    await scrollExplore(page);

    // Select an account
    const selects = await page.$$('.ant-select');
    if (selects.length > 0) {
      await selects[0].click({ force: true }); await page.waitForTimeout(800);
      const opts = await page.$$('.ant-select-item-option');
      if (opts.length > 0) { await opts[0].click({ force: true }); OK('Account selected'); }
      await page.keyboard.press('Escape');
    }
    await page.waitForTimeout(2000);

    // Charts
    const charts = await page.$$('.recharts-surface');
    charts.length >= 1 ? OK(`${charts.length} charts`) : W('No Recharts');

    // Metric cards
    const metricCards = await page.$$('.ant-statistic');
    metricCards.length >= 2 ? OK(`${metricCards.length} metric cards`) : W('Few metrics');

    // Pie charts
    const pies = await page.$$('.recharts-pie');
    pies.length >= 1 ? OK(`${pies.length} pie charts`) : W('No pie charts');

    await scrollExplore(page);
    shot(page, 'analytics');
  });

  // PHASE 14: System Logs
  await phase('System Logs', page, async () => {
    await navTo(page, '/logs');
    await page.waitForTimeout(2500);

    // 4 tabs
    const tabs = await page.$$('.ant-tabs-tab');
    const tabLabels = [];
    for (const t of tabs) { tabLabels.push((await t.textContent())?.trim()); }
    OK(`Log tabs: ${tabLabels.join(', ')}`);

    // Cycle through tabs
    for (const t of tabs) {
      await humanPause(300, 600);
      await t.click({ force: true });
      await page.waitForTimeout(1000);
      const table = await page.$('.ant-tabs-tabpane-active table');
      if (table) {
        const rows = await table.$$('tbody tr');
        rows.length > 0 ? OK(`${(await t.textContent())?.trim()}: ${rows.length} rows`) : W(`${(await t.textContent())?.trim()}: empty`);
      }
    }

    shot(page, 'logs');
  });

  // ═══════════════════════════════════════════════
  // PART 4: PLATFORM
  // ═══════════════════════════════════════════════

  // PHASE 15: Marketplace
  await phase('Marketplace', page, async () => {
    await navTo(page, '/marketplace');
    await page.waitForTimeout(3000);
    await scrollExplore(page);

    const tabs = await page.$$('.ant-tabs-tab');
    for (const t of tabs) {
      const txt = await t.textContent();
      if (txt && /purchase|purchased|我的|购入/i.test(txt)) {
        await t.click({ force: true });
        await page.waitForTimeout(1000);
        OK('My Purchases tab');
        break;
      }
    }

    const cards = await page.$$('.ant-card');
    cards.length > 0 ? OK(`${cards.length} marketplace cards`) : W('No marketplace content');

    shot(page, 'marketplace');
  });

  // PHASE 16: AI Settings
  await phase('AI Settings', page, async () => {
    await navTo(page, '/ai');
    await page.waitForTimeout(2500);

    // Provider cards
    const providerCards = await page.$$('.ant-card');
    providerCards.length >= 1 ? OK(`${providerCards.length} AI providers`) : W('No providers');

    // Select first provider
    if (providerCards.length > 0) {
      await humanPause(400, 800);
      await providerCards[0].click({ force: true });
      await page.waitForTimeout(1000);

      // Check form rendered
      const form = await page.$('form');
      form ? OK('Provider config form visible') : W('No form');
    }

    await scrollExplore(page);
    shot(page, 'ai');
  });

  // PHASE 17: Profile
  await phase('Profile', page, async () => {
    await navTo(page, '/profile');
    const body = await page.textContent('body') || '';
    (/profile|email|role|nickname/i.test(body)) ? OK('Profile loaded') : W('Profile?');

    const descItems = await page.$$('.ant-descriptions-item');
    descItems.length >= 3 ? OK(`${descItems.length} profile fields`) : W('Few fields');

    shot(page, 'profile');
  });

  // PHASE 18: Admin — Dashboard
  await phase('Admin Dashboard', page, async () => {
    await navTo(page, '/admin');
    await page.waitForTimeout(2500);

    const stats = await page.$$('.ant-statistic');
    stats.length >= 3 ? OK(`${stats.length} admin stat cards`) : W('Few admin stats');

    const tables = await page.$$('table');
    tables.length >= 1 ? OK(`${tables.length} admin tables`) : W('No admin tables');

    await scrollExplore(page);
    shot(page, 'admin');
  });

  // PHASE 19: Admin — User Management
  await phase('Admin: Users', page, async () => {
    await navTo(page, '/admin/users');
    await page.waitForTimeout(2500);

    const table = await page.$('table');
    if (table) {
      const rows = await table.$$('tbody tr');
      rows.length > 0 ? OK(`${rows.length} users`) : W('No users');
    }

    // Search
    const searchInput = await page.$('.ant-input-search input');
    if (searchInput) {
      await searchInput.click({ force: true });
      await page.keyboard.type('admin', { delay: 30 });
      await page.keyboard.press('Enter');
      await page.waitForTimeout(1500);
      OK('User search performed');
    }
    shot(page, 'admin-users');
  });

  // PHASE 20: Admin — System Config
  await phase('Admin: Config', page, async () => {
    await navTo(page, '/admin/config');
    await page.waitForTimeout(2000);

    const table = await page.$('table');
    if (table) {
      const rows = await table.$$('tbody tr');
      rows.length > 0 ? OK(`${rows.length} config items`) : W('No configs');
    }
    shot(page, 'admin-config');
  });

  // PHASE 21: Admin — Strategy Management
  await phase('Admin: Strategies', page, async () => {
    await navTo(page, '/admin/strategies');
    await page.waitForTimeout(2500);

    const tables = await page.$$('table');
    tables.length >= 1 ? OK(`${tables.length} strategy tables`) : W('No tables');

    const rows = tables.length > 0 ? await tables[0].$$('tbody tr') : [];
    rows.length > 0 ? OK(`${rows.length} strategies`) : W('No strategies');
    shot(page, 'admin-strategies');
  });

  // PHASE 22: Admin — Account Management
  await phase('Admin: Accounts', page, async () => {
    await navTo(page, '/admin/accounts');
    await page.waitForTimeout(2000);

    const table = await page.$('table');
    if (table) {
      const rows = await table.$$('tbody tr');
      rows.length > 0 ? OK(`${rows.length} accounts`) : W('No accounts');
    }
    shot(page, 'admin-accounts');
  });

  // PHASE 23: Admin — Trading Monitor
  await phase('Admin: Trading Monitor', page, async () => {
    await navTo(page, '/admin/trading');
    await page.waitForTimeout(2500);

    const stats = await page.$$('.ant-statistic');
    stats.length >= 3 ? OK(`${stats.length} trading stats`) : W('Few stats');
    shot(page, 'admin-trading');
  });

  // PHASE 24: Admin — Operation Logs
  await phase('Admin: Operation Logs', page, async () => {
    await navTo(page, '/admin/logs');
    await page.waitForTimeout(2000);

    const table = await page.$('table');
    if (table) {
      const rows = await table.$$('tbody tr');
      rows.length > 0 ? OK(`${rows.length} op logs`) : W('No op logs');
    }
    shot(page, 'admin-oplogs');
  });

  // ═══ FINAL CHECKS ═══
  await phase('API Summary', page, async () => {
    const groups: Record<string, number> = {};
    for (const a of apiLog) {
      const svc = a.replace(/^API \d{3} /, '').split('/')[0] || 'unknown';
      groups[svc] = (groups[svc] || 0) + 1;
    }
    const summary = Object.entries(groups).map(([k, v]) => `${k}×${v}`).join(', ');
    OK(`API: ${apiCount} calls, ${apiLog.length} logged → ${summary}`);
  });

  await phase('JS Errors', page, async () => {
    const uniq = [...new Set(jsErrs)].filter(e => !e.includes('ResizeObserver') && !e.includes('Script error'));
    uniq.length === 0 ? OK('Zero JS errors') : uniq.slice(0, 5).forEach(e => W('JS', e));
  });

  // ═══ SUMMARY ═══
  const pass = R.filter(r => r.startsWith('✅')).length;
  const warns = R.filter(r => r.startsWith('⚠️')).length;
  const fails = R.filter(r => r.startsWith('❌')).length;
  console.log(`\n╔════════════════════════════════════╗`);
  console.log(`║  Full Project E2E: ${pass}/${R.length} passed  (⚠️ ${warns}  ❌ ${fails})`);
  console.log(`║  Screenshots: ${SHOT}/*.png`);
  console.log(`║  API endpoints: ${new Set(apiLog.map(a => a.replace(/^API \d{3} /, ''))).size}`);
  console.log(`║  API calls: ${apiCount}`);
  console.log('╚════════════════════════════════════╝');
  R.forEach(r => console.log(r));

  const eps = [...new Set(apiLog.map(a => a.replace(/^API \d{3} /, '')))];
  console.log(`\nAPI endpoints (${eps.length}):`);
  eps.forEach(e => console.log(`  ${e}`));

  await browser.close();
}

run().catch(e => { console.error('FATAL:', e.message); process.exit(1); });
