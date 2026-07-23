import { test, expect } from '@playwright/test';
import { login } from './helpers/auth';

test.describe('Wallet page i18n check', () => {
  test.beforeEach(async ({ page }) => {
    await login(page, 'admin@1.com', '12345678');
  });

  test('Wallet page — no raw i18n keys visible', async ({ page }) => {
    await page.goto('http://localhost:8022/wallet');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    const bodyText = await page.locator('body').innerText();
    // Check for raw i18n key patterns (wallet.* or common.*)
    const rawKeyPattern = /\b(wallet|common)\.[a-zA-Z]+\.[a-zA-Z]+/g;
    const matches = bodyText.match(rawKeyPattern);
    if (matches) {
      console.log('Raw keys found:', matches);
    }
    expect(matches).toBeNull();

    // Take screenshot
    await page.screenshot({ path: 'tests/e2e/screenshots/wallet-page-i18n.png', fullPage: true });
  });
});
