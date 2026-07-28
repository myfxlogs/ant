import { test, expect, type Page } from '@playwright/test';

const PROMPT = `生成一套当前图表中商品的自动交易的代码，入场标准：四小时放量大跌后，强力反弹吞没前四小时K线位置；入场时机：1小时收线后再回调50%后，5分钟K线再出现反转吞没入场。交易手数：按照资金余额可以抗5美金空间来设置。盈利加仓：产生盈利立马加仓一倍后停止加仓，XAUUSD走出10美金空间后回调40%，再加仓前面手数的2倍，后面不再加仓。出场标准：小时图收线出现反向，或者总体盈利有200倍。`;

async function login(page: Page) {
  await page.goto('/login', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1500);
  await page.locator('#login_login').fill('admin@1.com');
  await page.locator('#login_password').fill('12345678');
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
  await page.waitForTimeout(2000);
}

test('AI Chat screenshot - compare with Windsurf', async ({ page }) => {
  await login(page);

  // Navigate to workspace
  await page.goto('/strategy/workspace', { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(3000);

  // Step 1: Select account (first .ant-select)
  // Use force:true to bypass SVG chart overlay that intercepts pointer events
  const accountSelect = page.locator('.ant-select').first();
  await accountSelect.waitFor({ state: 'visible', timeout: 10_000 });
  await accountSelect.click({ force: true });
  await page.waitForTimeout(1000);
  // Click the first available account option, or skip if no accounts bound
  const accountOption = page.locator('.ant-select-item-option').first();
  if (await accountOption.isVisible({ timeout: 5_000 }).catch(() => false)) {
    await accountOption.click();
    await page.waitForTimeout(2000);

    // Step 2: Select symbol BTCUSDm (second .ant-select in the top bar)
    const symbolSelect = page.locator('.ant-select').nth(1);
    await symbolSelect.click();
    await page.waitForTimeout(500);
    await page.keyboard.type('BTCUSD');
    await page.waitForTimeout(1500);
    // Try to find BTCUSDm option
    const btcOption = page.locator('.ant-select-item-option').filter({ hasText: /BTCUSDm/i }).first();
    if (await btcOption.isVisible({ timeout: 5000 }).catch(() => false)) {
      await btcOption.click();
    } else {
      // Just click first option available
      await page.locator('.ant-select-item-option').first().click();
    }
    await page.waitForTimeout(2000);
  } else {
    // No accounts bound — close dropdown and continue without account/symbol selection
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);
  }

  // Step 3: Find the AI chat textarea (already visible on page)
  const chatInput = page.locator('textarea').last();
  await chatInput.waitFor({ state: 'visible', timeout: 10_000 });
  await chatInput.fill(PROMPT);
  await page.waitForTimeout(500);

  // Screenshot: before sending
  await page.screenshot({ path: 'screenshots/ai-chat-before-send.png', fullPage: false });

  // Step 4: Click "Generate Strategy" button
  const sendBtn = page.locator('button').filter({ hasText: /Generate Strategy|Send|发送/i }).last();
  await sendBtn.click();

  // Wait for AI response to start streaming
  await page.waitForTimeout(5000);

  // Screenshot: during streaming / planning
  await page.screenshot({ path: 'screenshots/ai-chat-planning.png', fullPage: false });

  // Wait more for plan card or response
  await page.waitForTimeout(15000);
  await page.screenshot({ path: 'screenshots/ai-chat-after-plan.png', fullPage: false });

  // Wait even more for potential code generation
  await page.waitForTimeout(30000);
  await page.screenshot({ path: 'screenshots/ai-chat-final.png', fullPage: false });

  // Also capture the AI chat area specifically
  // The AI chat is the section with the textarea and "Strategy AI" header
  const aiSection = page.locator('text=Strategy AI').locator('..');
  if (await aiSection.isVisible().catch(() => false)) {
    const box = await aiSection.boundingBox();
    if (box) {
      await page.screenshot({
        path: 'screenshots/ai-chat-area-only.png',
        clip: { x: 0, y: box.y - 50, width: page.viewportSize()?.width || 1280, height: box.height + 100 },
      });
    }
  }
});
