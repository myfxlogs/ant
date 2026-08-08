# 前端测试基线施工 Spec

> **功能块**：frontend。**关联**：`docs/audits/launch-readiness-assessment.md`（上线缺口 ②）。**状态**：🏗 待施工。**日期**：2026-08-08

---

## 1. 背景（实情，非从零）

前端**测试基建半成品**——`frontend/package.json` 已有：
- `"test": "vitest run"`（scripts）
- `vitest@^4.1.7`、`@testing-library/react@^16`、`@testing-library/jest-dom@^6`（devDeps）

但**缺**：
- 零测试文件（`find frontend/src -name "*.test.tsx"` = 0）
- 无 `vitest.config.*`（find 空）
- 无 jsdom/happy-dom devDep（testing-library 需要 DOM 环境）

栈：React 19 + Vite 7 + Ant Design 6 + Zustand 5 + React Query 5 + react-router 7 + i18next + `@sentry/react`。

## 2. 目标 / 非目标

**目标**：建立可跑、进 CI 的前端测试基线，覆盖**纯逻辑（最高 ROI）**优先：Zustand store 状态机 + 关键 utils，再加少量关键组件冒烟测。`npm test` 绿、CI 跑。

**非目标**：不追求高覆盖率；不做 E2E（见 `e2e-suite-spec.md`）；不重构现有组件。先把"前端改动裸奔"这个零覆盖堵掉。

## 3. 设计

**框架**：vitest + jsdom + @testing-library/react（package.json 已选型，无争议）。

**关键基建——ConnectRPC client mock**：组件大量调用生成的 ConnectRPC 客户端（`frontend/src/client/*.ts`）。测试不能真发 RPC，必须有一个统一的 mock 层，否则带数据获取的组件测试全挂。这是本 spec 的基建核心。

## 4. 实现任务

| # | 任务 | 锚点/产出 |
|----|------|----------|
| 1 | 加 devDep `jsdom`（或 happy-dom） + `vitest.config.ts`（`environment:'jsdom'`、`setupFiles`、`globals:true`） | `frontend/vitest.config.ts` |
| 2 | `src/test/setup.ts`：注册 jest-dom matchers + 全局 mock（window.matchMedia/IntersectionObserver 等 Ant Design 常需） | `frontend/src/test/setup.ts` |
| 3 | **ConnectRPC client mock 工具**：一个 helper，按 service/method 注入 mock 响应（vi.fn），供组件测试用 | `frontend/src/test/mockClient.ts`（NEW，参考现有 `src/client/*.ts` 生成方式） |
| 4 | **Zustand store 测试**（最高 ROI，纯逻辑）：auth store、strategy workspace store、marketplace store 的状态转移 | `frontend/src/**/__tests__/*.test.ts` |
| 5 | **纯 utils 测试**：价格/Decimal 格式化、日期格式化（UTC ms）、i18n key 解析 | 对应 `src/utils/` |
| 6 | **关键组件冒烟测**（少量）：登录表单校验、账户选择器、回测参数表单（render + 一次交互） | testing-library |
| 7 | CI：`npm test`（vitest run）接进 CI，与后端 `go test` 并列 | `.github/workflows/ci.yml`（或等价） |

> **复用核对**：vitest 配置模式（Vite 项目通用）、现有 `src/client/*.ts`（mock 其接口，勿重写生成器）、`stream-pattern`/`ui-patterns` skill 描述的 store/组件结构。NEW：`mockClient.ts` + setup.ts + 测试文件。动工前 `bash scripts/cap.sh` 二次确认。

## 5. 验收 + 对抗证明

- `cd frontend && npm test` 绿；CI 跑前端测试。
- **对抗证明**：故意改坏一个 store 转移（如 auth store 的 logout 不清 token）→ 对应 store 测试必红；改坏 Decimal 格式化 → util 测试必红。"删了还绿 = 测试无效"。
- 覆盖目标（软）：至少覆盖全部 Zustand store + 关键 utils；组件冒烟 ≥5 个。

## 6. 边界

- Ant Design 6 组件的深层 DOM 测试脆——只测**我们自己的逻辑/交互**，不测 antd 内部。
- React Query 的 query 需 `QueryClient` provider 包裹（测试 util 提供）。
- i18next 测试用固定语言包，别依赖 detector。

## 7. 完工回填

`tech-debt-registry.md` 无对应债务条目（这是"没做过"非"继承债"）；完工后在 `launch-readiness-assessment.md` 把缺口 ② 划掉 + handover 变更日志加一行 + 对抗证明。`npm test` 绿为底线，不自行宣告完成。
