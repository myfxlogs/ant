# 施工提示词：QS-1.7-INV open outcomeUnknown 恢复可行性调研（只查不改）

> **[角色:施工]** — 你是本任务的施工方 agent（见 `.devin/rules/dual-terminal-roles.md`）。
> 本任务是**只读调研**：逐跳追踪代码链、收集证据、回答既定问题；**不改任何生产代码**。
> 严格按 S1–S3 执行，不做决策；超出提示词范围 = 违规，停下转 `[转交决策]`。
> 完成后先过自审（D-012）再交回；自报末行带 `[施工完成:QS-1.7-INV] @<commit-hash>`（D-014）。
> 勿部署、勿 push、禁 `--no-verify`；只显式 add 本任务文件（findings 文档）；commit 用 `ANT_ROLE=builder git commit` 前缀（D-015）；**不更新任何交接层文件**。
>
> **最终决策：Devin CLI（[角色:决策终] 激活）**

## 立项背景（触发 + 证据链）

- registry `QS-1.7-INV`；spec `docs/spec/vm-pipeline-quality-stability-improvement-plan.md` v2 §3.5；决策 D-009（模糊 magic+symbol+side+时间窗匹配**永久否决**；open 不 recovery 是 ④-② 刻意决策，`mutation_coordinator.go:260-266`）。
- 问题：open 动作 outcomeUnknown → 永久锁仓。唯一被批准的可能路径 = **ClientID 精确单匹配**：若 `strategyOrderClientID` 能随订单下发并被 broker 回显，则恢复时可按 ClientID 唯一匹配 open order 并释放锁仓。
- **决策方预核实的链路坐标**（施工时逐跳复核）：
  a. 起点：`live_dispatch.go:366` `ClientID: strategyOrderClientID(cfg.RunID, barOpenTime, sig.GetSignalType())`；格式 `start-<runID>-<barOpenTime>-<signalType>`（`live_helpers.go:37-39`）；**`req.Comment` 未设置**。
  b. `mthub.OrderRequest` 的 `Comment` 与 `ClientID` 是**两个独立字段**（`internal/mthub/order_types.go:17`）。
  c. ClientID 已知消费方：幂等 guard（`service_orders.go:32,61-67,110-111`）+ `publishOrderCreatedEvent`（`:299`）。
  d. Comment 下发路径：`broker_registry.go:43,111` `Comment: req.Comment` → `mt5/orders.go:45` `Comment: &req.Comment` 进 `pb.OrderSendRequest`（mt4 同构）。
  e. 回显侧已存在：`mt4/order_history.go:73,152`、`mt4/order_stream.go:178`、`mt4/profit.go:314,368`、mt5 同构文件——`Comment` 从 broker 读回。
  f. mtapi `pb.OrderSendRequest` 无专用 ClientID 字段，只有 `Comment`+`ExpertID`（`mt5/orders.go:38-46`）。

## 设计 SSOT 声明

- 设计文档：spec v2 §3.5。契约：AGENTS.md §1 fail-closed；决策 D-009/D-012/D-014/D-015。

## 调研问题（必须逐条给证据回答）

- **Q1**：`ClientID` 是否在 `PlaceOrder` 调用链任何一跳被复制进 `req.Comment` 或等价字段下发 broker？逐跳列证据：`live_dispatch.submitOrder` → `MtHubService.PlaceOrder` → executor/adapter（`broker_registry.go`）→ `oms.OrderRequest` → `mt4/mt5 Gateway.PlaceOrder` → `pb.OrderSendRequest`。若全程无复制 → Comment 空下发，ClientID 不可回显，Q2/Q3 可简答。
- **Q2**：若有 Comment 下发（或假设 ClientID 复制进 Comment）——MT4/MT5 broker 端 comment 长度限制与改写行为：代码中是否有截断/转义处理？`start-<uuid36>-<ts>-<signalType>`（~55+ 字符）是否会超 MT4/MT5 comment 限制被截断？proto/gateway 层有无注释说明？
- **Q3**：`OpenedOrders`/order stream 回读的 `Comment` 是否保证原样回显（broker 附加后缀如 `[sl]`/`[tp]`、改写、清空的已知行为在代码注释/proto/文档中是否有记录）？
- **Q4**：除 Comment 外，链路是否有其他 broker 可见字段可携带恢复标识（`ExpertID` 已占用于 magic；mtapi proto 是否有 `client_id`/`external_id`/`position_id` 类字段——查 `pb.OrderSendRequest`/`OpenedOrdersRequest` 响应 proto 定义）。

## 边界 / 不做

- **不改任何生产代码**（含测试）；不实现 recovery；不做模糊匹配方案设计（D-009 已否决）。
- 不外发网络请求、不连真实 broker；只读仓库代码 + proto 定义 + 仓库内文档。
- 不更新交接层文件；findings 文档是唯一交付文件。

## 施工指令

### S1 — 链路逐跳追踪（只读）

- **目标**：按 Q1 的链路逐跳读代码，确认 `ClientID` 是否到达 broker 下发的 `Comment`（或其他字段）。
- **落点**：列出每一跳的 file:line + 关键代码片段证据；标注每跳 ClientID/Comment 的去向。
- **验证**：findings 文档中每跳都有 file:line 引用。

### S2 — Q2/Q3/Q4 证据收集（只读）

- **目标**：comment 长度/改写约束、回显保真度、替代字段。
- **落点**：查 `pb` proto 定义（`api/proto` 或 vendored mtapi pb）中 `OrderSendRequest`/`OpenedOrdersRequest`/`OrderRecord` 字段；查 adapter 层对 Comment 的处理（截断/编码/UTF8 sanitize，如 `trade_event_store.go:109` `sanitizeUTF8` 模式）；查仓库内 MT4/MT5 行为注释。
- **验证**：每个结论附 file:line 或 proto 字段名证据；无证据的判断标"未知"。

### S3 — findings 文档落盘

- **坐标**：新建 `docs/audits/qs-1.7-inv-findings.md`。
- **落点**：Q1–Q4 逐条结论 + 证据链（file:line）+ 末尾给施工方观察到的可行性判断（"ClientID 可/不可经 Comment 回显"及理由），**但 go/no-go 决策留给决策方**。
- **验证**：文档自洽，每条结论可溯源。

## 对抗证明

- 只读调研无代码改动，mutation 不适用。验收以"每条结论附 file:line 证据、缺失证据标未知"为准。

## 验收标准

- [ ] findings 文档存在且 Q1–Q4 全部作答
- [ ] Q1 链路每一跳有 file:line 证据
- [ ] "未知"项显式标注，无凭空结论
- [ ] diff 仅含 `docs/audits/qs-1.7-inv-findings.md`，零代码改动、零交接层文件
- [ ] `git diff --check` 干净

## 施工完成自审（强制，D-012）

- [ ] 每个结论复核证据引用真实存在
- [ ] 无自审记录 = 复审直接退回

## 交付格式

自报：Q1–Q4 结论摘要 + findings 文档路径 + 施工完成自审记录 + 遗留疑问。
**自报最后一行固定为** `[施工完成:QS-1.7-INV] @<commit-hash>`。
**停手等 Devin CLI 复审；决策方据此裁定 QS-1.7 立项与否。**

[施工完成:QS-1.7-INV] @<本任务最终commit>
