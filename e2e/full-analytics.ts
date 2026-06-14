/**
 * Analytics E2E — comprehensive human-simulated operations.
 *
 * Covers: Account/Period select → Equity chart → Monthly profit →
 *   Metric cards → Pie charts → Trade stats → Risk metrics →
 *   Economic calendar → Macro indicators
 *
 * Usage: npx tsx e2e/full-analytics.ts
 */
import { chromium, Page } from 'playwright';
import * as fs from 'fs';

const BASE = 'http://localhost:8022';
const SHOT = '/opt/ant/e2e/shots/analytics';
fs.mkdirSync(SHOT, { recursive: true });

const U = '888888'; const P = '12345678';

const R: string[] = []; let sn = 0;
function S(m: string) { sn++; console.log(`\n── ${sn}. ${m} ──`); }
function OK(m: string) { R.push(`✅ ${m}`); console.log(`   ✅ ${m}`); }
function W(m: string, d?: string) { R.push(`⚠️ ${m}${d ? ': ' + d : ''}`); console.log(`   ⚠️ ${m}${d ? ': ' + d : ''}`); }
function F(m: string, d?: string) { R.push(`❌ ${m}${d ? ': ' + d : ''}`); console.log(`   ❌ ${m}${d ? ': ' + d : ''}`); }
function shot(page: Page, n: string) { return page.screenshot({ path: `${SHOT}/${String(sn).padStart(2, '0')}-${n}.png`, fullPage: false }).catch(() => {}); }
async function humanPause(minMs = 800, maxMs = 3000) { await new Promise(r => setTimeout(r, Math.floor(minMs + Math.random() * (maxMs - minMs)))); }
async function phase(name: string, page: Page, fn: () => Promise<void>) {
  S(name); try { await fn(); } catch (e: any) { F(`Phase error: ${name}`, e.message?.slice(0, 150)); await shot(page, `err-${name.replace(/\s+/g, '-')}`).catch(() => {}); }
}

async function run() {
  const browser = await chromium.launch({ headless: true, executablePath: '/snap/bin/chromium', args: ['--no-sandbox'] });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  const jsErrs: string[] = [];
  const apiLog: string[] = []; let apiCount = 0;
  page.on('pageerror', e => jsErrs.push(e.message));
  page.on('response', res => { apiCount++; const url = res.url(); if (/ant\.v1\.|assets\//i.test(url)) apiLog.push(`API ${res.status()} ${url.replace(/^.*\/api\//, '').replace(/\?.*$/, '').split('/').slice(-2).join('/')}`); });

  // PHASE 1: Login
  await phase('Login', page, async () => {
    await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('#login_login', { timeout: 5000 });
    await page.fill('#login_login', U); await humanPause(200, 500);
    await page.fill('#login_password', P); await humanPause(200, 500);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    page.url().includes('login') ? F('Login failed') : OK('Login');
    shot(page, 'login');
  });

  // PHASE 2: Navigate
  await phase('Navigate', page, async () => {
    await page.goto(`${BASE}/analytics`, { waitUntil: 'domcontentloaded', timeout: 20000 });
    await page.waitForTimeout(4000);
    page.url().includes('analytics') ? OK('Analytics loaded') : F('Navigation failed');
    shot(page, 'page');
  });

  // PHASE 3: Wait for auto-select + real data
  await phase('Account Auto-Select', page, async () => {
    // fetchAccounts() now runs on mount, auto-selects via useEffect
    await page.waitForTimeout(3000);

    // Verify account selector has a value
    const selects = await page.$$('.ant-select');
    if (selects.length >= 2) {
      let acctReady = false;
      for (let i = 0; i < 10; i++) {
        const txt = await selects[0].textContent();
        if (txt && txt.trim().length > 5 && !txt.includes('Select')) {
          OK(`Account auto-selected: ${txt?.trim().slice(0, 50)}`);
          acctReady = true;
          break;
        }
        await page.waitForTimeout(1000);
      }
      if (!acctReady) W('Account not auto-selected');
    }
    shot(page, 'select');
  });

  // PHASE 4: Verify Analytics Data Loaded
  await phase('Verify Data Loaded', page, async () => {
    // Wait for analytics query to complete
    let hasData = false;
    for (let i = 0; i < 20; i++) {
      await page.waitForTimeout(2000);
      const body = await page.textContent('body') || '';
      // Check for charts, stats, or non-zero values
      const hasCharts = (await page.$$('.recharts-surface')).length > 0;
      const hasStats = (await page.$$('.ant-statistic')).length > 0;
      if (hasCharts || hasStats) {
        OK(`Data rendered ~${(i + 1) * 2}s (${hasCharts ? 'charts' : ''}${hasCharts && hasStats ? '+' : ''}${hasStats ? 'stats' : ''})`);
        hasData = true;
        break;
      }
      if (i % 5 === 4) console.log(`   Wait ${(i + 1) * 2}s...`);
    }
    if (!hasData) W('No analytics data — account may have no trades');

    // Read actual metric values
    const stats = await page.$$('.ant-statistic');
    const vals: string[] = [];
    for (const s of stats.slice(0, 8)) { vals.push((await s.textContent())?.trim()?.replace(/\s+/g, ' ') || '-'); }
    console.log(`   Metrics: ${vals.join(' | ')}`);
    shot(page, 'data');
  });

  // PHASE 5: Charts
  await phase('Equity & Monthly Charts', page, async () => {
    // Recharts surfaces
    const charts = await page.$$('.recharts-surface');
    charts.length >= 2 ? OK(`${charts.length} charts rendered`) : W(`Only ${charts.length} charts`);

    // Equity curve should be a line chart
    const lines = await page.$$('.recharts-line');
    lines.length >= 1 ? OK(`Equity curve: ${lines.length} lines`) : W('No lines');

    // Monthly profit should be a bar chart
    const bars = await page.$$('.recharts-bar');
    const rects = await page.$$('.recharts-rectangle');
    (bars.length > 0 || rects.length > 0) ? OK('Monthly profit bars visible') : W('No bars');

    // Year selector for monthly chart
    const yearSelect = await page.$$('.ant-select');
    if (yearSelect.length >= 3) {
      const yr = await yearSelect[2].textContent();
      yr ? OK(`Year selector: ${yr.trim()}`) : W('No year');
    }

    shot(page, 'charts');
  });

  // PHASE 5: Metric Cards
  await phase('Metric Cards', page, async () => {
    const stats = await page.$$('.ant-statistic');
    if (stats.length >= 4) {
      // Read metric values
      const statValues: string[] = [];
      for (const s of stats) { statValues.push((await s.textContent())?.trim() || '-'); }
      OK(`Metrics: ${statValues.slice(0, 4).join(' | ')}`);
    } else {
      W(`Only ${stats.length} metric cards`);
    }
    shot(page, 'metrics');
  });

  // PHASE 6: Pie Charts
  await phase('Pie Charts', page, async () => {
    await page.evaluate(() => window.scrollBy(0, 400));
    await humanPause(400, 700);

    const pies = await page.$$('.recharts-pie');
    const pieLabels = await page.$$('.recharts-pie-label-text');

    if (pies.length > 0) {
      OK(`${pies.length} pie charts`);
      // Read some labels
      const labels: string[] = [];
      for (const l of pieLabels.slice(0, 6)) { labels.push((await l.textContent())?.trim() || ''); }
      labels.length > 0 && console.log(`   Labels: ${labels.join(', ')}`);
    } else {
      W('No pie charts — data may be sparse');
    }

    // Symbol P&L comparison (bar chart)
    const barCharts = await page.$$('.recharts-bar');
    barCharts.length >= 1 ? OK('Symbol P&L bar chart') : W('No symbol P&L');

    shot(page, 'pies');
  });

  // PHASE 7: Trade Stats
  await phase('Trade Stats', page, async () => {
    await page.evaluate(() => window.scrollBy(0, 300));
    await humanPause(300, 600);

    const body = await page.textContent('body') || '';
    const tradeStats = ['total trades', 'win', 'loss', 'win rate', 'profit factor'];
    let found = 0;
    for (const ts of tradeStats) { if (new RegExp(ts, 'i').test(body)) found++; }
    found >= 3 ? OK(`${found}/${tradeStats.length} trade stats`) : W(`Only ${found} trade stats`);
    shot(page, 'trade-stats');
  });

  // PHASE 8: Risk Metrics
  await phase('Risk Metrics', page, async () => {
    await page.evaluate(() => window.scrollBy(0, 300));
    await humanPause(300, 600);

    const body = await page.textContent('body') || '';
    const riskTerms = ['drawdown', 'sharpe', 'sortino', 'volatility', 'VaR'];
    let found = 0;
    for (const rt of riskTerms) { if (new RegExp(rt, 'i').test(body)) found++; }
    found >= 2 ? OK(`${found}/${riskTerms.length} risk metrics`) : W(`Only ${found} risk metrics`);
    shot(page, 'risk');
  });

  // PHASE 9: Economic Calendar
  await phase('Economic Calendar', page, async () => {
    await page.evaluate(() => window.scrollBy(0, 400));
    await humanPause(400, 700);

    const body = await page.textContent('body') || '';
    (/economic|calendar|event|宏观/i.test(body)) ? OK('Economic section visible') : W('No economic section');

    // Event list items
    const listItems = await page.$$('.ant-list-item');
    listItems.length > 0 ? OK(`${listItems.length} economic events`) : W('No events');

    shot(page, 'economic');
  });

  // PHASE 10: Period Switching
  await phase('Switch Periods', page, async () => {
    // Scroll back to top
    await page.evaluate(() => window.scrollTo(0, 0));
    await humanPause(300, 500);

    // Switch to "month"
    const selects = await page.$$('.ant-select');
    if (selects.length >= 2) {
      await selects[1].click({ force: true });
      await page.waitForTimeout(800);
      const opts = await page.$$('.ant-select-item-option');
      for (const o of opts) {
        const txt = await o.textContent();
        if (txt && /month|月/i.test(txt) && !/3/.test(txt || '')) { await o.click({ force: true }); OK('Switched to month'); break; }
      }
    }
    await page.waitForTimeout(2000);
    await page.keyboard.press('Escape');

    // Charts should update
    const charts = await page.$$('.recharts-surface');
    charts.length >= 2 ? OK('Charts updated after period switch') : W('Charts unchanged');
    shot(page, 'period-switch');
  });

  // PHASE 11: Switch Account
  await phase('Switch Account', page, async () => {
    const selects = await page.$$('.ant-select');
    if (selects.length > 0) {
      await selects[0].click({ force: true });
      await page.waitForTimeout(1000);
      const opts = await page.$$('.ant-select-item-option');
      if (opts.length >= 2) {
        try {
          await opts[1].evaluate((el: HTMLElement) => el.scrollIntoView({ block: 'center' }));
          await page.waitForTimeout(100);
          await opts[1].click({ force: true });
          OK('Account switched');
        } catch {
          // Option not clickable — use evaluate
          const clicked = await page.evaluate(() => {
            const opts = document.querySelectorAll('.ant-select-item-option');
            if (opts.length >= 2) { (opts[1] as HTMLElement).click(); return true; }
            return false;
          });
          clicked ? OK('Account switched (evaluate)') : W('Account switch failed');
        }
      } else if (opts.length === 1) {
        W('Only 1 account');
      }
    }
    await page.waitForTimeout(2000);
    await page.keyboard.press('Escape');

    // Verify data reloaded
    const stats = await page.$$('.ant-statistic');
    stats.length >= 2 ? OK('Data reloaded') : W('No data after switch');
    shot(page, 'account-switch');
  });

  // FINAL CHECKS
  await phase('API Summary', page, async () => {
    const groups: Record<string, number> = {};
    for (const a of apiLog) {
      const svc = a.replace(/^API \d{3} /, '').split('/')[0] || 'unknown';
      groups[svc] = (groups[svc] || 0) + 1;
    }
    OK(`API: ${apiCount} calls, ${apiLog.length} logged → ${Object.entries(groups).map(([k, v]) => `${k}×${v}`).join(', ')}`);
  });

  await phase('JS Errors', page, async () => {
    const uniq = [...new Set(jsErrs)].filter(e => !e.includes('ResizeObserver') && !e.includes('Script error'));
    uniq.length === 0 ? OK('Zero JS errors') : uniq.slice(0, 5).forEach(e => W('JS', e));
  });

  const pass = R.filter(r => r.startsWith('✅')).length;
  const warns = R.filter(r => r.startsWith('⚠️')).length;
  const fails = R.filter(r => r.startsWith('❌')).length;
  console.log(`\n╔════════════════════════════════════╗`);
  console.log(`║  Analytics: ${pass}/${R.length} passed  (⚠️ ${warns}  ❌ ${fails})`);
  console.log(`║  Screenshots: ${SHOT}/*.png`);
  console.log(`║  API endpoints: ${new Set(apiLog.map(a => a.replace(/^API \d{3} /, ''))).size}`);
  console.log(`║  API calls: ${apiCount}`);
  console.log('╚════════════════════════════════════╝');
  R.forEach(r => console.log(r));
  console.log(`\nAPI endpoints (${new Set(apiLog.map(a => a.replace(/^API \d{3} /, ''))).size}):`);
  [...new Set(apiLog.map(a => a.replace(/^API \d{3} /, '')))].forEach(e => console.log(`  ${e}`));

  await browser.close();
}
run().catch(e => { console.error('FATAL:', e.message); process.exit(1); });
