import { chromium } from 'playwright';
const B = 'http://localhost:8022';
async function main() {
  const b = await chromium.launch({headless: true, executablePath: '/snap/bin/chromium', args: ['--no-sandbox','--disable-setuid-sandbox']});
  const p = await b.newPage();
  await p.goto(`${B}/login`, {waitUntil:'domcontentloaded',timeout:10000});
  await p.fill('#login_email','admin@1.com');
  await p.fill('#login_password','12345678');
  const rp = p.waitForResponse(r=>r.url().includes('Login'),{timeout:10000});
  await p.click('button[type="submit"]');
  const lr = await rp;
  const tkn = (await lr.json()).accessToken;

  for (const acct of ['0eaf0332-9699-4aaf-8536-fc45770a9977','11c9b48d-b9c9-4b07-a6fb-30826d0cb925']) {
    const r = await p.evaluate(async ({tkn,id}:any)=>{
      const r=await fetch('/ant.v1.AccountService/GetAccount',{method:'POST',headers:{'Content-Type':'application/json','Authorization':`Bearer ${tkn}`,'Connect-Protocol-Version':'1'},body:JSON.stringify({id})});
      return r.json();
    },{tkn,id:acct});
    console.log(`${acct.slice(0,8)}... isInvestor=${r.isInvestor} mode=${r.isInvestor?'Investor':'Trader'}`);
  }
  await b.close();
}
main();
