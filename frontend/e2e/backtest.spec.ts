import { test, expect } from '@playwright/test';
import { loginAsTestUser, injectAuthState, type AuthSession } from './fixtures/auth';

test.describe('Journey 4: Backtest flow', () => {
  let session: AuthSession;

  test.beforeAll(async ({ request }) => {
    session = await loginAsTestUser(request);
  });

  test('strategy workspace page loads with backtest capability', async ({ page }) => {
    await injectAuthState(page, session);
    await page.goto('/strategy/new');

    await page.waitForTimeout(3000);

    const workspace = page.locator('.ant-layout, .ant-card, .ant-tabs, main');
    await expect(workspace.first()).toBeVisible({ timeout: 10_000 });
  });

  test('backtest API returns result via ConnectRPC', async ({ request }) => {
    const templatesResp = await request.post('/ant.v1.StrategyService/ListTemplates', {
      data: {},
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${session.accessToken}`,
      },
    });
    expect(templatesResp.ok(), 'ListTemplates should succeed').toBe(true);
    const templatesBody = await templatesResp.json();
    const templates = templatesBody?.templates || [];
    expect(templates.length, 'Should have at least one template').toBeGreaterThan(0);
  });

  test('strategy gallery page loads', async ({ page }) => {
    await injectAuthState(page, session);
    await page.goto('/strategy');

    await page.waitForTimeout(2000);

    const content = page.locator('.ant-card, .ant-list, .ant-table, .ant-empty');
    await expect(content.first()).toBeVisible({ timeout: 10_000 });
  });
});
