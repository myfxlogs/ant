/**
 * Full E2E: Bind MT5 account → observe Account Detail.
 */
import { chromium } from 'playwright';
import * as fs from 'fs';

const BASE = 'http://localhost:8022';
const SHOT = '/opt/ant/e2e/shots';
fs.mkdirSync(SHOT, { recursive: true });

const ACC = {
  broker: 'Exness',
  company: 'Exness Technologies Ltd',
  server: 'Exness-MT5Trial5',
  login: '277259925',
  password: 'HavEr7901$',
};

async function main() {
  const browser = await chromium.launch({
    headless: true,
    executablePath: '/snap/bin/chromium',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

  const t = () => new Date().toISOString().slice(11, 23);
  const log = (msg: string) => console.log(`[${t()}] ${msg}`);
  const tl: string[] = [];
  page.on('response', (r) => {
    const u = r.url();
    if (u.includes('ant.v1') && !u.includes('SubscribeEvents')) {
      tl.push(`${t()} ${r.status()} ${u.split('ant.v1')[1]?.slice(0, 100)}`);
    }
  });
  page.on('pageerror', (e) => tl.push(`${t()} PAGE_ERR ${e.message.slice(0, 200)}`));
  page.on('console', (msg) => {
    if (msg.type() === 'error') tl.push(`${t()} CONSOLE ${msg.text().slice(0, 200)}`);
  });

  // ═══ 1. Login ═══
  log('1. Login');
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.fill('#login_email', 'admin@1.com');
  await page.fill('#login_password', '12345678');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);
  log(`   URL: ${page.url()}`);

  // ═══ 2. Navigate to bind page ═══
  log('2. Go to bind page');
  await page.goto(`${BASE}/accounts/bind`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(1500);

  // ═══ 3. Step 1: Select platform + search broker ═══
  log('3. Step 1 — Select MT5');
  // Click MT5 card (second platform card)
  const platformCards = await page.$$('.flex.gap-4 > div');
  if (platformCards.length >= 2) {
    await platformCards[1].click(); // MT5 is the second card
    await page.waitForTimeout(300);
    log('   Clicked MT5');
  }

  // Type broker name
  const brokerInput = await page.$('input[placeholder*="broker"], input[placeholder*="Broker"]');
  if (brokerInput) {
    await brokerInput.fill(ACC.broker);
    log(`   Filled broker: ${ACC.broker}`);
  }

  // Click Search
  const searchBtn = await page.$('button:has-text("Search")');
  if (searchBtn) {
    await searchBtn.click();
    log('   Clicked Search');
    await page.waitForTimeout(2000); // wait for API response
  }

  // Select company — click antd Select, then pick option
  log('4. Select company');
  const companySelect = await page.$('.ant-select');
  if (companySelect) {
    await companySelect.click();
    await page.waitForTimeout(500);
    // In the dropdown, find and click company option
    const opts = await page.$$('.ant-select-item-option-content');
    for (const opt of opts) {
      const txt = await opt.textContent();
      if (txt?.includes(ACC.company)) {
        await opt.click();
        log(`   Selected: ${ACC.company}`);
        break;
      }
    }
    await page.waitForTimeout(800);
  }

  // Select server — now 2 antd Selects exist, click the second one
  log('5. Select server');
  const allSelects = await page.$$('.ant-select');
  log(`   Found ${allSelects.length} selects`);
  if (allSelects.length >= 2) {
    await allSelects[1].click();
    await page.waitForTimeout(800);
    const opts = await page.$$('.ant-select-item-option-content');
    for (const opt of opts) {
      const txt = await opt.textContent();
      if (txt?.includes(ACC.server)) {
        await opt.click();
        log(`   Selected: ${ACC.server}`);
        break;
      }
    }
    await page.waitForTimeout(500);
  }

  // Click Next
  log('6. Click Next → Step 2');
  const nextBtn = await page.$('button:has-text("Next")');
  if (nextBtn) {
    const isDisabled = await nextBtn.evaluate((e) => (e as HTMLButtonElement).disabled);
    log(`   Next button disabled: ${isDisabled}`);
    if (!isDisabled) {
      await nextBtn.click();
      await page.waitForTimeout(500);
    }
  }
  await page.screenshot({ path: `${SHOT}/bind-step2.png`, fullPage: true });

  // ═══ 7. Step 2: Enter credentials ═══
  log('7. Step 2 — Enter credentials');
  // Trading account input
  const accountInput = await page.$('input[placeholder*="account"], input[placeholder*="Account"], input[placeholder*="账号"]');
  if (accountInput) {
    await accountInput.fill(ACC.login);
    log(`   Filled account: ${ACC.login}`);
  }
  // Password input — use the input that's NOT the broker search (which is gone now)
  const textInputs = await page.$$('input[type="text"]');
  for (const inp of textInputs) {
    const ph = await inp.getAttribute('placeholder');
    log(`   Input: placeholder="${ph}"`);
  }
  const pwdInput = await page.$('input[placeholder*="password"], input[placeholder*="Password"]');
  if (pwdInput) {
    await pwdInput.fill(ACC.password);
    log('   Filled password');
  }

  // Click Next → Step 3
  const nextBtn2 = await page.$('button:has-text("Next")');
  if (nextBtn2) {
    await nextBtn2.click();
    await page.waitForTimeout(500);
    log('   Clicked Next → Step 3');
  }
  await page.screenshot({ path: `${SHOT}/bind-step3.png`, fullPage: true });

  // ═══ 8. Step 3: Verify & Confirm ═══
  log('8. Step 3 — Verify account');
  // Click "Verify account" button
  const verifyBtn = await page.$('button:has-text("Verify")');
  if (verifyBtn) {
    log('   Clicking Verify...');
    await verifyBtn.click();
    await page.waitForTimeout(5000); // wait for MT connection
  }

  // Check verification result
  let bodyText = await page.textContent('body');
  log(`   Page contains "Account verified": ${bodyText?.includes('Account verified')}`);
  log(`   Page contains "验证通过": ${bodyText?.includes('验证通过')}`);

  // If there's an error, log it
  const errorDiv = await page.$('[style*="E53935"], .ant-alert-error');
  if (errorDiv) {
    const errText = await errorDiv.textContent();
    log(`   ❌ Error: ${errText}`);
  }

  await page.screenshot({ path: `${SHOT}/bind-step3-after-verify.png`, fullPage: true });

  // Click "Confirm bind" if available
  const confirmBtn = await page.$('button:has-text("Confirm")');
  if (confirmBtn) {
    log('   Clicking Confirm bind...');
    await confirmBtn.click();
    // Wait for ConnectAccount + navigate to detail page
    await page.waitForTimeout(8000);
  } else {
    log('   Confirm button not found. Page state:');
    log(bodyText?.slice(0, 500) || 'EMPTY');
  }

  log(`   URL after bind: ${page.url()}`);

  // ═══ 9. Observe Account Detail ═══
  if (page.url().includes('/accounts/')) {
    log('\n9. Observing Account Detail over time:\n');
    console.log('Time    |Pos|OpenTime   |Hist|Equity|Stats|Canvas|Errors');

    for (let i = 0; i <= 12; i++) {
      await page.waitForTimeout(1500);
      const s = await getSnapshot(page);
      const elapsed = ((i + 1) * 1.5).toFixed(1).padStart(4);
      const ot = s.posOpenTimes.slice(0, 2).join('|').padEnd(10) || '--'.padEnd(10);
      console.log(`T+${elapsed}s | ${String(s.posCount).padEnd(2)} |${ot}| ${String(s.histCount).padEnd(3)} | ${String(s.hasEquity).padEnd(5)} | ${String(s.hasStats).padEnd(5)} | ${String(s.canvases).padEnd(5)} | ${s.visibleErrors.join(';') || '-'}`);

      if (s.allGood) {
        log(`\n✅ ALL DATA LOADED at T+${elapsed}s — no manual refresh needed!`);
        await page.screenshot({ path: `${SHOT}/detail-loaded.png`, fullPage: true });
        break;
      }
    }

    // Manual refresh comparison
    log('\n10. Manual refresh...');
    await page.reload({ waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(3000);
    const sr = await getSnapshot(page);
    log(`After refresh | pos=${sr.posCount} hist=${sr.histCount} equity=${sr.hasEquity} stats=${sr.hasStats} canvas=${sr.canvases}`);
  }

  log('\n📡 RPC Timeline:');
  for (const line of tl) console.log('   ' + line);
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
        if (cells.length >= 7 && /^\d{6,12}$/.test(cells[0]?.textContent?.trim() || '')) {
          posCount++;
          cells.forEach(c => {
            const txt = (c.textContent || '').trim();
            if ((/\d{4}[-/]\d{1,2}[-/]\d{1,2}/.test(txt) || /\d{1,2}:\d{2}:\d{2}/.test(txt)) && posOpenTimes.length < 5)
              posOpenTimes.push(txt);
          });
        }
      });
    });

    let histCount = -1;
    document.querySelectorAll('.ant-tabs-tab').forEach(t => {
      const txt = t.textContent || '';
      const m = txt.match(/\((\d+)\)/);
      if (m && /History|历史/i.test(txt)) histCount = parseInt(m[1]);
    });

    const canvases = document.querySelectorAll('canvas').length;
    const hasEquity = canvases > 0;
    const hasStats = /Sharpe|夏普|Win rate|胜率|Max drawdown|最大回撤|Profit factor|盈亏比/i.test(body);
    const visibleErrors: string[] = [];
    document.querySelectorAll('.ant-message-notice-content').forEach(e => {
      const txt = (e.textContent || '').trim().slice(0, 150);
      if (txt) visibleErrors.push(txt);
    });

    const allGood = posCount > 0 && histCount > 0 && hasEquity && hasStats;
    return { posCount, posOpenTimes, histCount, hasEquity, hasStats, canvases, visibleErrors, allGood };
  });
}

main().catch(e => { console.error(e); process.exit(1); });
