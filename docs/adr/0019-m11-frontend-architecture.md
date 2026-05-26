# ADR-0019 · M11 前端架构

- **状态**：Accepted
- **日期**：2026-05-26
- **决策者**：myfxlogs
- **关联**：ADR-0007 §7、ROADMAP M11

## 1. 背景

ADR-0007 记录了旧 53 页前端被删除的事实，并要求 M10 关闭后独立启动 M11 前端专项。当前前端已从"2 页最小实现"逐步扩展为 20+ 路由的功能性 SPA，但架构债务需要系统性偿还：loading/empty/error 三态缺失、数据获取分散、缺少错误边界。

## 2. 决策

**增量重构，不推倒重来。** 当前 React 19 + Vite 7 + TypeScript + Ant Design + Zustand + ConnectRPC 栈保持不变。在此基础上：

1. 引入 `@tanstack/react-query` 作为统一数据获取/缓存层
2. 每个页面必须有 loading/empty/error 三态
3. React ErrorBoundary 包裹每个路由页面
4. transport.ts 增加 token 过期自动刷新
5. 后端 mock/stub 服务按页面优先级逐步升级为真实实现

## 3. 备选方案

| 方案 | 优点 | 缺点 | 否决理由 |
|------|------|------|----------|
| A. 推倒重来 (Next.js/Remix) | 全新架构，SSR/SSG 能力 | 8-12 周工期，现有业务逻辑全部重写 | ROI 极低，当前页面已可用 |
| B. 继续修补 | 零风险，改哪坏哪 | 不解决系统性债务，质量继续劣化 | 用户要求的是"重建"非"修补" |
| C. 增量重构 (**选中**) | 保留可用代码，聚焦质量提升 | 需明确边界防止过度重写 | — |

## 4. 后果

- 正面：现有 20+ 页面业务逻辑零损失，TypeScript 编译始终不破
- 正面：数据获取层从 6 个松散 Zustand store 统一到 TanStack Query，缓存/去重/重试开箱即用
- 负面：无法获得 SSR/SSG（但 ant 是登录后 SPA，SSR 无收益）
- 中性：Ant Design 6.x 绑定，未来大版本升级需专项处理

## 5. 实施约束

- 不修改 `proto/ant/v1/*.proto`（API 合约稳定）
- 不删除现有页面文件（重构，非删除）
- 每次改动后 `npx tsc --noEmit` 必须通过
- 共用组件放 `src/components/`，页面私有组件放 `src/pages/<name>/components/`
- TanStack Query key 约定：`[serviceName, methodName, ...params]`

## 6. 验证方式

1. `npx tsc --noEmit` 零错误
2. `npm run build` 成功（vite build 无 warning）
3. `docker compose up -d --build frontend` 容器 healthy
4. Dashboard/AccountDetail/StrategyTemplate 三页三态手动验收
5. `curl localhost:8022` 返回 200 + HTML
