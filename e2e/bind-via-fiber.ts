/**
 * Use React fiber to directly manipulate state for the full bind flow.
 * Bypasses antd Select virtual list issues entirely.
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

  // ── Login ──
  log('Login');
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.fill('#login_email', 'admin@1.com');
  await page.fill('#login_password', '12345678');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);
  await page.goto(`${BASE}/accounts/bind`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(1000);

  // ── Step 1: Click MT5 + search + set company/server via fiber ──
  log('Step 1: MT5 + search + select company');
  // Click MT5
  await page.evaluate(() => {
    const divs = document.querySelectorAll('.flex.gap-4 > div');
    for (const d of divs) {
      if (d.textContent?.trim() === 'MT5') { (d as HTMLElement).click(); break; }
    }
  });
  await page.waitForTimeout(300);

  // Search
  await page.fill('input[placeholder*="broker"]', 'Exness');
  await page.click('button:has-text("Search")');
  await page.waitForTimeout(2000);

  // Use React fiber to directly set selectedCompany and selectedServer
  // This bypasses the antd Select virtual list issue
  log('Setting company via React fiber...');
  const fiberResult = await page.evaluate(() => {
    // Find the main card container that has the React root
    const selectEl = document.querySelector('.ant-select');
    if (!selectEl) return 'NO_SELECT';

    const fiberKey = Object.keys(selectEl).find(k => k.startsWith('__reactFiber'));
    if (!fiberKey) return 'NO_FIBER';

    let fiber: any = (selectEl as any)[fiberKey];
    // Walk up to find the BindAccount component
    while (fiber) {
      const props = fiber.memoizedProps;
      const state = fiber.memoizedState;
      // Look for the BindAccount component (has handleSearch, handleBind, etc.)
      if (props && 'handleVerify' in props) {
        return 'FOUND_BIND_ACCOUNT';
      }
      // Check hooks - look for useState pairs (companyName, broker, etc.)
      if (fiber.tag === 0 && state) { // Function component
        let s = state;
        let hookIdx = 0;
        while (s) {
          if (s.queue && hookIdx < 20) {
            // This is a useState hook
          }
          s = s.next;
          hookIdx++;
        }
      }
      fiber = fiber.return;
    }
    return 'NOT_FOUND depth_exceeded';
  });
  log(`Fiber result: ${fiberResult}`);

  // Alternative: just open the select and try clicking with force
  log('\nTrying force-click approach...');
  // Click select to open
  const selectHandle = await page.$('.ant-select');
  if (selectHandle) {
    // Get bounding box and click it
    const box = await selectHandle.boundingBox();
    if (box) {
      await page.mouse.click(box.x + box.width / 2, box.y + box.height / 2);
      await page.waitForTimeout(1000);
    }
  }

  // Now look for option items in the portal dropdown
  const dropdownOpts = await page.$$('.ant-select-dropdown .ant-select-item-option, .ant-select-item-option');
  log(`Dropdown options found: ${dropdownOpts.length}`);

  if (dropdownOpts.length > 0) {
    for (const opt of dropdownOpts) {
      const txt = await opt.textContent();
      if (txt?.includes('Exness Technologies Ltd')) {
        // Force click with mouse event
        const optBox = await opt.boundingBox();
        if (optBox) {
          await page.mouse.click(optBox.x + optBox.width / 2, optBox.y + optBox.height / 2);
          log('Force-clicked company option');
          break;
        }
      }
    }
  }
  await page.waitForTimeout(800);

  // Check if server select appeared
  const selectCount = await page.$$('.ant-select');
  log(`Selects after company: ${selectCount.length}`);

  await page.screenshot({ path: `${SHOT}/fiber-debug.png`, fullPage: true });

  log('\n📡 RPC:');
  for (const line of tl) console.log('  ' + line);
  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
