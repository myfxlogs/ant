import { test, expect, type Page, type ConsoleMessage } from '@playwright/test';

const ADMIN_EMAIL = 'admin@1.com';
const ADMIN_PASS = '12345678';

async function login(page: Page) {
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1500);
  await page.locator('#login_login').fill(ADMIN_EMAIL);
  await page.locator('#login_password').fill(ADMIN_PASS);
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
  await page.waitForTimeout(2000);
}

interface InteractionIssue {
  page: string;
  action: string;
  detail: string;
}

test.describe('Interaction Scan — click buttons, open modals, switch tabs', () => {
  test('verify key interactions do not crash', async ({ page }) => {
    await login(page);

    const issues: InteractionIssue[] = [];
    const pageErrors: string[] = [];
    const consoleErrors: ConsoleMessage[] = [];

    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        const txt = msg.text();
        if (/locize\.com/i.test(txt)) return;
        if (/Failed to load resource|net::ERR/i.test(txt)) return;
        consoleErrors.push(msg);
      }
    });
    page.on('pageerror', (err) => {
      if (/ResizeObserver|MutationObserver/i.test(err.message)) return;
      pageErrors.push(err.message);
    });

    function checkErrors(pageName: string, action: string) {
      for (const err of pageErrors) {
        issues.push({ page: pageName, action, detail: `PageError: ${err.substring(0, 200)}` });
      }
      for (const err of consoleErrors) {
        issues.push({ page: pageName, action, detail: `Console: ${err.text().substring(0, 200)}` });
      }
      pageErrors.length = 0;
      consoleErrors.length = 0;
    }

    // ── 1. Marketplace: open strategy detail modal ──
    console.log('\n--- Marketplace ---');
    await page.goto('/marketplace', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    // Click first strategy card to open detail modal
    const strategyCard = page.locator('.ant-card').first();
    if (await strategyCard.isVisible()) {
      await strategyCard.click();
      await page.waitForTimeout(1500);
      checkErrors('Marketplace', 'open strategy detail modal');

      // Check if modal appeared
      const modal = page.locator('.ant-modal-content').first();
      if (await modal.isVisible()) {
        // Try clicking tabs inside the modal
        const tabs = page.locator('.ant-modal-content .ant-tabs-tab');
        const tabCount = await tabs.count();
        for (let i = 0; i < Math.min(tabCount, 5); i++) {
          await tabs.nth(i).click();
          await page.waitForTimeout(800);
          checkErrors('Marketplace', `modal tab ${i}`);
        }

        // Try deploy button if visible
        const deployBtn = page.locator('.ant-modal-content button').filter({ hasText: /deploy|deploy|部署/i });
        if (await deployBtn.isVisible().catch(() => false)) {
          await deployBtn.first().click();
          await page.waitForTimeout(1500);
          checkErrors('Marketplace', 'click deploy button');
        }

        // Close modal
        await page.locator('.ant-modal-close').first().click().catch(() => {});
        await page.waitForTimeout(500);
      }
    }

    // ── 2. Strategy Gallery: click into strategy detail ──
    console.log('--- Strategy Gallery ---');
    await page.goto('/strategy', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);
    checkErrors('Strategy Gallery', 'page load');

    // Click first strategy item
    const strategyItem = page.locator('.ant-card, .ant-list-item, [class*="strategy"]').first();
    if (await strategyItem.isVisible()) {
      await strategyItem.click();
      await page.waitForTimeout(2000);
      checkErrors('Strategy Gallery', 'click strategy item');
      await page.goBack().catch(() => {});
      await page.waitForTimeout(1000);
    }

    // ── 3. Strategy Workspace: switch tabs ──
    console.log('--- Strategy Workspace ---');
    await page.goto('/strategy/new', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);
    checkErrors('Strategy Workspace', 'page load');

    const wsTabs = page.locator('.ant-tabs-tab:visible');
    const wsTabCount = await wsTabs.count();
    for (let i = 0; i < Math.min(wsTabCount, 6); i++) {
      try {
        await wsTabs.nth(i).click({ timeout: 5000 });
        await page.waitForTimeout(1000);
        checkErrors('Strategy Workspace', `tab ${i}`);
      } catch { /* skip invisible/clickable tabs */ }
    }

    // ── 4. Live Strategy: open schedule modal ──
    console.log('--- Live Strategy ---');
    await page.goto('/strategy/live', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);
    checkErrors('Live Strategy', 'page load');

    // Try clicking create/edit schedule buttons
    const createBtn = page.locator('button').filter({ hasText: /create|new|schedule|创建|新建/i });
    if (await createBtn.first().isVisible().catch(() => false)) {
      await createBtn.first().click();
      await page.waitForTimeout(1500);
      checkErrors('Live Strategy', 'click create schedule');

      // Close any modal that opened
      const modal2 = page.locator('.ant-modal-close').first();
      if (await modal2.isVisible().catch(() => false)) {
        await modal2.click();
        await page.waitForTimeout(500);
      }
    }

    // ── 5. Wallet: check tabs and actions ──
    console.log('--- Wallet ---');
    await page.goto('/wallet', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);
    checkErrors('Wallet', 'page load');

    const walletTabs = page.locator('.ant-tabs-tab:visible');
    const walletTabCount = await walletTabs.count();
    for (let i = 0; i < Math.min(walletTabCount, 4); i++) {
      try {
        await walletTabs.nth(i).click({ timeout: 5000 });
        await page.waitForTimeout(1000);
        checkErrors('Wallet', `tab ${i}`);
      } catch { /* skip */ }
    }

    // ── 6. Subscription: open subscribe modal ──
    console.log('--- Subscription ---');
    await page.goto('/subscription', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);
    checkErrors('Subscription', 'page load');

    const subscribeBtn = page.locator('button').filter({ hasText: /subscribe|choose|选择|订阅/i });
    if (await subscribeBtn.first().isVisible().catch(() => false)) {
      await subscribeBtn.first().click();
      await page.waitForTimeout(1500);
      checkErrors('Subscription', 'click subscribe');

      const modal3 = page.locator('.ant-modal-close').first();
      if (await modal3.isVisible().catch(() => false)) {
        await modal3.click();
        await page.waitForTimeout(500);
      }
    }

    // ── 7. Profile: edit profile ──
    console.log('--- Profile ---');
    await page.goto('/profile', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);
    checkErrors('Profile', 'page load');

    const profileTabs = page.locator('.ant-tabs-tab:visible');
    const profileTabCount = await profileTabs.count();
    for (let i = 0; i < Math.min(profileTabCount, 4); i++) {
      try {
        await profileTabs.nth(i).click({ timeout: 5000 });
        await page.waitForTimeout(1000);
        checkErrors('Profile', `tab ${i}`);
      } catch { /* skip */ }
    }

    // ── 8. Admin: key admin pages with tabs ──
    console.log('--- Admin pages ---');
    const adminPages = [
      { path: '/admin/users', name: 'Admin Users' },
      { path: '/admin/wallet', name: 'Admin Wallet' },
      { path: '/admin/marketplace', name: 'Admin Marketplace' },
      { path: '/admin/strategies', name: 'Admin Strategies' },
      { path: '/admin/config', name: 'Admin Config' },
    ];

    for (const ap of adminPages) {
      await page.goto(ap.path, { waitUntil: 'domcontentloaded' });
      await page.waitForTimeout(3000);
      checkErrors(ap.name, 'page load');

      // Click tabs if any
      const tabs = page.locator('.ant-tabs-tab:visible');
      const tabCount = await tabs.count();
      for (let i = 0; i < Math.min(tabCount, 4); i++) {
        try {
          await tabs.nth(i).click({ timeout: 5000 });
          await page.waitForTimeout(1000);
          checkErrors(ap.name, `tab ${i}`);
        } catch { /* skip */ }
      }

      // Try opening any modal-triggering buttons
      const actionBtns = page.locator('button').filter({ hasText: /create|add|new|edit|detail|创建|新增|编辑|详情/i });
      const btnCount = await actionBtns.count();
      for (let i = 0; i < Math.min(btnCount, 3); i++) {
        if (await actionBtns.nth(i).isVisible().catch(() => false)) {
          try {
            await actionBtns.nth(i).click({ timeout: 5000 });
          } catch { continue; }
          await page.waitForTimeout(1500);
          checkErrors(ap.name, `action button ${i}`);

          // Close modal/drawer if opened
          const closeBtn = page.locator('.ant-modal-close, .ant-drawer-close').first();
          if (await closeBtn.isVisible().catch(() => false)) {
            await closeBtn.click();
            await page.waitForTimeout(500);
          }
        }
      }
    }

    // ── Summary ──
    console.log('\n\n========== INTERACTION SCAN RESULTS ==========');
    console.log(`Total issues: ${issues.length}`);
    console.log('===============================================\n');

    if (issues.length > 0) {
      console.log('Issues found:');
      for (const i of issues) {
        console.log(`  [${i.page}] ${i.action}: ${i.detail}`);
      }
    } else {
      console.log('All interactions passed with no errors!');
    }

    // Write report
    const fs = await import('fs');
    const report = [
      '========== INTERACTION SCAN REPORT ==========',
      `Date: ${new Date().toISOString()}`,
      `Total issues: ${issues.length}`,
      '============================================',
      '',
    ];
    if (issues.length > 0) {
      for (const i of issues) {
        report.push(`  [${i.page}] ${i.action}: ${i.detail}`);
      }
    } else {
      report.push('All interactions passed with no errors!');
    }
    fs.writeFileSync('/opt/ant/tests/e2e/interaction-scan-report.txt', report.join('\n'));

    expect(issues.length).toBe(0);
  });
});
