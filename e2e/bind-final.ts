/**
 * Full bind flow — final working version.
 * Key findings:
 * - MT5 click must be by text content (not by index — hits step indicators)
 * - antd v5 uses .ant-select not .ant-select-selector
 * - Company selection works via keyboard (ArrowDown + Enter)
 * - Server select has 59 items in virtual list — scroll via mouse wheel
 */
import { chromium } from 'playwright';
import * as fs from 'fs';

const BASE = 'http://localhost:8022';
const SHOT = '/opt/ant/e2e/shots';
fs.mkdirSync(SHOT, { recursive: true });

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
    if (u.includes('ant.v1') && !u.includes('SubscribeEvents'))
      tl.push(`${t()} ${r.status()} ${u.split('ant.v1')[1]?.slice(0, 100)}`);
  });
  page.on('pageerror', (e) => tl.push(`${t()} ERR ${e.message.slice(0, 200)}`));

  // ═══════════════════════════════════════════
  // 1. LOGIN
  // ═══════════════════════════════════════════
  log('1. Login');
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.fill('#login_email', 'admin@1.com');
  await page.fill('#login_password', '12345678');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);

  // ═══════════════════════════════════════════
  // 2. BIND PAGE — Step 1
  // ═══════════════════════════════════════════
  log('2. Bind page');
  await page.goto(`${BASE}/accounts/bind`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(1000);

  // Click MT5 by text content (NOT by index — step indicators use same CSS)
  log('3. Click MT5');
  await page.evaluate(() => {
    const allDivs = document.querySelectorAll('.flex.gap-4 > div');
    for (const div of allDivs) {
      const h2 = div.querySelector('.text-2xl');
      if (h2 && h2.textContent?.trim() === 'MT5') {
        (div as HTMLElement).click();
        return;
      }
    }
  });
  await page.waitForTimeout(300);

  // Search broker
  log('4. Search Exness');
  await page.fill('input[placeholder*="broker"]', 'Exness');
  await page.click('button:has-text("Search")');
  await page.waitForTimeout(2000);

  // Select company via keyboard (ArrowDown + Enter is the only reliable way)
  log('5. Select company: Exness Technologies Ltd');
  await page.click('.ant-select');
  await page.waitForTimeout(800);
  // Navigate to 9th option (index 8)
  for (let i = 0; i < 9; i++) {
    await page.keyboard.press('ArrowDown');
    await page.waitForTimeout(50);
  }
  await page.keyboard.press('Enter');
  await page.waitForTimeout(800);

  // Verify server select appeared
  const selectsAfterCompany = await page.$$('.ant-select');
  log(`   Selects after company: ${selectsAfterCompany.length}`);
  if (selectsAfterCompany.length < 2) {
    log('   ❌ Server select not found!');
    await page.screenshot({ path: `${SHOT}/fail-company.png`, fullPage: true });
    await browser.close();
    return;
  }

  // Select server — now has showSearch, just type to filter!
  log('6. Select server: Exness-MT5Trial5');
  // Server select — now has custom filterOption matching server names.
  // Click, type to search, then click the filtered option.
  log('6. Select server: type "Exness-MT5Trial5" in search');
  await selectsAfterCompany[1].click();
  await page.waitForTimeout(500);
  // The server select has showSearch — the input is automatically focused
  await page.keyboard.type('Exness-MT5Trial5', { delay: 40 });
  await page.waitForTimeout(1000);

  // Find the filtered option and click it
  const serverOpts = await page.$$('.ant-select-item-option');
  log(`   Options after filter: ${serverOpts.length}`);
  for (const opt of serverOpts) {
    const txt = await opt.textContent();
    if (txt?.includes('Exness-MT5Trial5') || txt?.includes('MT5Trial5')) {
      await opt.click();
      log('   ✅ Selected server');
      break;
    }
  }
  await page.waitForTimeout(300);
  await page.waitForTimeout(300);
  await page.waitForTimeout(500);

  // ═══════════════════════════════════════════
  // 3. STEP 2 — Credentials
  // ═══════════════════════════════════════════
  log('7. Click Next → Step 2');
  // Check if Next is enabled
  const nextDisabled = await page.$eval('button:has-text("Next")', e => (e as HTMLButtonElement).disabled);
  if (!nextDisabled) {
    await page.click('button:has-text("Next")');
    await page.waitForTimeout(500);
  } else {
    log('   ❌ Next still disabled!');
  }
  log(`   URL: ${page.url()}`);
  await page.screenshot({ path: `${SHOT}/step2.png`, fullPage: true });

  // If on step 2, fill credentials
  if (!page.url().includes('bind')) {
    log('   Navigated away from bind page');
  } else {
    // Find account number input and password input on step 2
    const inputs = await page.$$('input[type="text"]');
    log(`   Found ${inputs.length} text inputs`);
    for (let i = 0; i < inputs.length; i++) {
      const ph = await inputs[i].getAttribute('placeholder');
      log(`   [${i}] placeholder="${ph}"`);
    }

    // Fill account number (should be the first input not for broker search)
    const acctInputs = await page.$$('input[placeholder*="account"], input[placeholder*="Account"], input[placeholder*="账号"]');
    if (acctInputs.length > 0) {
      await acctInputs[0].fill('277259925');
      log('   Filled account: 277259925');
    }

    // Fill password
    const pwdInputs = await page.$$('input[placeholder*="password"], input[placeholder*="Password"]');
    if (pwdInputs.length > 0) {
      await pwdInputs[0].fill('HavEr7901$');
      log('   Filled password');
    }

    await page.screenshot({ path: `${SHOT}/step2-filled.png`, fullPage: true });

    // Click Next → Step 3
    const nextBtn2 = await page.$('button:has-text("Next")');
    if (nextBtn2) {
      const dis = await nextBtn2.evaluate(e => (e as HTMLButtonElement).disabled);
      log(`   Next2 disabled: ${dis}`);
      if (!dis) {
        await nextBtn2.click();
        await page.waitForTimeout(500);
      }
    }
    log(`   After step2: URL=${page.url()}`);
  }

  // ═══════════════════════════════════════════
  // 4. STEP 3 — Verify & Confirm
  // ═══════════════════════════════════════════
  log('8. Step 3 — Verify');
  await page.screenshot({ path: `${SHOT}/step3.png`, fullPage: true });

  const verifyBtn = await page.$('button:has-text("Verify")');
  if (verifyBtn) {
    log('   Clicking Verify...');
    await verifyBtn.click();
    // Wait for MT verification (takes a few seconds)
    await page.waitForTimeout(6000);
  }

  await page.screenshot({ path: `${SHOT}/step3-after-verify.png`, fullPage: true });

  // Check for success
  const bodyText = await page.textContent('body');
  log(`   Verified: ${bodyText?.includes('Account verified') || bodyText?.includes('验证通过')}`);

  // Check for error message
  const errEls = await page.$$('[class*="error"], [style*="E53935"]');
  for (const el of errEls) {
    const txt = await el.textContent();
    if (txt && txt.length < 200) log(`   Error: ${txt.trim()}`);
  }

  // Click Confirm if available
  const confirmBtn = await page.$('button:has-text("Confirm")');
  if (confirmBtn) {
    log('   Clicking Confirm bind...');
    await confirmBtn.click();
    // Wait for ConnectAccount + navigation
    await page.waitForTimeout(8000);
  }

  log(`   URL after bind: ${page.url()}`);
  await page.screenshot({ path: `${SHOT}/after-bind.png`, fullPage: true });

  // ═══════════════════════════════════════════
  // 5. OBSERVE ACCOUNT DETAIL
  // ═══════════════════════════════════════════
  if (page.url().includes('/accounts/')) {
    log('\n9. Observing Account Detail:');
    const hdr = 'Time    |Pos|Hist|Equity|Stats|Canvas|Errors';
    console.log('\n' + hdr);
    console.log('-'.repeat(hdr.length));

    for (let i = 0; i <= 12; i++) {
      await page.waitForTimeout(1500);
      const s = await getSnapshot(page);
      const elapsed = ((i + 1) * 1.5).toFixed(1).padStart(4);
      console.log(`T+${elapsed}s | ${String(s.posCount).padEnd(2)} | ${String(s.histCount).padEnd(3)} | ${String(s.hasEquity).padEnd(5)} | ${String(s.hasStats).padEnd(5)} | ${String(s.canvases).padEnd(5)} | ${s.errs.join(';') || '-'}`);

      if (s.allGood) {
        log(`   ✅ ALL DATA LOADED at T+${elapsed}s!`);
        break;
      }
    }

    // Manual refresh comparison
    log('\n10. Manual refresh...');
    await page.reload({ waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(3000);
    const sr = await getSnapshot(page);
    log(`Refresh | pos=${sr.posCount} hist=${sr.histCount} eq=${sr.hasEquity} stats=${sr.hasStats} canvas=${sr.canvases}`);
  }

  log('\n📡 RPC Timeline:');
  for (const line of tl) console.log('  ' + line);
  await browser.close();
}

async function getSnapshot(page: any) {
  return page.evaluate(() => {
    const body = document.body?.textContent || '';
    let posCount = 0;
    document.querySelectorAll('table').forEach(t => {
      t.querySelectorAll('tbody tr').forEach(r => {
        const c = r.querySelectorAll('td');
        if (c.length >= 7 && /^\d{6,12}$/.test(c[0]?.textContent?.trim() || '')) posCount++;
      });
    });
    let histCount = -1;
    document.querySelectorAll('.ant-tabs-tab').forEach(t => {
      const m = (t.textContent || '').match(/\((\d+)\)/);
      if (m && /History|历史/i.test(t.textContent || '')) histCount = parseInt(m[1]);
    });
    const canv = document.querySelectorAll('canvas').length;
    const hasEquity = canv > 0;
    const hasStats = /Sharpe|夏普|Win rate|胜率|Max drawdown|最大回撤|Profit factor|盈亏比/i.test(body);
    const errs: string[] = [];
    document.querySelectorAll('.ant-message-notice-content').forEach(e => {
      const t = (e.textContent || '').trim().slice(0, 150);
      if (t) errs.push(t);
    });
    return { posCount, histCount, hasEquity, hasStats, canvases: canv, errs, allGood: posCount > 0 && histCount > 0 && hasEquity && hasStats };
  });
}

main().catch(e => { console.error(e); process.exit(1); });
