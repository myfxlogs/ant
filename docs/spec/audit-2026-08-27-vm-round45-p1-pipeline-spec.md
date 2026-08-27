# 审计 spec：VM round 4-5 遗留 + P1 资金/数据/报价管线（2026-08-27 Devin CLI）

> **审计方**：Devin CLI（独立审计，read-only on source code）
> **范围**：① VM round 4-5 遗留 5 ID 代码现状复审；② P1 资金/数据/报价管线 13 条目代码现状审计
> **方法**：registry 条目对照 + 代码文件存在性 + 关键函数 grep + 根因代码行验证

---

## 1. VM round 4-5 遗留 5 ID — 全部 FAIL（代码已被回滚）

### 结论

**5 个 ID 全部 FAIL（🟦open，需从零重做）**：round 5 修复代码已被 D-REVERT-SCOPE-DRIFT-001 回滚删除，当前代码库中不存在任何 round 5 描述的修复代码、测试文件或对抗证明。`vm-adversarial-proofs.md` 文件头部标记为 `SUPERSEDED`（2026-08-25），仅作历史记录。

### 验证证据

| 预期文件/函数 | 存在性 |
|---|---|
| `vm_live_validators.go` | ❌ 不存在 |
| `live_context_enums.go` | ❌ 不存在 |
| `compile_interp_decls.go` | ❌ 不存在 |
| `vm_trade_context6_round5_test.go` | ❌ 不存在 |
| `vm_api_truth3_round5_test.go` | ❌ 不存在 |
| `accountIsInvestorLookup` | ❌ 零匹配 |
| `validateLiveFinancialFields` | ❌ 零匹配 |
| `isInputDeclaration` | ❌ 零匹配 |
| `injectServerSideAccountTruth` | ❌ 零匹配 |
| `vm-adversarial-proofs.md` | SUPERSEDED 标记 |

### 5 ID 逐项

| ID | 状态 | 阻断原因 |
|---|---|---|
| VM-TRADE-CONTEXT-6 | 🟦open | round 5 代码被回滚，live context 注入 + validators 全部不存在 |
| VM-API-TRUTH-3 | 🟦open | round 5 代码被回滚，lookup fail-closed + investor gating 不存在 |
| VM-CACHE-INTEGRITY-5 | 🟦open | round 5 代码被回滚，coverage restore + payload limit 不存在 |
| VM-COMPILER-SEMANTICS-4 | 🟦open | round 5 代码被回滚，HasError guard + 结构化 input/extern 不存在 |
| VM-TEST-EVIDENCE-4 | 🟦open | proof 文档 SUPERSEDED，依赖的 4 ID 测试不存在 |

### 最小返工动作

按 D-VM-LIVE-001 设计规范从零重新实现 5 个 ID，实现完成后重新生成对抗证明文档，重新进行独立审计验收。

---

## 2. P1 资金/数据/报价管线 — 5 still-open + 8 fixed-acceptable

### 结论

13 个条目中：5 个 still-open（根因仍存在，需施工），8 个 fixed-acceptable（已修复并验收），0 个 false-alarm。

### still-open 条目（需施工）

| ID | 优先级 | 根因 | 验证证据 | 最小返工动作 |
|---|---|---|---|---|
| TRON-SECURITY-1 | P0 资金 | 提现冷签 MITM | `tron_client.go:34` 仍 `insecure.NewCredentials()`，xpubFingerprint 未绑 tx | 改用 TLS 凭据 + xpubFingerprint 绑 tx 内容 |
| DATA-TRUTH-1 | P0 数据 | orders 表与 broker 双向不一致 | `reconciliation.go:191` ghost 仍仅 `log.Warn` 从不补写 | 需架构决策：orphan 加 24h 下界 + ghost 是否自动补写 + reconciliation 定位 |
| QUOTE-RECONNECT-LOOP | P1 报价 | 报价流自持重连循环 | `connection.go:186-224` Disconnect 仍取消全 session 订阅 | Disconnect 改定向重连或级联退避 |
| BROKER-SEARCH-1 | P1 报价 | mtapi broker 搜索 host 硬编码 | `search.go:55,58` host 硬编码 + `handlers.go:67`/`pipeline.go:71` 传 `New("","")` | 接线 config 字段到两个 `New` 调用 |
| TRUST-1 | P2 业务 | Demo/真实账户战绩混展 | 前端战绩展示无 account_type 区分 | 需业务决策：real-only/标注/允许 |

### fixed-acceptable 条目（已修复）

| ID | 验证结论 | registry 状态 |
|---|---|---|
| DATA-TRUTH-2/2b | ✅ 已修复（DATA-TRUTH-2b-FIX） | ⚠️ 原条目仍 🟦open，修复条目 ✅done — 状态漂移需合并 |
| LIVE-PRICE-1 | ✅ 已修复 | ✅done |
| LIVE-PRICE-3 | ✅ 已修复（对抗测试需补强） | ✅done |
| LIVE-PRICE-4 | ✅ 已修复 | ✅done |
| STREAM-FREEZE-1 | ✅ 已修复 | ✅done |
| STREAM-KEEPALIVE | ✅ 已修复 | ✅done |
| LIVE-SSE-HEARTBEAT | ✅ 已修复 | ✅done |
| DEPLOY-LIVE-8 | ✅ 已修复 | ✅done |

### 状态漂移修正

DATA-TRUTH-2/2b 原条目（registry ~55/57）仍标记 🟦open，但修复条目 DATA-TRUTH-2b-FIX 已 ✅done（2026-08-20）。原条目应追加 ✅done 标注指向修复条目，消除状态漂移。

---

## 3. 施工优先级排序

### P0 — 资金安全（立即施工）
1. **TRON-SECURITY-1** — 提现冷签 MITM，改用 TLS 凭据
2. **DATA-TRUTH-1** — orders 表 reconciliation 不收敛（需架构决策）

### P1 — 功能阻断（高优先级）
3. **QUOTE-RECONNECT-LOOP** — 报价流自持重连循环
4. **BROKER-SEARCH-1** — broker 搜索 host 硬编码
5. **VM round 4-5 × 5 ID** — 从零重做（按 D-VM-LIVE-001）

### P2 — 业务风险（中优先级）
6. **TRUST-1** — Demo/真实账户战绩混展（需业务决策）

### 状态修正（无需施工）
7. **DATA-TRUTH-2/2b** — 合并状态漂移标注
