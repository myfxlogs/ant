/**
 * Capture the raw SearchBroker API response + test scrolling with mouse wheel.
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

  // Click MT5 card — find by "MT5" text (not by index, which hits step indicators)
  const platformCards = await page.$$('.flex.gap-4 > div');
  for (const card of platformCards) {
    const text = await card.textContent();
    if (text?.trim() === 'MT5' || text?.includes('MetaTrader 5')) {
      await card.click();
      console.log('Clicked MT5 card');
      break;
    }
  }
  await page.waitForTimeout(300);

  // Search
  await page.fill('input[placeholder*="broker"]', 'Exness');
  await page.click('button:has-text("Search")');

  // Capture the API response
  const respPromise = page.waitForResponse(r => r.url().includes('SearchBroker'), { timeout: 10000 });
  const resp = await respPromise;
  const body = await resp.json();
  const companies = body.companies || [];

  console.log(`SearchBroker returned ${companies.length} companies`);
  // Find Exness Technologies Ltd
  for (const c of companies) {
    const name = c.companyName || c.company_name || c.name || '';
    if (name.includes('Exness Technologies')) {
      const servers = c.servers || [];
      console.log(`\nCompany: ${name}`);
      console.log(`Servers count: ${servers.length}`);
      console.log('First 5 servers:');
      for (const s of servers.slice(0, 5)) {
        console.log(`  name="${s.name}" access=${JSON.stringify((s.access || []).slice(0, 2))}`);
      }
      console.log('Last 5 servers:');
      for (const s of servers.slice(-5)) {
        console.log(`  name="${s.name}" access=${JSON.stringify((s.access || []).slice(0, 2))}`);
      }
      // Find any server with "MT5" in name
      const mt5 = servers.filter((s: any) => (s.name || '').toLowerCase().includes('mt5'));
      console.log(`\nServers with "MT5": ${mt5.length}`);
      for (const s of mt5) console.log(`  name="${s.name}" access=${JSON.stringify((s.access || []).slice(0, 2))}`);
      // Also show trial servers for MT5
      const trial5 = servers.filter((s: any) => (s.name || '').toLowerCase().includes('trial'));
      console.log(`\nAll trial servers: ${trial5.length}`);
      for (const s of trial5) console.log(`  "${s.name}"`);
    }
  }

  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
