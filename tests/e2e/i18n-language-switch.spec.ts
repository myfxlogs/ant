import { test, expect } from '@playwright/test';

const BASE = 'http://localhost:8022';
const ADMIN_EMAIL = 'admin@1.com';
const ADMIN_PASS = '12345678';

test.describe('E2E: i18n Language Switch', () => {

  test('Language dropdown visible and switchable', async ({ page }) => {
    // Login
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.locator('#login_login').fill(ADMIN_EMAIL);
    await page.locator('#login_password').fill(ADMIN_PASS);
    await page.locator('form button[type="submit"]').click();
    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
    await page.waitForTimeout(2000);

    // Find the language dropdown trigger (GlobalOutlined icon)
    const langTrigger = page.locator('.anticon-global').first();
    await expect(langTrigger).toBeVisible({ timeout: 10_000 });

    // Click to open dropdown
    await langTrigger.click();
    await page.waitForTimeout(500);

    // Verify language options are visible in dropdown
    const dropdown = page.locator('.ant-dropdown').last();
    await expect(dropdown).toBeVisible({ timeout: 5_000 });

    // Check that at least English and Chinese options exist
    const dropdownText = await dropdown.textContent();
    expect(dropdownText).toMatch(/English|英语|英文/i);
    expect(dropdownText).toMatch(/中文|Chinese|简体/i);

    // Click English option
    const englishOption = dropdown.locator('text=/English|英语|英文/i').first();
    await englishOption.click();
    await page.waitForTimeout(1000);

    // Verify the page text changed (look for common English UI text)
    const bodyText = await page.locator('body').textContent();
    // After switching to English, some common UI text should be in English
    expect(bodyText).toMatch(/Dashboard|Strategy|Trading|Wallet/i);

    // Switch back to Chinese
    await langTrigger.click();
    await page.waitForTimeout(500);
    const dropdown2 = page.locator('.ant-dropdown').last();
    const chineseOption = dropdown2.locator('text=/简体中文|简体/i').first();
    await chineseOption.click();
    await page.waitForTimeout(1000);

    // Verify Chinese text appears
    const bodyText2 = await page.locator('body').textContent();
    expect(bodyText2).toMatch(/仪表盘|策略|交易|钱包|工作台/i);
  });

  test('Login page i18n — default language renders correctly', async ({ page }) => {
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    // The login page should display form labels
    const loginInput = page.locator('#login_login');
    await expect(loginInput).toBeVisible({ timeout: 10_000 });
    const passwordInput = page.locator('#login_password');
    await expect(passwordInput).toBeVisible();

    // Submit button should be visible
    const submitBtn = page.locator('form button[type="submit"]');
    await expect(submitBtn).toBeVisible();
  });
});
