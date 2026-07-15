import { test, expect, type Page } from '@playwright/test';

const ADMIN_EMAIL = 'admin@1.com';
const ADMIN_PASS = '12345678';

const PAGES: { path: string; name: string; needsAccount?: boolean }[] = [
  { path: '/', name: 'Dashboard' },
  { path: '/wallet', name: 'Wallet' },
  { path: '/subscription', name: 'Subscription' },
  { path: '/profile', name: 'Profile' },
  { path: '/marketplace', name: 'Marketplace' },
  { path: '/strategy/workspace', name: 'Strategy Workspace' },
  { path: '/strategy/live', name: 'Live Strategy' },
  { path: '/strategy/market-tools', name: 'Market Tools' },
  { path: '/logs', name: 'Log Management' },
  { path: '/auto-trading', name: 'Auto Trading Settings' },
  { path: '/trading/algos', name: 'Algo Dashboard' },
  { path: '/analytics', name: 'Analytics Summary' },
  { path: '/accounts/bind', name: 'Bind Account' },
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
  { path: '/admin/sre/killswitch', name: 'SRE Kill Switch' },
  { path: '/admin/sre/breakers', name: 'SRE Breakers' },
  { path: '/admin/sre/canary', name: 'SRE Canary' },
];

async function login(page: Page) {
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1500);
  await page.locator('#login_login').fill(ADMIN_EMAIL);
  await page.locator('#login_password').fill(ADMIN_PASS);
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
  await page.waitForTimeout(2000);
}

async function switchLanguageToZhCN(page: Page) {
  // Find the language switcher in the topbar and switch to zh-cn
  // Look for a Select or dropdown that contains language options
  const langSwitcher = page.locator('.ant-select').filter({ hasText: /English|中文|日本語|Việt/ }).first();
  if (await langSwitcher.isVisible().catch(() => false)) {
    await langSwitcher.click();
    await page.waitForTimeout(500);
    // Click the "简体中文" option
    const zhOption = page.locator('.ant-select-item-option').filter({ hasText: '简体中文' }).first();
    if (await zhOption.isVisible().catch(() => false)) {
      await zhOption.click();
      await page.waitForTimeout(1000);
    }
  } else {
    // Try setting via localStorage directly
    await page.evaluate(() => {
      localStorage.setItem('alphaforge_lang', 'zh-cn');
    });
    await page.reload();
    await page.waitForTimeout(2000);
  }
}

// English words/phrases that should NOT appear when language is zh-cn
// Excluding: brand names, technical terms, currency codes, proper nouns
const ENGLISH_PATTERNS = [
  // Common UI words
  /\bDashboard\b/,
  /\bSettings\b/,
  /\bProfile\b/,
  /\bWallet\b/,
  /\bSubscription\b/,
  /\bMarketplace\b/,
  /\bAnalytics\b/,
  /\bSearch\b/,
  /\bFilter\b/,
  /\bLoading\b/,
  /\bSubmit\b/,
  /\bCancel\b/,
  /\bConfirm\b/,
  /\bDelete\b/,
  /\bEdit\b/,
  /\bAdd\b/,
  /\bSave\b/,
  /\bClose\b/,
  /\bNext\b/,
  /\bPrevious\b/,
  /\bBack\b/,
  /\bRefresh\b/,
  /\bExport\b/,
  /\bImport\b/,
  /\bCreate\b/,
  /\bUpdate\b/,
  /\bRemove\b/,
  /\bEnable\b/,
  /\bDisable\b/,
  /\bActive\b/,
  /\bInactive\b/,
  /\bStatus\b/,
  /\bAmount\b/,
  /\bBalance\b/,
  /\bDeposit\b/,
  /\bWithdraw\b/,
  /\bTransaction\b/,
  /\bHistory\b/,
  /\bTotal\b/,
  /\bCurrent\b/,
  /\bPlan\b/,
  /\bBilling\b/,
  /\bUsage\b/,
  /\bMonitor\b/,
  /\bManagement\b/,
  /\bStrategy\b/,
  /\bTrading\b/,
  /\bAccount\b/,
  /\bReport\b/,
  /\bConfig\b/,
  /\bSystem\b/,
  /\bUser\b/,
  /\bLogs\b/,
  /\bOperation\b/,
  /\bJurisdiction\b/,
  /\bShares?\b/,
  /\bGateway\b/,
  // "Agent" is a common loanword in Chinese IT — "Agent 设置" is correct zh-cn
  // /\bAgent\b/,
  /\bKill\b/,
  /\bSwitch\b/,
  /\bBreakers?\b/,
  /\bCanary\b/,
  /\bIndicators?\b/,
  /\bWatchlist\b/,
  /\bSymbols?\b/,
  /\bCandle\b/,
  /\bArea\b/,
  /\bOverlay\b/,
  /\bMeasure\b/,
  /\bTrend Line\b/,
  /\bHorizontal\b/,
  /\bVertical\b/,
  /\bFibonacci\b/,
  /\bParallel\b/,
  /\bChannel\b/,
  /\bPrice Line\b/,
  /\bRay\b/,
  /\bExtended\b/,
  /\bCompliance\b/,
  /\bGenerate\b/,
  /\bAnalyze\b/,
  /\bBacktest\b/,
  /\bAuto.?renew\b/i,
  /\bMonthly\b/,
  /\bYearly\b/,
  /\bFree\b/,
  /\bChoose\b/,
  /\bSelect\b/,
  /\bNo data\b/,
  /\bNo results?\b/,
  /\bNot found\b/,
  /\bError\b/,
  /\bSuccess\b/,
  /\bFailed\b/,
  /\bWarning\b/,
  /\bPending\b/,
  /\bApproved\b/,
  /\bRejected\b/,
  /\bEnabled\b/,
  /\bDisabled\b/,
  /\bOnline\b/,
  /\bOffline\b/,
  /\bConnect\b/,
  /\bDisconnect\b/,
];

interface I18nIssue {
  page: string;
  path: string;
  englishText: string;
  context: string;
}

test.describe('i18n Audit — scan all pages for hardcoded English', () => {
  test('scan all pages in zh-cn mode', async ({ page }) => {
    await login(page);
    await switchLanguageToZhCN(page);

    const issues: I18nIssue[] = [];
    const pageTexts: Record<string, string> = {};

    for (const p of PAGES) {
      await page.goto(p.path, { waitUntil: 'domcontentloaded' });
      await page.waitForTimeout(3000);

      // Get all visible text content
      const bodyText = await page.locator('body').innerText();

      // Also check placeholder attributes
      const placeholders = await page.locator('input[placeholder], textarea[placeholder]').evaluateAll(
        (els) => els.map((el) => el.getAttribute('placeholder') || '').filter(Boolean),
      );

      // Check tooltip titles
      const tooltips = await page.locator('[title]').evaluateAll(
        (els) => els.map((el) => el.getAttribute('title') || '').filter(Boolean),
      );

      // Check antd Table column titles
      const tableHeaders = await page.locator('.ant-table-thead th').evaluateAll(
        (els) => els.map((el) => el.textContent || '').filter(Boolean),
      );

      // Check antd Menu items
      const menuItems = await page.locator('.ant-menu-item, .ant-menu-submenu-title').evaluateAll(
        (els) => els.map((el) => el.textContent || '').filter(Boolean),
      );

      // Check Tab labels
      const tabLabels = await page.locator('.ant-tabs-tab').evaluateAll(
        (els) => els.map((el) => el.textContent || '').filter(Boolean),
      );

      // Check Button labels
      const buttonLabels = await page.locator('button').evaluateAll(
        (els) => els.map((el) => el.textContent || '').filter((t) => t.trim().length > 0),
      );

      // Check Card titles
      const cardTitles = await page.locator('.ant-card-head-title').evaluateAll(
        (els) => els.map((el) => el.textContent || '').filter(Boolean),
      );

      // Check Modal/Drawer titles
      const modalTitles = await page.locator('.ant-modal-title, .ant-drawer-title').evaluateAll(
        (els) => els.map((el) => el.textContent || '').filter(Boolean),
      );

      // Check Form labels
      const formLabels = await page.locator('.ant-form-item-label label').evaluateAll(
        (els) => els.map((el) => el.textContent || '').filter(Boolean),
      );

      const allTexts = [
        ...placeholders,
        ...tooltips,
        ...tableHeaders,
        ...menuItems,
        ...tabLabels,
        ...buttonLabels,
        ...cardTitles,
        ...modalTitles,
        ...formLabels,
      ];

      // Also scan the full body text for English patterns
      const allContent = [bodyText, ...allTexts];

      for (const text of allContent) {
        for (const pattern of ENGLISH_PATTERNS) {
          const match = text.match(pattern);
          if (match) {
            const englishWord = match[0];
            // Skip if this is inside a longer Chinese sentence (likely a brand/technical term)
            const surrounding = text.substring(
              Math.max(0, (match.index || 0) - 20),
              Math.min(text.length, (match.index || 0) + englishWord.length + 20),
            );
            // If surrounding text is mostly Chinese, it might be a technical term used in context
            const chineseChars = (surrounding.match(/[\u4e00-\u9fff]/g) || []).length;
            if (chineseChars > 5) continue;

            // Skip common false positives
            if (englishWord === 'Area' && /Area Chart|面积图/.test(surrounding)) continue;
            if (englishWord === 'OHLC') continue;
            if (englishWord === 'MT4' || englishWord === 'MT5') continue;

            issues.push({
              page: p.name,
              path: p.path,
              englishText: englishWord,
              context: surrounding.trim(),
            });
          }
        }
      }

      pageTexts[p.name] = bodyText.substring(0, 200);
    }

    // Deduplicate issues
    const uniqueIssues = issues.filter((issue, idx, self) =>
      idx === self.findIndex((i) => i.page === issue.page && i.englishText === issue.englishText),
    );

    // Print results
    console.log('\n\n========== i18n AUDIT RESULTS ==========');
    console.log(`Pages scanned: ${PAGES.length}`);
    console.log(`Issues found: ${uniqueIssues.length}`);
    console.log('========================================\n');

    if (uniqueIssues.length > 0) {
      console.log('Hardcoded English strings found in zh-cn mode:\n');
      for (const issue of uniqueIssues) {
        console.log(`  [${issue.page}] "${issue.englishText}" — context: "${issue.context}"`);
      }
      console.log('\n');
    }

    // Write results to file for reference
    const fs = await import('fs');
    const reportPath = '/opt/ant/tests/e2e/i18n-audit-report.txt';
    const report = [
      '========== i18n AUDIT REPORT ==========',
      `Date: ${new Date().toISOString()}`,
      `Pages scanned: ${PAGES.length}`,
      `Issues found: ${uniqueIssues.length}`,
      '========================================',
      '',
    ];
    if (uniqueIssues.length > 0) {
      report.push('Hardcoded English strings found in zh-cn mode:');
      report.push('');
      for (const issue of uniqueIssues) {
        report.push(`  [${issue.page}] "${issue.englishText}" — context: "${issue.context}"`);
      }
    } else {
      report.push('No hardcoded English strings found. All text is properly internationalized.');
    }
    fs.writeFileSync(reportPath, report.join('\n'));
    console.log(`Report saved to: ${reportPath}`);

    // Don't fail the test — just report
    expect(uniqueIssues.length).toBeLessThanOrEqual(100);
  });
});
