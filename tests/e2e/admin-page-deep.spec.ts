import { test, expect, Page, ConsoleMessage } from '@playwright/test';

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

test.describe('Admin page deep load verification', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  for (const { path, name } of ADMIN_PAGES) {
    test(`${name} (${path}) — loads with no errors`, async ({ page }) => {
      // Track API failures, console errors, and error toasts
      const apiErrors: string[] = [];
      const consoleErrors: string[] = [];
      const errorToasts: string[] = [];

      // Monitor console errors
      page.on('console', (msg: ConsoleMessage) => {
        if (msg.type() === 'error') {
          const text = msg.text();
          // Ignore favicon, SSE abort, and browser extension noise
          if (!text.includes('favicon') && !text.includes('ERR_ABORTED') &&
              !text.includes('net::ERR_') && !text.includes('ResizeObserver')) {
            consoleErrors.push(text);
          }
        }
      });

      // Monitor API response failures
      page.on('response', (response) => {
        const url = response.url();
        const status = response.status();
        // Only check API calls (not static assets, not SSE streams)
        if (url.includes('/ant.v1.') || url.includes('/api/')) {
          if (status >= 400 && status !== 401) {
            apiErrors.push(`${status} ${url}`);
          }
          if (status === 401) {
            // 401 on non-auth endpoints = token issue
            if (!url.includes('RefreshToken') && !url.includes('Login')) {
              apiErrors.push(`401 ${url}`);
            }
          }
        }
      });

      // Monitor error toast messages (Ant Design message component)
      page.on('console', (msg: ConsoleMessage) => {
        if (msg.type() === 'error' && msg.text().includes('ant-message')) {
          errorToasts.push(msg.text());
        }
      });

      await page.goto(path, { waitUntil: 'domcontentloaded' });
      await page.waitForTimeout(4000);

      // Check for visible error toasts (Ant Design .ant-message-error)
      const visibleErrorToasts = await page.locator('.ant-message-error').allTextContents();
      errorToasts.push(...visibleErrorToasts);

      // Check for visible error alerts (.ant-alert-error)
      const visibleErrorAlerts = await page.locator('.ant-alert-error').allTextContents();

      // Check for "failed" / "error" / "加载失败" text in page content
      const bodyText = await page.locator('body').textContent() || '';
      const loadFailurePatterns = [
        '加载失败', 'Failed to load', 'failed to load',
        'Load failed', '请求失败', 'Request failed',
        '网络错误', 'Network error', 'network error',
        '服务器错误', 'Server error', 'server error',
        'Something went wrong', 'Application error',
        'TypeError:', 'ReferenceError:',
      ];
      const loadFailures = loadFailurePatterns.filter(p => bodyText.includes(p));

      // Check for Ant Design Result error component
      const hasErrorResult = await page.locator('.ant-result-error').first().isVisible().catch(() => false);

      // Check for Ant Design Empty with error-like description
      const emptyDescriptions = await page.locator('.ant-empty-description').allTextContents();
      const errorEmptyDescs = emptyDescriptions.filter(d =>
        d.includes('失败') || d.includes('failed') || d.includes('error') || d.includes('错误')
      );

      // Take screenshot
      await page.screenshot({
        path: `screenshots/admin-deep-${name.replace(/\s+/g, '-').toLowerCase()}.png`,
        fullPage: true,
      });

      // Assertions — all should be empty
      expect(apiErrors, `${name}: API errors: ${apiErrors.join('; ')}`).toEqual([]);
      expect(consoleErrors, `${name}: Console errors: ${consoleErrors.join('; ')}`).toEqual([]);
      expect(errorToasts, `${name}: Error toasts: ${errorToasts.join('; ')}`).toEqual([]);
      expect(visibleErrorAlerts, `${name}: Error alerts: ${visibleErrorAlerts.join('; ')}`).toEqual([]);
      expect(loadFailures, `${name}: Load failure text found: ${loadFailures.join('; ')}`).toEqual([]);
      expect(hasErrorResult, `${name}: Error result component visible`).toBe(false);
      expect(errorEmptyDescs, `${name}: Error empty descriptions: ${errorEmptyDescs.join('; ')}`).toEqual([]);

      // Log page info for debugging
      console.log(`\n[${name}] URL=${page.url()} | API errors=${apiErrors.length} | Console errors=${consoleErrors.length} | Toasts=${errorToasts.length} | Load failures=${loadFailures.length}`);
    });
  }
});
