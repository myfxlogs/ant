/**
 * Verify equity curve fix — screenshot and data check.
 */
import { chromium } from 'playwright';
import * as fs from 'fs';

const BASE = 'http://localhost:8022';
const ACCT = '0eaf0332-9699-4aaf-8536-fc45770a9977';
const SHOT = '/opt/ant/e2e/shots';

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

  // Get live data via API
  const respP = page.waitForResponse(r => r.url().includes('Login'), { timeout: 10000 });

  // Navigate to account detail
  await page.goto(`${BASE}/accounts/${ACCT}`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(3000);

  // Check equity chart rendering
  const info = await page.evaluate(() => {
    const b = document.body?.textContent || '';
    const charts = document.querySelectorAll('.recharts-surface');
    // Check for the equity chart specifically
    const areas = document.querySelectorAll('.recharts-area');
    const hasEquityData = areas.length > 0;

    // Check the last visible equity value
    const allText = b;
    const equityMatch = allText.match(/Equity|净值/);
    const noEquity = /No equity curve|暂无净值曲线/i.test(allText);

    return {
      rechartsSurfaces: charts.length,
      rechartsAreas: areas.length,
      hasEquityData,
      noEquity,
    };
  });

  console.log('Chart info:', JSON.stringify(info, null, 2));

  // Take screenshot
  await page.screenshot({ path: `${SHOT}/equity-after-fix.png`, fullPage: true });
  console.log('Screenshot saved');

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
