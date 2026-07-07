# Agent / Agent-Engine 模块审计报告

> 审计日期: 2026-07-03
> 范围: `internal/agent/`, `internal/connect/ai/`, admin agent handlers, repository, 前端

---

## §1 模块边界

### 后端

| 目录 | 职责 |
|------|------|
| `internal/agent/` (16源+5测试) | 核心引擎: Gateway, Bridge, Generator, Memory, Hooks, Settings, Permissions, Profiler, Interpreter, Cache |
| `internal/connect/ai/` (18源+3测试) | ConnectRPC handler: AgentLoop, ToolRegistry, Chat, CodeAssist, GateEval, StrategyPlan |
| `internal/connect/admin/` (2文件) | Admin: Settings管理, Hooks管理 |
| `internal/repository/` (2文件) | AgentToken/AuditRepo, AIAgentDefinitionRepo |

### 前端

| 文件 | 职责 |
|------|------|
| `client/agentGateway.ts` | SubmitStrategy RPC |
| `client/agentGen.ts` | GenerateStrategy SSE流 |
| `client/agentMemory.ts` | Memory CRUD RPC |
| `client/adminAgentSettings.ts` | Admin Settings RPC |
| `components/strategy/AgentGenChat.tsx` | Agent生成聊天(SSE) |
| `pages/strategy/components/workspace/MemoryContent.tsx` | Memory面板 |

**边界评估**: 清晰。建议将 `ToolRegistry`/`AgentLoop` 从 `connect/ai/` 迁移到 `internal/agent/`（属业务逻辑非协议层）。

---

## §2 合规扫描

| 规则 | 状态 | 违规数 |
|------|------|--------|
| REST端点 | ✅ | 0 |
| WebSocket | ✅ | 0 |
| JSON序列化 | ⚠️ | 6处 |
| float64精度 | ⚠️ | 11处(🔴2, 🟡9) |
| 轮询 | ✅ | 0 |
| 文件行数 | ✅ | 0 (最大317行) |
| 前端合规 | ✅ | 无polling/JSON/WS/fetch |

---

## §3 违规详情

### 3.1 🔴 JSON违规

| # | 文件:行 | 说明 |
|---|---------|------|
| J1 | `connect/ai/agent_loop.go:162,199,276` | LLM tool call args `json.Marshal/Unmarshal` (OpenAI协议要求) |
| J2 | `connect/ai/code_assist_handler.go:115,152,267` | `parametersJson`冗余——proto已有`Parameters`字段 |
| J3 | `connect/ai/code_assist_handler.go:197` | LLM翻译结果`json.Unmarshal` |
| J4 | `connect/ai/gate_eval_handler.go:53,66` | 通知payload `json.Marshal` |
| J5 | `connect/ai/ai_handler.go:59` | JWT claims解析(可豁免) |
| J6 | `agent/hooks.go:207` | Hook context手工拼JSON字符串 |

### 3.2 🔴 float64精度违规

| # | 文件:行 | 违规 | 严重度 |
|---|---------|------|--------|
| F6 | `agent/backtest_helpers.go:42` | `decimal.NewFromFloat(0.00001)` SwapRate | 🔴 |
| F7 | `agent/backtest_helpers.go:74` | `decimal.NewFromFloat(r.Metrics.TotalReturn)` TotalPnl | 🔴 |
| F1-F4 | `connect/ai/tool_registry.go:134-239` | barSummary/EMA/波动率全float64 | 🟡 LLM分析用 |
| F5 | `agent/settings.go:17` | `MaxCostCeilingUSD float64` | 🟡 非价格 |
| F8-F11 | 其余 | 日志/LLM解析/gate指标 | 🟢 |

### 3.3 🔴 死代码

| # | 文件 | 说明 |
|---|------|------|
| D1 | `repository/agent_repository.go` 全文202行 | `NewAgentRepository`无调用者，agent_tokens/audit_logs完全未接入 |
| D2 | `agent/hooks.go:24-25` | `HookPreLiveDeploy`/`HookPostStrategyGen`定义后从未Fire |
| D3 | `code_assist_handler.go` | `parametersJson`与proto`Parameters`冗余 |

### 3.4 🟡 BUG

| # | 文件:行 | 问题 | 修复 |
|---|---------|------|------|
| B1 | `gateway_memory_handlers.go:67` | `StoreExperience`传`nil` indicators + `""` conditionStructure | proto补字段 |
| B2 | `hooks.go:190` | webhook响应用`strings.Contains`手工匹配，`"allowed":false`会误匹配 | 改精确解析 |
| B3 | `hooks.go:187` | webhook响应只读1024字节会截断 | 用`io.ReadAll` |
| B4 | `permissions.go`全文 | `PermissionEngine`仅在`GetCapabilities` RPC使用，`SubmitStrategy`/`GenerateStrategy`不检查权限 | 入口加`Can()` |
| B6 | `agent_loop.go:147-151` | memory tool的`key`/`value`映射到`ToolInput.Symbol`/`Timeframe`，hack字段复用 | 加`Key`/`Value`字段 |
| B7 | `tool_registry_memory.go:95` | `list_strategies`用`json_agg`返回JSON字符串给LLM | 改分别查询 |

---

## §4 管线审计

### SubmitStrategy管线
```
Client→RPC→HookPreSubmit→Compile(VM)→FetchBars→Backtest→HookPostBacktest→Profiler(LLM)→Interpreter(LLM)→Bridge(if coverage<1.0)→Response
```
✅ 完整。问题: 无权限检查(B4)，HookPostStrategyGen未触发(D2)。

### GenerateStrategy管线 (SSE)
```
Client→RPC Stream→Generator→AgentLoop(LLM+Tools,max10rounds)→ExtractCode→Done
```
✅ 完整。问题: `history=nil`(待proto加session_id)，HookPostStrategyGen未触发。

### Memory管线
```
GatewayServer→MemoryStore(PG): Load/Search/Store/Save/List/Delete
```
✅ 三层记忆完整。问题: StoreExperience丢字段(B1)。

### Settings管线
```
SettingsStore(PG): Default→User→Managed分层→FailClosed→PermissionEngine
```
✅ 架构完善。问题: 权限未接入操作(B4)。

---

## §5 前端审计

✅ 无冗余Zustand store，无polling，无JSON序列化，无WS/fetch。使用组件级`useState`。无死代码。

---

## §6 修复任务书

| 优先级 | 任务 | 文件 | 操作 | 工作量 |
|--------|------|------|------|--------|
| P0 | SwapRate float64 | `backtest_helpers.go:42` | `NewFromFloat`→`NewFromString` | 1行 |
| P0 | 删除AgentRepository | `repository/agent_repository.go` | 删除整个文件 | 删文件 |
| P1 | 权限检查接入 | `agent/gateway.go` | SubmitStrategy/GenerateStrategy入口加`Can()` | 小 |
| P1 | StoreExperience补字段 | proto+`gateway_memory_handlers.go:67` | 添加indicators/condition_structure | 中 |
| P1 | 删parametersJson | `code_assist_handler.go`+`codeAssist.ts`+proto | 删JSON字段，统一用proto Parameters | 中 |
| P2 | HookPostStrategyGen | `generator_agent.go` | 完成后Fire | 小 |
| P2 | webhook buffer | `hooks.go:187` | `io.ReadAll` | 小 |
| P2 | ToolInput hack | `tool_registry.go`+`agent_loop.go` | 加Key/Value字段 | 中 |
| P3 | 通知payload proto化 | `gate_eval_handler.go` | 需扩展Sender | 中 |
| P3 | HookPreLiveDeploy | `hooks.go:24` | 保留+标注Phase3 TODO | 评估 |

每个任务需标注: `REUSE: <symbol>` 或 `NEW: 无现成能力(已搜:<关键词>)`

---

## §7 总结

**核心问题**:
1. `backtest_helpers.go` 两处`decimal.NewFromFloat`是真正精度bug
2. `agent_repository.go` 202行完全死代码
3. 权限系统装饰性——存在但未接入
4. Hook常量定义未全部触发
5. JSON违规主要在LLM协议交互(可考虑豁免)和冗余字段(应修)
6. `ToolInput`字段复用hack影响可维护性

**前端**: 完全合规，无需修改。
