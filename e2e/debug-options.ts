/**
 * Debug: dump server select options DOM + check React state
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

  // MT5
  await page.evaluate(() => {
    document.querySelectorAll('.flex.gap-4 > div').forEach(d => {
      const h2 = d.querySelector('.text-2xl');
      if (h2?.textContent?.trim() === 'MT5') (d as HTMLElement).click();
    });
  });
  await page.waitForTimeout(300);

  // Search
  await page.fill('input[placeholder*="broker"]', 'Exness');
  await page.click('button:has-text("Search")');
  await page.waitForTimeout(2000);

  // Select company
  await page.click('.ant-select');
  await page.waitForTimeout(500);
  for (let i = 0; i < 9; i++) { await page.keyboard.press('ArrowDown'); await page.waitForTimeout(50); }
  await page.keyboard.press('Enter');
  await page.waitForTimeout(800);

  // Check selectedCompany state via React fiber
  const state = await page.evaluate(() => {
    const root = document.getElementById('root');
    if (!root) return 'NO_ROOT';
    const key = Object.keys(root).find(k => k.startsWith('__reactContainer'));
    if (!key) return 'NO_KEY';

    function walk(fiber: any, depth: number): any {
      if (!fiber || depth > 40) return null;
      if (fiber.memoizedState) {
        let s = fiber.memoizedState;
        let hookIdx = 0;
        const hooks: any[] = [];
        while (s && hookIdx < 25) {
          const val = s.queue?.lastRenderedState;
          if (val !== undefined) {
            if (Array.isArray(val) && val.length > 0 && val[0]?.servers) {
              // This is searchResults
              hooks.push({ idx: hookIdx, type: 'searchResults', length: val.length });
            } else if (typeof val === 'object' && val !== null && 'companyName' in val && 'servers' in val) {
              // This is selectedCompany (single object)
              hooks.push({ idx: hookIdx, type: 'selectedCompany', company: val.companyName, serverCount: val.servers?.length });
            } else if (typeof val === 'object' && val !== null && 'name' in val && 'access' in val) {
              hooks.push({ idx: hookIdx, type: 'selectedServer', name: val.name });
            } else if (typeof val === 'string' && val.length > 3) {
              hooks.push({ idx: hookIdx, type: 'string', value: val.slice(0, 50) });
            }
          }
          s = s.next;
          hookIdx++;
        }
        if (hooks.length > 0) return hooks;
      }
      return walk(fiber.child, depth + 1) || walk(fiber.sibling, depth);
    }

    return walk((root as any)[key], 0);
  });

  console.log('React hooks state:', JSON.stringify(state, null, 2));

  // Now open server select and dump its options
  const selects = await page.$$('.ant-select');
  console.log(`\nSelects: ${selects.length}`);
  if (selects.length >= 2) {
    await selects[1].click();
    await page.waitForTimeout(500);

    // Type search
    await page.keyboard.type('Trial', { delay: 50 });
    await page.waitForTimeout(1000);

    // Dump all option DOM
    const opts = await page.$$eval('.ant-select-item-option', els => els.map(e => ({
      text: e.textContent?.trim().slice(0, 60),
      title: e.getAttribute('title'),
      label: e.getAttribute('label'),
      value: e.getAttribute('value'),
    })));
    console.log(`\nFiltered options (${opts.length}):`);
    for (const o of opts) console.log('  ', JSON.stringify(o));
  }

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
