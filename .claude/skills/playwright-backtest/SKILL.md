---
name: playwright-backtest
description: |
  Playwright E2E 回测自动化测试模式。涵盖 ConnectRPC API 登录、模板创建、
  UI 登录、Workspace 代码加载、账号/品种/时间周期选择、回测执行、报告提取的完整流程。
  当需要编写或修改 Playwright 回测 E2E 测试、调试回测自动化、或为新策略添加回测冒烟测试时使用此 skill。
  Triggers on: "playwright", "e2e test", "回测测试", "backtest automation",
  "playwright backtest", "automated backtest", "E2E 回测", "浏览器自动化测试"
---

# Playwright E2E 回测自动化模式

> 最后验证: 2026-07-02, 对标 tests/e2e/backtest-venus.spec.ts 通过的完整流程。

## 文件结构

```
tests/e2e/
  playwright.config.ts   — Playwright 配置 (chromium, headless, baseURL)
  package.json           — @playwright/test 依赖
  tsconfig.json          — TypeScript 配置
  backtest-venus.spec.ts — 完整回测 E2E 测试示例
```

## 核心流程 (7 步)

```
Step 1: API 登录 (Node.js fetch → ConnectRPC JSON)
  │ POST /ant.v1.AuthService/Login → accessToken
  ▼
Step 2: 创建模板 (ConnectRPC JSON API)
  │ POST /ant.v1.StrategyService/CreateTemplate → templateId
  │ body: { name, description, code, parameters:[], isPublic:false, tags:[], i18n:'' }
  ▼
Step 3: UI 登录 (Playwright 浏览器)
  │ goto /login → fill #login_login + #login_password → click submit
  │ waitForURL (离开 /login)
  ▼
Step 4: 导航到 Workspace 加载代码
  │ goto /strategy/workspace?templateId={id} → 自动加载 MQL 代码
  ▼
Step 5: 选择账号 + 品种 + 时间周期
  │ .ant-select (first) → type account number → click option
  │ .ant-select (nth 1) → type symbol → click option
  │ .ant-radio-button-wrapper:has-text("15m") → click
  ▼
Step 6: 运行回测
  │ button.ant-btn-primary:has-text("Run") → click
  ▼
Step 7: 等待完成 + 提取报告
  │ Poll for .ant-tag:has-text("Completed") or .ant-statistic-content-value
  │ Extract .ant-statistic title + value pairs
```

## 关键实现细节

### 1. API 登录 (Node.js, 非浏览器)

```typescript
async function loginViaAPI(): Promise<string> {
  const resp = await fetch('http://localhost:8022/ant.v1.AuthService/Login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login: 'admin@1.com', password: '12345678' }),
  });
  if (!resp.ok) throw new Error(`Login API failed: ${resp.status}`);
  const data = await resp.json();
  return data.accessToken;
}
```

**原因**: Zustand auth store 只持久化 `user`，不持久化 `accessToken`。无法从浏览器 localStorage 读取 token。用 Node.js fetch 直接调 ConnectRPC JSON API 获取 token。

### 2. 创建模板 (ConnectRPC JSON 协议)

```typescript
async function createTemplateViaAPI(token: string, code: string, name: string): Promise<string> {
  const resp = await fetch('http://localhost:8022/ant.v1.StrategyService/CreateTemplate', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
    body: JSON.stringify({ name, description, code, parameters: [], isPublic: false, tags: [], i18n: '' }),
  });
  const data = await resp.json();
  return data.id || data.template?.id;
}
```

**ConnectRPC JSON 端点格式**: `/{package}.{ServiceName}/{MethodName}`，POST + `Content-Type: application/json`

### 3. UI 登录 (Ant Design Form)

```typescript
async function login(page: Page) {
  await page.goto('/login', { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);
  await page.locator('#login_login').fill('admin@1.com');
  await page.locator('#login_password').fill('12345678');
  await page.locator('form button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes('/login'), { timeout: 15_000 });
}
```

**选择器**: Ant Design Form 根据 `name` 属性生成 ID: `{formName}_{fieldName}` → `login_login`, `login_password`

### 4. Workspace 代码加载

```typescript
await page.goto(`/strategy/workspace?templateId=${templateId}`, { waitUntil: 'domcontentloaded' });
await page.waitForTimeout(3000);
```

**原理**: `useStrategyWorkspaceState` hook 读取 `searchParams.get('templateId')`，自动调用 `handleSelectTemplate(tid)` 加载代码。无需操作 CodeMirror 或 React fiber。

**避坑**: 不要用 `networkidle`（SSE 连接永远不空闲），用 `domcontentloaded` + `waitForTimeout`。

### 5. 账号/品种/时间周期选择

```typescript
// 账号: 第一个 .ant-select
await page.locator('.ant-select').first().click();
await page.keyboard.type('95172262');
await page.locator('.ant-select-item-option').filter({ hasText: '95172262' }).first().click();

// 品种: 第二个 .ant-select
await page.locator('.ant-select').nth(1).click();
await page.keyboard.type('BTCUSD');
await page.locator('.ant-select-item-option').filter({ hasText: /BTCUSDm/i }).first().click();

// 时间周期: radio button
await page.locator('.ant-radio-button-wrapper').filter({ hasText: '15m' }).first().click();
```

**等待节奏**: 每步操作后 `waitForTimeout(800~1500)` 等待 Ant Design 下拉动画 + 数据加载。

### 6. 运行回测

```typescript
const runButton = page.locator('button.ant-btn-primary').filter({ hasText: /Run|运行/ }).last();
await runButton.waitFor({ state: 'visible', timeout: 10_000 });
await runButton.click();
```

### 7. 等待完成 + 提取报告

```typescript
// 轮询完成状态 (最多 100 秒)
for (let i = 0; i < 50; i++) {
  await page.waitForTimeout(2000);
  const completedTag = page.locator('.ant-tag').filter({ hasText: /^Completed$/ });
  if (await completedTag.first().isVisible().catch(() => false)) { completed = true; break; }
  const stats = page.locator('.ant-statistic-content-value');
  if (await stats.first().isVisible().catch(() => false)) { completed = true; break; }
}

// 提取指标
const metrics = page.locator('.ant-statistic');
for (let i = 0; i < await metrics.count(); i++) {
  const title = await metrics.nth(i).locator('.ant-statistic-title').textContent();
  const value = await metrics.nth(i).locator('.ant-statistic-content-value').textContent();
  report[title.trim()] = value.trim();
}
```

**完成信号** (三选一):
- `.ant-tag` 文本 "Completed" (绿色 success tag)
- `.ant-statistic-content-value` 可见 (指标已渲染)
- `.ant-tag-error` 可见 (回测出错)

## Playwright 配置要点

```typescript
export default defineConfig({
  testDir: '.',
  timeout: 180_000,          // 回测可能需要 30-60 秒
  expect: { timeout: 30_000 },
  fullyParallel: false,       // 串行，避免账号冲突
  workers: 1,
  use: {
    baseURL: 'http://localhost:8022',
    headless: true,
    screenshot: 'only-on-failure',
    trace: 'retain-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
});
```

## 运行命令

```bash
cd tests/e2e && npx playwright test --reporter=list
```

## 常见问题

| 问题 | 原因 | 解决方案 |
|------|------|---------|
| 找不到 `#login_login` | Ant Design Form ID 格式 | 确认 form `name` prop，ID = `{formName}_{fieldName}` |
| `networkidle` 超时 | SSE 连接永远不空闲 | 用 `domcontentloaded` + `waitForTimeout` |
| 无法从浏览器获取 token | Zustand 只持久化 `user` | 用 Node.js fetch 调 API 获取 token |
| React fiber 注入失败 | 生产构建函数名被 minify | 用 `?templateId=` URL 参数加载代码 |
| 回测 0 笔交易 | 策略参数导致（如虚拟下单） | 检查策略逻辑，非系统 bug |
| 选择器找不到 `.ant-select` | 页面未完全加载 | 增加 `waitForTimeout` 或 `waitFor({ state: 'visible' })` |

## 为新策略创建回测测试的步骤

1. 复制 `backtest-venus.spec.ts` 为新文件
2. 替换 `MQL_SOURCE` 常量为新策略源码
3. 修改测试中的账号 ID、品种、时间周期
4. 如需参数覆盖，在 Step 5 和 Step 6 之间添加 Validate + 参数编辑步骤
5. 运行 `npx playwright test <新文件> --reporter=list`
