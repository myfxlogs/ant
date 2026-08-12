import { test, expect } from '@playwright/test';

// ── E2E: Deploy Schedule → Navigate to Live Monitor → Highlight ──────────
// Verifies: login → strategy gallery → deploy modal → create schedule →
// auto-navigate to /strategy/live?tab=schedules&scheduleId=xxx → row highlight

const BASE = 'http://localhost:8022';
const USER_EMAIL = 'xianhua.chan@gmail.com';
const USER_PASS = 'Abc123456...';

// Known active account for this user (Exness-Trial, login 95262066 — connected)
const ACCOUNT_LOGIN = '95262066';
const SYMBOL = 'BTCUSDm';
const TIMEFRAME = '15m';

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

test.describe.serial('E2E: Deploy Schedule Two-Step Flow', () => {
  let token: string;
  let templateId: string;
  let createdScheduleId: string;

  test('1. API login as user', async () => {
    const resp = await rpc('/ant.v1.AuthService/Login', {
      login: USER_EMAIL,
      password: USER_PASS,
    });
    expect(resp.ok).toBe(true);
    token = resp.data.accessToken as string;
    expect(token.length).toBeGreaterThan(10);
  });

  test('2. Find MACD SAMPLE (Fork) template via API', async () => {
    const resp = await rpc('/ant.v1.StrategyService/ListSchedules', {}, token);
    // Just verify token works
    expect(resp.ok).toBe(true);

    // Query templates to find the one we want
    const tplResp = await rpc('/ant.v1.StrategyService/ListTemplates', {}, token);
    expect(tplResp.ok).toBe(true);
    const templates = tplResp.data.templates as Array<{ id: string; name: string }>;
    const found = templates.find(t => t.name === 'MACD SAMPLE (Fork)');
    expect(found).toBeTruthy();
    templateId = found!.id;
  });

  test('3. UI login and navigate to strategy gallery', async ({ page }) => {
    await page.goto('/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await page.locator('#login_login').fill(USER_EMAIL);
    await page.locator('#login_password').fill(USER_PASS);
    await page.locator('form button[type="submit"]').click();
    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });

    // Navigate to strategy gallery
    await page.goto('/strategy', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    // Verify we're on the strategy page
    expect(page.url()).toContain('/strategy');
  });

  test('4. Open Deploy modal for MACD SAMPLE (Fork)', async ({ page }) => {
    // Login first
    await page.goto('/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await page.locator('#login_login').fill(USER_EMAIL);
    await page.locator('#login_password').fill(USER_PASS);
    await page.locator('form button[type="submit"]').click();
    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });

    await page.goto('/strategy', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    // Find the MACD SAMPLE (Fork) card and click Deploy
    // Strategy cards display the name in a Text strong element
    const cardText = page.locator('text=MACD SAMPLE (Fork)').first();
    await expect(cardText).toBeVisible({ timeout: 10_000 });

    // Find the Deploy button near this card
    // The Deploy button has a RocketOutlined icon and "Deploy" text
    const deployBtn = page.locator('button:has-text("Deploy")').first();
    await expect(deployBtn).toBeVisible({ timeout: 5_000 });
    await deployBtn.click();

    // Verify the deploy modal appeared
    await expect(page.locator('.ant-modal')).toBeVisible({ timeout: 5_000 });
    // Verify modal title
    await expect(page.locator('.ant-modal-title')).toBeVisible({ timeout: 5_000 });
  });

  test('5. Fill deploy form and create schedule', async ({ page }) => {
    // Login
    await page.goto('/login', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(1000);
    await page.locator('#login_login').fill(USER_EMAIL);
    await page.locator('#login_password').fill(USER_PASS);
    await page.locator('form button[type="submit"]').click();
    await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });

    await page.goto('/strategy?_t=' + Date.now(), { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    // Open deploy modal
    const deployBtn = page.locator('button:has-text("Deploy")').first();
    await expect(deployBtn).toBeVisible({ timeout: 10_000 });
    await deployBtn.click();
    await expect(page.locator('.ant-modal')).toBeVisible({ timeout: 5_000 });

    // Wait for form to render and accounts to load
    await page.waitForTimeout(2000);

    // The modal form has 4 ant-select elements in order:
    // 0: account, 1: symbol (SymbolPicker), 2: timeframe, 3: scheduleType
    // Use form-item labels to be more robust

    // --- Select account ---
    const accountFormItem = page.locator('.ant-modal .ant-form-item').filter({ hasText: /account|账号|Account/i }).first();
    await accountFormItem.locator('.ant-select').click();
    await page.waitForTimeout(1000);
    const accountOption = page.locator('.ant-select-item:has-text("' + ACCOUNT_LOGIN + '")').first();
    await expect(accountOption).toBeVisible({ timeout: 10_000 });
    await accountOption.click();
    await page.waitForTimeout(1000);

    // --- MANUAL STEP: Select symbol and timeframe ---
    // The test will pause here. Manually:
    // 1. Select symbol "BTCUSDm" from the dropdown
    // 2. Select timeframe "15m"
    // 3. Click the Create/Deploy button
    // 4. The test will resume automatically after navigation
    console.log('⏸️  PAUSED: Please manually select symbol BTCUSDm, timeframe 15m, then click Create');
    console.log('⏸️  The test will continue after you click Create and navigation occurs');

    // Wait for navigation to /strategy/live (user will trigger this by clicking Create)
    const navPromise = page.waitForURL((url) => url.pathname === '/strategy/live', { timeout: 120_000 }).catch(() => null);

    // Pause for manual interaction
    await page.pause();

    // Wait for navigation to Live Strategy Monitor
    await navPromise;

    // Verify we navigated to /strategy/live
    expect(page.url()).toContain('/strategy/live');
    expect(page.url()).toContain('tab=schedules');
    expect(page.url()).toContain('scheduleId=');

    // Extract scheduleId from URL
    const url = new URL(page.url());
    const scheduleIdParam = url.searchParams.get('scheduleId');
    expect(scheduleIdParam).toBeTruthy();
    createdScheduleId = scheduleIdParam!;

    // Verify the Schedules tab is active
    await page.waitForTimeout(3000);
    const schedulesTab = page.locator('.ant-tabs-tab-active').filter({ hasText: /Schedules|调度/ });
    await expect(schedulesTab).toBeVisible({ timeout: 10_000 });

    // Verify the schedule row with the created ID is visible in the table
    const scheduleRow = page.locator(`tr:has-text("${createdScheduleId}")`).first();
    await expect(scheduleRow).toBeVisible({ timeout: 10_000 });

    // Verify highlight CSS class is applied
    const hasHighlight = await scheduleRow.evaluate((el) => {
      return el.classList.contains('schedule-row-highlight') ||
             el.querySelector('.schedule-row-highlight') !== null;
    });
    expect(hasHighlight).toBe(true);
  });

  test('6. Verify schedule is inactive (is_active=false) via API', async () => {
    expect(createdScheduleId).toBeTruthy();
    const resp = await rpc('/ant.v1.StrategyService/GetSchedule', {
      id: createdScheduleId,
    }, token);
    expect(resp.ok).toBe(true);
    expect(resp.data.isActive).toBe(false);
  });

  test('7. Cleanup — delete the test schedule', async () => {
    if (!createdScheduleId) return;
    const resp = await rpc('/ant.v1.StrategyService/DeleteSchedule', {
      id: createdScheduleId,
    }, token);
    expect(resp.ok).toBe(true);
  });
});
