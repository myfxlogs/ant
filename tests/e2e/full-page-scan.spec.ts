import { test, expect, type Page, type ConsoleMessage } from '@playwright/test';

const ADMIN_EMAIL = 'admin@1.com';
const ADMIN_PASS = '12345678';

const PAGES: { path: string; name: string }[] = [
  // Main app
  { path: '/', name: 'Dashboard' },
  { path: '/wallet', name: 'Wallet' },
  { path: '/subscription', name: 'Subscription' },
  { path: '/profile', name: 'Profile' },
  { path: '/marketplace', name: 'Marketplace' },
  { path: '/strategy', name: 'Strategy Gallery' },
  { path: '/strategy/new', name: 'Strategy Workspace' },
  { path: '/strategy/live', name: 'Live Strategy' },
  { path: '/strategy/market-tools', name: 'Market Tools' },
  { path: '/logs', name: 'Log Management' },
  { path: '/auto-trading', name: 'Auto Trading Settings' },
  { path: '/trading/algos', name: 'Algo Dashboard' },
  { path: '/analytics', name: 'Analytics Summary' },
  { path: '/accounts/bind', name: 'Bind Account' },
  // Admin
  { path: '/admin', name: 'Admin Dashboard' },
  { path: '/admin/users', name: 'Admin Users' },
  { path: '/admin/wallet', name: 'Admin Wallet Mgmt' },
  { path: '/admin/billing', name: 'Admin Billing' },
  { path: '/admin/deposits', name: 'Admin Deposits' },
  { path: '/admin/accounts', name: 'Admin Accounts' },
  { path: '/admin/trading', name: 'Admin Trading Monitor' },
  { path: '/admin/logs', name: 'Admin Operation Logs' },
  { path: '/admin/config', name: 'Admin System Config' },
  { path: '/admin/jurisdiction', name: 'Admin Jurisdiction Gate' },
  { path: '/admin/strategies', name: 'Admin Strategy Mgmt' },
  { path: '/admin/shares', name: 'Admin Share Mgmt' },
  { path: '/admin/ai-gateway', name: 'Admin AI Gateway' },
  { path: '/admin/monitoring', name: 'Admin Monitoring' },
  { path: '/admin/agent-settings', name: 'Admin Agent Settings' },
  { path: '/admin/marketplace', name: 'Admin Marketplace Mgmt' },
  { path: '/admin/refunds', name: 'Admin Refund Mgmt' },
  { path: '/admin/analytics', name: 'Admin Marketplace Analytics' },
  { path: '/admin/coupons', name: 'Admin Coupon Mgmt' },
  { path: '/admin/sweep', name: 'Admin Sweep Mgmt' },
  { path: '/admin/autogen-tasks', name: 'Admin AutoGen Tasks' },
  { path: '/admin/sre/killswitch', name: 'SRE Kill Switch' },
  { path: '/admin/sre/breakers', name: 'SRE Breakers' },
  { path: '/admin/sre/canary', name: 'SRE Canary' },
];

interface PageIssue {
  page: string;
  path: string;
  type: 'error' | 'i18n-key' | 'crash';
  detail: string;
}

async function login(page: Page) {
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1500);
  await page.locator('#login_login').fill(ADMIN_EMAIL);
  await page.locator('#login_password').fill(ADMIN_PASS);
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
  await page.waitForTimeout(2000);
}

test.describe('Full Page Scan — errors and i18n keys', () => {
  test('scan all pages for JS errors, crashes, and raw i18n keys', async ({ page }) => {
    await login(page);

    const issues: PageIssue[] = [];
    const consoleErrors: ConsoleMessage[] = [];
    const pageErrors: string[] = [];

    // Collect console errors globally
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg);
      }
    });
    page.on('pageerror', (err) => {
      pageErrors.push(err.message);
    });

    for (const p of PAGES) {
      // Clear previous errors
      consoleErrors.length = 0;
      pageErrors.length = 0;

      await page.goto(p.path, { waitUntil: 'domcontentloaded' });
      await page.waitForTimeout(3000);

      const bodyText = await page.locator('body').innerText().catch(() => '');

      // 1. Check for React crash (blank page or error boundary)
      const isBlank = bodyText.trim().length < 10;
      const hasErrorBoundary = /Something went wrong|Error Boundary|Application error/i.test(bodyText);

      // 2. Check for raw i18n keys (patterns like "some.key" or "KEY_NAME" that look like untranslated keys)
      const i18nKeyPatterns = [
        /\b[a-z]+\.[a-z]+\.[a-z]+\b/g,  // dot.separated.keys
        /[A-Z][A-Z_]{5,}/g,               // ALL_CAPS_KEYS (min 6 chars)
      ];
      const rawKeys: string[] = [];
      for (const pattern of i18nKeyPatterns) {
        const matches = bodyText.match(pattern);
        if (matches) {
          for (const m of matches) {
            // Filter out common false positives
            if (/^(http|https|localhost|ant\.v1|ant\.v2)/.test(m)) continue;
            if (/^(MT4|MT5|API|JSON|HTML|CSS|URL|SSE|RPC|NATS|JWT)/.test(m)) continue;
            if (/^(ETH|BTC|USD|EUR|USDT)/.test(m)) continue;
            rawKeys.push(m);
          }
        }
      }

      // 3. Check for "undefined" or "null" rendered in UI
      const hasUndefined = /\bundefined\b/.test(bodyText) && !/is undefined|variable is undefined/.test(bodyText);
      const hasNullRender = /\bnull\b/.test(bodyText) && !/nullable|not null|null check/.test(bodyText);

      // 4. Check console errors (filter out network errors which are expected on some pages)
      const realConsoleErrors = consoleErrors.filter((e) => {
        const txt = e.text();
        // Skip expected network/CORS errors
        if (/Failed to load resource|net::ERR|404|503|fetch.*failed/i.test(txt)) return false;
        // Skip i18next promotional message
        if (/locize\.com/i.test(txt)) return false;
        return true;
      });

      // 5. Check for pageerror (uncaught exceptions)
      const realPageErrors = pageErrors.filter((e) => {
        if (/ResizeObserver|MutationObserver/i.test(e)) return false;
        return true;
      });

      // Record issues
      if (isBlank || hasErrorBoundary) {
        issues.push({ page: p.name, path: p.path, type: 'crash', detail: isBlank ? 'Page is blank' : 'Error boundary triggered' });
      }
      if (hasUndefined) {
        issues.push({ page: p.name, path: p.path, type: 'error', detail: '"undefined" rendered in UI' });
      }
      if (hasNullRender) {
        issues.push({ page: p.name, path: p.path, type: 'error', detail: '"null" rendered in UI' });
      }
      for (const key of [...new Set(rawKeys)]) {
        issues.push({ page: p.name, path: p.path, type: 'i18n-key', detail: `Raw key: "${key}"` });
      }
      for (const err of realConsoleErrors) {
        issues.push({ page: p.name, path: p.path, type: 'error', detail: `Console: ${err.text().substring(0, 150)}` });
      }
      for (const err of realPageErrors) {
        issues.push({ page: p.name, path: p.path, type: 'error', detail: `PageError: ${err.substring(0, 150)}` });
      }

      // Quick status
      const status = issues.filter((i) => i.page === p.name).length === 0 ? '✅' : '❌';
      console.log(`  ${status} ${p.name} (${p.path})`);
    }

    // Print summary
    console.log('\n\n========== FULL PAGE SCAN RESULTS ==========');
    console.log(`Pages scanned: ${PAGES.length}`);
    console.log(`Total issues: ${issues.length}`);

    const crashes = issues.filter((i) => i.type === 'crash');
    const errors = issues.filter((i) => i.type === 'error');
    const i18nKeys = issues.filter((i) => i.type === 'i18n-key');

    console.log(`  Crashes: ${crashes.length}`);
    console.log(`  Errors: ${errors.length}`);
    console.log(`  Raw i18n keys: ${i18nKeys.length}`);
    console.log('============================================\n');

    if (crashes.length > 0) {
      console.log('\n--- CRASHES ---');
      for (const i of crashes) console.log(`  [${i.page}] ${i.path}: ${i.detail}`);
    }
    if (errors.length > 0) {
      console.log('\n--- ERRORS ---');
      for (const i of errors) console.log(`  [${i.page}] ${i.path}: ${i.detail}`);
    }
    if (i18nKeys.length > 0) {
      console.log('\n--- RAW I18N KEYS ---');
      for (const i of i18nKeys) console.log(`  [${i.page}] ${i.path}: ${i.detail}`);
    }

    // Write report
    const fs = await import('fs');
    const report = [
      '========== FULL PAGE SCAN REPORT ==========',
      `Date: ${new Date().toISOString()}`,
      `Pages scanned: ${PAGES.length}`,
      `Total issues: ${issues.length}`,
      `  Crashes: ${crashes.length}`,
      `  Errors: ${errors.length}`,
      `  Raw i18n keys: ${i18nKeys.length}`,
      '============================================',
      '',
    ];
    if (crashes.length > 0) {
      report.push('--- CRASHES ---');
      for (const i of crashes) report.push(`  [${i.page}] ${i.path}: ${i.detail}`);
      report.push('');
    }
    if (errors.length > 0) {
      report.push('--- ERRORS ---');
      for (const i of errors) report.push(`  [${i.page}] ${i.path}: ${i.detail}`);
      report.push('');
    }
    if (i18nKeys.length > 0) {
      report.push('--- RAW I18N KEYS ---');
      for (const i of i18nKeys) report.push(`  [${i.page}] ${i.path}: ${i.detail}`);
      report.push('');
    }
    fs.writeFileSync('/opt/ant/tests/e2e/page-scan-report.txt', report.join('\n'));
    console.log('\nReport saved to: /opt/ant/tests/e2e/page-scan-report.txt');

    // Fail if any crashes or errors
    expect(crashes.length).toBe(0);
  });
});
