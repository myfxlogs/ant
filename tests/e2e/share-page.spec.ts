import { test, expect } from '@playwright/test';

const BASE = 'http://localhost:8022';
const ADMIN_EMAIL = 'admin@1.com';
const ADMIN_PASS = '12345678';
const ADMIN_ACCOUNT_ID = 'fcca3414-d691-4a41-a1dc-53d914655059';

async function rpc(path: string, body: Record<string, unknown>, token?: string) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const r = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  const text = await r.text();
  let data: Record<string, unknown>;
  try { data = JSON.parse(text); } catch { data = { _raw: text }; }
  return { status: r.status, data, ok: r.ok };
}

test.describe.serial('E2E: Share Page Render', () => {
  const state: { token: string; shareToken: string } = { token: '', shareToken: '' };

  test('1. Create share token via API', async () => {
    // Login as admin
    const loginResp = await rpc('/ant.v1.AuthService/Login', {
      login: ADMIN_EMAIL,
      password: ADMIN_PASS,
    });
    expect(loginResp.ok).toBe(true);
    state.token = loginResp.data.accessToken as string;

    // Create share token for admin's account
    const resp = await rpc('/ant.v1.ShareService/CreateShareToken', {
      accountId: ADMIN_ACCOUNT_ID,
      showPositions: false,
    }, state.token);
    expect(resp.ok, `CreateShareToken should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    state.shareToken = resp.data.token as string;
    expect(state.shareToken.length).toBeGreaterThan(5);
    console.log(`Share token created: ${state.shareToken}`);
  });

  test('2. Share page renders without auth', async ({ page }) => {
    expect(state.shareToken, 'Share token must exist from previous test').toBeTruthy();

    // Navigate to public share page (no login required)
    await page.goto(`/share/${state.shareToken}`, { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    // The page should render — look for key elements
    // SharePerformancePage shows user name, performance metrics, equity curve
    const bodyText = await page.locator('body').textContent();

    // Should NOT show a login redirect or auth error
    expect(page.url()).toContain('/share/');

    // Should show some performance content (user name, metrics, or chart)
    // The page may show "no data" if account has no trades, but it should render
    expect(bodyText).toMatch(/performance|Equity|Balance|Profit|Drawdown|收益|净值|曲线/i);
  });

  test('3. Invalid share token shows error or empty state', async ({ page }) => {
    await page.goto('/share/invalid-token-12345', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(3000);

    // Should not crash — either error message or empty state
    const bodyText = await page.locator('body').textContent();
    expect(bodyText?.length).toBeGreaterThan(0);
    // Should NOT redirect to login
    expect(page.url()).toContain('/share/');
  });
});
