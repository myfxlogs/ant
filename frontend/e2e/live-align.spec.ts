import { test, expect } from '@playwright/test';

const USER = { email: 'xianhua.chan@gmail.com', password: 'Abc123456...' };

test('positions table Symbol column aligns with Positions tab label', async ({ page }) => {
  // UI login (API path via nginx returns HTML — not routed)
  await page.goto('/login');
  await page.getByPlaceholder(/email|account/i).fill(USER.email);
  await page.getByPlaceholder(/password/i).fill(USER.password);
  await page.getByRole('button', { name: /sign in|login/i }).click();
  await page.waitForURL(/\/$|\/dashboard/i, { timeout: 15_000 });

  await page.goto('/strategy/live');
  await page.waitForSelector('.ant-table-row', { timeout: 30_000 });
  await page.waitForTimeout(2000);
  await page.screenshot({ path: 'e2e/screenshots/live-page.png' });

  // Click the expand icon on the first row
  const expandBtn = page.locator('.ant-table-cell span[role="img"]').first();
  await expect(expandBtn).toBeVisible({ timeout: 15_000 });
  await expandBtn.click();
  await page.waitForTimeout(3000);

  // Outer "Strategy" column header: the outer table is the one with pagination
  // (inner expanded tables use pagination={false}); th[0] is the expand cell,
  // th[1] is the Strategy column.
  const strategyHeader = page.locator('.ant-table-wrapper:has(.ant-pagination) .ant-table-thead th').nth(1);
  await expect(strategyHeader).toBeVisible();
  const strategyBox = await strategyHeader.boundingBox();
  expect(strategyBox).not.toBeNull();

  // Positions tab label position
  const positionsTab = page.locator('[data-node-key="positions"] .ant-tabs-tab-btn');
  await expect(positionsTab).toBeVisible();
  const tabBox = await positionsTab.boundingBox();
  expect(tabBox).not.toBeNull();

  // Symbol column header position in the positions table
  const symbolHeader = page.locator('.ant-table-expanded-row .ant-tabs-content-active .ant-table-thead th').first();
  await expect(symbolHeader).toBeVisible();
  const headerBox = await symbolHeader.boundingBox();
  expect(headerBox).not.toBeNull();

  // Three-way alignment: Strategy header TEXT ≡ Positions tab ≡ Symbol header.
  // The th box includes cell padding — add computed padding-left to get the text edge.
  const strategyPad = await page.evaluate(() => {
    const th = document.querySelector('.ant-table-wrapper .ant-table-thead th:nth-child(2)') as HTMLElement | null;
    return th ? parseFloat(getComputedStyle(th).paddingLeft) : 0;
  });
  const strategyX = strategyBox!.x + strategyPad;
  const tabX = tabBox!.x;
  const headerX = headerBox!.x;
  console.log(`Strategy text x=${strategyX}, tab x=${tabX}, Symbol header x=${headerX}`);
  expect(Math.abs(tabX - strategyX)).toBeLessThan(3);
  expect(Math.abs(headerX - strategyX)).toBeLessThan(3);
  expect(Math.abs(headerX - tabX)).toBeLessThan(3);

  // Take screenshot for visual verification
  await page.screenshot({ path: 'e2e/screenshots/live-positions-align.png', fullPage: false });
});
