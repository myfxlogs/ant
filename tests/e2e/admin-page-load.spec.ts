import { test, expect, Page } from '@playwright/test';

const ADMIN_PAGES: { path: string; name: string }[] = [
  { path: '/admin', name: 'Dashboard' },
  { path: '/admin/users', name: 'User Management' },
  { path: '/admin/wallet', name: 'Wallet Management' },
  { path: '/admin/billing', name: 'Billing Management' },
  { path: '/admin/deposits', name: 'Deposit Management' },
  { path: '/admin/sweep', name: 'Sweep Management' },
  { path: '/admin/accounts', name: 'Account Management' },
  { path: '/admin/trading', name: 'Trading Monitor' },
  { path: '/admin/logs', name: 'Operation Logs' },
  { path: '/admin/config', name: 'System Config' },
  { path: '/admin/jurisdiction', name: 'Jurisdiction Gate' },
  { path: '/admin/strategies', name: 'Strategy Management' },
  { path: '/admin/shares', name: 'Share Management' },
  { path: '/admin/ai-gateway', name: 'AI Gateway' },
  { path: '/admin/monitoring', name: 'Monitoring' },
  { path: '/admin/agent-settings', name: 'Agent Settings' },
  { path: '/admin/autogen-tasks', name: 'AutoGen Tasks' },
  { path: '/admin/marketplace', name: 'Marketplace Management' },
  { path: '/admin/refunds', name: 'Refund Management' },
  { path: '/admin/analytics', name: 'Marketplace Analytics' },
  { path: '/admin/coupons', name: 'Coupon Management' },
  { path: '/admin/sre/killswitch', name: 'SRE Kill Switch' },
  { path: '/admin/sre/breakers', name: 'SRE Breakers' },
  { path: '/admin/sre/canary', name: 'SRE Canary' },
];

async function login(page: Page) {
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1500);
  await page.locator('#login_login').fill('admin@1.com');
  await page.locator('#login_password').fill('12345678');
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
}

test.describe('Admin page load verification', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  for (const { path, name } of ADMIN_PAGES) {
    test(`${name} (${path}) — page loads correctly`, async ({ page }) => {
      await page.goto(path, { waitUntil: 'domcontentloaded' });

      // 1. Wait for page to settle (content or error)
      await page.waitForTimeout(3000);

      // 2. Verify we stayed on the target path (not redirected away)
      expect(page.url()).toContain(path);

      // 3. Verify page has visible content — at least one of these should be present
      const hasCard = await page.locator('.ant-card').first().isVisible().catch(() => false);
      const hasTable = await page.locator('.ant-table').first().isVisible().catch(() => false);
      const hasForm = await page.locator('.ant-form').first().isVisible().catch(() => false);
      const hasStatistic = await page.locator('.ant-statistic').first().isVisible().catch(() => false);
      const hasTabs = await page.locator('.ant-tabs').first().isVisible().catch(() => false);
      const hasDescriptions = await page.locator('.ant-descriptions').first().isVisible().catch(() => false);
      const hasEmpty = await page.locator('.ant-empty').first().isVisible().catch(() => false);
      const hasAlert = await page.locator('.ant-alert').first().isVisible().catch(() => false);
      const hasResult = await page.locator('.ant-result').first().isVisible().catch(() => false);
      const hasList = await page.locator('.ant-list').first().isVisible().catch(() => false);
      const hasUpload = await page.locator('.ant-upload').first().isVisible().catch(() => false);
      const hasSegmented = await page.locator('.ant-segmented').first().isVisible().catch(() => false);

      const hasContent = hasCard || hasTable || hasForm || hasStatistic || hasTabs ||
        hasDescriptions || hasEmpty || hasAlert || hasResult || hasList || hasUpload || hasSegmented;

      // 4. Check for stuck loading spinner (Spin without content for too long)
      const hasSpinOnly = await page.locator('.ant-spin-spinning').first().isVisible().catch(() => false);
      const spinWithoutContent = hasSpinOnly && !hasContent;

      // 5. Check for error boundary / crash
      const bodyText = await page.locator('body').textContent() || '';
      const hasCrash = bodyText.includes('Something went wrong') ||
        bodyText.includes('Application error') ||
        bodyText.includes('TypeError') ||
        bodyText.includes('ReferenceError');

      // Take screenshot for evidence
      await page.screenshot({
        path: `screenshots/admin-page-${name.replace(/\s+/g, '-').toLowerCase()}.png`,
        fullPage: true,
      });

      // Assertions
      expect(hasCrash, `${name} crashed with error`).toBe(false);
      expect(spinWithoutContent, `${name} stuck on loading spinner with no content`).toBe(false);
      expect(hasContent, `${name} rendered no visible content (no card/table/form/statistic/empty/etc.)`).toBe(true);
    });
  }
});
