import { test, expect } from '@playwright/test';

test('History drawer should be empty on first visit', async ({ page }) => {
  // Step 1: Login via API to clear any session state
  const resp = await fetch('http://localhost:8022/ant.v1.AuthService/Login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login: 'admin@1.com', password: '12345678' }),
  });
  const data = await resp.json();
  const token = data.accessToken;
  console.log('Login token:', token ? 'OK' : 'FAILED');

  // Step 2: List current conversations via API
  const listResp = await fetch('http://localhost:8022/ant.v1.AIService/ListConversations', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
    body: JSON.stringify({}),
  });
  const listData = await listResp.json();
  console.log('Conversations before test:', JSON.stringify(listData, null, 2));

  // Step 3: UI login
  await page.goto('/login', { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);
  await page.locator('#login_login').fill('admin@1.com');
  await page.locator('#login_password').fill('12345678');
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });

  // Step 4: Navigate to workspace
  await page.goto('/strategy/workspace', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(3000);

  // Step 5: Click History button in AI Chat toolbar
  const historyBtn = page.locator('button').filter({ has: page.locator('.anticon-history') }).first();
  await historyBtn.waitFor({ state: 'visible', timeout: 10_000 });
  await historyBtn.click();
  await page.waitForTimeout(1500);

  // Step 6: Take screenshot of the history drawer
  await page.screenshot({ path: 'screenshots/ai-chat-history-drawer.png', fullPage: false });

  // Step 7: Check if there are conversation items in the drawer
  const convItems = page.locator('.ant-drawer-body').locator('> div > div'); // conversation rows
  const count = await convItems.count();
  console.log(`Conversation items in drawer: ${count}`);

  // List conversations via API again to see if any were auto-created
  const listResp2 = await fetch('http://localhost:8022/ant.v1.AIService/ListConversations', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
    body: JSON.stringify({}),
  });
  const listData2 = await listResp2.json();
  console.log('Conversations after page visit:', JSON.stringify(listData2, null, 2));
});
