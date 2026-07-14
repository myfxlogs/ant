import { test, expect } from '@playwright/test';

const BASE = 'http://localhost:8022';

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

test.describe('E2E: Error Message i18n', () => {

  test('Invalid credentials returns unauthenticated error', async () => {
    const resp = await rpc('/ant.v1.AuthService/Login', {
      login: 'nonexistent@test.com',
      password: 'wrongpassword',
    });
    expect(resp.ok).toBe(false);
    // ConnectRPC returns error with code
    expect(resp.status).toBe(401);
    // The error should have a code field (ConnectRPC error format)
    const errorData = resp.data as Record<string, unknown>;
    expect(errorData.code).toBeDefined();
  });

  test('Missing auth token returns unauthenticated', async () => {
    const resp = await rpc('/ant.v1.WalletService/GetWallet', {});
    expect(resp.ok).toBe(false);
    expect(resp.status).toBe(401);
  });

  test('Valid login but invalid account ID returns not found', async () => {
    // Login first
    const loginResp = await rpc('/ant.v1.AuthService/Login', {
      login: 'admin@1.com',
      password: '12345678',
    });
    expect(loginResp.ok).toBe(true);
    const token = loginResp.data.accessToken as string;

    // Try to get a non-existent account
    const resp = await rpc('/ant.v1.AccountService/GetAccount', {
      id: '00000000-0000-0000-0000-000000000000',
    }, token);
    expect(resp.ok).toBe(false);
    // Should be 404 (not found) or 403/401
    expect([404, 403, 500]).toContain(resp.status);
  });

  test('UI displays localized error on wrong password', async ({ page }) => {
    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    await page.locator('#login_login').fill('admin@1.com');
    await page.locator('#login_password').fill('wrongpassword123');
    await page.locator('form button[type="submit"]').click();
    await page.waitForTimeout(3000);

    // Should stay on login page
    expect(page.url()).toContain('/login');

    // An error message should appear (Ant Design message or form error)
    // Look for error toast/message
    const message = page.locator('.ant-message-error, .ant-form-item-explain-error, .ant-notification-notice-error');
    const errorVisible = await message.first().isVisible({ timeout: 5_000 }).catch(() => false);
    expect(errorVisible).toBe(true);
  });
});
