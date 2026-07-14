# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: backtest-venus.spec.ts >> backtest Venus EA on BTCUSDm 15m
- Location: backtest-venus.spec.ts:373:1

# Error details

```
Test timeout of 180000ms exceeded.
```

```
Error: locator.click: Test timeout of 180000ms exceeded.
Call log:
  - waiting for locator('.ant-radio-button-wrapper').filter({ hasText: '15m' }).first()
    - locator resolved to <label class="ant-radio-button-wrapper css-1enej14 css-var-_r_0_ ant-radio-css-var">…</label>
  - attempting click action
    2 × waiting for element to be visible, enabled and stable
      - element is not visible
    - retrying click action
    - waiting 20ms
    2 × waiting for element to be visible, enabled and stable
      - element is not visible
    - retrying click action
      - waiting 100ms
    313 × waiting for element to be visible, enabled and stable
        - element is not visible
      - retrying click action
        - waiting 500ms

```

# Page snapshot

```yaml
- generic [ref=e3]:
  - complementary [ref=e4]:
    - generic [ref=e5]:
      - generic [ref=e7]:
        - img "line-chart" [ref=e9]:
          - img [ref=e10]
        - generic [ref=e12]: AlphaForge
      - menu [ref=e13]:
        - menuitem "home Dashboard" [ref=e14] [cursor=pointer]:
          - img "home" [ref=e15]:
            - img [ref=e16]
          - generic [ref=e18]: Dashboard
        - menuitem "code Strategy" [expanded] [ref=e19] [cursor=pointer]:
          - img "code" [ref=e20]:
            - img [ref=e21]
          - generic [ref=e23]: Strategy
        - menu [ref=e24]:
          - menuitem "code Strategy Workspace" [ref=e25] [cursor=pointer]:
            - img "code" [ref=e26]:
              - img [ref=e27]
            - generic [ref=e29]: Strategy Workspace
          - menuitem "thunderbolt Live Monitor" [ref=e30] [cursor=pointer]:
            - img "thunderbolt" [ref=e31]:
              - img [ref=e32]
            - generic [ref=e34]: Live Monitor
          - menuitem "radar-chart Market Tools" [ref=e35] [cursor=pointer]:
            - img "radar-chart" [ref=e36]:
              - img [ref=e37]
            - generic [ref=e39]: Market Tools
        - menuitem "wallet Wallet" [ref=e40] [cursor=pointer]:
          - img "wallet" [ref=e41]:
            - img [ref=e42]
          - generic [ref=e44]: Wallet
        - menuitem "crown Subscription" [ref=e45] [cursor=pointer]:
          - img "crown" [ref=e46]:
            - img [ref=e47]
          - generic [ref=e49]: Subscription
        - menuitem "dashboard Algo Dashboard" [ref=e50] [cursor=pointer]:
          - img "dashboard" [ref=e51]:
            - img [ref=e52]
          - generic [ref=e54]: Algo Dashboard
        - menuitem "setting Auto Trading" [ref=e55] [cursor=pointer]:
          - img "setting" [ref=e56]:
            - img [ref=e57]
          - generic [ref=e59]: Auto Trading
        - menuitem "pie-chart Analytics" [ref=e60] [cursor=pointer]:
          - img "pie-chart" [ref=e61]:
            - img [ref=e62]
          - generic [ref=e64]: Analytics
        - menuitem "shop Marketplace" [ref=e65] [cursor=pointer]:
          - img "shop" [ref=e66]:
            - img [ref=e67]
          - generic [ref=e69]: Marketplace
        - menuitem "history System Logs" [ref=e70] [cursor=pointer]:
          - img "history" [ref=e71]:
            - img [ref=e72]
          - generic [ref=e74]: System Logs
  - generic [ref=e75]:
    - banner [ref=e76]:
      - generic [ref=e80]: System running normally
      - generic [ref=e81]:
        - generic [ref=e82] [cursor=pointer]:
          - img "wallet" [ref=e83]:
            - img [ref=e84]
          - generic [ref=e86]: "99.51504299"
          - generic [ref=e87]: USD
        - generic "Switch to dark mode" [ref=e88] [cursor=pointer]:
          - img "moon" [ref=e89]:
            - img [ref=e90]
        - img "global" [ref=e93] [cursor=pointer]:
          - img [ref=e94]
        - button "bell" [ref=e97] [cursor=pointer]:
          - img "bell" [ref=e99]:
            - img [ref=e100]
        - generic [ref=e102] [cursor=pointer]:
          - img "user" [ref=e104]:
            - img [ref=e105]
          - generic [ref=e107]: admin
    - main [ref=e108]:
      - generic [ref=e110]:
        - generic [ref=e111]:
          - generic [ref=e112]:
            - generic [ref=e113]:
              - generic "Exness-Trial · 95172262" [ref=e114]:
                - text: Exness-Trial · 95172262
                - combobox [ref=e115]
              - img "down" [ref=e117]:
                - img [ref=e118]
            - generic: BTCUSDm
            - generic [ref=e120]:
              - generic [ref=e121]:
                - generic [ref=e123]: BTCUSDm
                - combobox [active] [ref=e124]
              - img "down" [ref=e126]:
                - img [ref=e127]
          - generic [ref=e130]: Venus EA Backtest Test
          - generic [ref=e131]:
            - generic [ref=e132]:
              - generic [ref=e133]: Balance
              - generic [ref=e134]: $585.97
            - generic [ref=e135]:
              - generic [ref=e136]: Equity
              - generic [ref=e137]: $720.21
            - generic [ref=e138]:
              - generic [ref=e139]: Profit
              - generic [ref=e140]:
                - img "rise" [ref=e141]:
                  - img [ref=e142]
                - text: $134.24
            - generic [ref=e144]:
              - generic [ref=e145]: Free Margin
              - generic [ref=e146]: $720.21
          - button "Positions 1" [ref=e147] [cursor=pointer]:
            - generic [ref=e148]:
              - generic [ref=e149]: Positions
              - generic [ref=e150]: "1"
          - generic [ref=e151]:
            - generic [ref=e152]:
              - generic [ref=e153]: Platform
              - generic [ref=e154]: MT4
            - generic [ref=e155]:
              - generic [ref=e156]: Broker
              - generic [ref=e157]: Exness Technologies Ltd
            - generic [ref=e158]:
              - generic [ref=e159]: Server
              - generic [ref=e160]: Exness-Trial
            - generic [ref=e161]:
              - generic [ref=e162]: Permission
              - generic [ref=e163]: Master
            - generic [ref=e164]:
              - generic [ref=e165]: Leverage
              - generic [ref=e166]: 1:2100000000
        - generic [ref=e167]:
          - generic [ref=e168]:
            - generic [ref=e169]:
              - generic [ref=e170] [cursor=pointer]: 📈 Chart
              - generic [ref=e171] [cursor=pointer]: 📄 Strategy Code
              - generic [ref=e172] [cursor=pointer]: 📊 Backtest
              - generic [ref=e173]:
                - generic [ref=e174]: Venus EA Backtest Test
                - button "save Save" [ref=e175] [cursor=pointer]:
                  - img "save" [ref=e177]:
                    - img [ref=e178]
                  - generic [ref=e180]: Save
                - button "play-circle Backtest" [ref=e181] [cursor=pointer]:
                  - img "play-circle" [ref=e183]:
                    - img [ref=e184]
                  - generic [ref=e187]: Backtest
                - button "copy" [ref=e188] [cursor=pointer]:
                  - img "copy" [ref=e190]:
                    - img [ref=e191]
                - button "robot Send to AI" [ref=e193] [cursor=pointer]:
                  - img "robot" [ref=e195]:
                    - img [ref=e196]
                  - generic [ref=e198]: Send to AI
                - button "question-circle" [ref=e199] [cursor=pointer]:
                  - img "question-circle" [ref=e201]:
                    - img [ref=e202]
            - generic [ref=e208]:
              - generic [ref=e210]:
                - generic [ref=e211]: "1"
                - generic [ref=e212]: "2"
                - generic [ref=e213]: "3"
                - generic [ref=e214]: "4"
                - generic [ref=e215]: "5"
                - generic [ref=e216]: "6"
                - generic [ref=e217]: "7"
                - generic [ref=e218]: "8"
                - generic [ref=e219]: "9"
                - generic [ref=e220]: "10"
                - generic [ref=e221]: "11"
                - generic [ref=e222]: "12"
                - generic [ref=e223]: "13"
                - generic [ref=e224]: "14"
                - generic [ref=e225]: "15"
                - generic [ref=e226]: "16"
                - generic [ref=e227]: "17"
                - generic [ref=e228]: "18"
                - generic [ref=e229]: "19"
                - generic [ref=e230]: "20"
                - generic [ref=e231]: "21"
                - generic [ref=e232]: "22"
                - generic [ref=e233]: "23"
                - generic [ref=e234]: "24"
                - generic [ref=e235]: "25"
                - generic [ref=e236]: "26"
                - generic [ref=e237]: "27"
                - generic [ref=e238]: "28"
                - generic [ref=e239]: "29"
              - textbox [ref=e240]:
                - generic [ref=e241]: "#property copyright \"Venus\""
                - generic [ref=e242]: "#property link \"https://secure.tickmill.co.uk/redirect/index.php?cii=155&cis=1&lp=https%3A%2F%2Fsecure.tickmill.co.uk%2Ftrader%2Findex.php%3Ftask%3D1050%26lang%3D2\""
                - generic [ref=e243]: "#property description \"EA\""
                - generic [ref=e244]: int 帐号限制 = 0;
                - generic [ref=e245]: datetime 时间限制 = D'3020.12.31';
                - generic [ref=e246]: string 作者 = "https://secure.tickmill.co.uk";
                - generic [ref=e248]: extern string 平台选择0 = "IC TickMill FxPro Pepperstone ";
                - generic [ref=e249]: extern string 平台选择1 = "不要选择国产平台,否则后果自负";
                - generic [ref=e250]: extern string 加载货币M15 = "AUDUSD NZDUSD USDJPY GBPUSD EURGBP GBPJPY AUDJPY CHFJPY";
                - generic [ref=e251]: "enum BuyOrSell0 {只做多 = 0,只做空 = 1,多空都做 = 2};"
                - generic [ref=e252]: input BuyOrSell0 多空方向 = 2;
                - generic [ref=e253]: extern bool 显示止盈价格 = true;
                - generic [ref=e254]: extern int 显示浮亏单数 = 7;
                - generic [ref=e255]: extern bool 启用虚拟下单 = true;
                - generic [ref=e256]: extern int 虚拟下单单数 = 3;
                - generic [ref=e257]: extern int 单向最大单数 = 15;
                - generic [ref=e258]: extern double 资金2W单笔最大下单量 = 3;
                - generic [ref=e259]: extern double 起始下单量 = 0.01;
                - generic [ref=e260]: extern double 翻倍 = 1.6;
                - generic [ref=e261]: extern double 单K线限制点数 = 50;
                - generic [ref=e262]: extern int 达到限制暂停小时 = 0;
                - generic [ref=e263]: extern int 间隔单数 = 7;
                - generic [ref=e264]: extern int 单数以下间隔点数 = 20;
                - generic [ref=e265]: extern int 单数以上间隔点数 = 5;
                - generic [ref=e266]: extern int 总体盈利点数 = 10;
                - generic [ref=e267]: extern double 总亏损金额平仓 = 0.0;
                - generic [ref=e268]: extern double 总盈利金额平仓 = 0.0;
                - generic [ref=e269]: extern int 滑点 = 3;
            - generic [ref=e271] [cursor=pointer]:
              - generic [ref=e272]: ▲ Positions (1)
              - generic [ref=e273]: ·
              - generic [ref=e274]: History (1)
              - generic [ref=e275]: ·
              - generic [ref=e276]: Backtest
          - generic [ref=e278]:
            - generic [ref=e279]:
              - generic [ref=e280]: 🤖 AI Assistant
              - button "bulb" [ref=e281] [cursor=pointer]:
                - img "bulb" [ref=e283]:
                  - img [ref=e284]
            - generic [ref=e287]:
              - generic [ref=e288]:
                - generic [ref=e289]: BTCUSDm · 1h
                - generic [ref=e290] [cursor=pointer]:
                  - generic "deepseek-v4-flash (67ad4c3f-362d-4238-a8eb-6c8cd7892414)" [ref=e291]:
                    - text: deepseek-v4-flash (67ad4c3f-362d-4238-a8eb-6c8cd7892414)
                    - combobox [ref=e292]
                  - img "down" [ref=e294]:
                    - img [ref=e295]
                - button "history" [ref=e297] [cursor=pointer]:
                  - img "history" [ref=e299]:
                    - img [ref=e300]
                - button "file-text" [ref=e302] [cursor=pointer]:
                  - img "file-text" [ref=e304]:
                    - img [ref=e305]
                - button "setting" [ref=e307] [cursor=pointer]:
                  - img "setting" [ref=e309]:
                    - img [ref=e310]
              - generic [ref=e315]:
                - 'textbox "Describe the trading strategy you want to create, e.g.: \"Make a Bollinger Band mean-reversion strategy for EURUSD on 1H\"" [ref=e316]'
                - button "send Generate Strategy" [disabled] [ref=e317]:
                  - generic:
                    - img "send":
                      - img
                  - generic: Generate Strategy
```

# Test source

```ts
  320 | }
  321 | `;
  322 | 
  323 | async function login(page: Page) {
  324 |   await page.goto('/login', { waitUntil: 'networkidle' });
  325 |   await page.waitForTimeout(1000);
  326 |   // Ant Design Form generates IDs as {formName}_{fieldName}
  327 |   await page.locator('#login_login').fill('admin@1.com');
  328 |   await page.locator('#login_password').fill('12345678');
  329 |   // Submit button is inside the form
  330 |   await page.locator('form button[type="submit"]').click();
  331 |   // Wait for navigation away from /login
  332 |   await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
  333 | }
  334 | 
  335 | async function loginViaAPI(): Promise<string> {
  336 |   const resp = await fetch('http://localhost:8022/ant.v1.AuthService/Login', {
  337 |     method: 'POST',
  338 |     headers: { 'Content-Type': 'application/json' },
  339 |     body: JSON.stringify({ login: 'admin@1.com', password: '12345678' }),
  340 |   });
  341 |   if (!resp.ok) throw new Error(`Login API failed: ${resp.status}`);
  342 |   const data = await resp.json();
  343 |   return data.accessToken;
  344 | }
  345 | 
  346 | async function createTemplateViaAPI(token: string, code: string, name: string): Promise<string> {
  347 |   const resp = await fetch('http://localhost:8022/ant.v1.StrategyService/CreateTemplate', {
  348 |     method: 'POST',
  349 |     headers: {
  350 |       'Content-Type': 'application/json',
  351 |       'Authorization': `Bearer ${token}`,
  352 |     },
  353 |     body: JSON.stringify({
  354 |       name,
  355 |       description: 'Venus EA for backtest testing',
  356 |       code,
  357 |       parameters: [],
  358 |       isPublic: false,
  359 |       tags: [],
  360 |       i18n: '',
  361 |     }),
  362 |   });
  363 |   if (!resp.ok) {
  364 |     const text = await resp.text();
  365 |     throw new Error(`CreateTemplate failed: ${resp.status} ${text}`);
  366 |   }
  367 |   const data = await resp.json();
  368 |   const id = data.id || data.template?.id || '';
  369 |   if (!id) throw new Error(`CreateTemplate returned no ID: ${JSON.stringify(data)}`);
  370 |   return id;
  371 | }
  372 | 
  373 | test('backtest Venus EA on BTCUSDm 15m', async ({ page }) => {
  374 |   // ── 1. Login via API to get auth token (for template creation) ──
  375 |   const token = await loginViaAPI();
  376 |   console.log('API login successful');
  377 | 
  378 |   // ── 2. Create template via ConnectRPC API ──
  379 |   const templateId = await createTemplateViaAPI(token, MQL_SOURCE, 'Venus EA Backtest Test');
  380 |   console.log('Created template:', templateId);
  381 | 
  382 |   // ── 3. Login via UI (for browser session) ──
  383 |   await login(page);
  384 | 
  385 |   // ── 4. Navigate to workspace with templateId to auto-load code ──
  386 |   await page.goto(`/strategy/workspace?templateId=${templateId}`, { waitUntil: 'domcontentloaded' });
  387 |   await page.waitForTimeout(3000);
  388 | 
  389 |   // ── 5. Select trading account 95172262 ──
  390 |   // Account selector is in the toolbar — look for a Select with account-related placeholder
  391 |   const accountSelect = page.locator('.ant-select').first();
  392 |   await accountSelect.waitFor({ state: 'visible', timeout: 10_000 });
  393 |   await accountSelect.click();
  394 |   await page.waitForTimeout(800);
  395 |   // Type to filter the dropdown
  396 |   await page.keyboard.type('95172262');
  397 |   await page.waitForTimeout(800);
  398 |   // Click the matching option
  399 |   const accountOption = page.locator('.ant-select-item-option').filter({ hasText: '95172262' }).first();
  400 |   await accountOption.waitFor({ state: 'visible', timeout: 5000 });
  401 |   await accountOption.click();
  402 |   await page.waitForTimeout(1500);
  403 | 
  404 |   // ── 6. Select symbol BTCUSDm ──
  405 |   // The symbol picker is another Select in the toolbar
  406 |   const symbolSelect = page.locator('.ant-select').nth(1);
  407 |   await symbolSelect.waitFor({ state: 'visible', timeout: 10_000 });
  408 |   await symbolSelect.click();
  409 |   await page.waitForTimeout(800);
  410 |   await page.keyboard.type('BTCUSD');
  411 |   await page.waitForTimeout(1000);
  412 |   // Look for BTCUSDm in the dropdown
  413 |   const symbolOption = page.locator('.ant-select-item-option').filter({ hasText: /BTCUSDm/i }).first();
  414 |   await symbolOption.waitFor({ state: 'visible', timeout: 5000 });
  415 |   await symbolOption.click();
  416 |   await page.waitForTimeout(1500);
  417 | 
  418 |   // ── 7. Set timeframe to 15m ──
  419 |   const tf15m = page.locator('.ant-radio-button-wrapper').filter({ hasText: '15m' }).first();
> 420 |   await tf15m.click();
      |               ^ Error: locator.click: Test timeout of 180000ms exceeded.
  421 |   await page.waitForTimeout(500);
  422 | 
  423 |   // ── 8. Click Run backtest ──
  424 |   // The Run button is in the backtest panel — find by primary button with Run text
  425 |   const runButton = page.locator('button.ant-btn-primary').filter({ hasText: /Run|运行/ }).last();
  426 |   await runButton.waitFor({ state: 'visible', timeout: 10_000 });
  427 |   await runButton.click();
  428 | 
  429 |   // ── 9. Wait for backtest to complete ──
  430 |   // The "Completed" tag uses ant-tag with color="success" — text is "Completed"
  431 |   // The "Running" tag uses ant-tag with color="processing" — text is "Running"
  432 |   // The "Error" tag uses ant-tag with color="error"
  433 |   // Poll for completion (up to 100 seconds)
  434 |   let completed = false;
  435 |   let errored = false;
  436 |   for (let i = 0; i < 50; i++) {
  437 |     await page.waitForTimeout(2000);
  438 |     // Check for Completed tag (green/success)
  439 |     const completedTag = page.locator('.ant-tag').filter({ hasText: /^Completed$/ });
  440 |     if (await completedTag.first().isVisible().catch(() => false)) {
  441 |       completed = true;
  442 |       break;
  443 |     }
  444 |     // Check for error tag
  445 |     const errorTag = page.locator('.ant-tag-error, .ant-tag').filter({ hasText: /failed|error/i });
  446 |     if (await errorTag.first().isVisible().catch(() => false)) {
  447 |       errored = true;
  448 |       break;
  449 |     }
  450 |     // Also check if statistic values are present (metrics rendered = completed)
  451 |     const stats = page.locator('.ant-statistic-content-value');
  452 |     if (await stats.first().isVisible().catch(() => false)) {
  453 |       completed = true;
  454 |       break;
  455 |     }
  456 |   }
  457 | 
  458 |   // ── 10. Capture backtest report ──
  459 |   const report: Record<string, string> = {};
  460 | 
  461 |   // Extract metric values from Statistic components
  462 |   const metrics = page.locator('.ant-statistic');
  463 |   const metricCount = await metrics.count();
  464 |   for (let i = 0; i < metricCount; i++) {
  465 |     const title = await metrics.nth(i).locator('.ant-statistic-title').textContent().catch(() => '');
  466 |     const value = await metrics.nth(i).locator('.ant-statistic-content-value').textContent().catch(() => '');
  467 |     if (title && value) {
  468 |       report[title.trim()] = value.trim();
  469 |     }
  470 |   }
  471 | 
  472 |   // Also capture the collapsed metrics row if visible
  473 |   const collapsedMetrics = page.locator('div').filter({ hasText: /Total Return|总收益/ }).filter({ has: page.locator('b') });
  474 |   if (await collapsedMetrics.first().isVisible().catch(() => false)) {
  475 |     const text = await collapsedMetrics.first().textContent();
  476 |     if (text) report['collapsedMetrics'] = text.trim();
  477 |   }
  478 | 
  479 |   // Capture execution assumptions if visible
  480 |   const assumptions = page.locator('div').filter({ hasText: /Execution Assumptions|执行假设/ }).first();
  481 |   if (await assumptions.isVisible().catch(() => false)) {
  482 |     const text = await assumptions.textContent();
  483 |     if (text) report['executionAssumptions'] = text.trim();
  484 |   }
  485 | 
  486 |   // Capture error message if errored
  487 |   if (errored) {
  488 |     const errorTagEl = page.locator('.ant-tag-error').first();
  489 |     const errorText = await errorTagEl.textContent().catch(() => 'Unknown error');
  490 |     report['error'] = errorText?.trim() || 'Unknown error';
  491 |   }
  492 | 
  493 |   // ── 11. Output report ──
  494 |   console.log('\n========== BACKTEST REPORT ==========');
  495 |   console.log(`Status: ${completed ? 'COMPLETED' : errored ? 'ERROR' : 'TIMEOUT'}`);
  496 |   console.log(`Symbol: BTCUSDm | Timeframe: 15m`);
  497 |   console.log('---');
  498 |   for (const [key, value] of Object.entries(report)) {
  499 |     if (key === 'collapsedMetrics' || key === 'executionAssumptions') {
  500 |       console.log(`${key}: ${value}`);
  501 |     } else {
  502 |       console.log(`${key}: ${value}`);
  503 |     }
  504 |   }
  505 |   console.log('=====================================\n');
  506 | 
  507 |   // Take a screenshot for the record
  508 |   await page.screenshot({ path: 'backtest-result.png', fullPage: true });
  509 | 
  510 |   // Assert that we got some result
  511 |   if (errored) {
  512 |     console.log('Backtest returned an error — this may indicate a compilation or runtime issue with the MQL code.');
  513 |   }
  514 |   expect(completed || errored).toBeTruthy();
  515 | });
  516 | 
```