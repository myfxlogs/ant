// Shared helpers and page definitions for e2e-functional-check.ts
import type { Page } from 'playwright';
import * as path from 'path';

export interface PageDef {
  path: string;
  name: string;
  semanticSelectors: string[];
  interactWith?: string;
  needsAccount?: boolean;
}

export interface Issue {
  page: string;
  type: 'BLANK' | 'ERROR_BOUNDARY' | 'CONSOLE_ERROR' | 'NO_SEMANTIC' | 'NAV_FAIL' | 'TIMEOUT';
  detail: string;
}

export interface PageResult {
  page: string;
  status: 'ok' | 'warn' | 'fail';
  issues: Issue[];
  screenshot?: string;
  url: string;
}

export const MAIN_PAGES: PageDef[] = [
  { path: '/', name: 'Dashboard', semanticSelectors: ['h1, h2, h3, h4, h5, [class*="title"]', '.ant-card', '[class*="statistic"]'] },
  { path: '/accounts/bind', name: 'BindAccount', semanticSelectors: ['.ant-card', 'input, button, [class*="form"]'] },
  { path: '/profile', name: 'Profile', semanticSelectors: ['.ant-card', '.ant-descriptions, .ant-form'] },
  { path: '/strategy/templates', name: 'StrategyTemplates', semanticSelectors: ['.ant-card', '.ant-table, .ant-list, [class*="template"]'] },
  { path: '/strategy/workspace', name: 'StrategyWorkspace', semanticSelectors: ['.ant-btn', 'input, textarea, [class*="editor"]', '[class*="toolbar"]'] },
  { path: '/strategy/assets', name: 'StrategyAssets', semanticSelectors: ['.ant-card', '.ant-table, .ant-list'] },
  { path: '/strategy/schedules', name: 'StrategySchedules', semanticSelectors: ['.ant-card', '.ant-table, .ant-list, .ant-empty'] },
  { path: '/strategy/indicator-catalog', name: 'IndicatorCatalog', semanticSelectors: ['.ant-card', '.ant-table, .ant-list, [class*="indicator"]'] },
  { path: '/strategy/analysis', name: 'AssetAnalysis', semanticSelectors: ['.ant-card', 'input', '.ant-btn'] },
  { path: '/marketplace', name: 'Marketplace', semanticSelectors: ['.ant-card', '.ant-list', '.ant-tabs', '.ant-empty', '[class*="card"]'] },
  { path: '/logs', name: 'Logs', semanticSelectors: ['.ant-table, .ant-list, .ant-card, [class*="log"]'] },
  { path: '/auto-trading', name: 'AutoTrading', semanticSelectors: ['.ant-card', '.ant-table', '.ant-statistic', '.ant-switch', '.ant-form'] },
];

export const ADMIN_PAGES: PageDef[] = [
  { path: '/admin', name: 'AdminDashboard', semanticSelectors: ['.ant-card', '.ant-statistic, [class*="stat"]'] },
  { path: '/admin/users', name: 'UserManagement', semanticSelectors: ['.ant-table, .ant-list, .ant-card'] },
  { path: '/admin/accounts', name: 'AccountManagement', semanticSelectors: ['.ant-table, .ant-list, .ant-card'] },
  { path: '/admin/trading', name: 'TradingMonitor', semanticSelectors: ['.ant-table, .ant-list, .ant-card'] },
  { path: '/admin/logs', name: 'OperationLogs', semanticSelectors: ['.ant-table, .ant-list, .ant-card'] },
  { path: '/admin/config', name: 'SystemConfig', semanticSelectors: ['.ant-card', '.ant-form, .ant-table, .ant-list'] },
  { path: '/admin/jurisdiction', name: 'JurisdictionGate', semanticSelectors: ['.ant-card', '.ant-table, .ant-list, .ant-form'] },
  { path: '/admin/sre/killswitch', name: 'SREKillSwitch', semanticSelectors: ['.ant-card', '.ant-switch, .ant-btn, .ant-table'] },
  { path: '/admin/sre/breakers', name: 'SREBreakers', semanticSelectors: ['.ant-card', '.ant-switch, .ant-btn, .ant-table'] },
  { path: '/admin/sre/canary', name: 'SRECanary', semanticSelectors: ['.ant-card', '.ant-form, .ant-btn'] },
];

export async function checkPage(
  page: Page,
  baseUrl: string,
  def: PageDef,
  screenshotsDir: string,
): Promise<PageResult> {
  const issues: Issue[] = [];
  const consoleErrors: string[] = [];
  const onConsole = (msg: any) => { if (msg.type() === 'error') consoleErrors.push(msg.text()); };
  page.on('console', onConsole);

  try {
    await page.goto(baseUrl + def.path, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForTimeout(3000);
    const url = page.url();

    if (url.includes('/login')) {
      issues.push({ page: def.name, type: 'NAV_FAIL', detail: `Redirected to login — auth required or session expired. URL: ${url}` });
      page.off('console', onConsole);
      return { page: def.name, status: 'fail', issues, url };
    }

    const bodyText = await page.evaluate(() => document.body?.innerText?.trim() || '');
    if (bodyText.length < 20) {
      const hasEmpty = await page.evaluate(() =>
        !!document.querySelector('.ant-empty') || !!document.querySelector('[class*="empty"]') || !!document.querySelector('[class*="no-data"]'));
      if (!hasEmpty) issues.push({ page: def.name, type: 'BLANK', detail: `Body text: ${bodyText.length} chars — "${bodyText.substring(0, 80)}"` });
    }

    const errorText = await page.evaluate(() => {
      const el = document.querySelector('.ant-result-error, [class*="error-boundary"], [class*="ErrorBoundary"]');
      return el?.textContent || '';
    });
    if (errorText && (errorText.includes('Unexpected Error') || errorText.includes('Page Error'))) {
      issues.push({ page: def.name, type: 'ERROR_BOUNDARY', detail: errorText.substring(0, 120) });
    }

    const realErrors = consoleErrors.filter(e =>
      !e.includes('ResolveSession failed') && !e.includes('Failed to fetch') &&
      !e.includes('favicon') && !e.includes('Third-party cookie') && !e.includes('[PaperAccountPanel]'));
    for (const err of realErrors.slice(0, 3)) {
      issues.push({ page: def.name, type: 'CONSOLE_ERROR', detail: err.substring(0, 150) });
    }

    let hasSemantic = false;
    for (const sel of def.semanticSelectors) {
      try { if ((await page.locator(sel).count()) > 0) { hasSemantic = true; break; } } catch { /* skip invalid selector */ }
    }
    if (!hasSemantic && bodyText.length >= 20) {
      issues.push({ page: def.name, type: 'NO_SEMANTIC', detail: `No semantic selectors matched: ${def.semanticSelectors.join(', ')}` });
    }

    await page.screenshot({ path: path.join(screenshotsDir, `${def.name}.png`), fullPage: false });
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    issues.push({ page: def.name, type: 'NAV_FAIL', detail: msg.substring(0, 200) });
  }

  page.off('console', onConsole);
  const status = issues.length === 0 ? 'ok' : issues.some(i => ['BLANK', 'ERROR_BOUNDARY', 'NAV_FAIL'].includes(i.type)) ? 'fail' : 'warn';
  return { page: def.name, status, issues, url: page.url() };
}
