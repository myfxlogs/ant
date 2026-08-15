import { test, expect } from '@playwright/test';

test.use({ locale: 'zh-CN' });

const EMAIL = 'xianhua.chan@gmail.com';
const PASSWORD = 'Abc123456...';

test('system diagnostics tab renders live data (replaces manual monitoring)', async ({ page }) => {
  test.setTimeout(90_000);
  await page.goto('/login');
  await page.locator('input[type=password]').waitFor();
  const inputs = page.locator('form input:not([type=password])');
  await inputs.first().fill(EMAIL);
  await page.locator('input[type=password]').fill(PASSWORD);
  await page.locator('form button[type=submit], form button:not([type=button])').first().click();
  await page.waitForTimeout(3000);
  await page.waitForURL(/\/$|\/dashboard/i, { timeout: 15_000 });

  await page.goto('/strategy/live');
  // Wait for the strategies table to actually load rows (SSE initial snapshot)
  const row = page.locator('.ant-table-tbody .ant-table-row', { hasText: 'E2E' }).first();
  await expect(row).toBeVisible({ timeout: 30_000 });
  await page.waitForTimeout(2000);

  // Expand first row
  await row.locator('span[role="img"]').first().click();
  const expanded = page.locator('.ant-table-expanded-row');
  await expect(expanded).toBeVisible({ timeout: 10_000 });

  // Click the Diagnostics tab (5th, keyed)
  const diagTab = expanded.locator('[data-node-key="diagnostics"] .ant-tabs-tab-btn');
  await expect(diagTab).toBeVisible({ timeout: 10_000 });
  await diagTab.click();
  await page.waitForTimeout(1500);

  // Read the diagnostics panel (active pane inside expanded row)
  const debug = await page.evaluate(() => {
    const exp = document.querySelector('.ant-table-expanded-row');
    if (!exp) return 'NO_EXPANDED_ROW';
    const tabs = Array.from(exp.querySelectorAll('.ant-tabs-tab')).map(t => t.textContent || '');
    const panes = Array.from(exp.querySelectorAll('[role=tabpanel]')).map(p => (p.className.includes('active') ? '*' : '') + (p.textContent || '').slice(0, 80));
    return 'TABS=' + JSON.stringify(tabs) + ' PANES=' + JSON.stringify(panes);
  });
  console.log('DBG=' + debug);
  const panel = expanded.locator('[role=tabpanel]:visible, .ant-tabs-content-active').last();
  const body = (await panel.textContent()) || '';
  console.log('DIAG_PANEL=' + body.slice(0, 800));

  // Assertions — the data my cron monitoring used to fetch manually:
  const hasEvalCount = /\d{3,}/.test(body);              // eval counter (100+ by now)
  const hasIndicator = /iMACD\[|iMA\[/.test(body);        // indicator keys
  const hasWindow = /500|\d{2,}/.test(body);              // window bars
  const hasBadge = /评估|Evaluating|运行|Running|健康|Healthy/i.test(body);
  console.log(`SYS_EVAL=${hasEvalCount} SYS_INDICATOR=${hasIndicator} SYS_WINDOW=${hasWindow} SYS_BADGE=${hasBadge}`);

  // Screenshot for the record
  await page.screenshot({ path: 'e2e/screenshots/system-diagnostics.png' });

  expect(hasIndicator).toBeTruthy();
  expect(hasEvalCount).toBeTruthy();
});
