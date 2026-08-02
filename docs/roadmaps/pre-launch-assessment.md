# 上线前架构评估 — 2026-08-02

> 基于全项目审计（6 条管线 + 11 个功能块 + 9 维审计）的输出，
> 对照 `business-direction.md` 的商业目标，评估当前状态是否达标、
> 哪些缺口必须在上线前堵住、哪些商业点值得深挖。

---

## 总体结论

**项目骨架是对的，可以上线——但有三个缺口必须在上线前堵住。**

策略市场的定位清晰，三个飞轮逻辑自洽，技术选型正确（ConnectRPC+SSE、decimal.Decimal、Bytecode VM）。
全项目审计在核心安全/合规/正确性维度上无 CRITICAL 遗留缺陷。
剩余工作不是"修完所有 backlog"，是**把三个 launch-blocking 缺口堵了，够好就上**。

---

## 逐块评估：是否最优解、是否符合第一性

| 块 | 评级 | 判断 | 依据 |
|----|------|------|------|
| mt-gateway | ✅ 最优 | MT4/MT5 适配器分离，通过 mtapi.io 连接真实 broker。唯一可行方案 | 审计：零共享代码，ADR-0003 核心约束遵守 |
| strategy-runtime | ⚠️ 有缺口 | Bytecode VM 正确。但 GoExecutor（`go run`）仍在运行——用户代码写到磁盘编译执行，违反"代码不出平台" | 审计 A-008：双引擎并存 |
| mql-compiler | ⚠️ 有风险 | IR 是 MQL 语义的 1:1 映射。business-direction 要求 Year 1 内明确 IR 语言中性定位 | business-direction.md §技术前提 |
| **agent-engine** | 🔴 不够 | 供给侧飞轮的发动机。系统提示词刚重写但未验证效果。Agent 生成策略的质量直接决定开发者留存 | 审计 P4: text tool call parser bug 已修复，prompt quality 未知 |
| backtest-engine | ✅ 基本 OK | SimBroker 撮合正确，Decimal 全链路。metrics 精度已修复 | 审计 P5: 仅 metrics.go float64 往返问题（已修复） |
| risk-gate | ⚠️ 有冗余 | gate.Evaluate + risksvc.PreCheck 每单各跑一次，违反 D6-A"单一 chokepoint" | 审计 A-009：双风控引擎 |
| account-mgmt | ✅ 最优 | 架构最优，代码干净，零已知 BUG | 审计结论 + memory |
| market-data | ✅ 基本 OK | 6 阶段管线正确。Volume 精度/缓存雪崩已修复 | 审计 P1: 仅 bar 精度问题（已修复） |
| frontend | ✅ 可用 | 功能完整，有 dead code 但不影响业务 | 审计：knip 存量，eslint 已追踪 |
| api-gateway | ✅ 最优 | ConnectRPC+SSE，49/49 handler 鉴权，零 REST/WS | 审计：handler registry 100% auth |
| **strategy-marketplace** | 🔴 不够 | 收入来源。涉及钱的代码 0.1% 覆盖率——购买/分账/退款/订阅无测试守卫 | 审计 A-002/A-003: 分账 bug 已修，余量未知 |

---

## 三个上线前必须堵的缺口

### Gap 1: Agent 策略生成质量（供给侧飞轮）

**为什么是 launch-blocking**：

商业方向定义供给侧飞轮的第一步是"AI 帮开发者更快产策略"。
如果 Agent 生成的策略编译通不过、回测全是亏的、或者和用户需求完全不匹配——开发者来一次就不会再来。
这是**获客的第一步**，第一步失败 = 飞轮不转。

**当前状态**：

- 系统提示词从 3 句重写为 60 行（审计 P4 修复），但从未用真实策略需求验证过
- 文本工具调用解析器之前存在截断 bug（已修复，有 7 个单元测试）
- Chat Agent 路径的 prompt 引用 write_strategy 但工具未注册（审计 P4 发现，已修复）
- **没有端到端质量基准**：Agent 生成 100 个策略，编译通过率多少？回测夏普>0.5 的比例？不知道

**解决方案**：

1. **建立 Agent 质量基准测试套件**（上线前）
   - 准备 20 个不同难度和风格的策略需求（趋势跟踪/均值回归/突破/网格/多时间框架…）
   - 每个需求跑 3 次 Agent 生成，统计：
     - 编译通过率（目标 ≥90%）
     - 回测完成率（目标 ≥80%）
     - 回测夏普 >0 的比例（目标 ≥50%）
     - 策略与需求意图匹配度（人工评分 1-5）
   - 低于目标 → 调 prompt 和工具链 → 重跑 → 直到达标
   - 这套基准测试**保留为 CI pipeline 的一部分**，每次改 prompt 自动重跑

2. **Chat Agent 路径补齐**（上线后 Week 2-4）
   - Chat Agent（StrategizePlan/ExecutePlan）当前不注册 write_strategy——用户只能靠文本提取代码
   - 方案：在 ExecutePlan 路径中注册 write_strategy 工具，让 Agent 能编译+回测后再返回结果
   - 或者：明确 Chat Agent 的定位是"讨论"，策略生成走 Generator Agent 路径

3. **Agent 迭代闭环**（上线后 Month 1-3，商业深挖点）
   - 这是 MQL5 Market 做不到的——策略上线后 alpha 衰减，AI 自动检测并建议改进
   - 架构设计：见下方 §商业深挖点

### Gap 2: Marketplace 资金链路（信任飞轮）

**为什么是 launch-blocking**：

收入模型是"平台订阅 + 策略抽成 15-30%"。如果一笔购买的金额算错了、
分账没到账、退款没执行——损失的不是代码质量，是真金白银和信任。
信任飞轮的第一步是"买方的钱安全、卖方的钱准时到账"。

**当前状态**：

- `internal/marketplace/` 覆盖率 0.1% → 审计后提到 3.8%（仅 settlement 的集成测试）
- 分账静默失败 bug 已修复（审计 A-002，CRITICAL）
- 购买（purchase.go）、退款（refund.go）、订阅（service_subscription.go）、
  结算（settlement.go）、试用（trial.go）——**没有端到端集成测试**
- 验证方式：部署到服务器，手动操作一遍，看日志有没有报错——这在生产环境不可接受

**解决方案**：

1. **核心资金路径集成测试**（上线前，预计 2 天）
   - 购买流程：创建订单 → 扣款 → 发货（策略授权）→ 分账（provider + platform）
   - 退款流程：申请退款 → 冻结 → 到期自动退款 → 撤销策略授权
   - 订阅续费：到期自动续费 → 扣款 → 更新订阅状态
   - 试用到期：试用结束 → 自动冻结 → 通知用户
   - 每个路径**至少一个 happy-path + 一个失败场景**（如余额不足、策略已下架）
   - 复用 mthub 的 PG/Redis 集成测试模式（`idempotency_integration_test.go`）

2. **金额精度验证**（集成测试的一部分）
   - 所有涉及 decimal 的计算（分账比例、平台抽成、退款金额）断言精确到小数点后 8 位
   - 验证分账：providerAmt + platformFee = purchaseAmt（无精度损失）

3. **幂等性验证**（集成测试的一部分）
   - 同一笔购买重放 3 次：只扣一次款、只授权一次
   - 同一笔退款重放 3 次：只退一次款
   - 利用已有的 `IdemKeySettle` / `IdemKeyPurchase` 基础设施

### Gap 3: GoExecutor 移除（安全 + 核心承诺）

**为什么是 launch-blocking**：

商业方向的核心承诺是"代码不出平台"。GoExecutor 把用户代码写到临时文件、
调 `go run` 在宿主机上编译执行——这是在服务器上跑用户提交的任意 Go 代码。
既是安全风险，又是对核心承诺的违反。

**当前状态**：

- `internal/connect/strategy/go_executor.go` — 完整实现（~150 行）
- `cmd/server/handlers_strategy.go:55` — `SetGoExecutor` 依然在注入
- `strategy_execution_handler.go:31,52,188-191,268-271` — `isGoStrategy(code)` 匹配时走 GoExecutor 路径
- 审计 A-008（HIGH）：ADR-0023 要求 Go 代码生成不再用于运行时，但 GoExecutor 仍在生产路径中

**解决方案**：

1. **移除 GoExecutor 生产路径**（上线前，预计 4 小时）
   - 删除 `cmd/server/handlers_strategy.go` 中的 `SetGoExecutor` 注入
   - 在 `strategy_execution_handler.go` 中：`isGoStrategy(code)` 返回时走 Bytecode VM 路径而非 GoExecutor
   - 保留 `go_executor.go` 文件但加上 `//go:build ignore` 或移到 `tools/` 作为开发工具
   - 如果还有用户的 Go 策略需要兼容——在上线前手动转成 MQL/Bytecode

2. **验证 Bytecode VM 覆盖率**（确认替代方案可用）
   - 确认 Bytecode VM 支持当前 GoExecutor 支持的所有策略语义
   - 如果 VM 有缺失的 opcode → 补上后再切
   - 跑一轮回归：所有现有策略在 VM 上的回测结果与 GoExecutor 一致（或差异在可接受范围）

3. **Code Assist 路径检查**（确保没有其他代码执行入口）
   - `internal/connect/ai/code_assist_*.go` —— AI 代码辅助是否也有代码执行路径？
   - 确认所有策略执行统一走 `mql2go.CompileMQL` → `VMRunner` → SimBroker

---

## 商业深挖点（上线后）

### 1. AI 策略迭代闭环（MQL5 Market 做不到的护城河）

```
策略上线 → live performance 监控 → alpha 衰减检测
    → Agent 自动生成改进版本 → 回测对比
    → 如果改进版显著优于当前版 → 通知开发者审核 → 一键替换
```

**关键设计决策**：
- 衰减检测用什么指标？夏普下降 >50%？连续 N 天负收益？
- 改进生成：给 Agent 提供当前策略代码 + 近期回测/实盘数据 + 衰减分析 → 生成改进版
- 灰度替换：开发者审核后才生效，不能自动替换正在运行的策略

**实施路径**：上线后 Month 1 出设计方案，Month 2-3 实现 MVP。

### 2. 实盘战绩信任基础设施

"实盘战绩强制公开"是商业方向的核心差异——MQL5 Market 的回测可以美化。
但要真正建立信任，需要：

1. **战绩不可篡改**：live performance 数据写入后不可修改（append-only 表 + 哈希链）
2. **战绩可追溯**：每笔交易可追溯到具体的 MT broker 成交记录
3. **回测 vs 实盘对比**：展示策略的回测曲线和实盘曲线并排显示，差异一目了然
4. **Walk-forward 验证**：策略在上架前经过 walk-forward 测试——用最近 N 个月的数据做样本外验证

**当前状态**：`live_performance.go` 已实现基本的战绩记录。不可篡改性和回测对比待实施。

### 3. 多账户执行粘性

"买一份策略跑多个 MT 账户，离开平台就失去这个能力"——这是锁客逻辑。
上线后优先打磨多账户体验：

1. **多账户 P&L 对比 dashboard**：一个策略在 5 个账户上的收益对比，一目了然
2. **统一风控**：所有账户的风险敞口汇总，一键设置最大回撤
3. **一键平仓所有账户**：紧急情况下关闭所有账户的所有持仓

---

> **详细实施计划**: `docs/roadmaps/pre-launch-implementation-plan.md`（文件级、函数级的步骤拆解）

## 上线前优先级路线图

```
Week 1-2: 堵三个缺口
  Day 1-2:   GoExecutor 移除 + Bytecode VM 覆盖验证
  Day 3-4:   Marketplace 核心资金路径集成测试（购买/退款/订阅/试用 × happy path + error）
  Day 5-7:   Agent 质量基准测试套件 + prompt 调优
  Day 8-10:  真实用户验收 — 走完完整闭环（绑账户→生成策略→回测→上架→购买→分账）

Week 3: 验收 + 补漏
  - 基于验收发现修复
  - 上线 checklist 逐项确认

Week 4: 上线
  - 部署生产环境
  - 监控 + 告警就位
  - 用户反馈渠道就位
```

---

## 不做的事（上线前）

以下是有价值的改进，但**不阻塞上线**——记入 backlog，上线后按优先级排期：

- mdgateway/marketplace/agent 覆盖率提标到 rd.md 目标（已完成 P0 达标）
- deadcode 清理（588 函数）
- knip 存量清理（80 files + 96 exports）
- 双风控引擎合并
- IR 语言中性化
- ADR-0003 LOC 合规（adapter 超标 10x）
- 迁移文件 down 脚本补全（55 个缺失）
- webAuthn 提现授权恢复
- CSP 策略完善（当前 `unsafe-inline` 对 SPA 是标准实践）
