import { test, expect } from '@playwright/test';

const ADMIN_EMAIL = 'admin@1.com';
const ADMIN_PASSWORD = '12345678';

test.describe('Dark mode + performance audit', () => {
  test('login, switch pages, measure load time, screenshot dark mode', async ({ page }) => {
    test.setTimeout(120_000);

    // Enable dark mode via localStorage before login (key must match uiStore.ts: 'ant-theme')
    await page.addInitScript(() => {
      localStorage.setItem('ant-theme', 'dark');
    });

    await page.goto('/login');
    await page.getByPlaceholder(/email|account/i).fill(ADMIN_EMAIL);
    await page.getByPlaceholder(/password/i).fill(ADMIN_PASSWORD);
    await page.getByRole('button', { name: /sign in|login/i }).click();
    await expect(page).toHaveURL(/\/$|\/dashboard|\/strategy/i, { timeout: 20_000 });
    await page.waitForTimeout(1000);

    // Screenshot: home/dashboard in dark mode
    await page.screenshot({ path: 'e2e/screenshots/dark-home.png', fullPage: true });

    // Navigate to pages and measure load time
    const pages = [
      { name: 'strategy', path: '/strategy' },
      { name: 'accounts', path: '/accounts' },
      { name: 'marketplace', path: '/marketplace' },
    ];

    const timings: { page: string; ms: number }[] = [];

    for (const p of pages) {
      const start = Date.now();
      // Track API call timings
      const apiCalls: { url: string; duration: number }[] = [];
      page.on('requestfinished', req => {
        if (req.url().includes('/ant.v1.')) {
          apiCalls.push({ url: req.url().split('/ant.v1.')[1], duration: Math.round(req.timing().responseEnd - req.timing().startTime) });
        }
      });
      await page.goto(p.path);
      // Wait for network to settle (SSE streams stay open, so use domcontentloaded + timeout)
      await page.waitForLoadState('domcontentloaded', { timeout: 30_000 }).catch(() => {});
      await page.waitForTimeout(500);
      const elapsed = Date.now() - start;
      timings.push({ page: p.name, ms: elapsed });
      console.log(`[${p.name}] loaded in ${elapsed}ms`);
      console.log(`  API calls: ${apiCalls.length}`);
      apiCalls.forEach(c => console.log(`    ${c.url}: ${c.duration}ms`));

      await page.screenshot({ path: `e2e/screenshots/dark-${p.name}.png`, fullPage: true });
    }

    // Log all timings
    console.log('\n=== Page Load Times (dark mode) ===');
    for (const t of timings) {
      console.log(`${t.page}: ${t.ms}ms`);
    }

    // Capture detailed navigation timing for the last page
    const navTiming = await page.evaluate(() => {
      const entries = performance.getEntriesByType('navigation');
      if (entries.length === 0) return null;
      const e = entries[0] as PerformanceNavigationTiming;
      return {
        domContentLoaded: Math.round(e.domContentLoadedEventEnd - e.startTime),
        loadComplete: Math.round(e.loadEventEnd - e.startTime),
        domInteractive: Math.round(e.domInteractive - e.startTime),
        responseEnd: Math.round(e.responseEnd - e.startTime),
        transferSize: e.transferSize,
        encodedBodySize: e.encodedBodySize,
      };
    });
    console.log('\n=== Navigation Timing (last page) ===');
    if (navTiming) {
      console.log(`Response end: ${navTiming.responseEnd}ms`);
      console.log(`DOM interactive: ${navTiming.domInteractive}ms`);
      console.log(`DOM content loaded: ${navTiming.domContentLoaded}ms`);
      console.log(`Load complete: ${navTiming.loadComplete}ms`);
    }

    // Count resource sizes and parse timings
    const resources = await page.evaluate(() => {
      const entries = performance.getEntriesByType('resource');
      const jsEntries = entries.filter(e => e.name.endsWith('.js') || e.name.includes('.js?'));
      const totalSize = jsEntries.reduce((sum, e) => sum + (e as PerformanceResourceTiming).transferSize, 0);
      const totalDuration = jsEntries.reduce((max, e) => Math.max(max, e.duration), 0);
      // Sum up all JS durations (not just max) to approximate total parse+exec time
      const totalJSDuration = jsEntries.reduce((sum, e) => sum + e.duration, 0);
      // Get the largest JS files by duration
      const topJS = jsEntries
        .map(e => ({ name: (e.name.split('/').pop() || '').substring(0, 40), dur: Math.round(e.duration), size: Math.round((e as PerformanceResourceTiming).transferSize / 1024) }))
        .sort((a, b) => b.dur - a.dur)
        .slice(0, 10);
      return { count: jsEntries.length, totalKB: Math.round(totalSize / 1024), maxDuration: Math.round(totalDuration), totalJSDuration: Math.round(totalJSDuration), topJS };
    });
    console.log(`\n=== Resources ===`);
    console.log(`JS files: ${resources.count}, total: ${resources.totalKB}KB, max duration: ${resources.maxDuration}ms, total JS duration: ${resources.totalJSDuration}ms`);
    console.log(`Top JS by duration:`);
    resources.topJS.forEach(j => console.log(`  ${j.name}: ${j.dur}ms, ${j.size}KB`));

    // Check computed styles for contrast issues
    const bodyStyles = await page.evaluate(() => {
      const body = document.body;
      const cs = window.getComputedStyle(body);
      return {
        background: cs.backgroundColor,
        color: cs.color,
        classList: document.documentElement.className,
      };
    });
    console.log('\n=== Body Computed Styles ===');
    console.log('html class:', bodyStyles.classList);
    console.log('background:', bodyStyles.background);
    console.log('color:', bodyStyles.color);

    // Check CSS variable values in dark mode
    const cssVars = await page.evaluate(() => {
      const root = document.documentElement;
      const styles = window.getComputedStyle(root);
      return {
        '--color-bg-main': styles.getPropertyValue('--color-bg-main'),
        '--color-text': styles.getPropertyValue('--color-text'),
        '--color-text-secondary': styles.getPropertyValue('--color-text-secondary'),
        '--color-text-muted': styles.getPropertyValue('--color-text-muted'),
        '--color-bg-elevated': styles.getPropertyValue('--color-bg-elevated'),
        '--color-bg-secondary': styles.getPropertyValue('--color-bg-secondary'),
        '--color-border': styles.getPropertyValue('--color-border'),
        '--color-info': styles.getPropertyValue('--color-info'),
        '--color-success': styles.getPropertyValue('--color-success'),
        '--color-danger': styles.getPropertyValue('--color-danger'),
        '--color-warning': styles.getPropertyValue('--color-warning'),
      };
    });
    console.log('\n=== CSS Variables (dark mode) ===');
    for (const [key, val] of Object.entries(cssVars)) {
      console.log(`${key}: ${val}`);
    }

    // Verify dark class is applied
    expect(bodyStyles.classList).toContain('dark');
  });
});
