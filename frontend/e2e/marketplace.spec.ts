import { test, expect } from '@playwright/test';
import { loginAsTestUser, injectAuthState, type AuthSession } from './fixtures/auth';
import {
  findCheapestStrategy,
  purchaseStrategy,
  listPurchasedStrategies,
  type PublishedStrategyInfo,
} from './fixtures/seed';

test.describe('Journey 2: Marketplace browse → purchase → verify', () => {
  let session: AuthSession;
  let strategy: PublishedStrategyInfo | null;

  test.beforeAll(async ({ request }) => {
    session = await loginAsTestUser(request);
    strategy = await findCheapestStrategy(request, session);
  });

  test('marketplace page loads with strategy listings', async ({ page }) => {
    await injectAuthState(page, session);
    await page.goto('/marketplace');

    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10_000 });

    await page.waitForTimeout(2000);

    const cards = page.locator('.ant-card');
    const count = await cards.count();
    expect(count).toBeGreaterThan(0);
  });

  test('purchase cheapest strategy and verify in subscriptions', async ({ page, request }) => {
    test.skip(!strategy, 'No strategy available in marketplace');

    const alreadyPurchased = await listPurchasedStrategies(request, session);
    if (!alreadyPurchased.includes(strategy!.strategyId)) {
      const result = await purchaseStrategy(request, session, strategy!);
      expect(result.ok, `Purchase should succeed, error: ${result.error}`).toBe(true);
    }

    const verifyPurchased = await listPurchasedStrategies(request, session);
    expect(verifyPurchased).toContain(strategy!.strategyId);

    await injectAuthState(page, session);
    await page.goto('/marketplace');

    await expect(page.locator('.ant-tabs')).toBeVisible({ timeout: 10_000 });

    const purchasesTab = page.getByRole('tab', { name: /我的购买|purchases|my.*purchase/i });
    if (await purchasesTab.count() > 0) {
      await purchasesTab.click();
      await page.waitForTimeout(1000);
    }
  });
});
