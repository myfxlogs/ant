import { test, expect } from '@playwright/test';

// ── E2E: MQL Import → Workspace Integration ──────────────────────────────
// Verifies the full user journey: login → open workspace → import MQL →
// verify code loaded in editor → verify strategy ID persisted → query via API.

const BASE = 'http://localhost:8022';
const ADMIN_EMAIL = 'admin@1.com';
const ADMIN_PASS = '12345678';

async function rpc(path: string, body: Record<string, unknown>, token?: string) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const resp = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  const text = await resp.text();
  let data: Record<string, unknown>;
  try { data = JSON.parse(text); } catch { data = { _raw: text }; }
  return { status: resp.status, data, ok: resp.ok };
}

// Simple MQL4 EA with #property indicator_name for name extraction test
const MQL4_SOURCE = `#property indicator_name "E2EImportTest"
extern int FastPeriod = 5;
extern double Lots = 0.1;

int OnInit() { return INIT_SUCCEEDED; }
void OnDeinit(const int reason) {}
void OnTick() {
  double fastMA = iMA(Symbol(), 0, FastPeriod, 0, MODE_SMA, PRICE_CLOSE, 0);
  if (fastMA > 0) {
    OrderSend(Symbol(), OP_BUY, Lots, Ask, 5, 0, 0, "E2E", 123, 0, 0);
  }
}`;

test.describe.serial('E2E: MQL Import → Workspace Integration', () => {
  let adminToken: string;

  test('1. Admin login', async () => {
    const resp = await rpc('/ant.v1.AuthService/Login', {
      login: ADMIN_EMAIL,
      password: ADMIN_PASS,
    });
    expect(resp.ok).toBe(true);
    adminToken = resp.data.accessToken as string;
    expect(adminToken.length).toBeGreaterThan(10);
  });

  test('2. SubmitStrategy — import MQL4 (no backtest)', async () => {
    const resp = await rpc('/ant.v1.AgentGatewayService/SubmitStrategy', {
      sourceCode: MQL4_SOURCE,
      language: 'mql4',
      mode: 1, // SYNC
    }, adminToken);

    expect(resp.ok).toBe(true);
    expect(resp.data.compileSuccess).toBe(true);
    expect(resp.data.strategyId).toBeTruthy();
    expect(typeof resp.data.coverageScore).toBe('number');
    expect(resp.data.bridgeStatus).toBe('not_attempted');

    // Store for next test
    (test as any).strategyId = resp.data.strategyId;
    (test as any).coverageScore = resp.data.coverageScore;
  });

  test('3. GetImportedStrategy — verify persistence', async () => {
    const sid = (test as any).strategyId as string;
    expect(sid).toBeTruthy();

    const resp = await rpc('/ant.v1.StrategyRuntimeService/GetImportedStrategy', {
      strategyId: sid,
    }, adminToken);

    expect(resp.ok).toBe(true);
    expect(resp.data.strategyId).toBe(sid);
    expect(resp.data.strategyName).toBe('E2EImportTest');
    expect(resp.data.sourceLang).toBe('mql4');
    expect(resp.data.sourceCode).toContain('#property indicator_name');
    expect(resp.data.coverageScore).toBe((test as any).coverageScore);
  });

  test('4. DB verification — imported_strategies row', async () => {
    // This test runs via the API; DB check is done implicitly through GetImportedStrategy
    // If the row doesn't exist, GetImportedStrategy would return NotFound
    const sid = (test as any).strategyId as string;
    const resp = await rpc('/ant.v1.StrategyRuntimeService/GetImportedStrategy', {
      strategyId: sid,
    }, adminToken);
    expect(resp.ok).toBe(true);
    expect(resp.data.strategyName).toBe('E2EImportTest');
  });

  // Helper: login via UI and navigate to strategy workspace
  async function loginAndGoToWorkspace(page: import('@playwright/test').Page) {
    await page.goto('/login');
    await page.fill('input[type="email"], input[id*="email"], input[placeholder*="mail"]', ADMIN_EMAIL);
    await page.fill('input[type="password"], input[id*="password"], input[placeholder*="assword"]', ADMIN_PASS);
    await page.click('button[type="submit"]');
    // Login redirects to / (dashboard), not /strategy
    await page.waitForURL('http://localhost:8022/', { timeout: 15000 });

    // Dismiss welcome dialog if present
    const dismissBtn = page.locator('button:has-text("Got it"), button:has-text("dismiss")');
    if (await dismissBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await dismissBtn.click();
      await page.waitForTimeout(500);
    }

    // Navigate to strategy workspace
    await page.goto('/strategy/workspace');
    await page.waitForTimeout(3000);

    // Dismiss product tour if present (overlay blocks buttons)
    const tourClose = page.locator('button:has-text("Close"), button.ant-tour-close').first();
    if (await tourClose.isVisible({ timeout: 3000 }).catch(() => false)) {
      await tourClose.click();
      await page.waitForTimeout(500);
    }

    // Switch to "Strategy Code" tab so the import button becomes visible
    const codeTab = page.locator('text=Strategy Code').first();
    if (await codeTab.isVisible({ timeout: 5000 }).catch(() => false)) {
      await codeTab.click();
      await page.waitForTimeout(1000);
    }
  }

  test('5. UI: Workspace loads and import drawer opens', async ({ page }) => {
    await loginAndGoToWorkspace(page);

    // Look for import button in the workspace toolbar ("Import MQL" or i18n variant)
    const importBtn = page.locator('button:has-text("Import MQL"), button:has-text("Import"), button:has-text("导入")').first();
    await expect(importBtn).toBeVisible({ timeout: 10000 });
    await importBtn.click();
    await page.waitForTimeout(1000);

    // ImportEAPanel textarea should be visible inside the drawer
    const textarea = page.locator('textarea[placeholder*="MQL"], textarea[placeholder*="Paste"]').first();
    await expect(textarea).toBeVisible({ timeout: 5000 });
  });

  test('6. UI: Paste code → MQL version tag appears', async ({ page }) => {
    await loginAndGoToWorkspace(page);

    // Open import drawer
    const importBtn = page.locator('button:has-text("Import MQL"), button:has-text("Import"), button:has-text("导入")').first();
    await expect(importBtn).toBeVisible({ timeout: 10000 });
    await importBtn.click();
    await page.waitForTimeout(1000);

    // Find textarea in the import panel (placeholder contains "MQL4/MQL5")
    const textarea = page.locator('textarea[placeholder*="MQL"], textarea[placeholder*="Paste"]').first();
    await expect(textarea).toBeVisible({ timeout: 5000 });
    await textarea.fill(MQL4_SOURCE);
    await page.waitForTimeout(500);

    // MQL4 tag should appear (blue tag)
    const mqlTag = page.locator('.ant-tag:has-text("MQL4")').first();
    await expect(mqlTag).toBeVisible({ timeout: 3000 });

    // Import button should be enabled (not disabled)
    const importBtn2 = page.locator('button:has-text("导入"), button:has-text("Import")').last();
    await expect(importBtn2).toBeEnabled({ timeout: 3000 });
  });

  test('7. UI: Empty code → no MQL version tag', async ({ page }) => {
    await loginAndGoToWorkspace(page);

    const importBtn = page.locator('button:has-text("Import MQL"), button:has-text("Import"), button:has-text("导入")').first();
    await expect(importBtn).toBeVisible({ timeout: 10000 });
    await importBtn.click();
    await page.waitForTimeout(1000);

    const textarea = page.locator('textarea[placeholder*="MQL"], textarea[placeholder*="Paste"]').first();
    await expect(textarea).toBeVisible({ timeout: 5000 });
    await textarea.fill('');
    await page.waitForTimeout(500);

    // MQL4/MQL5 tag should NOT be visible
    const mqlTag = page.locator('.ant-tag:has-text("MQL4")').first();
    await expect(mqlTag).not.toBeVisible({ timeout: 2000 });
  });
});
