/**
 * Debug: dump all DOM after search to understand antd Select structure.
 */
import { chromium } from 'playwright';

const BASE = 'http://localhost:8022';

async function main() {
  const browser = await chromium.launch({
    headless: true,
    executablePath: '/snap/bin/chromium',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.fill('#login_email', 'admin@1.com');
  await page.fill('#login_password', '12345678');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);

  await page.goto(`${BASE}/accounts/bind`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(1000);

  // Click MT5
  await page.click('.flex.gap-4 > div:nth-child(2)');
  await page.waitForTimeout(300);

  // Search
  await page.fill('input[placeholder*="broker"]', 'Exness');
  await page.click('button:has-text("Search")');
  await page.waitForTimeout(2000);

  // Dump all select-related DOM
  console.log('=== All .ant-select elements ===');
  const selects = await page.$$('.ant-select');
  for (let i = 0; i < selects.length; i++) {
    const html = await selects[i].evaluate(el => el.outerHTML.slice(0, 300));
    console.log(`[${i}] ${html}`);
  }

  console.log('\n=== All .ant-select-selector elements ===');
  const selectors = await page.$$('.ant-select-selector');
  for (let i = 0; i < selectors.length; i++) {
    const html = await selectors[i].evaluate(el => el.outerHTML.slice(0, 200));
    console.log(`[${i}] ${html}`);
  }

  // Check body for "Select company" text
  const bodyText = await page.textContent('body');
  console.log('\nBody contains "Select company":', bodyText?.includes('Select company'));
  console.log('Body contains "ant-select":', bodyText?.includes('ant-select'));

  // Just grab all divs with class containing "select"
  console.log('\n=== All elements with "select" in class ===');
  const selectEls = await page.$$('[class*="select"]');
  console.log(`Found ${selectEls.length} elements`);
  for (let i = 0; i < Math.min(selectEls.length, 5); i++) {
    const cls = await selectEls[i].evaluate(el => el.className.slice(0, 60));
    const txt = await selectEls[i].evaluate(el => (el as HTMLElement).innerText?.slice(0, 60));
    console.log(`[${i}] class="${cls}" text="${txt}"`);
  }

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
