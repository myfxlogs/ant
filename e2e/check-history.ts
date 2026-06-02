import { chromium } from 'playwright';

async function main() {
  const b = await chromium.launch({headless: true, executablePath: '/snap/bin/chromium', args: ['--no-sandbox','--disable-setuid-sandbox']});
  const p = await b.newPage();
  await p.goto('http://localhost:8022/login',{waitUntil:'domcontentloaded',timeout:10000});
  await p.fill('#login_email','admin@1.com');
  await p.fill('#login_password','12345678');
  const rp = p.waitForResponse(r=>r.url().includes('Login'),{timeout:10000});
  await p.click('button[type="submit"]');
  const lr = await rp; const tkn = (await lr.json()).accessToken;

  // Get recent trades
  const r = await p.evaluate(async({tkn}:any)=>{
    const r=await fetch('/ant.v1.AnalyticsService/GetRecentTrades',{method:'POST',headers:{'Content-Type':'application/json','Authorization':`Bearer ${tkn}`,'Connect-Protocol-Version':'1'},body:JSON.stringify({accountId:'70dbefac-d98b-47a4-81e8-d32ab7d4f46d',page:1,pageSize:20})});
    return r.json();
  },{tkn});

  console.log(`Total trades: ${r.total}`);
  console.log(`Sample (first 20):`);
  for (const t of (r.trades||[]).slice(0,20)) {
    console.log(`  ticket=${t.ticket} symbol="${t.symbol}" type="${t.type}" openTime="${t.openTime}" closeTime="${t.closeTime}" volume=${t.volume} profit=${t.profit}`);
  }

  await b.close();
}
main();
