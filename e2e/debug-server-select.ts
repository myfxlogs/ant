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

  // Now click the server select
  const selects = await page.$$('.ant-select');
  console.log(`Selects: ${selects.length}`);
  await selects[1].click();
  await page.waitForTimeout(1000);

  // Dump ALL inputs in the entire document (including portal dropdown)
  const allInputs = await page.$$eval('input', els => els.map(e => ({
    class: e.className.slice(0, 80),
    type: e.getAttribute('type'),
    placeholder: e.getAttribute('placeholder'),
    role: e.getAttribute('role'),
    ariaAutocomplete: e.getAttribute('aria-autocomplete'),
    parent: e.parentElement?.className?.slice(0, 60),
  })));
  console.log(`Total inputs in DOM: ${allInputs.length}`);
  for (const inp of allInputs) console.log('  ', JSON.stringify(inp));

  // Check all elements with "search" class
  const searchEls = await page.$$eval('[class*="search"]', els => els.map(e => ({
    tag: e.tagName,
    class: e.className.slice(0, 80),
    text: (e as HTMLElement).innerText?.slice(0, 30),
  })));
  console.log(`\nSearch elements: ${searchEls.length}`);
  for (const el of searchEls) console.log('  ', JSON.stringify(el));

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
