/**
 * Observe the previously created account.
 */
import { chromium } from 'playwright';

const BASE = 'http://localhost:8022';
const ACCT = '77292b6f-95d7-496a-a382-a54f31b82fbd';

async function main() {
  const browser = await chromium.launch({
    headless: true,
    executablePath: '/snap/bin/chromium',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

  const t = () => new Date().toISOString().slice(11, 23);
  const log = (msg: string) => console.log(`[${t()}] ${msg}`);

  // Login
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.fill('#login_email', 'admin@1.com');
  await page.fill('#login_password', '12345678');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);

  // Navigate to account detail
  log(`Navigating to /accounts/${ACCT}`);
  await page.goto(`${BASE}/accounts/${ACCT}`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(2000);

  log('\nObserving (no manual refresh):\n');
  console.log('Time    |Pos|OpenTime|Hist|Equity|Stats|Errors');
  console.log('-'.repeat(55));

  for (let i = 0; i <= 10; i++) {
    await page.waitForTimeout(1500);
    const s = await page.evaluate(() => {
      const b = document.body?.textContent || '';
      let pos = 0;
      const ots: string[] = [];
      document.querySelectorAll('table').forEach(t => {
        t.querySelectorAll('tbody tr').forEach(r => {
          const cells = r.querySelectorAll('td');
          if (cells.length >= 7 && /^\d{6,12}$/.test(cells[0]?.textContent?.trim() || '')) {
            pos++;
            cells.forEach(c => {
              const txt = c.textContent?.trim() || '';
              if (/\d{4}[-/]\d{1,2}[-/]\d{1,2}|\d{1,2}:\d{2}:\d{2}/.test(txt) && ots.length < 3) ots.push(txt);
            });
          }
        });
      });
      let hist = -1;
      document.querySelectorAll('.ant-tabs-tab').forEach(t => {
        const m = (t.textContent || '').match(/\((\d+)\)/);
        if (m && /History|历史/i.test(t.textContent || '')) hist = parseInt(m[1]);
      });
      const hasChart = document.querySelectorAll('.recharts-surface').length > 0;
      const noEquity = /No equity curve|暂无净值曲线/i.test(b);
      const eq = hasChart && !noEquity;
      const stats = /Sharpe|夏普|Win rate|胜率/i.test(b);
      const errs: string[] = [];
      document.querySelectorAll('.ant-message-notice-content').forEach(e => {
        const t = (e.textContent || '').trim().slice(0, 100);
        if (t) errs.push(t);
      });
      return { pos, ots, hist, eq, stats, errs, allGood: pos > 0 && hist >= 0 && eq && stats };
    });

    const elapsed = ((i + 1) * 1.5).toFixed(1).padStart(4);
    const ot = s.ots.slice(0, 2).join('|').padEnd(12).slice(0, 12);
    console.log(`T+${elapsed}s | ${String(s.pos).padEnd(2)} | ${ot} | ${String(s.hist).padEnd(3)} | ${String(s.eq)}  | ${String(s.stats)}  | ${s.errs.join(';') || '-'}`);

    if (s.allGood) {
      log(`\n✅ ALL DATA LOADED at T+${elapsed}s WITHOUT manual refresh!`);
      break;
    }
  }

  // Manual refresh comparison
  log('\nManual refresh...');
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(3000);
  const sr = await page.evaluate(() => {
    const b = document.body?.textContent || '';
    let pos = 0;
    document.querySelectorAll('table').forEach(t => {
      t.querySelectorAll('tbody tr').forEach(r => {
        if (r.querySelectorAll('td').length >= 7 && /^\d{6,12}$/.test(r.querySelector('td')?.textContent?.trim() || '')) pos++;
      });
    });
    const hasChart = document.querySelectorAll('.recharts-surface').length > 0;
    return { pos, hasChart, stats: /Sharpe|夏普|Win rate|胜率/i.test(b), noEquity: /No equity|暂无净值/i.test(b) };
  });
  log(`After refresh: pos=${sr.pos} chart=${sr.hasChart} stats=${sr.stats} noEquity=${sr.noEquity}`);

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
