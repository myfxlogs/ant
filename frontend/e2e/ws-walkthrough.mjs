// 真实浏览器走查：策略工作台分区导航 + 来源选择 + 回测历史。
// 账号：e2e 专用种子账号。登录态注入与 authStore 现行持久化一致。
import { chromium } from 'playwright';

const BASE = 'http://localhost:8022';
const SHOT_DIR = '/tmp/ws-shots';

const browser = await chromium.launch({ headless: true });
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: 'zh-CN' });
const page = await ctx.newPage();

const errors = [];
page.on('pageerror', (e) => errors.push('pageerror: ' + e.message.slice(0, 160)));
page.on('console', (m) => { if (m.type() === 'error') errors.push('console: ' + m.text().slice(0, 160)); });

let shot = 0;
async function step(name, fn) {
  try {
    await fn();
    console.log('STEP OK:', name);
    await page.screenshot({ path: `${SHOT_DIR}/${String(++shot).padStart(2, '0')}-${name}.png` });
  } catch (e) {
    console.log('STEP FAIL:', name, '-', String(e).slice(0, 200));
    await page.screenshot({ path: `${SHOT_DIR}/fail-${name}.png` }).catch(() => {});
  }
}

// 1) API 登录
const loginResp = await page.request.post(`${BASE}/ant.v1.AuthService/Login`, {
  data: { login: 'e2e@test.com', password: 'E2etest123!' },
  headers: { 'Content-Type': 'application/json' },
});
if (!loginResp.ok()) { console.log('LOGIN FAILED', loginResp.status()); process.exit(1); }
const session = await loginResp.json();
console.log('LOGIN OK:', session.user?.email);

// 2) 注入登录态（与 authStore 现行持久化一致：token 单独存 localStorage）
await ctx.addInitScript((auth) => {
  localStorage.setItem('auth-access-token', JSON.stringify(auth.accessToken));
  localStorage.setItem('auth-remember-me', 'true');
  localStorage.setItem('auth-storage', JSON.stringify({
    state: { user: { id: auth.userId, email: auth.email, username: auth.username, role: auth.role }, _rememberMe: true },
    version: 0,
  }));
}, { userId: session.userId, email: session.user.email, username: session.user.username, role: session.user.role, accessToken: session.accessToken });

await page.goto(`${BASE}/strategy/new`, { waitUntil: 'domcontentloaded' });
await page.waitForTimeout(2500);

// 0) 关闭新手引导浮层（会拦截点击）：连点"下一步"直到消失
await step('关闭新手引导', async () => {
  const nextBtn = page.getByText('下一步', { exact: true }).first();
  for (let i = 0; i < 8; i++) {
    if (!(await nextBtn.isVisible().catch(() => false))) break;
    await nextBtn.click();
    await page.waitForTimeout(250);
  }
  const finish = page.getByText('结束导览', { exact: true }).first();
  if (await finish.isVisible().catch(() => false)) {
    await finish.click();
    await page.waitForTimeout(500);
  }
});

// 3) 展开新建策略分区 → 4 来源项（中文）
await step('展开新建策略分区', async () => {
  await page.getByRole('button', { name: /新建策略/ }).first().click();
  await page.waitForTimeout(400);
  for (const label of ['AI 生成', '手动编写', '导入 MQL', '使用模板']) {
    if (!(await page.getByText(label, { exact: true }).first().isVisible().catch(() => false))) {
      throw new Error(`来源项缺失: ${label}`);
    }
  }
});

// 4) 手动编写 → 编辑器
await step('手动编写 → 空白编辑器', async () => {
  await page.getByText('手动编写', { exact: true }).first().click();
  await page.waitForTimeout(1200);
  const n = await page.locator('.monaco-editor, .cm-editor, textarea').count();
  if (n === 0) throw new Error('编辑器未出现（落在空态页）');
});

// 5) 新建策略 → AI 生成 → AI 面板
await step('AI 生成 → AI 面板', async () => {
  await page.getByRole('button', { name: /新建策略/ }).first().click();
  await page.waitForTimeout(400);
  await page.getByText('AI 生成', { exact: true }).first().click();
  await page.waitForTimeout(1200);
});

// 6) 关键回归：AI 面板开着时点分区头 → 应回到来源选择卡
await step('AI 打开时切回新建策略', async () => {
  await page.getByRole('button', { name: /新建策略/ }).first().click();
  await page.waitForTimeout(800);
  if (!(await page.getByTestId('new-source-ai').isVisible().catch(() => false))) {
    throw new Error('AI 面板未让位给来源选择面板');
  }
});

// 7) 来源卡：导入 MQL → 导入面板
await step('来源卡导入 MQL → 导入面板', async () => {
  await page.getByTestId('new-source-import').click();
  await page.waitForTimeout(1000);
  if (!(await page.getByText(/MQL4|MQL5|导入|源码/).first().isVisible().catch(() => false))) {
    throw new Error('导入面板未出现');
  }
});

// 8) 展开回测历史 → 主区历史面板（真实 20 条数据，protobuf 时间）
await step('回测历史主区面板', async () => {
  await page.getByRole('button', { name: /回测历史/ }).first().click();
  await page.waitForTimeout(1200);
  const empty = await page.getByText(/暂无回测记录/).count();
  const list = await page.locator('[data-testid^="history-run-"]').count();
  if (empty === 0 && list === 0) throw new Error('历史面板既无空态也无列表');
});

// 9) 我的策略分区 → 编辑器
await step('我的策略分区 → 编辑器', async () => {
  await page.getByRole('button', { name: /我的策略/ }).first().click();
  await page.waitForTimeout(800);
});

console.log('PAGE ERRORS:', errors.length);
errors.slice(0, 8).forEach((e) => console.log('  ', e));
await browser.close();
