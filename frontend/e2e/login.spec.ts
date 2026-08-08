import { test, expect } from '@playwright/test';
import { E2E_TEST_USER } from './fixtures/auth';

test.describe('Journey 1: Login → Dashboard', () => {
  test('UI login succeeds and redirects to dashboard', async ({ page }) => {
    await page.goto('/login');

    await page.getByPlaceholder(/email|account/i).fill(E2E_TEST_USER.email);
    await page.getByPlaceholder(/password/i).fill(E2E_TEST_USER.password);
    await page.getByRole('button', { name: /sign in|login/i }).click();

    await expect(page).toHaveURL(/\/$|\/dashboard/i, { timeout: 10_000 });

    const authStorage = await page.evaluate(() => localStorage.getItem('auth-storage'));
    expect(authStorage).toBeTruthy();
    const parsed = JSON.parse(authStorage!);
    expect(parsed.state.accessToken).toBeTruthy();
    expect(parsed.state.user?.email).toBe(E2E_TEST_USER.email);
  });

  test('invalid credentials show error', async ({ page }) => {
    await page.goto('/login');

    await page.getByPlaceholder(/email|account/i).fill('wrong@user.com');
    await page.getByPlaceholder(/password/i).fill('wrongpassword');
    await page.getByRole('button', { name: /sign in|login/i }).click();

    await expect(page.locator('.ant-message-notice')).toBeVisible({ timeout: 5000 });
    await expect(page).toHaveURL(/\/login/);
  });

  test('unauthenticated user sees landing page at root', async ({ page }) => {
    await page.goto('/');
    await page.waitForTimeout(3000);
    await expect(page.locator('body')).toContainText(/AI-Powered|MT4|MT5|Strategy/i, { timeout: 10_000 });
  });

  test('login page is accessible directly when unauthenticated', async ({ page }) => {
    await page.goto('/login');
    await expect(page.getByPlaceholder(/email|account/i)).toBeVisible({ timeout: 10_000 });
    await expect(page.getByPlaceholder(/password/i)).toBeVisible({ timeout: 10_000 });
  });
});
