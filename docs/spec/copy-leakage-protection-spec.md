# 策略跟单外泄防护 spec（账号绑定 + 跟单检测）

> **功能块**：strategy-marketplace + mt-gateway + risk-gate
> **来源**：`docs/roadmaps/market-strategy-review.md` §10（MT 跟单外泄死结）
> **决策**：用户 2026-08-08 拍板——① 接受"放弃完美防复制"（无法改变）；② 采纳战术层，且**跟单异常检测=技术缺失，必须补**。
> **角色**：审计方（Claude Code）出 spec；施工方实现+回填，**不自行宣告 ✅，等审计方实测**。

---

## 1. 背景

MT 跟单外泄：策略在用户 MT 账户跑 → 交易可被别处开终端**跟单复制** → 策略变相外泄。"代码不出平台"防的是源码，**防不住行为（交易流）**。

战略定调（§10）：放弃"完美防复制"，承认 determined actor 能跟单；**作者真正的保护是平台绑定资产**（验证战绩 + AI 迭代 + 订阅受众），跟单偷不走这些。本 spec 落地**战术层**——把外泄成本抬到"不值得"，不追求"不可能"。

## 2. 设计决策（施工方遵循；如与代码现实冲突，标 ⚠️ 并回填真实根因）

1. **账号绑定 = 按档位限账号数，非严格 1:1。** 暂定默认：**免费层=1，Pro=5**（Enterprise 待定）。**档位账号上限必须是管理端可改的配置项**（system_config + admin 面板控制，**非硬编码**——这才是正确设计，与现有 fee tier / min_deposit_amount 等配置一致）。与"多账户执行"卖点兼容（买一份跑自己的 N 个账户是合法）。订阅绑定到指定 account login(s)，超额/未绑定 → 拒绝执行。
2. **检测 = 监控 + 告警/标记，非硬阻断。** 误报会伤合法用户（如用户正当跟自己的信号）。检测抬摩擦 + admin 复查严重案例；不自动断（避免误杀）。与"抬成本不追求不可能"一致。
3. **检测范围 = 仅 bound 账号的"供给侧"指标。** **Non-goal：检测外部被动跟单者**（fundamentally unobservable——我们只看自己的账户，看不到跟单者的账户）。这是诚实边界，写在脸上，别过承诺。

## 3. 技术现状（审计方已查实，2026-08-08）

- **账号绑定**：`marketplace/live_performance.go:42 LinkLiveAccount(strategy→account)` 是**业绩追踪的松绑定**，**不是**执行的强绑定。**档位账号上限是否已强制 = 待核（Phase 1 task 1.1 先做）。**
- **MT 可观测性**：mt5 adapter 里全是 market session（交易时段），**无"账号是否开了信号广播/多终端"查询**。mt5 proto 有 `rpc Account` / `rpc AccountSummary` + order/deal 的 **`PlacedType_Signal = 10`**（该笔交易由跟单信号触发）。
- **跟单检测代码：零**（copytrade 早删，检测是全新绿地）。

## 4. Phase 1（P0，具体可落地）— 账号绑定强制

**目标**：限制每个订阅最多绑定 N 个账号（free=1），超额/未绑定账号**不执行**策略。这是最强、最简单的降 blast-radius 杠杆。

| 任务 | 内容 | 锚点（施工方确认） |
|---|---|---|
| 1.1 核验现状 | 查 purchase/subscribe → schedule 创建链，确认是否已强制"free 层 1 账号"。跑 `bash scripts/cap.sh subscribe` / `cap.sh schedule`。 | `connect/strategy/*`（StartStrategy/resolveModeAndAccount）+ `service/subscription*` |
| 1.2 绑定模型 + 档位配置 | subscription ↔ bound account logins。migration：新增 `subscription_bound_accounts` 或 `strategy_schedules` 加约束。**档位账号上限入 system_config（`subscription.account_limit.free=1` / `.pro=5` / `.enterprise=N`），admin 面板可改**（复用现有 system_config + admin billing/config UI 模式，非硬编码）。 | `backend/migrations/`（下一个未用号）+ `adminRepo.GetConfig/SetConfigValue` |
| 1.3 执行闸强制 | 在 **Gate**（risk-gate 单一 chokepoint，D6-A）或 schedule 创建处校验"该订阅绑定账号数 < tier_limit（从 system_config 读）且当前 account 已绑定"。超额 → 拒绝（错误："超出档位账号上限，升级 Pro"）。 | `risk/gate.go` Evaluate 链 + `strategy_schedules` 创建处 |
| 1.4 前端 | ① 购买/启动时显示档位账号额度 + 绑定管理 UI；② **admin 面板：档位账号上限可配置**（admin billing/config 页加控件）。 | `frontend/src/pages/marketplace/` + admin 页 |

**REUSE**：`Gate.Evaluate`（单一 chokepoint）、`LinkLiveAccount`（绑定基础）、subscription tier 配置（plans）。

**对抗证明**：free 用户绑第 2 个账号 → 必须被拒；移除 1.3 校验 → 测试必红。

## 5. Phase 2（P1，探索性）— 跟单外泄检测

> **❌ DESCOPED（2026-08-08 第一性核验后取消，不施工）。**
> 用户质疑"MetaTrader 官方有无跟单检测功能"——核验后确认**官方没有，本节检测是空谈**：
> - mt5/mt4 proto 唯一 signal 字段 `PlacedType_Signal` = 账户**订阅方**在接收跟单，非"被跟单的提供方"；**无字段查提供方/订阅者**（`Subscribe*` RPC 全是行情订阅）。
> - MetaQuotes MQL5 Signals **不暴露公开 API 检测/列订阅者**（服务端管理）。
> - 真实外泄（外部第三方读账户 A 镜像到 B）**我们只连 A 看不到 B，MetaQuotes 也看不到** → 无人能检测。
> - `PlacedType_Signal` 检测错对象（订阅方非提供方），用反误报。
>
> **真实保护 = LEAKAGE-1 账号绑定（已 ✅，抬成本）+ 信任护城河（跟单偷不走验证战绩/AI/受众，§6.7/§10）。** 以下 Phase 2 任务**保留作历史记录，不执行**。

**目标**：监控 bound 账号的供给侧外泄指标，告警/标记可疑。**需 mtapi runtime 探查（proto 不够）。**

| 任务 | 内容 | 锚点 |
|---|---|---|
| 2.1 mtapi 可观测性探查（施工方先做） | runtime 调 `Account`/`AccountSummary`，确认是否暴露"账号是否为 signal provider"。**若不暴露 → 降级为交易模式 + 连接模式检测（2.2/2.4）。** | `mdgateway/adapter/mt5/account.go` + `reference/grpc/mt5.proto` |
| 2.2 PlacedType_Signal 监控 | bound 账户的 order/deal 流出现 `PlacedType=Signal` → 账号在**接收**跟单（异常：本该只跑我们策略）→ 标记。 | order update 回调 `pipeline_callbacks.go` + `mt5/orders.go` PlacedType |
| 2.3 signal-provider 状态检测（若 2.1 暴露） | bound 账号开了信号广播 → 强外泄指标 → admin 告警 + 通知作者。 | account 轮询/事件 |
| 2.4 多会话/多终端异常 | 同一 bound 账号多个并发 mtapi 连接/终端 → 外泄使能 → 标记。 | `mdgateway` 连接管理 |
| 2.5 告警通道 | 复用 `deploy/prometheus/alerts.yml` + 通知系统。**warn-not-block。** | alerts.yml + 通知 SSE |

**REUSE**：mt5 `Account`/`AccountSummary` RPC、order history（`PlacedType`）、`deploy/prometheus/alerts.yml`、通知 SSE 管道、pgListen pattern。

**对抗证明**：模拟 bound 账号出现 `PlacedType_Signal` 交易 → 检测必触发告警；移除监控 → 不触发 → 红。

## 6. Non-goals（明确不做，防过承诺）

- 检测**外部被动跟单者**（不可观测，我们看不到跟单方账户）。
- **硬阻断**合法多账户使用（误伤风险）。
- 100% 防复制（已战略放弃，见 §10）。

## 7. 验收（审计方实测）

- **Phase 1**：free 用户超额绑账号被拒 + 对抗证明成立 + `go build`/`go test`/`check-file-lines` 绿。
- **Phase 2**：`PlacedType_Signal` / 多会话 检测触发告警 + 对抗证明成立 + 2.1 runtime 探查结论回填（mtapi 是否暴露 signal-provider）。

## 8. 完工回填纪律（施工方，不做=任务失败）

1. `tech-debt-registry.md` 新增条目状态 🟦→✅（标日期）+ 真实根因/修复/对抗证明/测试结果。若真根因与 spec 假设不同，**如实写明**。
2. `handover-audit-plan.md` 变更日志加一行。
3. **不自行宣告完成**——等审计方核对状态 + 实测。

---

> **审计方备注**：Phase 1（账号绑定）是 P0，具体、低风险、高杠杆，建议施工方先做。Phase 2（检测）有 mtapi runtime 不确定性（2.1），需施工方先探查再定检测信号集——若 mtapi 不暴露 signal-provider，降级为 2.2+2.4 仍有效（PlacedType_Signal + 多会话）。两 phase 可独立 PR。
