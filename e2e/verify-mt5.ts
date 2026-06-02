/**
 * Verify MT5 selection works on the bind page.
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

  // Check initial state
  let mt4Style = await page.$eval('.flex.gap-4 > div:first-child', el => ({
    bg: (el as HTMLElement).style.background,
    border: (el as HTMLElement).style.border,
    color: (el as HTMLElement).style.color,
  }));
  let mt5Style = await page.$eval('.flex.gap-4 > div:last-child', el => ({
    bg: (el as HTMLElement).style.background,
    border: (el as HTMLElement).style.border,
    color: (el as HTMLElement).style.color,
  }));
  console.log('INITIAL:');
  console.log('  MT4:', JSON.stringify(mt4Style));
  console.log('  MT5:', JSON.stringify(mt5Style));

  // Click MT5
  const cards = await page.$$('.flex.gap-4 > div');
  console.log(`  Found ${cards.length} platform cards`);
  await cards[1].click();
  await page.waitForTimeout(500);

  // Check after click
  mt4Style = await page.$eval('.flex.gap-4 > div:first-child', el => ({
    bg: (el as HTMLElement).style.background,
    border: (el as HTMLElement).style.border,
    color: (el as HTMLElement).style.color,
  }));
  mt5Style = await page.$eval('.flex.gap-4 > div:last-child', el => ({
    bg: (el as HTMLElement).style.background,
    border: (el as HTMLElement).style.border,
    color: (el as HTMLElement).style.color,
  }));
  console.log('AFTER CLICK MT5:');
  console.log('  MT4:', JSON.stringify(mt4Style));
  console.log('  MT5:', JSON.stringify(mt5Style));

  // Verify by checking which one has the gold color
  const mt5TextColor = await page.$eval('.flex.gap-4 > div:last-child .text-2xl', el =>
    (el as HTMLElement).style.color
  );
  console.log(`  MT5 text color: ${mt5TextColor}`);
  console.log(`  MT5 is selected: ${mt5TextColor === 'rgb(212, 175, 55)'}`);

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
