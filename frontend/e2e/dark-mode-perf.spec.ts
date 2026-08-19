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
    await page.waitForTimeout(3000);

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
      await page.goto(p.path);
      // Wait for network to settle (SSE streams stay open, so use domcontentloaded + timeout)
      await page.waitForLoadState('domcontentloaded', { timeout: 30_000 }).catch(() => {});
      await page.waitForTimeout(3000);
      const elapsed = Date.now() - start;
      timings.push({ page: p.name, ms: elapsed });
      console.log(`[${p.name}] loaded in ${elapsed}ms`);

      await page.screenshot({ path: `e2e/screenshots/dark-${p.name}.png`, fullPage: true });
    }

    // Log all timings
    console.log('\n=== Page Load Times (dark mode) ===');
    for (const t of timings) {
      console.log(`${t.page}: ${t.ms}ms`);
    }

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
