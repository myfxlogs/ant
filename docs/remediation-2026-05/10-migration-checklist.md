# AntTrader 整改执行清单

## 1. 使用方式

本文档用于跟踪整改执行进度。

状态约定：

- `[ ]` 未开始。
- `[~]` 进行中。
- `[x]` 已完成。
- `[!]` 阻塞或需要决策。

## 2. 文档体系

- [x] 创建 `docs/remediation-2026-05/README.md`。
- [x] 创建工程治理文档。
- [x] 创建整体整改规划。
- [x] 创建 ConnectRPC 回归专项方案。
- [x] 创建旧文档清理方案。
- [x] 创建目标架构规划。
- [x] 创建系统级优化方案。
- [x] 创建 REST / fetch 调用清点基线。
- [x] 创建 `docs/api/connectrpc-style-guide.md`。
- [x] 创建 `docs/api/proto-splitting-guide.md`。
- [x] 创建 `docs/domains/trading-core.md`。
- [x] 创建 `docs/domains/strategy-lifecycle.md`。
- [x] 创建 `docs/domains/risk-engine.md`。
- [x] 创建 `docs/ops/container-restart-checklist.md`。

## 3. 工程规则

- [x] 更新 `.windsurf/rules/rules.md`，明确 800 行硬约束。
- [x] 更新 `.windsurf/rules/rules.md`，明确前端只负责展示和渲染。
- [x] 更新 `.windsurf/rules/rules.md`，明确业务通信统一 ConnectRPC。
- [x] 增加行数检查脚本。
- [x] 将行数检查加入 `make verify`。
- [x] 将 proto 生成代码行数检查加入 CI。

## 4. REST 业务接口迁移

### 4.1 后端 REST 注册清点

- [x] 清点 `server.go` 中所有 `mux.HandleFunc('/api/...')`。
- [x] 为每个 REST handler 标注：业务接口 / 健康接口 / 第三方代理 / 临时兼容。
- [x] 建立 REST 到 ConnectRPC 映射表。

### 4.2 高优先级迁移

- [x] `frontend/src/client/strategyService.ts` 直连 `strategy-service /api/objective-score` → `ObjectiveScoreService.CalculateObjectiveScore`。
- [x] `/api/debate/v2/start` → `DebateV2Service.StartDebateV2`。
- [x] `/api/debate/v2/chat` → `DebateV2Service.ChatDebateV2`。
- [x] `/api/debate/v2/advance` → `DebateV2Service.AdvanceDebateV2`。
- [x] `/api/debate/v2/back` → `DebateV2Service.BackDebateV2`。
- [x] `/api/debate/v2/params` → `DebateV2Service.SetDebateV2Params`。
- [x] `/api/debate/v2/code/reject` → `DebateV2Service.RejectDebateV2Code`。
- [x] `/api/debate/v2/sessions` → `DebateV2Service.ListDebateV2Sessions` / `GetDebateV2Session` / `DeleteDebateV2Session`。
- [x] `/api/ai/code/revise` → `CodeAssistService.ReviseCode`。
- [x] `/api/ai/code/explain` → `CodeAssistService.ExplainCode`。
- [x] `/api/ai/primary` → `AIPrimaryService` ConnectRPC。
- [x] `/api/strategy/validate-extended` → `CodeAssistService.ValidateStrategyExtended`。
- [x] `/api/backtest-runs/*` → `BacktestTradesService.ListBacktestRunTrades`。

### 4.3 中优先级迁移

- [x] `/api/strategy/indicator-catalog` → `IndicatorCatalogService.GetIndicatorCatalog`。
- [x] `/api/economic-calendar` → `EconomicDataService.ListEconomicCalendarEvents`。
- [x] `/api/economic-indicators` → `EconomicDataService.ListEconomicIndicators`。

### 4.4 删除 REST

- [x] 前端调用替换完成。
- [x] ConnectRPC 对应 RPC 验证完成。
- [x] 删除 REST handler 注册。
- [x] 删除 REST handler 实现。
- [x] 删除前端 REST client 封装。
- [x] 更新 Nginx 规则，只保留必要代理路径。

## 5. 前端计算下沉

- [x] 搜索前端金额计算。
- [x] 搜索前端盈亏计算。
- [x] 搜索前端回撤计算。
- [x] 搜索前端风险等级判断。
- [x] 搜索前端策略状态推导。
- [x] 搜索前端回测指标二次计算。
- [x] 为首批风险/回测成交统计确定后端 owner service：`StrategyService.RunBacktest` / `BacktestTradesService.ListBacktestRunTrades`。
- [x] 在 proto response 中补齐首批展示字段：`BacktestTradeSummary`。
- [x] 替换首批前端风险评分/回测成交统计为后端结果渲染。
- [x] 替换首批回测运行终态/成功态判断为后端 `BacktestRun.is_terminal` / `is_succeeded`。

## 6. proto 与生成代码拆分

- [x] 统计 `proto/*.proto` 行数。
- [x] 统计 `backend/gen/proto/**` 行数。
- [x] 统计 `frontend/src/gen/**` 行数。
- [x] 拆分超 800 行 proto（项目自有 proto 已完成；MT4/MT5 外部上游协议按治理例外隔离）。
- [x] 确保生成代码也全部低于 800 行（项目自有 ConnectRPC 生成物已完成；MT4/MT5 外部上游生成物按治理例外隔离）。
- [x] 将 AI proto 拆成 AI chat / AI debate / System AI 等领域。
- [x] 将 strategy proto 拆成 template / schedule / backtest 等领域。
- [x] 将 `backtest_run.proto` 拆成实体、启动、查询、控制消息，`backtest_run.pb.go` 降至 800 行内。

## 7. 后端模块化

- [x] 拆分 `server.New()` 依赖组装。
- [x] 建立 `internal/app/container.go`。
- [x] 建立 `internal/app/lifecycle.go`。
- [x] 建立 `internal/transport/connect/register.go`。
- [x] 建立 `internal/transport/rest/health.go`。
- [x] 将业务 REST 从 transport/rest 删除。
- [x] 将 worker 启动逻辑从 API server 启动逻辑中分离。

## 8. 交易核心重构

- [x] 定义统一 `RiskDecision`。
- [x] 手动交易接入统一 RiskEngine。
- [x] 自动交易接入统一 RiskEngine。
- [x] 策略调度接入统一 RiskEngine。
- [x] 强化 `AutoTradingService.CheckRiskLimits()`。
- [x] 统一执行日志、订单日志、审计日志关系。
- [x] 明确 MT4/MT5 adapter 边界。

## 9. 策略生命周期重构

- [x] 定义策略模板状态机。
- [x] 定义调度配置态与运行态拆分方案。
- [x] AI 生成策略接入校验、回测、发布流程。
- [x] Online 启动前强制检查模板状态。
- [x] 调度运行日志结构化。

## 10. 旧文档处理

- [x] 给旧文档增加迁移状态索引。
- [x] 删除已被 `docs/api/connectrpc-style-guide.md` 替代的 `API_STYLE.md`。
- [x] 删除已被 `docs/domains/backtest-system.md` 替代的 `backtest_engine.md`。
- [x] 迁移运维部署手册。
- [x] 删除 2026-04 历史评估文档。
- [x] 删除整改前全景说明快照。
- [x] 删除确认无引用、无价值、已有替代的旧文档。

## 11. 每轮代码整改验收

每轮涉及代码修改必须完成：

- [x] 行数检查。
- [x] 格式化。
- [x] 静态检查。
- [x] 编译。
- [x] 单元测试。
- [x] 前端构建。
- [x] proto 重新生成。
- [x] 容器重建。
- [x] 容器重启。
- [ ] 前端人工验收（由用户评估）。

Markdown-only 修改至少完成：

- [x] `git diff --check`。
- [x] 文档链接与目录核对。
