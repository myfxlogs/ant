import { test, expect, type Page } from '@playwright/test';

/**
 * QC-CACHE-LEAK 实测：A 登出 → B 登录（全程 SPA 客户端导航，无整页刷新），
 * Dashboard 账户列表必须立即显示 B 的账户，不得残留 A 的。
 *
 * 基准（API 核实 2026-08-16）：
 *   A xianhua.chan@gmail.com → 2 个交易账户（登录时动态捕获）
 *   B admin@1.com           → 3 个：95172262 / 277259925 / 80057439
 */

const USER_A = { login: 'xianhua.chan@gmail.com', password: 'Abc123456...' };
const USER_B = { login: 'admin@1.com', password: '12345678' };
const B_LOGINS = ['95172262', '277259925', '80057439'];

async function uiLogin(page: Page, user: { login: string; password: string }) {
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1200);
  await page.locator('#login_login').fill(user.login);
  await page.locator('#login_password').fill(user.password);
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((u) => !u.pathname.includes('/login'), { timeout: 20_000 });
  await page.waitForTimeout(3000); // Dashboard 挂载 + fetchAccounts 完成
}

async function uiLogout(page: Page) {
  await page.locator('header .ant-avatar').first().click();
  await page.waitForTimeout(600);
  await page.locator('.ant-dropdown-menu-item-danger, .ant-dropdown-menu-item:has-text("退出"), .ant-dropdown-menu-item:has-text("Logout"), .ant-dropdown-menu-item:has-text("登出")').first().click();
  await page.waitForURL((u) => u.pathname.includes('/login'), { timeout: 20_000 });
  await page.waitForTimeout(1200);
}

/** DashboardAccountList 的 item.login 渲染为独立 span 数字串 */
async function visibleAccountLogins(page: Page): Promise<string[]> {
  const spans = await page.locator('main span, [class*="account"] span').allTextContents();
  const logins = spans
    .map((s) => s.trim())
    .filter((s) => /^\d{6,}$/.test(s));
  return [...new Set(logins)];
}

test('user switch: B must not see A accounts without manual refresh', async ({ page }) => {
  test.setTimeout(180_000);

  // 诊断：记录 bundle 指纹 + ListAccounts 调用时序
  const listAccountsCalls: string[] = [];
  page.on('request', (r) => {
    if (r.url().includes('ListAccounts')) {
      listAccountsCalls.push(`${new Date().toISOString().slice(11, 23)} ${r.url().split('/').pop()}`);
    }
  });

  // ── 1. A 登录，记录基线 ──
  await uiLogin(page, USER_A);
  const bundle = await page.evaluate(() =>
    Array.from(document.scripts).map((s) => s.src).filter((s) => s.includes('/assets/index')),
  );
  console.log('[diag] bundle:', bundle);
  const aLogins = await visibleAccountLogins(page);
  console.log('[diag] A sees:', aLogins.join(','), '| ListAccounts so far:', listAccountsCalls.length);
  await page.screenshot({ path: 'test-results/leak-1-A-dashboard.png', fullPage: true });
  expect(aLogins.length, `A 应有 2 个账户，实际 ${aLogins.join(',')}`).toBe(2);

  // ── 2. A 登出 ──
  await uiLogout(page);
  const callsAfterLogout = listAccountsCalls.length;
  console.log('[diag] after logout, ListAccounts total:', callsAfterLogout);

  // ── 3. B 登录（此后零整页刷新）──
  await uiLogin(page, USER_B);
  // 不做任何 reload/goto，等 UI 稳定
  await page.waitForTimeout(3000);
  const bLogins = await visibleAccountLogins(page);
  console.log('[diag] B sees:', bLogins.join(','), '| expected:', B_LOGINS.join(','));
  console.log('[diag] ListAccounts timeline:', JSON.stringify(listAccountsCalls, null, 2));
  const authStorage = await page.evaluate(() => localStorage.getItem('auth-storage'));
  console.log('[diag] auth-storage user:', authStorage?.slice(0, 200));
  await page.screenshot({ path: 'test-results/leak-2-B-dashboard.png', fullPage: true });

  // 核心断言：B 必须看到 B 的 3 个账户（顺序无关）
  expect(new Set(bLogins)).toEqual(new Set(B_LOGINS));

  // ── 4. 反向 B → A 再验一遍 ──
  await uiLogout(page);
  await uiLogin(page, USER_A);
  await page.waitForTimeout(3000);
  const aLoginsAgain = await visibleAccountLogins(page);
  console.log('[diag] A sees again:', aLoginsAgain.join(','));
  await page.screenshot({ path: 'test-results/leak-3-A-again.png', fullPage: true });
  expect(new Set(aLoginsAgain)).toEqual(new Set(aLogins));
});
