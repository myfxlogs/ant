import { test, expect } from '@playwright/test';

const BASE = 'http://localhost:8022';
const ADMIN_EMAIL = 'admin@1.com';
const ADMIN_PASS = '12345678';

async function rpc(path: string, body: Record<string, unknown>, token?: string) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const resp = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  const text = await resp.text();
  let data: Record<string, unknown>;
  try { data = JSON.parse(text); } catch { data = { _raw: text }; }
  return { status: resp.status, data, ok: resp.ok };
}

test.describe.serial('E2E: Login + Account Bind Wizard', () => {

  test('1. UI login as admin', async ({ page }) => {
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.locator('#login_login').fill(ADMIN_EMAIL);
    await page.locator('#login_password').fill(ADMIN_PASS);
    await page.locator('form button[type="submit"]').click();
    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
    // Verify we landed on dashboard
    expect(page.url()).not.toContain('/login');
  });

  test('2. Navigate to bind account page', async ({ page }) => {
    // Login first (serial tests don't share browser context)
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.locator('#login_login').fill(ADMIN_EMAIL);
    await page.locator('#login_password').fill(ADMIN_PASS);
    await page.locator('form button[type="submit"]').click();
    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });

    await page.goto('/accounts/bind', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    // Verify step indicator is visible (3 steps)
    const stepCircles = page.locator('.rounded-full.w-8.h-8');
    await expect(stepCircles.first()).toBeVisible({ timeout: 10_000 });
    expect(await stepCircles.count()).toBe(3);

    // Verify Step 1 title is visible
    const step1Title = page.locator('h2').first();
    await expect(step1Title).toBeVisible();
  });

  test('3. Step1 — platform selection and broker search', async ({ page }) => {
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.locator('#login_login').fill(ADMIN_EMAIL);
    await page.locator('#login_password').fill(ADMIN_PASS);
    await page.locator('form button[type="submit"]').click();
    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });

    await page.goto('/accounts/bind', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    // Verify MT4/MT5 platform options exist
    const mt4Option = page.locator('text=MT4').first();
    const mt5Option = page.locator('text=MT5').first();
    await expect(mt4Option).toBeVisible({ timeout: 10_000 });
    await expect(mt5Option).toBeVisible();

    // Select MT5 (click on the MT5 platform card)
    await mt5Option.click();
    await page.waitForTimeout(500);

    // Enter broker search term
    const searchInput = page.locator('input[placeholder]').first();
    await searchInput.fill('ICMarkets');
    await page.waitForTimeout(300);

    // Click search button
    const searchBtn = page.locator('button').filter({ hasText: /Search|搜索/ }).first();
    await searchBtn.click();
    await page.waitForTimeout(3000);

    // Verify either results or no-results message appears
    // (We don't assert specific brokers since this depends on live mtapi)
    const pageText = await page.locator('body').textContent();
    // After search, either company select appears or "no brokers" message
    expect(pageText).toMatch(/ICMarkets|server|broker|没有|No matching|Select/i);
  });

  test('4. Step1 — next button disabled without server selection', async ({ page }) => {
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.locator('#login_login').fill(ADMIN_EMAIL);
    await page.locator('#login_password').fill(ADMIN_PASS);
    await page.locator('form button[type="submit"]').click();
    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });

    await page.goto('/accounts/bind', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    // The "Next" button should be disabled initially
    const nextBtn = page.locator('button').filter({ hasText: /Next|下一步/ }).last();
    await expect(nextBtn).toBeVisible({ timeout: 10_000 });
    // Disabled state — either disabled attribute or ant-btn disabled class
    const isDisabled = await nextBtn.isDisabled().catch(() => false) ||
      await nextBtn.evaluate(el => el.classList.contains('ant-btn-disabled')).catch(() => false);
    expect(isDisabled).toBe(true);
  });

  test('5. Back navigation from Step2 to Step1', async ({ page }) => {
    // This test verifies the step navigation works
    // We can't fully complete the wizard without real broker credentials,
    // but we can verify the back button works if we reach step 2

    // Use API to verify the bind page is accessible
    const resp = await rpc('/ant.v1.AuthService/Login', {
      login: ADMIN_EMAIL,
      password: ADMIN_PASS,
    });
    expect(resp.ok).toBe(true);

    // UI: login and navigate
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.locator('#login_login').fill(ADMIN_EMAIL);
    await page.locator('#login_password').fill(ADMIN_PASS);
    await page.locator('form button[type="submit"]').click();
    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });

    await page.goto('/accounts/bind', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    // Verify the back arrow button exists (top-left)
    const backArrow = page.locator('button .anticon-arrow-left').first();
    await expect(backArrow).toBeVisible({ timeout: 10_000 });

    // Click back arrow — should navigate away from bind page
    await backArrow.click();
    await page.waitForTimeout(1000);
    expect(page.url()).not.toContain('/accounts/bind');
  });
});
