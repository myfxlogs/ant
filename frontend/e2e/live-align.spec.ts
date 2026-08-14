import { test, expect, type APIRequestContext } from '@playwright/test';

const USER = { email: 'xianhua.chan@gmail.com', password: 'Abc123456...' };

async function apiLogin(request: APIRequestContext): Promise<{ accessToken: string; userId: string; username: string; role: string }> {
  const resp = await request.post('/ant.v1.AuthService/Login', {
    data: { login: USER.email, password: USER.password },
    headers: { 'Content-Type': 'application/json' },
  });
  if (!resp.ok()) {
    throw new Error(`Login failed: ${resp.status()} ${await resp.text()}`);
  }
  const body = await resp.json();
  return {
    accessToken: body.accessToken,
    userId: body.user.id,
    username: body.user.username,
    role: body.user.role,
  };
}

test('positions table Symbol column aligns with Positions tab label', async ({ page, request }) => {
  const session = await apiLogin(request);
  // Set auth state via evaluate after initial page load, then reload
  await page.goto('/');
  await page.evaluate((auth) => {
    const state = {
      state: {
        user: { id: auth.userId, email: auth.email, username: auth.username, role: auth.role },
        accessToken: auth.accessToken,
        isAuthenticated: true,
        _hasHydrated: true,
        _rememberMe: false,
      },
      version: 0,
    };
    localStorage.setItem('auth-storage', JSON.stringify(state));
  }, { ...session, email: USER.email });
  await page.goto('/strategy/live');
  await page.waitForSelector('.ant-table-row', { timeout: 30_000 });
  await page.waitForTimeout(2000);
  await page.screenshot({ path: 'e2e/screenshots/live-page.png' });

  // Click the expand icon on the first row
  const expandBtn = page.locator('.ant-table-cell span[role="img"]').first();
  await expect(expandBtn).toBeVisible({ timeout: 15_000 });
  await expandBtn.click();
  await page.waitForTimeout(3000);

  // Get the Positions tab label position
  const positionsTab = page.locator('[data-node-key="positions"] .ant-tabs-tab-btn');
  await expect(positionsTab).toBeVisible();
  const tabBox = await positionsTab.boundingBox();
  expect(tabBox).not.toBeNull();

  // Get the Symbol column header position in the positions table
  const symbolHeader = page.locator('.ant-tabs-tabpane-active .ant-table-thead th').first();
  await expect(symbolHeader).toBeVisible();
  const headerBox = await symbolHeader.boundingBox();
  expect(headerBox).not.toBeNull();

  // The tab label and the Symbol header should start at approximately the same x position
  const tabX = tabBox!.x;
  const headerX = headerBox!.x;
  const diff = Math.abs(tabX - headerX);
  console.log(`Tab label x=${tabX}, Symbol header x=${headerX}, diff=${diff}`);
  expect(diff).toBeLessThan(3);

  // Take screenshot for visual verification
  await page.screenshot({ path: 'e2e/screenshots/live-positions-align.png', fullPage: false });
});
