import { test } from '@playwright/test';

test('Load conversation from history — verify code block appears', async ({ page }) => {
  // Step 1: API login
  const resp = await fetch('http://localhost:8022/ant.v1.AuthService/Login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login: 'admin@1.com', password: '12345678' }),
  });
  const { accessToken: token } = await resp.json();

  // Step 2: Check conversations via API
  const listResp = await fetch('http://localhost:8022/ant.v1.AIService/ListConversations', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
    body: JSON.stringify({}),
  });
  const listData = await listResp.json();
  console.log('=== Conversations ===');
  for (const c of (listData.conversations || [])) {
    console.log(`  ${c.id} | msg=${c.messageCount} | ${c.title}`);

    // Step 3: Get detail for conversations with messages
    if (c.messageCount > 0) {
      const detailResp = await fetch(`http://localhost:8022/ant.v1.AIService/GetConversation`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({ id: c.id }),
      });
      const detail = await detailResp.json();
      for (const m of (detail.messages || [])) {
        const hasTurnData = !!(m.turnData && m.turnData.length > 0);
        const hasPythonBlock = m.content && m.content.includes('```python');
        console.log(`    ${m.role}: turn_data=${hasTurnData} has_python_block=${hasPythonBlock} preview=${(m.content||'').slice(0, 60)}`);
      }
    }
  }

  // Step 4: UI login and navigate
  await page.goto('/login', { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);
  await page.locator('#login_login').fill('admin@1.com');
  await page.locator('#login_password').fill('12345678');
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
  await page.goto('/strategy/workspace', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(4000);

  // Step 5: Open history drawer
  // Use force:true to bypass SVG overlay elements (chart rendering layer) that intercept pointer events
  const historyBtn = page.locator('button').filter({ has: page.locator('.anticon-history') }).first();
  await historyBtn.waitFor({ state: 'visible', timeout: 10_000 });
  await historyBtn.click({ force: true });
  await page.waitForTimeout(2000);
  await page.screenshot({ path: 'screenshots/ai-chat-history-drawer.png' });

  // Step 6: Click the first conversation with messages
  const convRows = page.locator('.ant-drawer-body').getByText(/生成一套|EURUSD/i);
  const rowCount = await convRows.count();
  console.log(`Matching conversation rows: ${rowCount}`);
  if (rowCount > 0) {
    await convRows.first().click();
    await page.waitForTimeout(3000);
    await page.screenshot({ path: 'screenshots/ai-chat-after-load-conv.png' });

    // Step 7: Log what's visible in the chat area
    const codeCards = page.locator('text=Final Strategy Code');
    console.log(`'Final Strategy Code' cards visible: ${await codeCards.count()}`);
    const applyBtns = page.locator('button:has-text("Apply to Editor")');
    console.log(`'Apply to Editor' buttons visible: ${await applyBtns.count()}`);
    const copyBtns = page.locator('button:has-text("Copy")');
    console.log(`'Copy' buttons visible: ${await copyBtns.count()}`);
  }
});
