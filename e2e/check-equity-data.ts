/**
 * Check equity curve data for the new account.
 */
import { chromium } from 'playwright';

const BASE = 'http://localhost:8022';
const ACCT = '0eaf0332-9699-4aaf-8536-fc45770a9977';

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
  const respP = page.waitForResponse(r => r.url().includes('Login'), { timeout: 10000 });
  await page.click('button[type="submit"]');
  const lr = await respP;
  const token = (await lr.json()).accessToken;

  // Call GetAccountAnalytics directly
  const analyticsResp = await page.evaluate(async ({ tkn, acct }: any) => {
    const res = await fetch('/ant.v1.AnalyticsService/GetAccountAnalytics', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${tkn}`,
        'Connect-Protocol-Version': '1',
      },
      body: JSON.stringify({ accountId: acct, equityCurvePeriod: 4 }), // ALL
    });
    return res.json();
  }, { tkn: token, acct: ACCT });

  const eq = analyticsResp.equityCurve || [];
  console.log(`Equity curve points: ${eq.length}`);

  if (eq.length > 0) {
    console.log('First 5 points:');
    for (const p of eq.slice(0, 5)) {
      console.log(`  date=${p.date} equity=${p.equity} balance=${p.balance} profit=${p.profit}`);
    }
    console.log('...');
    console.log('Last 5 points:');
    for (const p of eq.slice(-5)) {
      console.log(`  date=${p.date} equity=${p.equity} balance=${p.balance} profit=${p.profit}`);
    }

    // Check if all points have the same value
    const equities = eq.map((p: any) => p.equity);
    const unique = new Set(equities);
    console.log(`\nUnique equity values: ${unique.size} out of ${eq.length}`);
    if (unique.size === 1) {
      console.log(`All points have equity=${equities[0]} — flat line!`);
    }
  }

  // Also check the account snapshots in DB
  console.log('\n--- Checking snapshots via API ---');
  // Call GetAccountSnapshots if available, or just look at the balance
  const accountResp = await page.evaluate(async ({ tkn, acct }: any) => {
    const res = await fetch('/ant.v1.AccountService/GetAccount', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${tkn}`,
        'Connect-Protocol-Version': '1',
      },
      body: JSON.stringify({ id: acct }),
    });
    return res.json();
  }, { tkn: token, acct: ACCT });

  console.log(`Account balance: ${accountResp.balance}, equity: ${accountResp.equity}`);

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
