import { test, expect } from '@playwright/test';

// ── E2E Regression: Date Range Filter (SQL syntax bug H-1) ──────────────────
// Verifies that the 4 endpoints affected by the addFilter("created_at >=", val)
// SQL syntax bug now work correctly with date range parameters.
//
// Bug: addFilter generated "AND created_at >= = $N" (double =, invalid SQL)
// Fix: inline parameterized SQL for date range conditions
//
// Affected endpoints:
//   1. /ant.v1.LogService/GetConnectionLogs
//   2. /ant.v1.LogService/GetOrderLogHistory
//   3. /ant.v1.LogService/GetOperationLogs
//   4. /ant.v1.AutoTradingService/GetTradingLogs

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

async function login(): Promise<string> {
  const { status, data, ok } = await rpc('/ant.v1.AuthService/Login', {
    login: ADMIN_EMAIL,
    password: ADMIN_PASS,
  });
  if (!ok) throw new Error(`Login failed: ${status} ${JSON.stringify(data)}`);
  return (data as { accessToken: string }).accessToken;
}

test.describe.serial('Date Range Filter — SQL H-1 regression', () => {
  let token: string;

  test.beforeAll(async () => {
    token = await login();
  });

  // Helper: call an endpoint with date range and assert no 500 error.
  async function testDateRange(
    endpoint: string,
    baseBody: Record<string, unknown>,
    startDate: string,
    endDate: string,
  ) {
    const body = { ...baseBody, startDate, endDate };
    const { status, data, ok } = await rpc(endpoint, body, token);

    // 500 = SQL syntax error (the bug we fixed)
    // 200 = query succeeded (even if 0 rows returned)
    expect(status, `${endpoint} with date range should not 500 (SQL bug H-1)`).not.toBe(500);
    expect(ok, `${endpoint} should succeed: ${JSON.stringify(data)}`).toBe(true);
  }

  test('GetConnectionLogs with date range', async () => {
    await testDateRange(
      '/ant.v1.LogService/GetConnectionLogs',
      { page: 1, pageSize: 10 },
      '2025-01-01',
      '2026-12-31',
    );
  });

  test('GetConnectionLogs without date range (baseline)', async () => {
    const { status, ok } = await rpc(
      '/ant.v1.LogService/GetConnectionLogs',
      { page: 1, pageSize: 10 },
      token,
    );
    expect(status).not.toBe(500);
    expect(ok).toBe(true);
  });

  test('GetOrderLogHistory with date range', async () => {
    await testDateRange(
      '/ant.v1.LogService/GetOrderLogHistory',
      { page: 1, pageSize: 10 },
      '2025-01-01',
      '2026-12-31',
    );
  });

  test('GetOperationLogs with date range', async () => {
    await testDateRange(
      '/ant.v1.LogService/GetOperationLogs',
      { page: 1, pageSize: 10 },
      '2025-01-01',
      '2026-12-31',
    );
  });

  test('GetTradingLogs with date range', async () => {
    await testDateRange(
      '/ant.v1.AutoTradingService/GetTradingLogs',
      { page: 1, pageSize: 10 },
      '2025-01-01',
      '2026-12-31',
    );
  });

  test('GetConnectionLogs with only startDate', async () => {
    const body = { page: 1, pageSize: 10, startDate: '2025-01-01' };
    const { status, ok } = await rpc('/ant.v1.LogService/GetConnectionLogs', body, token);
    expect(status, 'startDate-only should not 500').not.toBe(500);
    expect(ok).toBe(true);
  });

  test('GetConnectionLogs with only endDate', async () => {
    const body = { page: 1, pageSize: 10, endDate: '2026-12-31' };
    const { status, ok } = await rpc('/ant.v1.LogService/GetConnectionLogs', body, token);
    expect(status, 'endDate-only should not 500').not.toBe(500);
    expect(ok).toBe(true);
  });

  test('GetConnectionLogs with empty date range (baseline)', async () => {
    const body = { page: 1, pageSize: 10, startDate: '', endDate: '' };
    const { status, ok } = await rpc('/ant.v1.LogService/GetConnectionLogs', body, token);
    expect(status).not.toBe(500);
    expect(ok).toBe(true);
  });
});
