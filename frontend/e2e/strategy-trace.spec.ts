import { test, expect } from '@playwright/test';

const ADMIN_EMAIL = 'admin@1.com';
const ADMIN_PASSWORD = '12345678';

test.describe('Strategy page bottleneck analysis', () => {
  test('measure SPA navigation to /strategy (real user experience)', async ({ page }) => {
    test.setTimeout(120_000);

    await page.addInitScript(() => {
      localStorage.setItem('ant-theme', 'dark');
    });

    await page.goto('/login');
    await page.getByPlaceholder(/email|account/i).fill(ADMIN_EMAIL);
    await page.getByPlaceholder(/password/i).fill(ADMIN_PASSWORD);
    await page.getByRole('button', { name: /sign in|login/i }).click();
    await expect(page).toHaveURL(/\/$|\/dashboard|\/strategy/i, { timeout: 20_000 });
    await page.waitForTimeout(2000);

    // Navigate to accounts first (warm up), then SPA-navigate to strategy
    await page.evaluate(() => {
      window.history.pushState({}, '', '/accounts');
      window.dispatchEvent(new PopStateEvent('popstate'));
    });
    await page.waitForURL('/accounts', { timeout: 10_000 });
    await page.waitForTimeout(1000);

    // Now SPA-navigate to /strategy and measure
    const start = Date.now();
    await page.evaluate(() => {
      window.history.pushState({}, '', '/strategy');
      window.dispatchEvent(new PopStateEvent('popstate'));
    });
    await page.waitForURL('/strategy', { timeout: 10_000 });
    await page.waitForTimeout(2000);
    const elapsed = Date.now() - start;

    const data = await page.evaluate(() => {
      const resources = performance.getEntriesByType('resource');
      // Only look at resources that started after our navigation (last 5s)
      const recent = resources.filter(r => r.startTime > (performance.now() - 5000));

      const jsFiles = recent
        .filter(r => r.name.endsWith('.js'))
        .map(r => {
          const rr = r as PerformanceResourceTiming;
          return {
            name: rr.name.split('/assets/')[1] || rr.name.split('/').pop() || '',
            duration: Math.round(rr.duration),
            start: Math.round(rr.startTime),
            end: Math.round(rr.responseEnd),
          };
        })
        .sort((a, b) => b.duration - a.duration);

      const apiCalls = recent
        .filter(r => r.name.includes('/ant.v1.'))
        .map(r => {
          const rr = r as PerformanceResourceTiming;
          return {
            name: rr.name.split('/ant.v1.')[1].split('?')[0],
            duration: Math.round(rr.duration),
            start: Math.round(rr.startTime),
            end: Math.round(rr.responseEnd),
          };
        })
        .sort((a, b) => b.duration - a.duration);

      return {
        jsFiles,
        apiCalls,
        maxJSEnd: jsFiles.length > 0 ? Math.max(...jsFiles.map(f => f.end)) : 0,
        maxAPIEnd: apiCalls.length > 0 ? Math.max(...apiCalls.map(c => c.end)) : 0,
      };
    });

    console.log(`\n=== SPA nav to /strategy: ${elapsed}ms ===`);
    console.log(`\n--- New JS chunks loaded (${data.jsFiles.length}) ---`);
    data.jsFiles.forEach(f => console.log(`  ${f.name}: ${f.duration}ms (start=${f.start}ms, end=${f.end}ms)`));
    console.log(`\n--- API calls (${data.apiCalls.length}) ---`);
    data.apiCalls.forEach(c => console.log(`  ${c.name}: ${c.duration}ms (start=${c.start}ms, end=${c.end}ms)`));
    console.log(`\n--- Summary ---`);
    console.log(`  JS chunks finish: ${data.maxJSEnd}ms (relative to page start)`);
    console.log(`  API calls finish: ${data.maxAPIEnd}ms (relative to page start)`);
    console.log(`  Total SPA nav time: ${elapsed}ms`);
  });
});
