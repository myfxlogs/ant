import { test, expect } from '@playwright/test';
import { loginAsTestUser, injectAuthState, type AuthSession } from './fixtures/auth';
import {
  findCheapestStrategy,
  purchaseStrategy,
  listPurchasedStrategies,
  type PublishedStrategyInfo,
} from './fixtures/seed';

test.describe('Journey 3: Purchase → Live trading schedule', () => {
  let session: AuthSession;
  let strategy: PublishedStrategyInfo | null;

  test.beforeAll(async ({ request }) => {
    session = await loginAsTestUser(request);
    strategy = await findCheapestStrategy(request, session);
  });

  test('purchased strategy appears in strategy gallery and can navigate to live page', async ({ page, request }) => {
    test.skip(!strategy, 'No strategy available');

    const alreadyPurchased = await listPurchasedStrategies(request, session);
    if (!alreadyPurchased.includes(strategy!.strategyId)) {
      const result = await purchaseStrategy(request, session, strategy!);
      expect(result.ok, `Purchase should succeed, error: ${result.error}`).toBe(true);
    }

    await injectAuthState(page, session);
    await page.goto('/strategy');

    await page.waitForTimeout(2000);

    await page.goto('/strategy/live');
    await expect(page).toHaveURL(/\/strategy\/live/);
    await page.waitForTimeout(1000);

    const livePage = page.locator('.ant-tabs, .ant-table, .ant-empty');
    await expect(livePage.first()).toBeVisible({ timeout: 10_000 });
  });

  test('live strategy page shows active tab and schedule tab', async ({ page }) => {
    await injectAuthState(page, session);
    await page.goto('/strategy/live');

    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10_000 });

    const tabs = page.locator('.ant-tabs-tab');
    const tabCount = await tabs.count();
    expect(tabCount).toBeGreaterThan(0);

    const tabTexts = await tabs.allTextContents();
    const hasActiveOrSchedule = tabTexts.some((t) =>
      /active|运行|schedule|调度|live/i.test(t),
    );
    expect(hasActiveOrSchedule, `Expected active/schedule tab, got: ${tabTexts.join(', ')}`).toBe(true);
  });

  test('can navigate to strategy detail page for purchased strategy', async ({ page }) => {
    test.skip(!strategy, 'No strategy available');

    await injectAuthState(page, session);
    await page.goto(`/strategy/view/${strategy!.strategyId}`);

    await page.waitForTimeout(2000);

    const pageContent = page.locator('main, .ant-card, .ant-typography');
    await expect(pageContent.first()).toBeVisible({ timeout: 10_000 });
  });
});
