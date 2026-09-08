// 真实用户完整旅程（xianhua.chan，真实数据）
import { chromium } from 'playwright';
import { mkdirSync } from 'node:fs';
const BASE = 'http://localhost:8022';
const SHOT = '/tmp/real-shots';
mkdirSync(SHOT, { recursive: true });

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'zh-CN' });
const page = await ctx.newPage();
const errors = [];
page.on('pageerror', (e) => errors.push('pageerror: ' + e.message.slice(0, 200)));
let shotNo = 0;
async function shot(name) { await page.screenshot({ path: `${SHOT}/${String(++shotNo).padStart(2, '0')}-${name}.png` }); console.log('SHOT:', name); }
async function step(name, fn) {
  try { await fn(); console.log('STEP OK:', name); }
  catch (e) { console.log('STEP FAIL:', name, '-', String(e).slice(0, 160)); await page.screenshot({ path: `${SHOT}/FAIL-${name}.png` }).catch(() => {}); }
}

// 1) 真实 UI 登录
await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded' });
await page.waitForTimeout(1200);
await step('UI 登录', async () => {
  await page.getByPlaceholder('邮箱/账号').fill('xianhua.chan@gmail.com');
  await page.getByPlaceholder('密码').fill(process.env.REAL_PASS || '');
  await page.getByRole('button', { name: /立即登录/ }).click();
  await page.waitForTimeout(2500);
  if (page.url().includes('/login')) throw new Error('登录未成功');
});

// 2) 进入工作台 + 关新手引导
await page.goto(`${BASE}/strategy/new`, { waitUntil: 'domcontentloaded' });
await page.waitForTimeout(2500);
await step('关闭新手引导', async () => {
  const nextBtn = page.getByText('下一步', { exact: true }).first();
  for (let i = 0; i < 8; i++) {
    if (!(await nextBtn.isVisible().catch(() => false))) break;
    await nextBtn.click(); await page.waitForTimeout(250);
  }
  const fin = page.getByText('结束导览', { exact: true }).first();
  if (await fin.isVisible().catch(() => false)) { await fin.click(); await page.waitForTimeout(500); }
});
await shot('1-工作台落地');

// 3) 展开我的策略 → 打开聪明的牛马
await step('展开我的策略', async () => {
  await page.getByRole('button', { name: /我的策略/ }).first().click();
  await page.waitForTimeout(500);
});
await shot('2-我的策略列表');
await step('打开策略', async () => {
  await page.getByText('聪明的牛马', { exact: true }).first().click();
  await page.waitForTimeout(2500);
});
await shot('3-策略打开');

// 4) 粘性导入态回归：导入面板打开 → 点我的策略 → 应回编辑器
await step('粘性导入态回归', async () => {
  await page.getByRole('button', { name: /新建策略/ }).first().click();
  await page.waitForTimeout(400);
  await page.getByText('导入 MQL', { exact: true }).first().click();
  await page.waitForTimeout(800);
  await page.getByRole('button', { name: /我的策略/ }).first().click();
  await page.waitForTimeout(500);
  await page.getByText('聪明的牛马', { exact: true }).first().click();
  await page.waitForTimeout(2000);
  const importVisible = await page.getByText('导入 MQL', { exact: true }).first().isVisible().catch(() => false);
  const editorBack = await page.locator('.monaco-editor, .cm-editor').count();
  if (importVisible) throw new Error('导入面板仍粘滞显示');
  if (editorBack === 0) throw new Error('未回到编辑器');
});
await shot('4-粘性回归通过');

// 5) 展开回测历史 → 主区面板（真实 20 条）
await step('回测历史主区', async () => {
  await page.getByRole('button', { name: /回测历史/ }).first().click();
  await page.waitForTimeout(1500);
});
await shot('5-回测历史主区');
const runCount = await page.locator('[data-testid^="history-run-"]').count();
console.log('HISTORY RUNS RENDERED:', runCount);

// 6) 点第一条历史
await step('打开一条历史', async () => {
  const first = page.locator('[data-testid^="history-run-"]').first();
  if (await first.count()) await first.click();
  await page.waitForTimeout(2000);
});
await shot('6-历史条目加载');

// 7) 切回新建策略分区
await step('切回新建策略分区', async () => {
  await page.getByRole('button', { name: /新建策略/ }).first().click();
  await page.waitForTimeout(1000);
  if (!(await page.getByTestId('new-source-ai').isVisible().catch(() => false))) throw new Error('来源面板未出现');
});
await shot('7-新建策略来源');

console.log('PAGE ERRORS:', errors.length);
errors.slice(0, 6).forEach((e) => console.log('  ', e));
await browser.close();
