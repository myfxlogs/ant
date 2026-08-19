import { test, expect } from '@playwright/test';

const ADMIN_EMAIL = 'admin@1.com';
const ADMIN_PASSWORD = '12345678';

test.describe('SPA navigation performance', () => {
  test('measure client-side route transitions (no full page reload)', async ({ page }) => {
    test.setTimeout(120_000);

    await page.addInitScript(() => {
      localStorage.setItem('ant-theme', 'dark');
    });

    await page.goto('/login');
    await page.getByPlaceholder(/email|account/i).fill(ADMIN_EMAIL);
    await page.getByPlaceholder(/password/i).fill(ADMIN_PASSWORD);
    await page.getByRole('button', { name: /sign in|login/i }).click();
    await expect(page).toHaveURL(/\/$|\/dashboard|\/strategy/i, { timeout: 20_000 });
    await page.waitForTimeout(1500);

    const routes = [
      { name: 'strategy', path: '/strategy' },
      { name: 'accounts', path: '/accounts' },
      { name: 'marketplace', path: '/marketplace' },
      { name: 'dashboard', path: '/dashboard' },
    ];

    const results: { route: string; ms: number; apiCalls: number; apiMs: number; chunkLoads: number }[] = [];

    for (const route of routes) {
      let apiCalls = 0;
      let apiMs = 0;
      let chunkLoads = 0;

      const apiListener = (req: import('@playwright/test').Request) => {
        if (req.url().includes('/ant.v1.')) {
          apiCalls++;
          apiMs += Math.round(req.timing().responseEnd - req.timing().startTime);
        }
      };
      const chunkListener = (req: import('@playwright/test').Request) => {
        if (req.url().includes('/assets/') && req.url().endsWith('.js')) {
          chunkLoads++;
        }
      };

      page.on('requestfinished', apiListener);
      page.on('requestfinished', chunkListener);

      const start = Date.now();
      await page.evaluate((path) => {
        window.history.pushState({}, '', path);
        window.dispatchEvent(new PopStateEvent('popstate'));
      }, route.path);

      await page.waitForURL(route.path, { timeout: 15_000 });
      await page.waitForTimeout(800);

      const elapsed = Date.now() - start;

      page.off('requestfinished', apiListener);
      page.off('requestfinished', chunkListener);

      results.push({ route: route.name, ms: elapsed, apiCalls, apiMs, chunkLoads });
      console.log(`[${route.name}] SPA nav: ${elapsed}ms | API: ${apiCalls} calls, ${apiMs}ms | chunks: ${chunkLoads}`);
    }

    console.log('\n=== SPA Navigation Times ===');
    for (const r of results) {
      console.log(`${r.route}: ${r.ms}ms (API: ${r.apiCalls}/${r.apiMs}ms, chunks: ${r.chunkLoads})`);
    }

    const allUnder2s = results.every(r => r.ms < 2000);
    console.log(`\nAll routes < 2s: ${allUnder2s ? 'YES' : 'NO'}`);

    const avg = Math.round(results.reduce((s, r) => s + r.ms, 0) / results.length);
    console.log(`Average SPA nav: ${avg}ms`);

    const bodyStyles = await page.evaluate(() => {
      const cs = window.getComputedStyle(document.body);
      return {
        background: cs.backgroundColor,
        color: cs.color,
        classList: document.documentElement.className,
      };
    });
    console.log('\n=== Final Page State ===');
    console.log('html class:', bodyStyles.classList);
    console.log('background:', bodyStyles.background);
    console.log('color:', bodyStyles.color);

    expect(bodyStyles.classList).toContain('dark');
    expect(allUnder2s).toBe(true);
  });
});
