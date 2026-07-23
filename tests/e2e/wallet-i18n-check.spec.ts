import { test, expect, Page } from '@playwright/test';

async function login(page: Page) {
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1500);
  await page.locator('#login_login').fill('admin@1.com');
  await page.locator('#login_password').fill('12345678');
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
}

test.describe('Wallet page i18n check', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('Wallet page — no raw i18n keys visible', async ({ page }) => {
    await page.goto('/wallet', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);
    const bodyText = await page.locator('body').innerText();
    const matches = bodyText.match(/\b(wallet|common)\.[a-zA-Z]+\.[a-zA-Z]+/g);
    if (matches) console.log('Raw keys found:', matches);
    expect(matches).toBeNull();
    await page.screenshot({ path: 'tests/e2e/screenshots/wallet-page-i18n.png', fullPage: true });
  });
});
