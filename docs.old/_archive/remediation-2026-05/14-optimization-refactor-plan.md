# 优化和精简重构方案

## 1. 文档目标

本文档记录整改完成后的下一阶段优化和精简重构任务。

本阶段目标不是继续扩大功能范围，而是在当前代码真实状态基础上降低维护成本、减少误导性入口、控制文件膨胀风险。

## 2. 范围边界

纳入评估范围：

- `backend/`
- `frontend/src/`
- `strategy-service/app/`
- `scripts/`
- `proto/`
- `.windsurf/`

排除范围：

- `Emulator/`：外部文件，不作为本项目源码整改对象。
- `mtproto/`：外部上游协议契约，不拆分、不改写。
- `backend/mt4` 与 `backend/mt5` 中由外部 MT 协议生成的超限文件：按外部生成物基线隔离管理。

## 3. 当前结论

当前整改主线已经收敛：

- 浏览器到后端业务通信已回归 ConnectRPC。
- 项目自有 proto 与 ConnectRPC 生成物已低于 800 行。
- 前端首批业务计算已下沉到后端或策略服务。
- 后端依赖组装、生命周期、Connect 注册和健康检查已模块化。
- 手动交易、自动交易、策略调度已接入统一 RiskEngine。

因此，后续优化以小批次精简为主，不做一次性大重构。

交付后第一批低风险精简重构以 `15-refactor-batch1-implementation-guide.md` 为实施依据。任何 Batch 1 代码改动必须先满足该文档的范围、顺序和验收要求。

## 4. 交付前必须项

### 4.1 工作区纳入检查

交付前必须确认所有新增、修改、删除文件均符合预期：

- 新拆分 proto 必须纳入版本。
- 新生成 Go/TS 代码必须纳入版本。
- 删除的旧生成文件、旧 REST handler、旧文档必须与替代文档一致。
- 私钥、缓存、测试产物不得纳入版本。

特别禁止提交：

- `id_ed25519`
- `__pycache__/`
- `.pytest_cache/`
- `node_modules/`
- Playwright 报告和临时 trace

### 4.2 前端人工验收

自动化检查通过后，仍需用户完成前端人工验收：

- 登录、账户、交易、策略、AI、回测、管理后台主流程可打开。
- 浏览器控制台无关键错误。
- ConnectRPC 迁移后的功能行为与原业务语义一致。
- 容器健康检查通过。

## 5. 第一批优化任务

### 5.1 精简 `AdminService`

状态：已完成。

现状：

- `backend/internal/service/admin_service.go` 接近 800 行。
- 仍存在未实现的告警与缓存操作入口。

要求：

- 将告警、缓存、指标聚合拆到独立文件。
- 未实现能力不得返回假成功。
- 无真实调用价值的入口应从 API facade 中移除或标为不可用。

验收：

- 单文件低于 800 行并留出增长空间。
- 空实现清零。
- 管理后台相关构建和测试通过。

### 5.2 拆分 `strategy-service/app/main.py`

状态：已完成。

现状：

- `main.py` 同时承担 app 初始化、schema、路由、回测适配、memory 接口和 objective score 计算。
- 文件接近 800 行。

目标结构：

```text
strategy-service/app/main.py
strategy-service/app/schemas.py
strategy-service/app/routes/strategy.py
strategy-service/app/routes/backtest.py
strategy-service/app/routes/memory.py
strategy-service/app/routes/objective_score.py
strategy-service/app/services/objective_score.py
```

要求：

- `main.py` 只负责创建 FastAPI app、注册 middleware、注册 router。
- 路由只做请求响应适配。
- 指标计算放入 service 纯函数并补测试。

### 5.3 消除静默吞错

状态：已完成首批治理。

重点文件：

- `strategy-service/app/main.py`
- `strategy-service/app/memory.py`

要求：

- 非关键路径可以不阻断主流程，但必须记录日志。
- 不允许新增 `except Exception: pass`。
- 内部错误响应应可定位模块和操作。

## 6. 第二批优化任务

### 6.1 前端大页面继续拆分

状态：已继续拆分中等复杂度组件。

优先候选：

- `frontend/src/pages/ai/debate/DebatePageV2.tsx`
- `frontend/src/pages/strategy/StrategyTemplatePage.tsx`
- `frontend/src/pages/accounts/components/MonthlyAnalysisCard.tsx`
- `frontend/src/pages/analytics/Summary.tsx`

原则：

- 页面文件只保留容器逻辑。
- 表格、弹窗、抽屉、状态流订阅、表单校验拆成组件或 hook。
- 不把业务计算重新放回前端。

### 6.2 清理前端 Connect client 兼容层

状态：已完成 facade 收敛。

现状：

- `frontend/src/client/connect.ts` 仍保留 `createXService()` 兼容旧调用。

要求：

- 新代码直接使用 `xxxClient`。
- 旧调用分批迁移。
- 移除兼容层前必须确认无引用。

### 6.3 明确策略服务外部数据占位接口

状态：已确认保留并作为财经数据接入预留接口。

现状：

- `strategy-service` 中 macro/news 接口当前返回空结果。

要求：

- 若保留，响应必须表达 `unavailable` 或 `disabled` 状态。
- 若无调用，删除接口或只保留内部规划说明。
- 禁止把占位接口描述为已完成外部数据能力。

## 7. 文档治理任务

文档是项目实施依据，只保留与当前代码和逻辑相符的文档：

- 当前权威入口保留在 `docs/remediation-2026-05/`。
- API 规则保留在 `docs/api/`。
- 领域说明保留在 `docs/domains/`。
- 运维说明保留在 `docs/ops/`。
- 已过时、与当前实现不符、已有替代的新旧重复文档应删除。

删除文档前必须满足：

- 当前能力已有替代说明。
- 不再作为实施依据。
- 不包含唯一的部署命令、协议规则或业务边界。
- 删除后不会让新成员误读历史计划为当前状态。

## 8. 每批验收要求

每批代码优化后必须完成：

- 格式化。
- 静态检查。
- 单元测试。
- 前端构建。
- 行数检查。
- `git diff --check`。
- 必要时重建并重启容器。

Markdown-only 修改至少完成：

- `git diff --check`。
- 文档链接核对。
- 当前文档索引更新。
