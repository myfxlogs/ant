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

test.describe('Admin i18n check', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  for (const { path, name } of ADMIN_PAGES) {
    test(`${name} (${path}) — no raw i18n keys visible`, async ({ page }) => {
      await page.goto(path, { waitUntil: 'domcontentloaded' });
      await page.waitForTimeout(2000);

      // Check for raw i18n keys leaking as visible text (e.g. "admin.sidebar.dashboard")
      const bodyText = await page.locator('body').textContent() || '';
      
      // Raw key patterns: dotted paths that look like i18n keys
      const rawKeyPattern = /[a-z]+(?:\.[a-z][a-zA-Z0-9]*){2,}/g;
      const matches = bodyText.match(rawKeyPattern) || [];
      
      // Filter out known false positives (CSS classes, file paths, URLs, etc.)
      const i18nKeyLeaks = matches.filter((m) => {
        // Ignore things that are clearly not i18n keys
        if (m.includes('www.') || m.includes('http') || m.includes('.com') || m.includes('.org')) return false;
        if (m.includes('antd.') || m.includes('css.') || m.includes('font.')) return false;
        if (m.includes('.png') || m.includes('.svg') || m.includes('.js') || m.includes('.ts')) return false;
        // Real i18n keys: lowercase segments with dots, e.g. "admin.sidebar.dashboard"
        // Exclude false positives: domain names, config key names, API URLs
        if (m.match(/\.(cn|com|org|net|io|ai)$/)) return false;
        if (m.startsWith('ditmarketplace.')) return false;
        if (m.startsWith('open.') || m.startsWith('api.')) return false;
        return /^[a-z]+\.[a-z]/.test(m) && !m.includes(' ') && m.length < 80;
      });

      // Deduplicate
      const uniqueLeaks = [...new Set(i18nKeyLeaks)];
      
      if (uniqueLeaks.length > 0) {
        // Take screenshot for evidence
        await page.screenshot({ path: `screenshots/i18n-leak-${name.replace(/\s+/g, '-').toLowerCase()}.png`, fullPage: true });
      }

      expect(uniqueLeaks, `Raw i18n keys found on ${name}: ${uniqueLeaks.join(', ')}`).toEqual([]);
    });
  }

  test('Admin sidebar — all menu labels translated', async ({ page }) => {
    await page.goto('/admin', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(2000);

    // Get all menu item labels
    const menuLabels = await page.locator('.ant-menu-item .ant-menu-title-content').allTextContents();
    
    // Check none are raw keys
    const rawKeys = menuLabels.filter((label) => /^[a-z]+\.[a-z]+\./.test(label.trim()));
    expect(rawKeys, `Raw i18n keys in sidebar: ${rawKeys.join(', ')}`).toEqual([]);

    // Check all menu items have non-empty text
    const emptyLabels = menuLabels.filter((label) => !label.trim());
    expect(emptyLabels, `Empty menu labels found`).toEqual([]);
  });
});
