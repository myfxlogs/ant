/**
 * Bind account by calling API directly. Bypass antd Select UI issues entirely.
 * 1. SearchBroker → get company+server
 * 2. CreateAccount with those params
 * 3. Navigate to Account Detail
 * 4. Observe data loading
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

  const t = () => new Date().toISOString().slice(11, 23);
  const log = (msg: string) => console.log(`[${t()}] ${msg}`);
  const tl: string[] = [];
  page.on('response', (r) => {
    const u = r.url();
    if (u.includes('ant.v1')) {
      const p = u.split('ant.v1')[1]?.slice(0, 100);
      tl.push(`${t()} ${r.status()} ${p}`);
      if (r.status() !== 200) {
        r.text().then(body => tl.push(`${t()}   BODY: ${body.slice(0, 300)}`));
      }
    }
  });

  // ── Login ──
  log('Login');
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 10000 });
  await page.fill('#login_email', 'admin@1.com');
  await page.fill('#login_password', '12345678');
  const loginResp = page.waitForResponse(r => r.url().includes('Login'), { timeout: 10000 });
  await page.click('button[type="submit"]');
  const lr = await loginResp;
  const token = (await lr.json()).accessToken;
  log(`Got token: ${token?.slice(0, 20)}...`);

  // ── SearchBroker (need company + server + host) ──
  log('SearchBroker API...');
  const searchResp = await page.evaluate(async (tkn) => {
    const res = await fetch('/ant.v1.AccountService/SearchBroker', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${tkn}`,
        'Connect-Protocol-Version': '1',
      },
      body: JSON.stringify({ company: 'Exness', mtType: 'MT5' }),
    });
    return res.json();
  }, token);

  const companies = searchResp.companies || [];
  log(`Found ${companies.length} companies`);
  const company = companies.find((c: any) =>
    (c.companyName || c.company_name || '').includes('Exness Technologies Ltd'));
  if (!company) { log('❌ Company not found'); await browser.close(); return; }

  const servers = company.servers || [];
  const server = servers.find((s: any) => s.name === 'Exness-MT5Trial5');
  if (!server) {
    log(`❌ Server not found. Available: ${servers.slice(0, 3).map((s: any) => s.name).join(', ')}...`);
    await browser.close(); return;
  }
  const host = (server.access || [])[0] || '';
  log(`Server: ${server.name}, host: ${host}`);

  // ── CreateAccount ──
  log('CreateAccount API...');
  const createParams = {
    login: '277259925',
    password: 'HavEr7901$',
    mtType: 'MT5',
    brokerCompany: company.companyName || company.company_name,
    brokerServer: server.name,
    brokerHost: host,
  };

  const createResp = await page.evaluate(async ({ tkn, params }: any) => {
    const res = await fetch('/ant.v1.AccountService/CreateAccount', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${tkn}`,
        'Connect-Protocol-Version': '1',
      },
      body: JSON.stringify(params),
    });
    return { status: res.status, body: await res.text() };
  }, { tkn: token, params: createParams });

  log(`CreateAccount response: ${createResp.status}`);
  if (createResp.status !== 200) {
    log(`Body: ${createResp.body.slice(0, 300)}`);
  } else {
    const account = JSON.parse(createResp.body);
    log(`Account created: ${account.id}`);

    // Connect the account
    log('ConnectAccount API...');
    const connectResp = await page.evaluate(async ({ tkn, aid }: any) => {
      const res = await fetch('/ant.v1.AccountService/ConnectAccount', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${tkn}`,
          'Connect-Protocol-Version': '1',
        },
        body: JSON.stringify({ id: aid }),
      });
      return { status: res.status, body: await res.text() };
    }, { tkn: token, aid: account.id });
    log(`ConnectAccount response: ${connectResp.status}`);

    // Navigate to account detail
    log(`Navigating to /accounts/${account.id}`);
    await page.goto(`${BASE}/accounts/${account.id}`, { waitUntil: 'domcontentloaded', timeout: 10000 });
    await page.waitForTimeout(2000);

    // Observe
    log('\nObserving Account Detail:\n');
    for (let i = 0; i <= 12; i++) {
      await page.waitForTimeout(1500);
      const s = await page.evaluate(() => {
        const b = document.body?.textContent || '';
        let pos = 0;
        document.querySelectorAll('table').forEach(t => {
          t.querySelectorAll('tbody tr').forEach(r => {
            if (r.querySelectorAll('td').length >= 7 && /^\d{6,12}$/.test(r.querySelector('td')?.textContent?.trim() || '')) pos++;
          });
        });
        let hist = -1;
        document.querySelectorAll('.ant-tabs-tab').forEach(t => {
          const m = (t.textContent || '').match(/\((\d+)\)/);
          if (m && /History|历史/i.test(t.textContent || '')) hist = parseInt(m[1]);
        });
        // Recharts renders as SVG, not canvas
        const hasChart = document.querySelectorAll('.recharts-surface, svg.recharts-surface, .recharts-wrapper').length > 0;
        const hasNoEquity = /No equity curve|暂无净值曲线/i.test(b);
        const eq = hasChart && !hasNoEquity;
        const stats = /Sharpe|夏普|Win rate|胜率|Max drawdown|最大回撤/i.test(b);
        const errs: string[] = [];
        document.querySelectorAll('.ant-message-notice-content').forEach(e => {
          const t = (e.textContent || '').trim().slice(0, 120);
          if (t) errs.push(t);
        });
        return { pos, hist, eq, stats, errs, allGood: pos > 0 && hist >= 0 && eq && stats };
      });
      const elapsed = ((i + 1) * 1.5).toFixed(1).padStart(4);
      console.log(`T+${elapsed}s | pos=${s.pos} hist=${s.hist} eq=${s.eq} stats=${s.stats} canvas=${s.canv} errs=${s.errs.join(';') || '-'}`);
      if (s.allGood) { log(`✅ ALL DATA at T+${elapsed}s!`); break; }
    }
  }

  log('\n📡 Timeline:');
  for (const line of tl) console.log('  ' + line);
  await browser.close();
}

main().catch(e => { console.error(e); process.exit(1); });
