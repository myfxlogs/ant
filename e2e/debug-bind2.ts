/**
 * Debug: After selecting company, dump server options.
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

  // Login
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.fill('#login_email', 'admin@1.com');
  await page.fill('#login_password', '12345678');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);

  // Go to bind page
  await page.goto(`${BASE}/accounts/bind`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(1000);

  // Click MT5
  const platformCards = await page.$$('.flex.gap-4 > div');
  if (platformCards.length >= 2) { await platformCards[1].click(); await page.waitForTimeout(300); }

  // Search Exness
  const brokerInput = await page.$('input[placeholder*="broker"], input[placeholder*="Broker"]');
  if (brokerInput) { await brokerInput.fill('Exness'); }
  const searchBtn = await page.$('button:has-text("Search")');
  if (searchBtn) { await searchBtn.click(); await page.waitForTimeout(2000); }

  // Select company
  const companySelect = await page.$('.ant-select');
  if (companySelect) {
    await companySelect.click();
    await page.waitForTimeout(500);
    const opts = await page.$$('.ant-select-item-option-content');
    for (const opt of opts) {
      const txt = await opt.textContent();
      console.log(`Company option: "${txt}"`);
      if (txt?.includes('Exness Technologies')) { await opt.click(); break; }
    }
    await page.waitForTimeout(800);
  }

  // Now click server select and dump ALL options
  const allSelects = await page.$$('.ant-select');
  console.log(`\nNumber of selects: ${allSelects.length}`);
  if (allSelects.length >= 2) {
    // Check what each select shows
    for (let i = 0; i < allSelects.length; i++) {
      const text = await allSelects[i].textContent();
      console.log(`Select ${i}: "${text?.trim().slice(0, 60)}"`);
    }

    await allSelects[1].click();
    await page.waitForTimeout(1000);

    // Dump ALL options in the dropdown
    const serverOpts = await page.$$('.ant-select-item-option-content');
    console.log(`\nServer options (${serverOpts.length}):`);
    for (const opt of serverOpts) {
      const txt = await opt.textContent();
      console.log(`  "${txt}"`);
    }

    // Also check for items with title attribute
    const items = await page.$$('.ant-select-item-option');
    console.log(`\nAll option items (${items.length}):`);
    for (const item of items) {
      const title = await item.getAttribute('title');
      const text = await item.textContent();
      console.log(`  title="${title}" text="${text?.trim().slice(0, 60)}"`);
    }
  }

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
