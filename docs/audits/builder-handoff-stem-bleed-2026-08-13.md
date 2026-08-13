# 施工交接：止血批次（外部边界同款缺陷）+ 补网（live 冒烟）

> **⚠️ 审计方自我审计标注（2026-08-13）**：本文件的 MDGATEWAY-1~5 / TRON-GRID-1 / TRON-SECURITY-1 / BROKER-SEARCH-1 等均为**审计 agent 报告，审计方尚未逐条独立读码复核**（Tier 0 的 EXEC-1 就是被 agent 误诊、复核才发现的）。**动工前审计方须先复核确认**，不要直接照 agent 结论改。当前优先级已降（TRON 延后、MDGATEWAY 待 launch 后），复核留到动工前。

> **⚠️ 2026-08-13 优先级重排（用户业务决策）**：主业产品（策略市场）先完成并上线是第一优先；TRON 充值/提现**延后到最后**（未收费、未上线，硬化未启用的资金通道非当务之急）。故本批次**实盘交易相关项（A3 LIVE-PRICE-4 / A4 MDGATEWAY-1 / A5 MDGATEWAY-3）= launch-critical 最先做**（实盘交易是策略市场的核心价值"实盘战绩公开"）；**A1 TRON-GRID-1 / A2 TRON-SECURITY-1 / A6 BROKER-SEARCH-1 延后**（等收费/上线后）。下面原文保留 A1/A2 在前的资金优先序，但**执行序按本注：A3→A4→A5 先，A1/A2/A6 延后**。
>
> **审计方根因定论，2026-08-13。** 触发：用户判定 LIVE-PRICE 是"低级错误叠加"、担心项目质量 → 审计方先补网摸爆炸半径（2 agent 并行只读扫 mdgateway + 其他外部边界）。**结论：LIVE-PRICE 三模式（硬编码外部数据 / 吞外部响应错误 / 流式无死检测）系统性重复，资金边界最危险。** 本文件=止血（修第一梯队）+ 补网（防再犯）。
>
> 完整发现见 `docs/audits/tech-debt-registry.md` §「止血扫描」+ changelog。规则已落 CLAUDE.md/AGENTS.md/.windsurfrules（禁硬编码外部可变数据）。

---

## Part A — 止血批次（第一梯队，按资金/可用性优先级）

> **共性**：6 项都是 LIVE-PRICE 同款三模式。每项必带对抗证明（删行必红）+ 回填 registry 状态，不自行宣告完成（审计方验收）。部署唯一方式 `docker compose build backend && docker compose up -d backend`。

### A1. TRON-GRID-1（P1，最高优先——可丢用户充值/双花）
- **位置**：`backend/internal/chain/tron_grid.go:102-109`（GetBlockEvents）/ `:182-189`（HasOutgoingTRC20Transfer）/ `:232-249`（GetLatestBlock）。
- **根因**：TronGrid 限流/内部错误返回 **HTTP 200 + `success:false` + 空 data**；这三处只查 `resp.StatusCode != 200`，**不查 `result.Success`**（同文件 `GetTRC20Balance:288` 查了，属遗漏）。后果：GetBlockEvents→`(nil,nil)`→monitor `saveCheckpoint` 推进过该块→**充值永久跳过不入账**；HasOutgoingTRC20Transfer→`(false,nil)`→CheckDoubleSpend 通过→**双花**。
- **修法**：三处 unmarshal 后补 `if !result.Success { return ..., fmt.Errorf("trongrid: success=false code=%d msg=%s", result.Code, result.Message) }`，镜像 `GetTRC20Balance:288`。
- **对抗证明**：mock TronGrid 返回 `{200, success:false}` → 旧代码返回 `(nil,nil)` 被当正常（RED，断言应 error）；新代码返回 error（GREEN）。**补回归测试 `{200, success:false}` 必 error**。

### A2. TRON-SECURITY-1（P1——提现冷签 MITM）
- **位置**：`backend/internal/sweep/tron_client.go:34`（`insecure.NewCredentials()` 明文 gRPC 到公网）+ xpubFingerprint 未绑 tx 内容。
- **根因**：构建 raw tx（`BuildTRC20Transfer`/`BuildDelegateResource`，冷签输入）走明文无认证 gRPC；MITM 可改返回的 raw tx → 冷签签出转账到攻击者地址，xpub 指纹不绑 tx 内容无法发现。
- **修法**（二选一，推荐后者）：① 公网 endpoint 用 TLS gRPC；② **冷签侧从解析的 RawData 重算 txid 并显示，与返回的 raw tx 不匹配则拒签**（不信任传输层）。优先做 ②（端到端防篡改）。
- **对抗证明**：mock 篡改返回的 raw tx → 旧代码照签（RED）；新代码重算 txid 不匹配拒签（GREEN）。

### A3. LIVE-PRICE-4（P1——实盘无法开仓，用户原始报告的真正根因）
- 详见 `builder-handoff-live-price-2026-08-13.md` 任务 5。核心：删 `defaultQuoteSymbols()`，`postConnectSetup` 订阅改 `FetchAllSymbols` 过滤 + 逐 symbol `Subscribe`（非原子），MT4+MT5。对抗证明 mock FetchAllSymbols 含 XAUUSDm 不含 XAUJPYm → 旧整批失败 RED / 新逐 symbol GREEN。

### A4. MDGATEWAY-1（P1——FetchAccountInfo 吞错误判只读）
- **位置**：`backend/internal/mdgateway/adapter/mt4/connection_account.go:32` + `mt5/connection_account.go:31`（FetchAccountInfo）。
- **根因**：`AccountSummary` 不查 `resp.GetError()` → mtapi app error 时 result nil → 落入"investor 只读"分支 → **会话过期/权限错被当成只读账户 → 实盘被错误阻断 + 误诊**。
- **修法**：加 `if resp.GetError() != nil && resp.GetError().GetCode() != 0 { return error }`（参照 connection_extra.go:32）。
- **对抗证明**：mock AccountSummary 返回 gRPC nil-error + body Error → 旧代码误判 investor（RED）；新代码返回 error（GREEN）。

### A5. MDGATEWAY-3（P1——订单事件流无死检测）
- **位置**：`backend/internal/mdgateway/adapter/mt4/order_stream.go:71` + `mt5/order_stream.go:71`（orderUpdateRecvLoop）。
- **根因**：`stream.Recv()` 长循环无 no-data 超时（quote/profit 流有 90s，这条对称兄弟流漏了）→ broker 停推时**订单成交/SL/TP/平仓事件静默停止到达**。
- **修法**：仿 `quotes.go:164` profit/quote 流的 `select{recv/err/case <-time.After(quoteTimeout)}` 模式。
- **对抗证明**：mock Recv 阻塞 → 超时触发重连（GREEN）；删超时分支 → 永不重连（RED）。
- **附带 MDGATEWAY-2/4**（同区域顺手）：HealthCheck 吞错（connection_account.go:79/113）补 GetError；hub 订单 goroutine（orders.go:310/341）加外层重连+超时。

### A6. BROKER-SEARCH-1（P1——mtapi host 硬编码+配置未接线）
- **位置**：`brokersearch/search.go:55,58`（host 硬编码默认）+ `cmd/server/handlers.go:67` + `cmd/server/pipeline.go:70`（两处 `brokersearch.New("","")`）。
- **根因**：mtapi 搜索 host 硬编码为默认值，且两个调用点传空串 → "可配置"构造器从未接 env/config，host 漂移零覆盖（LIVE-PRICE-4 硬编码复刻）。
- **修法**：两个 `New` 调用传 `cfg.MT4SearchGateway`/`MT5SearchGateway`（config 已有字段则接线，无则加）；default-fallback 可保留但 call-site 空串去掉。

---

## Part B — 补网（防再犯，止血后立即做）

### B1. live 行情端到端冒烟测试（最高性价比的网）
**这一个测试能一次性抓出 LIVE-PRICE-1/2/3/4 全部**。spec：
1. 连真实 broker 账户（用现有配置）。
2. 触发订阅（gateway connect）。
3. **断言 `md_e2e_latency_seconds_count > 0`**（HandleTick 真在触发 = 报价真在流）——这是 LIVE-PRICE 的决定性指标。
4. 断言 NATS `md.tick.>`/`md.bar.>` 有消息（非 40 条死水）。
5. 断言 Active Runs 价格列收到 bid/ask（WatchAllTicks + tick 桥）。
6. 启一个 paper 策略（BTCUSDm），断言它收到 bar/tick、产生信号、能下 paper 单。
7. **断言无 `subscribe symbols rejected by mtapi` 日志**（订阅真成功）。
- 落为可重复脚本（参考既有 `/tmp/smoke_*` 模式，但持久化进 repo `scripts/smoke/`），CI 可触发或上线前手跑。

### B2. 资金边界端到端冒烟（TRON 充值/提现）
- TronGrid `{200, success:false}` 必 fail（验证 A1）。
- 冷签 raw tx 篡改必拒签（验证 A2）。
- 充值入账端到端（块事件 → 入账）、提现端到端（构建→冷签→广播→确认）。

### B3. 外部边界 PR checklist（规则 enforcement）
外部边界 PR 必过：
- [ ] 所有外部响应查**业务级 error**（HTTP 状态码 + body error + RPC `.GetError()`），不只 transport error。
- [ ] 所有流式/长连/轮询外部有 **no-data 超时/死检测 + 重连**。
- [ ] 无硬编码外部可变数据（host/symbol/地址/费率/模型名）——有权威查询则查、否则 config。
- [ ] 资金/门控路径 **fail-closed**（错误默认拒绝，不放行）。

---

## 红队自审（动手前自查）
- [ ] A1/A4 的 `.GetError()` / `result.Success` 检查必须 fail-closed（error → 返回 error，绝不降级返回空/nil/false）。
- [ ] A2 冷签防篡改必须在**冷签侧**重算（不信任 sweep 返回的 raw tx），否则 MITM 仍在。
- [ ] A3/A5 的流超时阈值可注入测试（勿真等 90s）。
- [ ] B1 冒烟用真账户，但只下 **paper 单**（不碰真钱）。

## 回填纪律
每项 `🟦open → ✅done`（标日期+commit+对抗证明结果+部署后实测），不删条目不改审计方事实，不自行宣告完成（审计方验收）。**A1/A2 部署后必须回填真实资金路径实测**（充值入账/冷签防篡改），不能只靠单测。
