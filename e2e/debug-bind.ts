/**
 * Debug: Go to bind page, screenshot step 1, dump DOM structure.
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

  // Login
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.fill('#login_email', 'admin@1.com');
  await page.fill('#login_password', '12345678');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);

  // Go to bind page
  await page.goto(`${BASE}/accounts/bind`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(1500);
  await page.screenshot({ path: `${SHOT}/debug-step1-initial.png`, fullPage: true });

  // Dump all interactive elements in the card
  const cardHtml = await page.$eval('.rounded-2xl, [class*="card"], [class*="form"]', el => el.outerHTML.slice(0, 3000)).catch(() => 'NO_CARD');
  console.log('Card HTML:', cardHtml.slice(0, 2000));

  // Dump all buttons
  const btns = await page.$$eval('button, [role="button"]', els => els.map(e => ({
    text: e.textContent?.trim().slice(0, 40),
    tag: e.tagName,
    disabled: (e as HTMLButtonElement).disabled,
    className: e.className.slice(0, 60),
  })));
  console.log('\nButtons:', JSON.stringify(btns, null, 2));

  // Dump selects
  const selects = await page.$$eval('.ant-select', els => els.map(e => ({
    visible: (e as HTMLElement).offsetParent !== null,
    text: e.textContent?.trim().slice(0, 40),
  })));
  console.log('\nSelects:', JSON.stringify(selects, null, 2));

  // Try clicking MT5
  console.log('\nClicking MT5...');
  const mt5div = await page.$('text=MT5');
  if (mt5div) {
    await mt5div.click();
    await page.waitForTimeout(300);
    console.log('Clicked MT5');
  }

  // Find search input and fill
  console.log('\nFilling broker search...');
  const searchInput = await page.$('input[placeholder*="broker"], input[placeholder*="Broker"], input[placeholder*="经纪"]');
  if (searchInput) {
    await searchInput.click();
    await searchInput.fill('Exness');
    console.log('Filled "Exness"');
  } else {
    // Try by getting all inputs
    const inputs = await page.$$eval('input', els => els.map(e => ({ ph: e.getAttribute('placeholder'), type: e.getAttribute('type') })));
    console.log('All inputs:', JSON.stringify(inputs));
  }

  await page.screenshot({ path: `${SHOT}/debug-step1-typed.png`, fullPage: true });

  // Click search — try multiple selectors
  for (const sel of ['button:has-text("Search")', 'button:has-text("搜索")', '[role="button"]:has-text("Search")', '.gradient-button:has-text("Search")']) {
    const btn = await page.$(sel);
    if (btn) {
      console.log(`\nClicking search via: ${sel}`);
      await btn.click();
      await page.waitForTimeout(2000);
      break;
    }
  }

  await page.screenshot({ path: `${SHOT}/debug-step1-after-search.png`, fullPage: true });

  // Check what appeared
  const newSelects = await page.$$eval('.ant-select', els => els.map(e => ({
    visible: (e as HTMLElement).offsetParent !== null,
    text: e.textContent?.trim().slice(0, 50),
  })));
  console.log('\nAfter search, selects:', JSON.stringify(newSelects, null, 2));

  // Dump body text around selects
  const bodyExcerpt = await page.textContent('body');
  const idx = bodyExcerpt?.indexOf('Exness') || 0;
  console.log('\nBody around "Exness":', bodyExcerpt?.slice(Math.max(0, idx - 50), idx + 200));

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
