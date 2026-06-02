/**
 * Full observation v3: Login → find account card and click → observe detail page.
 */
import { chromium } from 'playwright';
import * as fs from 'fs';

const BASE = 'http://localhost:8022';
const SHOT_DIR = '/opt/ant/e2e/shots';
fs.mkdirSync(SHOT_DIR, { recursive: true });

async function main() {
  const browser = await chromium.launch({
    headless: true,
    executablePath: '/snap/bin/chromium',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await ctx.newPage();

  const timeline: string[] = [];
  const t = () => new Date().toISOString().slice(11, 23);
  page.on('response', (res) => {
    const u = res.url();
    if (u.includes('ant.v1') && !u.includes('SubscribeEvents')) {
      timeline.push(`${t()} ${res.status()} ${u.split('ant.v1')[1]?.slice(0, 80)}`);
    }
  });
  page.on('pageerror', (err) => timeline.push(`${t()} PAGE_ERR ${err.message.slice(0, 200)}`));

  // ── 1. Login ──
  console.log('1️⃣  Login');
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.fill('#login_email', 'admin@1.com');
  await page.fill('#login_password', '12345678');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);
  console.log(`   URL: ${page.url()}`);

  // ── 2. Find all clickable elements ──
  console.log('2️⃣  Finding account cards...');

  // Look for elements containing MT account numbers (ticket-like 6-12 digit numbers)
  const elements = await page.$$('div, a, button, span, tr, td');
  let foundTicket = '';
  for (const el of elements) {
    const text = (await el.textContent()) || '';
    // Match a known ticket from the dashboard
    const match = text.match(/\b(95172262|80057439)\b/);
    if (match && text.length < 60) {
      foundTicket = match[1];
      console.log(`   Found element with ticket ${foundTicket}: <${await el.evaluate(e => e.tagName)}> "${text.trim().slice(0, 50)}"`);
      // Click the parent card/row
      const parent = await el.evaluateHandle((e) => {
        let p = e.parentElement;
        while (p && p.tagName !== 'BODY') {
          if (p.className && (p.className.includes('card') || p.className.includes('row') || p.className.includes('item') || p.className.includes('account'))) return p;
          p = p.parentElement;
        }
        return e;
      });
      const parentEl = parent.asElement();
      if (parentEl) {
        await parentEl.click();
        break;
      }
    }
  }

  if (!foundTicket) {
    console.log('   ❌ Could not find any ticket numbers');
    // Try clicking anything that looks like an account card
    const cards = await page.$$('[class*="account"], [class*="card"], [class*="AccountCard"]');
    console.log(`   Found ${cards.length} potential account card elements`);
    if (cards.length > 0) {
      await cards[0].click();
    }
  }

  await page.waitForTimeout(3000);
  console.log(`3️⃣  URL after click: ${page.url()}`);

  if (!page.url().includes('/accounts/')) {
    console.log('   ❌ Did not navigate to account detail page');
    // Try direct navigation to known account
    console.log(`   Trying direct navigation to /accounts/${foundTicket || '95172262'}`);
    await page.goto(`${BASE}/accounts/${foundTicket || '95172262'}`, {
      waitUntil: 'domcontentloaded', timeout: 10000
    });
    await page.waitForTimeout(2000);
    console.log(`   URL: ${page.url()}`);
  }

  // ── 3. Observe over time ──
  console.log('\n4️⃣  Observing page state...\n');
  for (let i = 0; i <= 10; i++) {
    await page.waitForTimeout(1500);
    const s = await getSnapshot(page);
    const elapsed = ((i + 1) * 1.5).toFixed(1);
    console.log(`T+${elapsed.padStart(4)}s | pos=${s.posCount} ot=${s.posOpenTimes.slice(0,2).join('|').padEnd(14) || '--'.padEnd(14)} hist=${s.histCount} eq=${s.hasEquity} stats=${s.hasStats} canvas=${s.canvases} errs=${s.visibleErrors.join(';') || '-'}`);

    if (s.allGood) { console.log(`   ✅ All good at T+${elapsed}s`); break; }
  }

  // ── 4. Manual refresh ──
  console.log('\n5️⃣  Manual refresh...');
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(3000);
  const sr = await getSnapshot(page);
  console.log(`Refreshed | pos=${sr.posCount} ot=${sr.posOpenTimes.slice(0,2).join('|')} hist=${sr.histCount} eq=${sr.hasEquity} stats=${sr.hasStats}\n`);

  // ── Timeline ──
  console.log('📡 Timeline:');
  for (const line of timeline) console.log('   ' + line);

  await page.screenshot({ path: `${SHOT_DIR}/final-state.png`, fullPage: true });
  await browser.close();
}

async function getSnapshot(page: any) {
  return page.evaluate(() => {
    const body = document.body?.textContent || '';

    let posCount = 0;
    const posOpenTimes: string[] = [];
    document.querySelectorAll('table').forEach(table => {
      table.querySelectorAll('tbody tr').forEach(row => {
        const cells = row.querySelectorAll('td');
        const first = cells[0]?.textContent?.trim() || '';
        if (/^\d{6,12}$/.test(first)) {
          posCount++;
          // Check ALL cells for date-like patterns
          cells.forEach((c) => {
            const txt = (c.textContent || '').trim();
            if ((/\d{4}-\d{2}-\d{2}/.test(txt) || /\d{1,2}\/\d{1,2}\/\d{4}/.test(txt) || /\d{1,2}:\d{2}:\d{2}/.test(txt)) && posOpenTimes.length < 5) {
              posOpenTimes.push(txt);
            }
          });
        }
      });
    });

    // History tab count
    let histCount = -1;
    document.querySelectorAll('.ant-tabs-tab').forEach(t => {
      const txt = t.textContent || '';
      const m = txt.match(/\((\d+)\)/);
      if (m && /History|历史/i.test(txt)) histCount = parseInt(m[1]);
    });

    // Equity/Stats — check for chart canvas and stat text
    const hasEquity = /Equity|净值/i.test(body) && document.querySelectorAll('canvas').length > 0;
    const hasStats = /Sharpe|夏普|Win rate|胜率|Max drawdown|最大回撤|Profit factor|盈亏比/i.test(body);
    const canvases = document.querySelectorAll('canvas').length;

    // Error toasts
    const visibleErrors: string[] = [];
    document.querySelectorAll('.ant-message-notice-content').forEach(e => {
      const txt = (e.textContent || '').trim().slice(0, 150);
      if (txt) visibleErrors.push(txt);
    });

    const allGood = posCount > 0 && histCount > 0 && hasEquity && hasStats;
    return { posCount, posOpenTimes, histCount, hasEquity, hasStats, canvases, visibleErrors, allGood };
  });
}

main().catch(console.error);
