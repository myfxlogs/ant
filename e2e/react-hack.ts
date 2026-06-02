/**
 * Debug: use React internals to set selectedCompany directly.
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

  // Click MT5 by text
  const cards = await page.$$('.flex.gap-4 > div');
  for (const card of cards) {
    const t = await card.textContent();
    if (t?.trim() === 'MT5') { await card.click(); break; }
  }
  await page.waitForTimeout(300);

  // Search Exness
  await page.fill('input[placeholder*="broker"]', 'Exness');
  await page.click('button:has-text("Search")');
  await page.waitForTimeout(2000);

  // Use page.evaluate to directly set selectedCompany via React state
  // Look for the company data in the searchResults variable
  await page.evaluate(() => {
    // Find the antd Select and click it to open
    const select: any = document.querySelector('.ant-select');
    if (!select) return;
    select.click();
  });
  await page.waitForTimeout(1000);

  // Dump all option DOM attributes
  const optionInfo = await page.$$eval('.ant-select-item-option', els => els.map(e => ({
    title: e.getAttribute('title'),
    value: e.getAttribute('value'),
    label: e.getAttribute('label'),
    text: e.textContent?.trim().slice(0, 60),
    ariaLabel: e.getAttribute('aria-label'),
  })));
  console.log('Options:', JSON.stringify(optionInfo.slice(0, 5), null, 2));

  // Try clicking by aria-label or by evaluating a programmatic select
  // Use the Antd Select API through the DOM
  const result = await page.evaluate(() => {
    // Try to find the React fiber and access the onChange
    const selectEl = document.querySelector('.ant-select');
    if (!selectEl) return 'no select found';

    // Walk up the fiber
    const fiberKey = Object.keys(selectEl).find(k => k.startsWith('__reactFiber'));
    if (!fiberKey) return 'no fiber key';

    let fiber = (selectEl as any)[fiberKey];
    // Walk up to find the Select component with onChange
    let depth = 0;
    while (fiber && depth < 20) {
      if (fiber.memoizedProps?.onChange) {
        // Found it! Get the search results from the parent component
        return 'found onChange at depth ' + depth;
      }
      fiber = fiber.return;
      depth++;
    }
    return 'no onChange found, depth=' + depth;
  });
  console.log('React fiber result:', result);

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
