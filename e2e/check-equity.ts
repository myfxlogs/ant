/**
 * Check equity chart rendering on existing account.
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

  // Go to the new account detail
  const acctId = '77292b6f-95d7-496a-a382-a54f31b82fbd';
  await page.goto(`${BASE}/accounts/${acctId}`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.waitForTimeout(3000);

  // Check equity chart data
  const eqData = await page.evaluate(() => {
    // Check if EquityChart rendered or showed empty state
    const chartSection = document.querySelector('.rounded-2xl.p-5')?.parentElement;
    const emptyText = document.body?.textContent?.match(/No equity curve|暂无净值曲线/) || [];
    const canvasCount = document.querySelectorAll('canvas').length;

    // Check the chart area specifically
    const chartDivs = document.querySelectorAll('.h-\\[280px\\], [style*="280px"]');
    const chartTexts: string[] = [];
    chartDivs.forEach(d => chartTexts.push(d.textContent?.trim().slice(0, 80) || ''));

    return { emptyText: emptyText[0] || null, canvasCount, chartTexts };
  });

  console.log('Equity data:', JSON.stringify(eqData, null, 2));

  // Check GetAccountAnalytics response
  // Intercept the next call
  const analyticsPromise = page.waitForResponse(r => r.url().includes('GetAccountAnalytics'), { timeout: 10000 });
  await page.reload({ waitUntil: 'domcontentloaded', timeout: 10000 });
  const ar = await analyticsPromise;
  const body = await ar.json();
  console.log(`\nGetAccountAnalytics status: ${ar.status()}`);
  console.log(`equityCurve length: ${body.equityCurve?.length || 0}`);
  console.log(`tradeStats: ${JSON.stringify(body.tradeStats)}`);
  console.log(`First equity point: ${JSON.stringify(body.equityCurve?.[0])}`);
  console.log(`Last equity point: ${JSON.stringify(body.equityCurve?.slice(-1)[0])}`);

  // After refresh, check if chart renders
  await page.waitForTimeout(3000);
  const afterRefresh = await page.evaluate(() => ({
    canvasCount: document.querySelectorAll('canvas').length,
    noData: document.body?.textContent?.includes('No equity curve') || false,
  }));
  console.log(`\nAfter refresh - canvas: ${afterRefresh.canvasCount}, noData: ${afterRefresh.noData}`);

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
