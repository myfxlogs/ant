import { test, expect } from '@playwright/test';

const BASE = 'http://localhost:8022';

const CREDENTIALS = {
  login: 'admin@1.com',
  password: '12345678',
};

// All major app pages to test
const PAGES = [
  { path: '/', name: 'Home/Dashboard', requiresAuth: true },
  { path: '/login', name: 'Login', requiresAuth: false },
  { path: '/dashboard', name: 'Dashboard', requiresAuth: true },
  { path: '/accounts', name: 'Accounts', requiresAuth: true },
  { path: '/strategy', name: 'Strategy List', requiresAuth: true },
  { path: '/marketplace', name: 'Marketplace', requiresAuth: true },
  { path: '/trading', name: 'Trading', requiresAuth: true },
  { path: '/wallet', name: 'Wallet', requiresAuth: true },
  { path: '/ai', name: 'AI Settings', requiresAuth: true },
  { path: '/admin', name: 'Admin Dashboard', requiresAuth: true },
  { path: '/admin/users', name: 'Admin Users', requiresAuth: true },
  { path: '/admin/accounts', name: 'Admin Accounts', requiresAuth: true },
  { path: '/admin/strategies', name: 'Admin Strategies', requiresAuth: true },
  { path: '/admin/logs', name: 'Admin Logs', requiresAuth: true },
  { path: '/admin/config', name: 'Admin Config', requiresAuth: true },
  { path: '/admin/wallet', name: 'Admin Wallet', requiresAuth: true },
  { path: '/admin/trading', name: 'Admin Trading Monitor', requiresAuth: true },
  { path: '/analytics', name: 'Analytics', requiresAuth: true },
  { path: '/settings', name: 'Settings', requiresAuth: true },
  { path: '/notifications', name: 'Notifications', requiresAuth: true },
];

async function loginViaAPI(page) {
  const resp = await page.request.post(`${BASE}/ant.v1.AuthService/Login`, {
    headers: { 'Content-Type': 'application/json' },
    data: CREDENTIALS,
  });
  const body = await resp.json();
  const token = body.accessToken;

  // Inject token into localStorage/sessionStorage for the app
  await page.addInitScript((t) => {
    window.__TEST_TOKEN = t;
  }, token);

  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' });

  // Set the token in the app's auth store via localStorage
  await page.evaluate((t) => {
    // The app reads token from in-memory store, but we can set it via the login flow
    window.localStorage.setItem('alphaforge_test_token', t);
  }, token);

  // Login via UI
  await page.locator('#login_login').fill(CREDENTIALS.login);
  await page.locator('#login_password').fill(CREDENTIALS.password);
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((url) => !url.toString().includes('/login'), { timeout: 10000 }).catch(() => {});
  await page.waitForTimeout(2000);
}

test.describe('Full Page Load Test', () => {
  test.beforeEach(async ({ page }) => {
    await loginViaAPI(page);
  });

  for (const p of PAGES) {
    if (!p.requiresAuth) continue;

    test(`${p.name} (${p.path}) loads without errors`, async ({ page }) => {
      // Navigate to the page
      await page.goto(`${BASE}${p.path}`, { waitUntil: 'networkidle', timeout: 15000 }).catch(() => {});
      await page.waitForTimeout(3000);

      // Check for error boundaries / blank pages
      const bodyText = await page.locator('body').innerText().catch(() => '');
      const hasContent = bodyText.length > 50;

      // Check for console errors
      const consoleErrors: string[] = [];
      page.on('console', (msg) => {
        if (msg.type() === 'error') consoleErrors.push(msg.text());
      });

      // Check for visible error messages
      const errorElements = await page.locator('.ant-result-error, .ant-message-error, .ant-notification-notice-error').count();
      const hasVisibleError = errorElements > 0;

      // Check for "no data" or "loading forever" states
      const spinners = await page.locator('.ant-spin-spinning').count();

      // Check for raw i18n keys (untranslated)
      const rawKeys = await page.locator('text=/[a-z]+\.[a-z]+\.[a-z]+/').count();

      expect(hasContent, `${p.name} should have visible content`).toBe(true);
      expect(hasVisibleError, `${p.name} should not show error messages`).toBe(false);

      // Log warnings but don't fail
      if (spinners > 0) console.log(`  ⚠ ${p.name}: ${spinners} spinners still spinning`);
      if (rawKeys > 0) console.log(`  ⚠ ${p.name}: ${rawKeys} possible raw i18n keys`);
      if (consoleErrors.length > 0) console.log(`  ⚠ ${p.name}: ${consoleErrors.length} console errors`);
    });
  }

  test('Login page renders correctly', async ({ browser }) => {
    const context = await browser.newContext();
    const page = await context.newPage();
    await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    const loginInput = await page.locator('#login_login').isVisible();
    const passwordInput = await page.locator('#login_password').isVisible();
    const submitButton = await page.locator('form button[type="submit"]').isVisible();

    expect(loginInput, 'Login input visible').toBe(true);
    expect(passwordInput, 'Password input visible').toBe(true);
    expect(submitButton, 'Submit button visible').toBe(true);
    await context.close();
  });

  test('Marketplace page accessible without login', async ({ page }) => {
    // Clear any auth state
    await page.context().clearCookies();
    await page.goto(`${BASE}/marketplace`, { waitUntil: 'networkidle', timeout: 15000 }).catch(() => {});
    await page.waitForTimeout(2000);

    // Marketplace should either show content or redirect to login
    const url = page.url();
    const hasContent = (await page.locator('body').innerText().catch(() => '')).length > 50;

    expect(url.includes('/login') || hasContent, 'Marketplace should show content or redirect to login').toBe(true);
  });
});
