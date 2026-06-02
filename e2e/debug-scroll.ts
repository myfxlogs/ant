/**
 * Debug: find the right way to scroll the antd Select virtual list.
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

  // Click MT5 by finding the card with "MT5" text
  const cards = await page.$$('.flex.gap-4 > div');
  for (const card of cards) {
    const text = await card.textContent();
    if (text?.trim() === 'MT5' || text?.includes('MetaTrader 5')) {
      await card.click();
      console.log('Clicked MT5');
      break;
    }
  }
  await page.waitForTimeout(300);

  // Search
  await page.fill('input[placeholder*="broker"]', 'Exness');
  await page.click('button:has-text("Search")');
  await page.waitForTimeout(2000);

  // Select company by clicking the option's parent container (not just the text span)
  await page.click('.ant-select');  // open dropdown
  await page.waitForTimeout(500);
  // Click the option div directly (not the inner span)
  const companyOption = await page.$('.ant-select-item-option[title*="Exness Technologies"]');
  if (companyOption) {
    await companyOption.click();
    console.log('Clicked company option via title');
  } else {
    // Fallback: click by iterating options
    const items = await page.$$('.ant-select-item-option');
    for (const item of items) {
      const txt = await item.textContent();
      if (txt?.includes('Exness Technologies Ltd')) {
        await item.click();
        console.log('Clicked company option via text');
        break;
      }
    }
  }
  await page.waitForTimeout(800);

  // Now open server select
  const selectEls = await page.$$('.ant-select');
  console.log(`Selects: ${selectEls.length}`);
  await selectEls[1].click();
  await page.waitForTimeout(500);

  // Find the scrollable container in the dropdown
  const scrollInfo = await page.evaluate(() => {
    // Check various possible scroll containers
    const holders = document.querySelectorAll('.rc-virtual-list-holder, .rc-virtual-list-holder-inner, [class*="virtual"]');
    const results: any[] = [];
    holders.forEach((h, i) => {
      results.push({
        index: i,
        className: h.className,
        scrollTop: (h as HTMLElement).scrollTop,
        scrollHeight: (h as HTMLElement).scrollHeight,
        clientHeight: (h as HTMLElement).clientHeight,
        childCount: h.children.length,
      });
    });
    return results;
  });
  console.log('Scroll containers:', JSON.stringify(scrollInfo, null, 2));

  // Try dispatching wheel event on the dropdown
  console.log('\nDispatching wheel events...');
  for (let i = 0; i < 30; i++) {
    await page.mouse.wheel(0, 100);
    await page.waitForTimeout(100);
  }

  // Check what items are now visible
  const visibleItems = await page.$$('.ant-select-item-option');
  console.log(`\nAfter wheel: ${visibleItems.length} items visible`);
  for (const item of visibleItems.slice(0, 5)) {
    console.log(`  "${(await item.textContent())?.trim().slice(0, 60)}"`);
  }
  console.log('  ...');
  for (const item of visibleItems.slice(-3)) {
    console.log(`  "${(await item.textContent())?.trim().slice(0, 60)}"`);
  }

  // Look for MT5Trial5
  let found = false;
  for (const item of visibleItems) {
    const txt = await item.textContent();
    if (txt?.includes('MT5Trial5')) {
      console.log(`\n✅ Found MT5Trial5 in visible items!`);
      found = true;
      break;
    }
  }
  if (!found) {
    console.log('\n❌ MT5Trial5 not visible after scrolling');
  }

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
