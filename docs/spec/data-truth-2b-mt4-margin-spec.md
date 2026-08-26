# Spec：DATA-TRUTH-2b MT4 margin 从 AccountSummary 补齐（P1）

> **状态**：✅done（修复 + 对抗证明在 revert 后完整存活，2026-08-26 Devin 审计方验收）
> **registry 条目**：`DATA-TRUTH-2b`（`docs/audits/tech-debt-registry.md:57`）、`DATA-TRUTH-2b-FIX`（:58，✅done 2026-08-20）
> **优先级**：P1（保证金风控全线失明——MT4 账户 margin 恒为 0）

## 1. 问题陈述

MT4 profit 帧的 `margin:0 / free_margin==equity / margin_level:0` ——字段根本未填（`free_margin` 恰等于 equity 是铁证，不是"真的为 0"）。

**危害**：
- `pipeline.go:224` 爆仓检查门槛是 `pr.MarginLevel.GreaterThan(0)`，MT4 永远进不了 margin call 分支
- `marginCallThresholds`/`broker_stop_out_pct` 形同虚设
- MT4 账户的保证金风控全线失明

**已纠正的错判**："MT4 平台不返回 margin"是错的——`account_balance_history` 里 MT4 账户 deed9f79 有 718 行 margin>0（最高 1867.23），证明 MT4 的 `AccountSummary` 含 margin，只是 profit 流不含。错的是数据源归属，不是平台能力。

## 2. revert 后验证结果（2026-08-26 Devin 审计方）

### 2.1 修复代码完整存活

- `profit.go:34` `refreshAccountSummary` — 每帧刷新 AccountSummary
- `profit.go:110-114` — stream active 时发初始快照
- `profit.go:200` `fetchAndPublish` — 调用 AccountSummary 取权威值
- `profit.go:228` `margin := decimal.NewFromFloat(s.GetMargin())` — 从 AccountSummary 取 margin
- `pipeline.go:224` `pr.MarginLevel.GreaterThan(0)` 爆仓检查 — 逻辑正确

### 2.2 对抗证明测试完整存活且通过

```
TestDATATRUTH2_MarginFromAccountSummary — PASS (0.00s)
TestDATATRUTH2_AccountSummaryFailureRejectsFinancialSnapshot — PASS (0.20s)
```

测试位于 `mt4_test.go:740-760`，验证 margin 从 AccountSummary 取得（非 0），且 AccountSummary 失败时拒绝发布 financial snapshot（fail-closed）。

### 2.3 结论

DATA-TRUTH-2b-FIX 的修复和对抗证明在 revert `830b2c79` 后完整存活。registry 标记 ✅done 是准确的。**本 spec 验收通过，无需重新施工。**

## 3. 设计决策（保留供参考）

### D1：margin/free_margin/margin_level 取自 AccountSummary
MT4 的 margin 取自 `AccountSummary`（`s.GetMargin()`），不从 profit 帧取。用户原则"服务器有的数据一律以服务器为准、禁止本地计算"。

### D2：profit 帧的未填值禁止覆盖权威值
profit 帧的 margin=0 是"未填"而非"真的为 0"，禁止用这些值覆盖从 `AccountSummary` 取得的权威值。这是 bug 本体。

### D3：禁止按 contractSize 本地反推 margin
margin 是 broker 权威值，本地反推会引入计算误差和假设偏差。

### D4：push-first 豁免
mtapi MT4 无 margin 推送源，命中"无 push 能力"豁免条款，允许 pull 模式（每帧 `AccountSummary`）。

## 4. 不做

- 不按 contractSize 本地反推 margin（D3）
- 不改 MT5 的 margin 取值（MT5 profit 帧已含 margin）
- 不改 `pipeline.go` 的爆仓检查逻辑（只需 margin 值正确，逻辑本身没问题）
- 不部署（D-COMMIT-SCOPE-001 部署闸仍有效）
