/**
 * Debug v3: Try different broker/server combinations.
 * The user specified MT5 + Exness-MT5Trial5 but search returned only MT4 servers.
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

  await page.goto(`${BASE}/accounts/bind`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(1000);

  // Click MT5
  const cards = await page.$$('.flex.gap-4 > div');
  if (cards.length >= 2) {
    await cards[1].click();
    await page.waitForTimeout(300);
    console.log('Selected MT5');
  }

  // Search for "Exness"
  const input = await page.$('input[placeholder*="broker"]');
  if (input) await input.fill('Exness');
  const searchBtn = await page.$('button:has-text("Search")');
  if (searchBtn) await searchBtn.click();
  await page.waitForTimeout(2000);

  // Get company list
  const companySelect = await page.$('.ant-select');
  if (companySelect) {
    await companySelect.click();
    await page.waitForTimeout(500);
    const opts = await page.$$('.ant-select-item-option-content');
    for (const opt of opts) {
      const txt = await opt.textContent();
      if (txt?.includes('Exness Technologies Ltd')) { await opt.click(); break; }
    }
    await page.waitForTimeout(800);
    console.log('Selected company');
  }

  // Now for the server select, try TYPEING in it to filter
  const selects = await page.$$('.ant-select');
  console.log(`Selects: ${selects.length}`);

  if (selects.length >= 2) {
    // The server select should have search functionality — click and type
    await selects[1].click();
    await page.waitForTimeout(500);

    // Try typing "MT5" in the search field that may appear in the dropdown
    const searchInDropdown = await page.$('.ant-select-selection-search-input');
    if (searchInDropdown) {
      await searchInDropdown.fill('MT5');
      await page.waitForTimeout(1000);
      console.log('Typed "MT5" in server dropdown search');
    }

    // Check all options after filtering
    const opts = await page.$$('.ant-select-item-option-content');
    console.log(`Server options after filter (${opts.length}):`);
    for (const opt of opts) {
      const txt = await opt.textContent();
      console.log(`  "${txt}"`);
    }

    // Also try search for "Trial"
    const searchInput2 = await page.$('.ant-select-selection-search-input');
    if (searchInput2) {
      await searchInput2.fill('');
      await searchInput2.fill('Trial');
      await page.waitForTimeout(1000);
      const opts2 = await page.$$('.ant-select-item-option-content');
      console.log(`\nServer options with "Trial" (${opts2.length}):`);
      for (const opt of opts2) {
        const txt = await opt.textContent();
        console.log(`  "${txt}"`);
      }
    }
  }

  // Try searching specifically for "Exness-MT5Trial5"
  console.log('\n--- Try direct search for "Exness-MT5Trial5" ---');
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(1000);

  // Select MT5
  const cards2 = await page.$$('.flex.gap-4 > div');
  if (cards2.length >= 2) { await cards2[1].click(); await page.waitForTimeout(300); }

  // Search for the exact server name
  const input2 = await page.$('input[placeholder*="broker"]');
  if (input2) await input2.fill('Exness-MT5Trial5');
  const searchBtn2 = await page.$('button:has-text("Search")');
  if (searchBtn2) await searchBtn2.click();
  await page.waitForTimeout(2000);

  // Check results
  const selectCount = await page.$$('.ant-select');
  console.log(`Selects after direct search: ${selectCount.length}`);
  const bodyText = await page.textContent('body');
  const idx = bodyText?.indexOf('Exness-MT5') || 0;
  console.log('Body around "Exness-MT5":', bodyText?.slice(Math.max(0, idx - 50), idx + 150));

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
