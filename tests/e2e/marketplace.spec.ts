import { test, expect, type Page } from '@playwright/test';

const _env = (globalThis as any).process?.env ?? {};
const TEST_USER = _env.TEST_USER || 'admin@1.com';
const TEST_PASSWORD = _env.TEST_PASSWORD || '12345678';
const BASE = 'http://localhost:8022';

async function loginViaAPI(): Promise<string> {
  const resp = await fetch(`${BASE}/ant.v1.AuthService/Login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login: TEST_USER, password: TEST_PASSWORD }),
  });
  if (!resp.ok) throw new Error(`Login API failed: ${resp.status}`);
  const data = await resp.json();
  return data.accessToken;
}

async function login(page: Page) {
  await page.goto('/login', { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);
  await page.locator('#login_login').fill(TEST_USER);
  await page.locator('#login_password').fill(TEST_PASSWORD);
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
}

// ── Test 1: ListBundles works without authentication (regression: was 401) ──

test('ListBundles returns 200 without auth token', async () => {
  const resp = await fetch(`${BASE}/ant.v1.MarketplaceService/ListBundles`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({}),
  });
  expect(resp.status).toBe(200);
  const data = await resp.json();
  expect(typeof data).toBe('object');
  expect(data).not.toBeNull();
});

// ── Test 2: GetProviderEarnings works with auth (regression: was 500) ──

test('GetProviderEarnings returns 200 with auth token', async () => {
  const token = await loginViaAPI();
  const resp = await fetch(`${BASE}/ant.v1.MarketplaceService/GetProviderEarnings`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({}),
  });
  expect(resp.status).toBe(200);
  const data = await resp.json();
  expect(data).toHaveProperty('totalEarnings');
  expect(data).toHaveProperty('availableBalance');
  expect(data).toHaveProperty('pendingWithdrawal');
  expect(data).toHaveProperty('lifetimeWithdrawn');
  expect(data).toHaveProperty('activeStrategies');
  expect(data).toHaveProperty('pendingSettlement');
});

// ── Test 3: GetProviderEarnings returns 401 without auth ──

test('GetProviderEarnings returns 401 without auth token', async () => {
  const resp = await fetch(`${BASE}/ant.v1.MarketplaceService/GetProviderEarnings`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({}),
  });
  expect(resp.status).toBe(401);
});

// ── Test 4: Marketplace page loads and i18n renders correctly ──

test('Marketplace page: tabs render with localized text, not raw keys', async ({ page }) => {
  await login(page);
  await page.goto('/marketplace', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(3000);

  // Verify tab labels are localized — check for known localized text, not raw keys
  const tabLabels = page.locator('.ant-tabs-tab .ant-tabs-tab-btn');
  await expect(tabLabels.first()).toBeVisible({ timeout: 15_000 });
  const tabCount = await tabLabels.count();
  expect(tabCount).toBeGreaterThan(0);

  for (let i = 0; i < tabCount; i++) {
    const text = await tabLabels.nth(i).textContent();
    expect(text).toBeTruthy();
    // Raw i18n keys contain dots like "marketplace.tabs.purchases"
    expect(text!).not.toMatch(/^[a-z]+\.[a-z]+\./);
  }

  await page.screenshot({ path: 'screenshots/marketplace-tabs.png', fullPage: false });
});

// ── Test 5: My Purchases tab renders without raw i18n keys ──

test('Marketplace My Purchases tab: no raw i18n keys visible', async ({ page }) => {
  await login(page);
  await page.goto('/marketplace', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);

  // Click the "My Purchases" tab — match by localized text (en/zh)
  const purchasesTab = page.locator('.ant-tabs-tab').filter({ hasText: /My Purchases|我的购买/i }).first();
  await expect(purchasesTab).toBeVisible({ timeout: 10_000 });
  await purchasesTab.click();
  await page.waitForTimeout(2000);

  // Check page body for raw i18n keys (use entire body, not just tab pane)
  const bodyText = await page.locator('body').textContent() || '';
  expect(bodyText).not.toContain('marketplace.purchases.');
  expect(bodyText).not.toContain('strategy.backtestHistory.');
  expect(bodyText).not.toContain('[object Object]');

  await page.screenshot({ path: 'screenshots/marketplace-purchases.png', fullPage: false });
});

// ── Test 6: Author Center tab loads without error ──

test('Marketplace Author Center tab: earnings data loads without error', async ({ page }) => {
  await login(page);
  await page.goto('/marketplace', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);

  // Click the "Author Center" tab
  const authorTab = page.locator('.ant-tabs-tab').filter({ hasText: /Author Center|作者中心/i }).first();
  await expect(authorTab).toBeVisible({ timeout: 10_000 });
  await authorTab.click();
  await page.waitForTimeout(3000);

  // Verify no error message is displayed
  const errorMsg = page.locator('.ant-message-error, .ant-result-error, .ant-alert-error');
  const errorVisible = await errorMsg.first().isVisible().catch(() => false);
  expect(errorVisible).toBeFalsy();

  // Check body text doesn't contain error indicators
  const bodyText = await page.locator('body').textContent() || '';
  expect(bodyText).not.toContain('Failed to load earnings');
  expect(bodyText).not.toContain('Internal Server Error');

  await page.screenshot({ path: 'screenshots/marketplace-author.png', fullPage: false });
});

// ── Test 7: Bundles tab loads without auth error ──

test('Marketplace Bundles tab: loads without 401 error', async ({ page }) => {
  await login(page);
  await page.goto('/marketplace', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(2000);

  // Click the "Bundles" tab — may not exist if no bundles tab is rendered
  const bundlesTab = page.locator('.ant-tabs-tab').filter({ hasText: /Bundles|捆绑/i }).first();
  const hasBundlesTab = await bundlesTab.isVisible().catch(() => false);
  
  if (hasBundlesTab) {
    await bundlesTab.click();
    await page.waitForTimeout(3000);

    // Verify no auth error message is displayed
    const errorMsg = page.locator('.ant-message-error, .ant-result-error');
    const errorVisible = await errorMsg.first().isVisible().catch(() => false);
    expect(errorVisible).toBeFalsy();

    const bodyText = await page.locator('body').textContent() || '';
    expect(bodyText).not.toContain('sign in to continue');
    expect(bodyText).not.toContain('Please sign in');
    expect(bodyText).not.toContain('unauthenticated');
  }

  // Also verify via API that ListBundles works without auth
  const resp = await fetch(`${BASE}/ant.v1.MarketplaceService/ListBundles`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({}),
  });
  expect(resp.status).toBe(200);

  await page.screenshot({ path: 'screenshots/marketplace-bundles.png', fullPage: false });
});

// ── Test 4: ListPublished works without auth (public browsing) ──

test('ListPublished returns 200 without auth token', async () => {
  const resp = await fetch(`${BASE}/ant.v1.MarketplaceService/ListPublished`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ limit: 10, offset: 0 }),
  });
  expect(resp.status).toBe(200);
  const data = await resp.json();
  expect(data).toHaveProperty('strategies');
  expect(Array.isArray(data.strategies)).toBeTruthy();
});
